package report

import (
	"fmt"
	"io"
	"strings"

	"disk-usage/internal/dossier"
	"disk-usage/internal/facts"
)

// WriteTerminal prints a factual summary. It does not say what to keep or delete.
func WriteTerminal(w io.Writer, scan dossier.Scan, htmlPath string) {
	fmt.Fprintf(w, "掃描根目錄：%s\n", scan.Root)
	if scan.Volume.Error != "" {
		fmt.Fprintf(w, "磁碟容量：未能讀取 diskutil（%s）\n", scan.Volume.Error)
	} else {
		fmt.Fprintf(w, "容器總容量：%s\n", Format(scan.Volume.ContainerSize))
		fmt.Fprintf(w, "可用空間：%s\n", Format(scan.Volume.Free))
		fmt.Fprintf(w, "可清除空間：%s\n", Format(scan.Volume.Purgeable))
		fmt.Fprintf(w, "作業系統本體：%s（%s）\n", Format(scan.Volume.SystemVolume), scan.Volume.SystemNote)
	}
	fmt.Fprintf(w, "掃到的檔案數：%d\n", scan.ScannedFiles)
	fmt.Fprintf(w, "掃到的配置大小：%s\n", Format(scan.ScannedAllocated))
	fmt.Fprintf(w, "目錄筆數：%d", len(scan.Directories))
	if scan.Truncated {
		fmt.Fprintf(w, "（已達上限，較小的目錄沒有全部列入）")
	}
	fmt.Fprintln(w)
	if scan.UnreadableCount > 0 {
		fmt.Fprintf(w, "未能讀取：%d 個路徑\n", scan.UnreadableCount)
	}
	if scan.FullDiskAccessHint {
		fmt.Fprintln(w, "有些路徑在桌面、文件、郵件或訊息裡。若要掃到這些位置，請在已授予完整磁碟取用權限的終端機執行。")
	}
	fmt.Fprintln(w, "較大的目錄（只有大小，沒有去留結論）：")
	limit := len(scan.Directories)
	if limit > 20 {
		limit = 20
	}
	for _, d := range scan.Directories[:limit] {
		extra := ""
		if len(d.NameSignals) > 0 {
			extra = "  名稱訊號：" + strings.Join(d.NameSignals, "、")
		}
		fmt.Fprintf(w, "  %s  %s  檔案 %d  目錄 %d%s\n", Format(d.Allocated), d.Path, d.Files, d.Dirs, extra)
	}
	fmt.Fprintf(w, "\n事實報告：%s\n", htmlPath)
	fmt.Fprintln(w, "這份輸出沒有判斷哪些該留或可丟。請用 disk-analysis skill 讀 scan.json。")
}

func markerText(m facts.Markers) string {
	var parts []string
	if m.Documents > 0 {
		parts = append(parts, "文件 "+Format(m.Documents))
	}
	if m.Images > 0 {
		parts = append(parts, "圖片 "+Format(m.Images))
	}
	if m.Video > 0 {
		parts = append(parts, "影片 "+Format(m.Video))
	}
	if m.Databases > 0 {
		parts = append(parts, "資料庫 "+Format(m.Databases))
	}
	if m.SourceMarkers > 0 {
		parts = append(parts, "原始碼標記 "+Format(m.SourceMarkers))
	}
	if m.Installers > 0 {
		parts = append(parts, "安裝檔 "+Format(m.Installers))
	}
	if len(parts) == 0 {
		return "—"
	}
	return strings.Join(parts, "、")
}
