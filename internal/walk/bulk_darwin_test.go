//go:build darwin

package walk

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOSReaderTempDir(t *testing.T) {
	dir := t.TempDir()
	body := []byte("hello")
	file := filepath.Join(dir, "hello.txt")
	if err := os.WriteFile(file, body, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Link(file, filepath.Join(dir, "hello-link")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("hello.txt", filepath.Join(dir, "hello.lnk")); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "sub", "note.pdf"), []byte("pdf"), 0o644); err != nil {
		t.Fatal(err)
	}

	reader := NewOSReader()
	fsid, err := reader.FSID(dir)
	if err != nil {
		t.Fatal(err)
	}
	res, err := Walk(reader, dir, 2, nil)
	if err != nil {
		t.Fatal(err)
	}
	root := res.Dirs[dir]
	if root == nil {
		t.Fatal("missing root")
	}
	var sawFile, sawLink, sawSym, sawSub bool
	ents, err := reader.Read(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range ents {
		if e.FSID != fsid {
			t.Fatalf("%s fsid %d != root %d", e.Name, e.FSID, fsid)
		}
		switch e.Name {
		case "hello.txt", "hello-link":
			if e.Allocated == 0 || e.Allocated%512 != 0 {
				t.Fatalf("%s allocated %d", e.Name, e.Allocated)
			}
			if e.LinkCount != 2 {
				t.Fatalf("%s links %d", e.Name, e.LinkCount)
			}
			if e.Name == "hello.txt" {
				sawFile = true
			} else {
				sawLink = true
			}
		case "hello.lnk":
			sawSym = true
			if !e.IsSymlink {
				t.Fatal("symlink not marked")
			}
		case "sub":
			sawSub = true
			if !e.IsDir {
				t.Fatal("sub is not a directory")
			}
		}
	}
	if !sawFile || !sawLink || !sawSym || !sawSub {
		t.Fatalf("entries = %+v", ents)
	}
	if root.DirectAllocated == 0 {
		t.Fatal("expected allocated bytes")
	}
	// Two hard links share one allocation.
	var one int64
	for _, e := range ents {
		if e.Name == "hello.txt" {
			one = e.Allocated
		}
	}
	sub := res.Dirs[filepath.Join(dir, "sub")]
	if sub == nil || sub.Markers.Documents == 0 {
		t.Fatalf("sub = %+v", sub)
	}
	if root.DirectAllocated != one {
		t.Fatalf("hardlink allocated %d, want %d (symlink and names only once)", root.DirectAllocated, one)
	}
}
