package walk

import (
	"path/filepath"
	"testing"
)

type fakeReader struct {
	fsid  uint64
	dirs  map[string][]Entry
	other uint64
}

func (f fakeReader) FSID(string) (uint64, error) { return f.fsid, nil }

func (f fakeReader) Read(path string) ([]Entry, error) {
	return f.dirs[path], nil
}

func TestWalkHardlinkAndFilesystemBoundary(t *testing.T) {
	root := filepath.Join(string(filepath.Separator), "data")
	other := filepath.Join(root, "other")
	sub := filepath.Join(root, "sub")
	r := fakeReader{
		fsid:  1,
		other: 2,
		dirs: map[string][]Entry{
			root: {
				{Name: "a.txt", Allocated: 100, FileID: 7, FSID: 1, LinkCount: 2},
				{Name: "b.txt", Allocated: 100, FileID: 7, FSID: 1, LinkCount: 2},
				{Name: "sub", IsDir: true, FileID: 3, FSID: 1},
				{Name: "other", IsDir: true, FileID: 9, FSID: 2},
				{Name: "link", IsSymlink: true, Allocated: 0, FSID: 1},
			},
			sub: {
				{Name: "c.pdf", Allocated: 50, FileID: 4, FSID: 1, LinkCount: 1},
			},
			other: {
				{Name: "secret", Allocated: 9999, FileID: 8, FSID: 2, LinkCount: 1},
			},
		},
	}
	res, err := Walk(r, root, 2, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := res.Dirs[other]; ok {
		t.Fatal("crossed onto another filesystem")
	}
	got := res.Dirs[root]
	if got.DirectAllocated != 100 {
		t.Fatalf("allocated %d, want 100", got.DirectAllocated)
	}
	if got.DirectFiles != 3 { // a, b, symlink; b adds no bytes
		t.Fatalf("files %d", got.DirectFiles)
	}
	if len(got.ChildDirs) != 1 || got.ChildDirs[0] != sub {
		t.Fatalf("children %v", got.ChildDirs)
	}
	if res.Dirs[sub].Markers.Documents != 50 {
		t.Fatalf("markers %+v", res.Dirs[sub].Markers)
	}
	if res.Dirs[sub].Ext[".pdf"].Allocated != 50 {
		t.Fatalf("ext %+v", res.Dirs[sub].Ext)
	}
}
