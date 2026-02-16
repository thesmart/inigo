package inigo

import (
	"fmt"
	"os"
)

// RequireFile verifies that a file exists at path and is readable.
// The name parameter is used in error messages to describe what the path represents.
func RequireFile(path, name string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("%s not found: %s", name, path)
	}
	if info.IsDir() {
		return fmt.Errorf("%s is a directory, expected a file: %s", name, path)
	}
	return nil
}

// RequireDir verifies that a directory exists at path.
// The name parameter is used in error messages to describe what the path represents.
func RequireDir(path, name string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("%s not found: %s", name, path)
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is a file, expected a directory: %s", name, path)
	}
	return nil
}
