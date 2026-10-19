package walk

import "computer-usage/internal/facts"

// Entry is one directory entry returned by a reader.
type Entry struct {
	Name      string
	IsDir     bool
	IsSymlink bool
	Allocated int64
	FileID    uint64
	FSID      uint64
	LinkCount uint32
	Err       error
}

// Unreadable is a path the walk could not list.
type Unreadable struct {
	Path  string
	Error string
}

// Dir is the direct contents of one directory, before subtree rollup.
type Dir struct {
	Path            string
	DirectAllocated int64
	DirectFiles     int64
	ChildDirs       []string
	TopFiles        []facts.Child
	Ext             map[string]facts.ExtStat
	Markers         facts.Markers
	NameSignals     []string
}

// Result is a finished walk. Sizes are allocated bytes, with hard links
// counted once. Directories on another filesystem are not entered.
type Result struct {
	Root       string
	RootFSID   uint64
	Dirs       map[string]*Dir
	Unreadable []Unreadable
}

// DirReader lists one directory and reports the filesystem id of a path.
type DirReader interface {
	Read(path string) ([]Entry, error)
	FSID(path string) (uint64, error)
}
