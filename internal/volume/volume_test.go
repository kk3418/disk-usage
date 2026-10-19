package volume

import "testing"

func TestParsePlistIntegers(t *testing.T) {
	const doc = `<?xml version="1.0" encoding="UTF-8"?>
<plist version="1.0"><dict>
<key>APFSContainerSize</key><integer>100</integer>
<key>APFSContainerFree</key><integer>40</integer>
<key>CapacityInUse</key><integer>12</integer>
<key>PurgeableSpace</key><integer>3</integer>
<key>VolumeName</key><string>Data</string>
</dict></plist>`
	got, err := ParsePlistIntegers([]byte(doc))
	if err != nil {
		t.Fatal(err)
	}
	if got["APFSContainerSize"] != 100 || got["APFSContainerFree"] != 40 || got["CapacityInUse"] != 12 || got["PurgeableSpace"] != 3 {
		t.Fatalf("%v", got)
	}
	if _, ok := got["VolumeName"]; ok {
		t.Fatal("string values are not integers")
	}
}
