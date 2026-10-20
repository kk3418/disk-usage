package walk

import (
	"fmt"
	"path/filepath"
	"sync"
	"sync/atomic"

	"disk-usage/internal/facts"
)

const topFiles = 30

// Progress counts files and allocated bytes while a walk is running.
type Progress struct {
	Files atomic.Int64
	Bytes atomic.Int64
	Dirs  atomic.Int64
}

type dirResult struct {
	path    string
	entries []Entry
	err     error
}

type linkKey struct {
	fsid   uint64
	fileID uint64
}

// Walk lists root with a bounded worker pool. Allocated sizes of hard-linked
// regular files are counted once. Symlinks are not followed. A directory
// whose filesystem id differs from root is not entered.
func Walk(reader DirReader, root string, workers int, prog *Progress) (*Result, error) {
	if workers < 1 {
		workers = 1
	}
	root = filepath.Clean(root)
	fsid, err := reader.FSID(root)
	if err != nil {
		return nil, fmt.Errorf("讀取檔案系統編號 %s: %w", root, err)
	}
	res := &Result{
		Root:     root,
		RootFSID: fsid,
		Dirs:     map[string]*Dir{},
	}

	jobs := make(chan string, workers*2)
	results := make(chan dirResult, workers*2)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for path := range jobs {
				entries, err := reader.Read(path)
				results <- dirResult{path: path, entries: entries, err: err}
			}
		}()
	}

	seenLinks := map[linkKey]struct{}{}
	seenDirID := map[linkKey]struct{}{}
	enqueued := map[string]bool{root: true}
	pending := []string{root}
	inflight := 0

	handle := func(dr dirResult) {
		inflight--
		if prog != nil {
			prog.Dirs.Add(1)
		}
		if dr.err != nil {
			res.Unreadable = append(res.Unreadable, Unreadable{Path: dr.path, Error: dr.err.Error()})
			res.Dirs[dr.path] = &Dir{
				Path:        dr.path,
				Ext:         map[string]facts.ExtStat{},
				NameSignals: facts.NameSignals(filepath.Base(dr.path)),
			}
			return
		}
		d := &Dir{
			Path:        dr.path,
			Ext:         map[string]facts.ExtStat{},
			NameSignals: facts.NameSignals(filepath.Base(dr.path)),
		}
		for _, e := range dr.entries {
			if e.Name == "" || e.Name == "." || e.Name == ".." || stringsHasSlash(e.Name) {
				continue
			}
			child := filepath.Join(dr.path, e.Name)
			if e.Err != nil {
				res.Unreadable = append(res.Unreadable, Unreadable{Path: child, Error: e.Err.Error()})
				continue
			}
			if prog != nil && !(e.IsDir && !e.IsSymlink) {
				prog.Files.Add(1)
			}
			if e.IsDir && !e.IsSymlink {
				if e.FSID != 0 && res.RootFSID != 0 && e.FSID != res.RootFSID {
					continue
				}
				if e.FileID != 0 {
					id := linkKey{e.FSID, e.FileID}
					if _, ok := seenDirID[id]; ok {
						continue
					}
					seenDirID[id] = struct{}{}
				}
				if enqueued[child] {
					continue
				}
				enqueued[child] = true
				pending = append(pending, child)
				d.ChildDirs = append(d.ChildDirs, child)
				continue
			}
			size := e.Allocated
			if size < 0 {
				size = 0
			}
			if !e.IsSymlink && e.LinkCount > 1 && e.FileID != 0 {
				id := linkKey{e.FSID, e.FileID}
				if _, ok := seenLinks[id]; ok {
					d.DirectFiles++
					continue
				}
				seenLinks[id] = struct{}{}
			}
			d.DirectFiles++
			d.DirectAllocated += size
			if prog != nil {
				prog.Bytes.Add(size)
			}
			if !e.IsSymlink {
				facts.Absorb(e.Name, size, &d.Markers, d.Ext)
			}
			d.TopFiles = keepTop(d.TopFiles, facts.Child{Name: e.Name, Allocated: size, IsDir: false}, topFiles)
		}
		res.Dirs[dr.path] = d
	}

	// Sending jobs and receiving results in one select keeps the pool from
	// deadlocking when the job buffer fills up.
	for inflight > 0 || len(pending) > 0 {
		for len(pending) > 0 {
			select {
			case jobs <- pending[0]:
				pending = pending[1:]
				inflight++
			case dr := <-results:
				handle(dr)
			}
		}
		if inflight == 0 {
			break
		}
		handle(<-results)
	}
	close(jobs)
	wg.Wait()
	return res, nil
}

func keepTop(files []facts.Child, c facts.Child, n int) []facts.Child {
	if n < 1 {
		return files
	}
	if len(files) == n && c.Allocated <= files[len(files)-1].Allocated {
		return files
	}
	files = append(files, c)
	for i := len(files) - 1; i > 0; i-- {
		if files[i].Allocated <= files[i-1].Allocated {
			break
		}
		files[i], files[i-1] = files[i-1], files[i]
	}
	if len(files) > n {
		files = files[:n]
	}
	return files
}

func stringsHasSlash(name string) bool {
	for i := 0; i < len(name); i++ {
		if name[i] == '/' {
			return true
		}
	}
	return false
}
