package report

import (
	"embed"
	"html/template"
	"io"
	"strings"

	"disk-usage/internal/dossier"
)

//go:embed scan.html
var files embed.FS

var page = template.Must(template.New("scan.html").Funcs(template.FuncMap{
	"fmt":     Format,
	"join":    func(s []string) string { return strings.Join(s, "、") },
	"markers": markerText,
}).ParseFS(files, "scan.html"))

// WriteHTML writes a self-contained factual report.
func WriteHTML(w io.Writer, scan dossier.Scan) error {
	return page.Execute(w, scan)
}
