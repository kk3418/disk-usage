package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"disk-usage/internal/dossier"
	"disk-usage/internal/report"
	"disk-usage/internal/volume"
	"disk-usage/internal/walk"
)

func main() {
	var (
		output  string
		root    string
		minFlag string
		workers int
	)
	flag.StringVar(&output, "output", "./disk-scan", "掃描結果目錄（scan.json 與 scan.html）")
	flag.StringVar(&root, "root", "", "掃描根目錄，預設為 /System/Volumes/Data")
	flag.StringVar(&minFlag, "min-size", "50MB", "列入結果的目錄配置大小門檻")
	flag.IntVar(&workers, "workers", 0, "平行讀目錄的 worker 數，預設為 min(8, CPU 數)")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "用法: diskusage [--output 目錄] [--root 路徑] [--min-size 50MB] [--workers N]\n\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	minSize, err := parseSize(minFlag)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if root == "" {
		root = defaultRoot()
	}
	root, err = filepath.Abs(root)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		fmt.Fprintf(os.Stderr, "無法掃描 %s\n", root)
		os.Exit(1)
	}
	if workers < 1 {
		workers = runtime.NumCPU()
		if workers > 8 {
			workers = 8
		}
		if workers < 1 {
			workers = 1
		}
	}

	vol := volume.Load()
	prog := &walk.Progress{}
	done := make(chan struct{})
	go printProgress(prog, done)

	res, err := walk.Walk(walk.NewOSReader(), root, workers, prog)
	close(done)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	scan := dossier.Build(res, vol, prog.Files.Load(), prog.Bytes.Load(), dossier.DefaultOptions(minSize))
	if err := os.MkdirAll(output, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	jsonPath := filepath.Join(output, "scan.json")
	htmlPath := filepath.Join(output, "scan.html")
	if err := writeJSON(jsonPath, scan); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	htmlFile, err := os.Create(htmlPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	err = report.WriteHTML(htmlFile, scan)
	closeErr := htmlFile.Close()
	if err != nil || closeErr != nil {
		if err == nil {
			err = closeErr
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	report.WriteTerminal(os.Stdout, scan, htmlPath)
	fmt.Printf("scan.json：%s\n", jsonPath)
}

func defaultRoot() string {
	const data = "/System/Volumes/Data"
	if info, err := os.Stat(data); err == nil && info.IsDir() {
		return data
	}
	return "/"
}

func printProgress(prog *walk.Progress, done <-chan struct{}) {
	tick := time.NewTicker(time.Second)
	defer tick.Stop()
	printed := false
	for {
		select {
		case <-tick.C:
			fmt.Fprintf(os.Stderr, "\r已掃描 %d 個檔案，%s", prog.Files.Load(), report.Format(prog.Bytes.Load()))
			printed = true
		case <-done:
			if printed {
				fmt.Fprintln(os.Stderr)
			}
			return
		}
	}
}

func writeJSON(path string, scan dossier.Scan) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	return enc.Encode(scan)
}

func parseSize(s string) (int64, error) {
	s = strings.TrimSpace(strings.ToUpper(strings.ReplaceAll(s, " ", "")))
	mult := int64(1)
	switch {
	case strings.HasSuffix(s, "TB"):
		mult = 1 << 40
		s = strings.TrimSuffix(s, "TB")
	case strings.HasSuffix(s, "GB"):
		mult = 1 << 30
		s = strings.TrimSuffix(s, "GB")
	case strings.HasSuffix(s, "MB"):
		mult = 1 << 20
		s = strings.TrimSuffix(s, "MB")
	case strings.HasSuffix(s, "KB"):
		mult = 1 << 10
		s = strings.TrimSuffix(s, "KB")
	case strings.HasSuffix(s, "B"):
		s = strings.TrimSuffix(s, "B")
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil || n < 0 {
		return 0, fmt.Errorf("無法解析大小 %q", s)
	}
	return n * mult, nil
}
