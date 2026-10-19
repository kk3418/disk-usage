// Package dossier rolls a walk up into a bounded JSON document.
package dossier

import (
	"path/filepath"
	"sort"

	"computer-usage/internal/facts"
	"computer-usage/internal/volume"
	"computer-usage/internal/walk"
)

const (
	defaultMaxDirs       = 2000
	defaultMaxChildren   = 30
	defaultMaxExt        = 15
	defaultMaxUnreadable = 200
)

// Options controls which directories are kept in the document.
type Options struct {
	MinSize       int64
	HotMinSize    int64
	MaxDirs       int
	MaxChildren   int
	MaxExt        int
	MaxUnreadable int
}

// DefaultOptions uses a hot-zone threshold of one fifth of minSize.
func DefaultOptions(minSize int64) Options {
	hot := minSize / 5
	if hot < 1 {
		hot = 1
	}
	if minSize > 0 && hot > minSize {
		hot = minSize
	}
	return Options{
		MinSize:       minSize,
		HotMinSize:    hot,
		MaxDirs:       defaultMaxDirs,
		MaxChildren:   defaultMaxChildren,
		MaxExt:        defaultMaxExt,
		MaxUnreadable: defaultMaxUnreadable,
	}
}

// Scan is the document an agent reads. It contains sizes and evidence, not verdicts.
type Scan struct {
	Root               string       `json:"root"`
	MinSize            int64        `json:"minSize"`
	HotMinSize         int64        `json:"hotMinSize"`
	Volume             volume.Info  `json:"volume"`
	ScannedFiles       int64        `json:"scannedFiles"`
	ScannedAllocated   int64        `json:"scannedAllocated"`
	Unreadable         []Unreadable `json:"unreadable"`
	UnreadableCount    int          `json:"unreadableCount"`
	FullDiskAccessHint bool         `json:"fullDiskAccessHint"`
	Directories        []Directory  `json:"directories"`
	Truncated          bool         `json:"truncated"`
}

// Unreadable is a path that could not be listed.
type Unreadable struct {
	Path  string `json:"path"`
	Error string `json:"error"`
}

// Directory is one kept directory and its rolled-up facts.
type Directory struct {
	Path        string            `json:"path"`
	Allocated   int64             `json:"allocated"`
	Files       int64             `json:"files"`
	Dirs        int64             `json:"dirs"`
	Children    []facts.Child     `json:"children"`
	Extensions  []facts.Extension `json:"extensions"`
	Markers     facts.Markers     `json:"markers"`
	NameSignals []string          `json:"nameSignals"`
}

type rolled struct {
	path      string
	allocated int64
	files     int64
	dirs      int64
	ext       map[string]facts.ExtStat
	markers   facts.Markers
	signals   []string
	children  []facts.Child
}

// Build rolls up subtree sizes and keeps a bounded set of directories.
func Build(res *walk.Result, vol volume.Info, scannedFiles, scannedAllocated int64, opt Options) Scan {
	if opt.MaxDirs < 1 {
		opt.MaxDirs = defaultMaxDirs
	}
	if opt.MaxChildren < 1 {
		opt.MaxChildren = defaultMaxChildren
	}
	if opt.MaxExt < 1 {
		opt.MaxExt = defaultMaxExt
	}
	if opt.MaxUnreadable < 1 {
		opt.MaxUnreadable = defaultMaxUnreadable
	}
	if opt.HotMinSize < 1 {
		opt.HotMinSize = 1
	}

	nodes := rollup(res)
	hot, large := candidates(nodes, res.Root, opt.MinSize, opt.HotMinSize)
	bySize := func(list []cand) {
		sort.Slice(list, func(i, j int) bool {
			if list[i].allocated == list[j].allocated {
				return list[i].path < list[j].path
			}
			return list[i].allocated > list[j].allocated
		})
	}
	bySize(hot)
	bySize(large)

	chosen := make([]string, 0, opt.MaxDirs)
	seen := map[string]bool{}
	truncated := false
	add := func(list []cand) {
		for _, c := range list {
			if seen[c.path] {
				continue
			}
			if len(chosen) >= opt.MaxDirs {
				truncated = true
				return
			}
			seen[c.path] = true
			chosen = append(chosen, c.path)
		}
	}
	add(hot)
	add(large)

	dirs := make([]Directory, 0, len(chosen))
	for _, p := range chosen {
		n := nodes[p]
		dirs = append(dirs, Directory{
			Path:        n.path,
			Allocated:   n.allocated,
			Files:       n.files,
			Dirs:        n.dirs,
			Children:    capChildren(n.children, opt.MaxChildren),
			Extensions:  capExt(n.ext, opt.MaxExt),
			Markers:     n.markers,
			NameSignals: nonNil(n.signals),
		})
	}
	sort.Slice(dirs, func(i, j int) bool {
		if dirs[i].Allocated == dirs[j].Allocated {
			return dirs[i].Path < dirs[j].Path
		}
		return dirs[i].Allocated > dirs[j].Allocated
	})

	unread := make([]Unreadable, 0, len(res.Unreadable))
	hint := false
	for _, u := range res.Unreadable {
		if fdaPath(u.Path) {
			hint = true
		}
		unread = append(unread, Unreadable{Path: u.Path, Error: u.Error})
	}
	count := len(unread)
	if len(unread) > opt.MaxUnreadable {
		unread = unread[:opt.MaxUnreadable]
	}
	if unread == nil {
		unread = []Unreadable{}
	}

	return Scan{
		Root:               res.Root,
		MinSize:            opt.MinSize,
		HotMinSize:         opt.HotMinSize,
		Volume:             vol,
		ScannedFiles:       scannedFiles,
		ScannedAllocated:   scannedAllocated,
		Unreadable:         unread,
		UnreadableCount:    count,
		FullDiskAccessHint: hint,
		Directories:        dirs,
		Truncated:          truncated,
	}
}

type cand struct {
	path      string
	allocated int64
}

func candidates(nodes map[string]*rolled, root string, minSize, hotMin int64) (hot, large []cand) {
	for p, n := range nodes {
		if p == root || n.allocated >= minSize {
			large = append(large, cand{p, n.allocated})
		}
		parent := filepath.Dir(p)
		if parent != p && isHot(parent) && n.allocated >= hotMin {
			hot = append(hot, cand{p, n.allocated})
		}
	}
	return hot, large
}

func rollup(res *walk.Result) map[string]*rolled {
	memo := map[string]*rolled{}
	stack := map[string]bool{}
	var walkPath func(string) *rolled
	walkPath = func(path string) *rolled {
		if r, ok := memo[path]; ok {
			return r
		}
		if stack[path] {
			return &rolled{path: path, ext: map[string]facts.ExtStat{}}
		}
		stack[path] = true
		d := res.Dirs[path]
		r := &rolled{path: path, ext: map[string]facts.ExtStat{}}
		if d != nil {
			r.allocated = d.DirectAllocated
			r.files = d.DirectFiles
			r.markers = d.Markers
			r.signals = d.NameSignals
			for k, v := range d.Ext {
				r.ext[k] = v
			}
			for _, f := range d.TopFiles {
				r.children = append(r.children, f)
			}
			for _, child := range d.ChildDirs {
				cr := walkPath(child)
				r.allocated += cr.allocated
				r.files += cr.files
				r.dirs += 1 + cr.dirs
				r.markers.Add(cr.markers)
				for k, v := range cr.ext {
					st := r.ext[k]
					st.Count += v.Count
					st.Allocated += v.Allocated
					r.ext[k] = st
				}
				r.children = append(r.children, facts.Child{
					Name:      filepath.Base(child),
					Allocated: cr.allocated,
					IsDir:     true,
				})
			}
		}
		delete(stack, path)
		memo[path] = r
		return r
	}
	if res.Dirs[res.Root] != nil || res.Root != "" {
		walkPath(res.Root)
	}
	for p := range res.Dirs {
		walkPath(p)
	}
	return memo
}

func isHot(path string) bool {
	switch filepath.Base(path) {
	case "Application Support", "Containers", "Group Containers", "Caches", "Logs", "Downloads":
		return true
	case "Library":
		return filepath.Base(filepath.Dir(filepath.Dir(path))) == "Users"
	default:
		return false
	}
}

func capChildren(in []facts.Child, n int) []facts.Child {
	sort.Slice(in, func(i, j int) bool {
		if in[i].Allocated == in[j].Allocated {
			return in[i].Name < in[j].Name
		}
		return in[i].Allocated > in[j].Allocated
	})
	if len(in) > n {
		in = in[:n]
	}
	if in == nil {
		return []facts.Child{}
	}
	return in
}

func capExt(ext map[string]facts.ExtStat, n int) []facts.Extension {
	out := make([]facts.Extension, 0, len(ext))
	for k, v := range ext {
		out = append(out, facts.Extension{Ext: k, Count: v.Count, Allocated: v.Allocated})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Allocated == out[j].Allocated {
			return out[i].Ext < out[j].Ext
		}
		return out[i].Allocated > out[j].Allocated
	})
	if len(out) > n {
		out = out[:n]
	}
	return out
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

func fdaPath(path string) bool {
	for _, part := range []string{"/Desktop/", "/Documents/", "/Library/Mail/", "/Library/Messages/"} {
		if containsPart(path, part) {
			return true
		}
	}
	return false
}

func containsPart(path, part string) bool {
	if len(path) >= len(part) {
		for i := 0; i+len(part) <= len(path); i++ {
			if path[i:i+len(part)] == part {
				return true
			}
		}
	}
	trim := part[:len(part)-1]
	return len(path) >= len(trim) && path[len(path)-len(trim):] == trim
}
