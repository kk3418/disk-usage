# disk-usage

[繁體中文](README.md)

Scan a macOS disk and write directory allocated sizes plus content markers to `scan.json` and `scan.html`. The scan records facts. To list the files and folders to keep before reinstalling, have the repo’s `disk-analysis` skill read `scan.json`. This repo does not delete or move files.

## Requirements

- macOS (`diskutil` for capacity, `getattrlistbulk` for directory listing)
- Go 1.26, or the version in `go.mod`

## Scan

```bash
go build -o diskusage ./cmd/diskusage
./diskusage --output ./disk-scan
```

The default root is `/System/Volumes/Data`. On a machine without that path, the scan starts at `/`. Results go to `./disk-scan/scan.json` and `./disk-scan/scan.html`. When the walk finishes, the terminal prints a capacity summary and the larger directories. During the scan, stderr updates once a second with the file count and allocated size so far.

To scan only your home directory and leave smaller directories out of the report:

```bash
./diskusage --root "$HOME" --output ./home-scan --min-size 100MB --workers 4
```

## Options

| Flag | Default | Meaning |
| --- | --- | --- |
| `--output` | `./disk-scan` | Directory for `scan.json` and `scan.html` |
| `--root` | `/System/Volumes/Data`, otherwise `/` | Scan root |
| `--min-size` | `50MB` | Minimum allocated size for a directory to be kept. Units: `B`, `KB`, `MB`, `GB`, `TB` |
| `--workers` | `min(8, CPU count)` | Parallel directory readers |

One fifth of `--min-size` is the hot-zone threshold. Directories in that zone are kept first. At most 2000 directories are written.

## What the result contains

Sizes are allocated bytes. A hard-linked regular file is counted once. Symlink targets are left unfollowed. A directory whose filesystem id differs from the scan root is left unentered, so a scan of the Data volume stays on that volume. The operating system’s own usage is a single figure from `diskutil`; those files are not classified one by one.

Each kept directory includes its larger direct children, extension totals, and allocated bytes for these markers: documents, images, video, databases, source markers (such as `go.mod` and `package.json`), and installers (`.dmg`, `.pkg`, `.iso`). Paths that could not be listed are in `unreadable`.

Open `scan.html` in a browser. `scan.json` is the same set of facts for a person or an agent.

## Full Disk Access

Desktop, Documents, Downloads, Mail, and Messages need Full Disk Access for the app that actually runs the scan (Terminal, iTerm, or Cursor’s terminal): System Settings → Privacy & Security → Full Disk Access. Paths without permission show up in `unreadable`; the rest of the walk still finishes. If the terminal summary mentions those locations, run the scan again from an authorized terminal.

## Analysis

After a scan, ask the Cursor agent to follow the `disk-analysis` skill on `./disk-scan/scan.json`. The skill writes Traditional Chinese `analysis.md` beside the scan. It lists files and folders to copy off before a reinstall, app data to confirm, and caches or dependencies that a reinstall will recreate. If the scan file is already there and you did not ask for a rescan, analysis can start from that file.

## Tests

```bash
go test ./...
```

The `diskusage` binary and `disk-scan/` are in `.gitignore`. Leave scan output on the machine that produced it.
