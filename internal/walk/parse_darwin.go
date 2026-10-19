//go:build darwin

package walk

import (
	"encoding/binary"
	"fmt"

	"golang.org/x/sys/unix"
)

const (
	objDir = 2
	objLnk = 5
)

// ParseBulk decodes count getattrlistbulk records.
// macOS packs this attribute set tightly; the byte fixtures in parse_darwin_test.go lock that layout.
func ParseBulk(buf []byte, count int) ([]Entry, error) {
	out := make([]Entry, 0, count)
	off := 0
	for len(out) < count {
		if off+4 > len(buf) {
			return nil, fmt.Errorf("屬性紀錄在偏移 %d 被截斷", off)
		}
		recLen := int(binary.LittleEndian.Uint32(buf[off:]))
		if recLen < 4 || off+recLen > len(buf) {
			return nil, fmt.Errorf("屬性紀錄長度 %d 無效", recLen)
		}
		ent, err := parseRecord(buf[off : off+recLen])
		if err != nil {
			return nil, err
		}
		out = append(out, ent)
		off += recLen
	}
	return out, nil
}

func parseRecord(rec []byte) (Entry, error) {
	if len(rec) < 4+20 {
		return Entry{}, fmt.Errorf("屬性紀錄過短")
	}
	common := binary.LittleEndian.Uint32(rec[4:])
	file := binary.LittleEndian.Uint32(rec[16:])
	cursor := 24
	var ent Entry

	take := func(n int) ([]byte, error) {
		if cursor+n > len(rec) {
			return nil, fmt.Errorf("屬性欄位超出紀錄")
		}
		b := rec[cursor : cursor+n]
		cursor += n
		return b, nil
	}

	if common&unix.ATTR_CMN_NAME != 0 {
		ref, err := take(8)
		if err != nil {
			return Entry{}, err
		}
		rel := int32(binary.LittleEndian.Uint32(ref[0:4]))
		nlen := int(binary.LittleEndian.Uint32(ref[4:8]))
		start := (cursor - 8) + int(rel)
		if nlen < 1 || start < 0 || start+nlen > len(rec) {
			return Entry{}, fmt.Errorf("檔名參考無效")
		}
		name := rec[start : start+nlen]
		if name[len(name)-1] == 0 {
			name = name[:len(name)-1]
		}
		ent.Name = string(name)
	}
	if common&unix.ATTR_CMN_FSID != 0 {
		b, err := take(8)
		if err != nil {
			return Entry{}, err
		}
		ent.FSID = binary.LittleEndian.Uint64(b)
	}
	if common&unix.ATTR_CMN_OBJTYPE != 0 {
		b, err := take(4)
		if err != nil {
			return Entry{}, err
		}
		switch binary.LittleEndian.Uint32(b) {
		case objDir:
			ent.IsDir = true
		case objLnk:
			ent.IsSymlink = true
		}
	}
	if common&unix.ATTR_CMN_FILEID != 0 {
		b, err := take(8)
		if err != nil {
			return Entry{}, err
		}
		ent.FileID = binary.LittleEndian.Uint64(b)
	}
	if common&unix.ATTR_CMN_ERROR != 0 {
		b, err := take(4)
		if err != nil {
			return Entry{}, err
		}
		if code := binary.LittleEndian.Uint32(b); code != 0 {
			ent.Err = unix.Errno(code)
		}
	}
	if file&unix.ATTR_FILE_LINKCOUNT != 0 {
		b, err := take(4)
		if err != nil {
			return Entry{}, err
		}
		ent.LinkCount = binary.LittleEndian.Uint32(b)
	}
	if file&unix.ATTR_FILE_ALLOCSIZE != 0 {
		b, err := take(8)
		if err != nil {
			return Entry{}, err
		}
		ent.Allocated = int64(binary.LittleEndian.Uint64(b))
	}
	return ent, nil
}
