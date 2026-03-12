package pgini

import "path/filepath"

// testDir is a non-existent directory under /tmp used as a base for all
// IniFile paths in this test file, avoiding collisions with real files.
var testDir = filepath.Join("/tmp", "175d1c03/b11c/42f1/b571/1737d4fcd594")

// testPath returns a full path under testDir for the given filename.
func testPath(name string) string {
	return filepath.Join(testDir, name)
}
