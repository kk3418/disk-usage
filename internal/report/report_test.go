package report

import (
	"bytes"
	"strings"
	"testing"

	"computer-usage/internal/dossier"
)

func TestFormat(t *testing.T) {
	if got := Format(50 << 20); got != "50.0 MB" {
		t.Fatal(got)
	}
	if got := Format(4096); got != "4.0 KB" {
		t.Fatal(got)
	}
}

func TestHTMLHasNoVerdict(t *testing.T) {
	var buf bytes.Buffer
	scan := dossier.Scan{
		Root: "/data",
		Directories: []dossier.Directory{{
			Path:        "/data/Library/Caches",
			Allocated:   4096,
			NameSignals: []string{"caches"},
			Children:    nil,
			Extensions:  nil,
		}},
		Unreadable: []dossier.Unreadable{},
	}
	if err := WriteHTML(&buf, scan); err != nil {
		t.Fatal(err)
	}
	body := buf.String()
	if !strings.Contains(body, "/data/Library/Caches") {
		t.Fatal(body)
	}
	for _, word := range []string{"通常可丟", "應保留", "要你判斷"} {
		if strings.Contains(body, word) {
			t.Fatalf("report contains verdict %s", word)
		}
	}
}
