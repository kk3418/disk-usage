//go:build !darwin

package walk

import "fmt"

// OSReader is unavailable off darwin.
type OSReader struct{}

// NewOSReader returns a reader that reports the unsupported platform.
func NewOSReader() DirReader { return OSReader{} }

func (OSReader) FSID(string) (uint64, error) {
	return 0, fmt.Errorf("getattrlistbulk 只支援 macOS")
}

func (OSReader) Read(string) ([]Entry, error) {
	return nil, fmt.Errorf("getattrlistbulk 只支援 macOS")
}
