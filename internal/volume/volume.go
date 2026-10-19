// Package volume reads whole-disk capacity figures from diskutil.
package volume

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// Info is the container summary shown beside a file walk.
type Info struct {
	ContainerSize int64  `json:"containerSize"`
	Free          int64  `json:"free"`
	Purgeable     int64  `json:"purgeable"`
	SystemVolume  int64  `json:"systemVolume"`
	SystemNote    string `json:"systemNote"`
	Error         string `json:"error,omitempty"`
}

// Load asks diskutil for the container and the sealed system volume.
// The file walk does not enter the system volume; SystemVolume is that
// volume's in-use size reported as one figure.
func Load() Info {
	info := Info{SystemNote: "作業系統本體，未逐檔分類"}
	root, err := plist("/")
	if err != nil {
		info.Error = err.Error()
		return info
	}
	info.ContainerSize = pick(root, "APFSContainerSize", "TotalSize", "Size")
	info.Free = pick(root, "APFSContainerFree", "FreeSpace")
	info.Purgeable = pick(root, "PurgeableSpace", "APFSContainerPurgeable", "Purgeable")
	info.SystemVolume = pick(root, "CapacityInUse")
	return info
}

func plist(path string) (map[string]int64, error) {
	cmd := exec.Command("diskutil", "info", "-plist", path)
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok && len(ee.Stderr) > 0 {
			msg := strings.Join(strings.Fields(string(ee.Stderr)), " ")
			return nil, fmt.Errorf("diskutil info %s: %s", path, msg)
		}
		return nil, fmt.Errorf("diskutil info %s: %w", path, err)
	}
	vals, err := ParsePlistIntegers(out)
	if err != nil {
		return nil, fmt.Errorf("解析 diskutil plist: %w", err)
	}
	return vals, nil
}

func pick(m map[string]int64, keys ...string) int64 {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			return v
		}
	}
	return 0
}

// ParsePlistIntegers reads integer values from an XML property list.
func ParsePlistIntegers(data []byte) (map[string]int64, error) {
	dec := xml.NewDecoder(bytes.NewReader(data))
	vals := map[string]int64{}
	var pending string
	for {
		tok, err := dec.Token()
		if err != nil {
			if err.Error() == "EOF" {
				return vals, nil
			}
			return nil, err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if t.Name.Local == "key" {
				var key string
				if err := dec.DecodeElement(&key, &t); err != nil {
					return nil, err
				}
				pending = key
				continue
			}
			if pending != "" && (t.Name.Local == "integer" || t.Name.Local == "real") {
				var raw string
				if err := dec.DecodeElement(&raw, &t); err != nil {
					return nil, err
				}
				n, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
				if err == nil {
					vals[pending] = n
				}
				pending = ""
			}
		}
	}
}
