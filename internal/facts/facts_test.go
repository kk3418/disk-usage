package facts

import "testing"

func TestAbsorb(t *testing.T) {
	var m Markers
	ext := map[string]ExtStat{}
	Absorb("notes.pdf", 10, &m, ext)
	Absorb("package.json", 4, &m, ext)
	Absorb("setup.dmg", 8, &m, ext)
	Absorb("data.sqlite", 16, &m, ext)
	if m.Documents != 10 || m.SourceMarkers != 4 || m.Installers != 8 || m.Databases != 16 {
		t.Fatalf("markers = %+v", m)
	}
	if ext[".pdf"].Allocated != 10 || ext[".json"].Count != 1 {
		t.Fatalf("ext = %+v", ext)
	}
}

func TestNameSignals(t *testing.T) {
	if got := NameSignals("Caches"); len(got) != 1 || got[0] != "caches" {
		t.Fatalf("Caches: %v", got)
	}
	if got := NameSignals("node_modules"); len(got) != 1 || got[0] != "node_modules" {
		t.Fatalf("node_modules: %v", got)
	}
	if got := NameSignals("App.xcodeproj"); len(got) != 1 || got[0] != "xcodeproj" {
		t.Fatalf("xcodeproj: %v", got)
	}
	if NameSignals("Photos") != nil {
		t.Fatal("Photos should have no name signal")
	}
}
