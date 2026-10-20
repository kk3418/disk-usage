# disk-usage

[English](README.en.md)

在 macOS 上掃描磁碟，把目錄的配置大小與內容標記寫成 `scan.json` 與 `scan.html`。掃描只記錄事實。要列出重灌前要保留的檔案與資料夾，請再用專案裡的 `disk-analysis` skill 讀 `scan.json`。這個 repo 不刪除、不移動檔案。

## 需要

- macOS（讀取容量用 `diskutil`，列目錄用 `getattrlistbulk`）
- Go 1.26 或 `go.mod` 所寫的版本

## 掃描

```bash
go build -o diskusage ./cmd/diskusage
./diskusage --output ./disk-scan
```

預設從 `/System/Volumes/Data` 開始。這台機器沒有該路徑時，改從 `/` 開始。結果寫入 `./disk-scan/scan.json` 與 `./disk-scan/scan.html`。結束時終端機會印出容量摘要，以及配置較大的目錄。掃描過程中，標準錯誤每秒更新已掃過的檔案數與配置大小。

只掃家目錄、並把較小的目錄留在結果外：

```bash
./diskusage --root "$HOME" --output ./home-scan --min-size 100MB --workers 4
```

## 選項

| 選項 | 預設 | 說明 |
| --- | --- | --- |
| `--output` | `./disk-scan` | 放置 `scan.json` 與 `scan.html` 的目錄 |
| `--root` | `/System/Volumes/Data`，否則 `/` | 掃描根目錄 |
| `--min-size` | `50MB` | 目錄配置大小達到這個值才列入結果。單位可用 `B`、`KB`、`MB`、`GB`、`TB` |
| `--workers` | `min(8, CPU 數)` | 平行讀取目錄的 worker 數 |

`--min-size` 的五分之一是熱區門檻。熱區裡的目錄會優先保留。列入的目錄最多 2000 筆。

## 結果裡有什麼

大小是配置大小（allocated）。硬連結的一般檔案只計一次。符號連結的目標不會被跟著走進去。檔案系統編號與掃描根目錄不同的目錄不會進入，所以從 Data 磁碟掃描時不會走進另一個 volume。作業系統本體的佔用來自 `diskutil` 的一個數字，程式不會逐檔分類。

每個列入的目錄帶有較大的直接子項目、副檔名統計，以及這些內容標記的配置大小：文件、圖片、影片、資料庫、原始碼標記（例如 `go.mod`、`package.json`）、安裝檔（`.dmg`、`.pkg`、`.iso`）。讀取失敗的路徑在 `unreadable`。

`scan.html` 用瀏覽器打開即可。`scan.json` 是給人或 agent 讀的同一份事實。

## 完整磁碟取用權限

桌面、文件、下載、郵件、訊息等位置，需要在「系統設定 → 隱私權與安全性 → 完整磁碟取用權限」授權給實際執行掃描的 App（Terminal、iTerm，或 Cursor 的終端機）。沒有權限的路徑會出現在 `unreadable`，其餘目錄仍會掃完。終端摘要看到相關提示時，用已授權的終端機再跑一次。

## 分析

掃描完成後，在 Cursor 裡請 agent 依 `disk-analysis` skill 處理 `./disk-scan/scan.json`。skill 會在同一目錄寫出繁體中文 `analysis.md`，列出重灌前要帶走的檔案與資料夾、要再確認的 App 資料，以及重灌後會重建、不必帶走的快取與依賴。檔案已在且沒有要求重掃時，直接分析即可。

## 測試

```bash
go test ./...
```

執行檔 `diskusage` 與 `disk-scan/` 已列入 `.gitignore`。掃描結果留在本機即可。
