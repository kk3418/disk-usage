//go:build darwin

package walk

import (
	"fmt"
	"runtime"
	"unsafe"

	"golang.org/x/sys/unix"
)

// OSReader lists directories with getattrlistbulk.
type OSReader struct{}

// NewOSReader returns the darwin directory reader.
func NewOSReader() DirReader { return OSReader{} }

func (OSReader) FSID(path string) (uint64, error) {
	al := unix.Attrlist{
		Bitmapcount: unix.ATTR_BIT_MAP_COUNT,
		Commonattr:  unix.ATTR_CMN_RETURNED_ATTRS | unix.ATTR_CMN_FSID,
	}
	buf := make([]byte, 128)
	p, err := unix.BytePtrFromString(path)
	if err != nil {
		return 0, err
	}
	_, _, errno := unix.Syscall6(
		unix.SYS_GETATTRLIST,
		uintptr(unsafe.Pointer(p)),
		uintptr(unsafe.Pointer(&al)),
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(len(buf)),
		uintptr(unix.FSOPT_NOFOLLOW),
		0,
	)
	runtime.KeepAlive(p)
	if errno != 0 {
		return 0, errno
	}
	ents, err := ParseBulk(buf, 1)
	if err != nil {
		return 0, err
	}
	if len(ents) != 1 {
		return 0, fmt.Errorf("沒有檔案系統編號")
	}
	return ents[0].FSID, nil
}

func (OSReader) Read(path string) ([]Entry, error) {
	fd, err := unix.Open(path, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, err
	}
	defer unix.Close(fd)

	al := unix.Attrlist{
		Bitmapcount: unix.ATTR_BIT_MAP_COUNT,
		Commonattr:  unix.ATTR_CMN_RETURNED_ATTRS | unix.ATTR_CMN_NAME | unix.ATTR_CMN_ERROR | unix.ATTR_CMN_OBJTYPE | unix.ATTR_CMN_FILEID | unix.ATTR_CMN_FSID,
		Fileattr:    unix.ATTR_FILE_LINKCOUNT | unix.ATTR_FILE_ALLOCSIZE,
	}
	buf := make([]byte, 256*1024)
	var all []Entry
	for {
		r1, _, errno := unix.Syscall6(
			unix.SYS_GETATTRLISTBULK,
			uintptr(fd),
			uintptr(unsafe.Pointer(&al)),
			uintptr(unsafe.Pointer(&buf[0])),
			uintptr(len(buf)),
			uintptr(unix.FSOPT_NOFOLLOW),
			0,
		)
		if errno == unix.EINTR {
			continue
		}
		if errno == unix.ERANGE && len(buf) < 4*1024*1024 {
			buf = make([]byte, len(buf)*2)
			continue
		}
		if errno != 0 {
			return nil, errno
		}
		n := int(r1)
		if n == 0 {
			return all, nil
		}
		ents, err := ParseBulk(buf, n)
		if err != nil {
			return nil, err
		}
		all = append(all, ents...)
	}
}
