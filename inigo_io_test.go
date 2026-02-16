package inigo

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRequireFile(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "test.conf")
	os.WriteFile(filePath, []byte("data"), 0o644)

	t.Run("existing file", func(t *testing.T) {
		if err := RequireFile(filePath, "config"); err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})
	t.Run("missing file", func(t *testing.T) {
		err := RequireFile(filepath.Join(dir, "nope.conf"), "config")
		if err == nil {
			t.Fatal("expected error for missing file")
		}
	})
	t.Run("directory instead of file", func(t *testing.T) {
		err := RequireFile(dir, "config")
		if err == nil {
			t.Fatal("expected error when path is a directory")
		}
	})
}

func TestRequireDir(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "test.conf")
	os.WriteFile(filePath, []byte("data"), 0o644)

	t.Run("existing directory", func(t *testing.T) {
		if err := RequireDir(dir, "conf dir"); err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})
	t.Run("missing directory", func(t *testing.T) {
		err := RequireDir(filepath.Join(dir, "nope"), "conf dir")
		if err == nil {
			t.Fatal("expected error for missing directory")
		}
	})
	t.Run("file instead of directory", func(t *testing.T) {
		err := RequireDir(filePath, "conf dir")
		if err == nil {
			t.Fatal("expected error when path is a file")
		}
	})
}
