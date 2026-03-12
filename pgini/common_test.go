package pgini

import "path/filepath"

// nonExistingDir is a non-existent directory under /tmp used as a base for all
// IniFile paths in this test file, avoiding collisions with real files.
var nonExistingDir = filepath.Join("/tmp", "175d1c03/b11c/42f1/b571/1737d4fcd594")

// nonExistingPath returns a full path under testDir for the given filename.
func nonExistingPath(name string) string {
	return filepath.Join(nonExistingDir, name)
}
