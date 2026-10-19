// Package facts records what a scan saw in a directory. These fields are
// evidence for an agent; they are not keep-or-delete verdicts.
package facts

import (
	"path/filepath"
	"strings"
)

// Markers are allocated-byte totals for recognizable file kinds.
type Markers struct {
	Documents     int64 `json:"documents"`
	Images        int64 `json:"images"`
	Video         int64 `json:"video"`
	Databases     int64 `json:"databases"`
	SourceMarkers int64 `json:"sourceMarkers"`
	Installers    int64 `json:"installers"`
}

// ExtStat is one extension's direct or rolled-up total.
type ExtStat struct {
	Count     int64
	Allocated int64
}

// Child is a direct child file or directory.
type Child struct {
	Name      string `json:"name"`
	Allocated int64  `json:"allocated"`
	IsDir     bool   `json:"isDir"`
}

// Extension is the JSON form of an extension total.
type Extension struct {
	Ext       string `json:"ext"`
	Count     int64  `json:"count"`
	Allocated int64  `json:"allocated"`
}

// Absorb adds one non-symlink file to the extension histogram and markers.
func Absorb(name string, size int64, m *Markers, ext map[string]ExtStat) {
	if size < 0 {
		size = 0
	}
	key := strings.ToLower(filepath.Ext(name))
	st := ext[key]
	st.Count++
	st.Allocated += size
	ext[key] = st

	base := strings.ToLower(filepath.Base(name))
	switch base {
	case "package.json", "go.mod", "cargo.toml", "pyproject.toml", "gemfile", "package.swift", "composer.json":
		m.SourceMarkers += size
	}
	switch strings.ToLower(filepath.Ext(name)) {
	case ".pdf", ".doc", ".docx", ".docm", ".xls", ".xlsx", ".xlsm", ".ppt", ".pptx", ".pptm",
		".pages", ".numbers", ".key", ".txt", ".md", ".rtf", ".odt", ".ods", ".odp", ".csv":
		m.Documents += size
	case ".jpg", ".jpeg", ".png", ".gif", ".heic", ".heif", ".tif", ".tiff", ".webp", ".bmp",
		".raw", ".dng", ".psd", ".ai":
		m.Images += size
	case ".mov", ".mp4", ".m4v", ".mkv", ".avi", ".mpg", ".mpeg":
		m.Video += size
	case ".sqlite", ".sqlite3", ".db", ".sqlitedb":
		m.Databases += size
	case ".dmg", ".pkg", ".iso":
		m.Installers += size
	}
}

// Add merges rolled-up marker totals.
func (m *Markers) Add(o Markers) {
	m.Documents += o.Documents
	m.Images += o.Images
	m.Video += o.Video
	m.Databases += o.Databases
	m.SourceMarkers += o.SourceMarkers
	m.Installers += o.Installers
}

// NameSignals reports structural name evidence for a single path component.
// The values are observations, not verdicts.
func NameSignals(name string) []string {
	switch strings.ToLower(name) {
	case "caches", "cache", "gpucache", "code cache", "shadercache":
		return []string{"caches"}
	case "logs", "log":
		return []string{"logs"}
	case "node_modules":
		return []string{"node_modules"}
	case "deriveddata":
		return []string{"deriveddata"}
	case "trash", ".trash", ".trashes":
		return []string{"trash"}
	case "tmp", "temp", "temporary":
		return []string{"tmp"}
	case "crashreporter", "diagnosticreports":
		return []string{"crash"}
	case "coresimulator":
		return []string{"simulator"}
	case "pods":
		return []string{"pods"}
	case ".venv", "venv", "virtualenv":
		return []string{"venv"}
	case "build", "target":
		return []string{"build"}
	}
	lower := strings.ToLower(name)
	if strings.HasSuffix(lower, ".xcodeproj") || strings.HasSuffix(lower, ".xcworkspace") {
		return []string{"xcodeproj"}
	}
	return nil
}
