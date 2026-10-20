package dossier

import (
	"path/filepath"
	"testing"

	"disk-usage/internal/facts"
	"disk-usage/internal/volume"
	"disk-usage/internal/walk"
)

func TestBuildRollupHotZoneAndCap(t *testing.T) {
	me := filepath.Join(string(filepath.Separator), "Users", "me")
	lib := filepath.Join(me, "Library")
	caches := filepath.Join(lib, "Caches")
	support := filepath.Join(lib, "Application Support")
	app := filepath.Join(support, "App")
	cache := filepath.Join(app, "Cache")
	proj := filepath.Join(me, "proj")
	nm := filepath.Join(proj, "node_modules")
	src := filepath.Join(proj, "src")
	const mb = int64(1 << 20)

	dirs := map[string]*walk.Dir{
		me:      {Path: me, ChildDirs: []string{lib, proj}, Ext: map[string]facts.ExtStat{}},
		lib:     {Path: lib, ChildDirs: []string{caches, support}, Ext: map[string]facts.ExtStat{}},
		caches:  {Path: caches, DirectAllocated: 60 * mb, DirectFiles: 4, NameSignals: []string{"caches"}, Ext: map[string]facts.ExtStat{".tmp": {Count: 4, Allocated: 60 * mb}}},
		support: {Path: support, ChildDirs: []string{app}, Ext: map[string]facts.ExtStat{}},
		app:     {Path: app, DirectAllocated: 20 * mb, DirectFiles: 1, ChildDirs: []string{cache}, Markers: facts.Markers{Databases: 20 * mb}, TopFiles: []facts.Child{{Name: "data.sqlite", Allocated: 20 * mb}}, Ext: map[string]facts.ExtStat{".sqlite": {Count: 1, Allocated: 20 * mb}}},
		cache:   {Path: cache, DirectAllocated: 5 * mb, DirectFiles: 1, NameSignals: []string{"caches"}, Ext: map[string]facts.ExtStat{}},
		proj:    {Path: proj, ChildDirs: []string{nm, src}, Ext: map[string]facts.ExtStat{}},
		nm:      {Path: nm, DirectAllocated: 80 * mb, DirectFiles: 10, NameSignals: []string{"node_modules"}, Ext: map[string]facts.ExtStat{}},
		src:     {Path: src, DirectAllocated: 1 * mb, DirectFiles: 1, Markers: facts.Markers{SourceMarkers: 100}, Ext: map[string]facts.ExtStat{}},
	}
	for i := 0; i < 6; i++ {
		p := filepath.Join(proj, "big"+string(rune('a'+i)))
		dirs[p] = &walk.Dir{Path: p, DirectAllocated: int64(70-i) * mb, DirectFiles: 1, Ext: map[string]facts.ExtStat{}}
		dirs[proj].ChildDirs = append(dirs[proj].ChildDirs, p)
	}
	res := &walk.Result{Root: me, Dirs: dirs, Unreadable: []walk.Unreadable{{Path: filepath.Join(me, "Desktop", "x"), Error: "permission denied"}}}
	opt := DefaultOptions(50 * mb)
	opt.MaxDirs = 8
	scan := Build(res, volume.Info{}, 20, 200*mb, opt)

	if !scan.FullDiskAccessHint {
		t.Fatal("expected full disk access hint")
	}
	byPath := map[string]Directory{}
	for _, d := range scan.Directories {
		byPath[d.Path] = d
	}
	if _, ok := byPath[cache]; ok {
		t.Fatal("5MB cache dir is below the hot threshold and should be omitted as its own row")
	}
	gotApp, ok := byPath[app]
	if !ok {
		t.Fatal("Application Support child should be kept below the normal threshold")
	}
	if gotApp.Allocated != 25*mb || gotApp.Markers.Databases != 20*mb {
		t.Fatalf("app rollup %+v", gotApp)
	}
	var sawCache bool
	for _, c := range gotApp.Children {
		if c.Name == "Cache" && c.IsDir && c.Allocated == 5*mb {
			sawCache = true
		}
	}
	if !sawCache {
		t.Fatalf("app children %+v", gotApp.Children)
	}
	if byPath[lib].Allocated != 85*mb {
		t.Fatalf("library %d", byPath[lib].Allocated)
	}
	if byPath[nm].NameSignals[0] != "node_modules" {
		t.Fatalf("signals %+v", byPath[nm].NameSignals)
	}
	if !scan.Truncated || len(scan.Directories) != 8 {
		t.Fatalf("truncated=%v dirs=%d", scan.Truncated, len(scan.Directories))
	}
	exts := map[string]facts.ExtStat{}
	for i := 0; i < 20; i++ {
		exts[string(rune('a'+i))] = facts.ExtStat{Count: 1, Allocated: int64(i + 1)}
	}
	capped := capExt(exts, 15)
	if len(capped) != 15 || capped[0].Allocated != 20 {
		t.Fatalf("ext cap %+v", capped)
	}
}
