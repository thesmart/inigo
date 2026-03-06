package ini

import (
	"os"
	"path/filepath"
	"testing"
)

func TestUnmarshalIniFileIntermediate_Basic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.conf")
	err := os.WriteFile(path, []byte("key = value\n"), 0o644)
	if err != nil {
		t.Fatal(err)
	}
	iniFile, err := unmarshalIniFileIntermediate(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if iniFile.defaultSection == nil {
		t.Fatal("expected default section")
	}
	p, ok := iniFile.defaultSection.params["key"]
	if !ok {
		t.Fatal("expected param 'key'")
	}
	if p.value != "value" {
		t.Errorf("expected value='value', got %q", p.value)
	}
}

func TestUnmarshalIniFileIntermediate_Include(t *testing.T) {
	dir := t.TempDir()

	// Main file
	mainPath := filepath.Join(dir, "main.conf")
	includePath := filepath.Join(dir, "included.conf")

	mainContent := "host = mainhost\ninclude '" + includePath + "'\n"
	includeContent := "port = 9999\n"

	os.WriteFile(mainPath, []byte(mainContent), 0o644)
	os.WriteFile(includePath, []byte(includeContent), 0o644)

	iniFile, err := unmarshalIniFileIntermediate(mainPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if p, ok := iniFile.defaultSection.params["host"]; !ok || p.value != "mainhost" {
		t.Errorf("expected host=mainhost")
	}
	if p, ok := iniFile.defaultSection.params["port"]; !ok || p.value != "9999" {
		t.Errorf("expected port=9999")
	}
}

func TestUnmarshalIniFileIntermediate_IncludeOverrides(t *testing.T) {
	dir := t.TempDir()

	mainPath := filepath.Join(dir, "main.conf")
	includePath := filepath.Join(dir, "override.conf")

	mainContent := "host = original\ninclude '" + includePath + "'\n"
	includeContent := "host = overridden\n"

	os.WriteFile(mainPath, []byte(mainContent), 0o644)
	os.WriteFile(includePath, []byte(includeContent), 0o644)

	iniFile, err := unmarshalIniFileIntermediate(mainPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	p, ok := iniFile.defaultSection.params["host"]
	if !ok || p.value != "overridden" {
		t.Errorf("expected host=overridden, got %q", p.value)
	}
}

func TestUnmarshalIniFileIntermediate_IncludeIfExists(t *testing.T) {
	dir := t.TempDir()
	mainPath := filepath.Join(dir, "main.conf")

	// include_if_exists for a file that doesn't exist should not error
	mainContent := "host = val\ninclude_if_exists '" + filepath.Join(dir, "nope.conf") + "'\n"
	os.WriteFile(mainPath, []byte(mainContent), 0o644)

	iniFile, err := unmarshalIniFileIntermediate(mainPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p, ok := iniFile.defaultSection.params["host"]; !ok || p.value != "val" {
		t.Error("expected host=val")
	}
}

func TestUnmarshalIniFileIntermediate_IncludeDir(t *testing.T) {
	dir := t.TempDir()
	confDir := filepath.Join(dir, "conf.d")
	os.MkdirAll(confDir, 0o755)

	// Create files in lexicographic order
	os.WriteFile(filepath.Join(confDir, "00base.conf"), []byte("host = base\n"), 0o644)
	os.WriteFile(filepath.Join(confDir, "10override.conf"), []byte("host = override\n"), 0o644)
	os.WriteFile(filepath.Join(confDir, ".hidden.conf"), []byte("host = hidden\n"), 0o644)
	os.WriteFile(filepath.Join(confDir, "readme.txt"), []byte("host = txt\n"), 0o644)

	mainPath := filepath.Join(dir, "main.conf")
	mainContent := "include_dir '" + confDir + "'\n"
	os.WriteFile(mainPath, []byte(mainContent), 0o644)

	iniFile, err := unmarshalIniFileIntermediate(mainPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Last file wins: 10override.conf
	p, ok := iniFile.defaultSection.params["host"]
	if !ok || p.value != "override" {
		t.Errorf("expected host=override (from 10override.conf), got %q", p.value)
	}
}

func TestUnmarshalIniFileIntermediate_CircularInclude(t *testing.T) {
	dir := t.TempDir()
	pathA := filepath.Join(dir, "a.conf")
	pathB := filepath.Join(dir, "b.conf")

	os.WriteFile(pathA, []byte("include '"+pathB+"'\n"), 0o644)
	os.WriteFile(pathB, []byte("include '"+pathA+"'\n"), 0o644)

	_, err := unmarshalIniFileIntermediate(pathA)
	if err == nil {
		t.Error("expected circular include error")
	}
}

func TestUnmarshalIniFileIntermediate_IncludeNotFound(t *testing.T) {
	dir := t.TempDir()
	mainPath := filepath.Join(dir, "main.conf")
	mainContent := "include '/nonexistent/file.conf'\n"
	os.WriteFile(mainPath, []byte(mainContent), 0o644)

	_, err := unmarshalIniFileIntermediate(mainPath)
	if err == nil {
		t.Error("expected error for missing include file")
	}
}

func TestUnmarshalIniFileIntermediate_IncludeDirectory(t *testing.T) {
	dir := t.TempDir()
	subDir := filepath.Join(dir, "subdir")
	os.MkdirAll(subDir, 0o755)

	mainPath := filepath.Join(dir, "main.conf")
	mainContent := "include '" + subDir + "'\n"
	os.WriteFile(mainPath, []byte(mainContent), 0o644)

	_, err := unmarshalIniFileIntermediate(mainPath)
	if err == nil {
		t.Error("expected error for including a directory with include (not include_dir)")
	}
}

func TestUnmarshalIniFileIntermediate_RelativeInclude(t *testing.T) {
	dir := t.TempDir()
	subDir := filepath.Join(dir, "sub")
	os.MkdirAll(subDir, 0o755)

	mainPath := filepath.Join(dir, "main.conf")
	includePath := filepath.Join(subDir, "child.conf")

	mainContent := "include 'sub/child.conf'\n"
	includeContent := "port = 1234\n"

	os.WriteFile(mainPath, []byte(mainContent), 0o644)
	os.WriteFile(includePath, []byte(includeContent), 0o644)

	iniFile, err := unmarshalIniFileIntermediate(mainPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	p, ok := iniFile.defaultSection.params["port"]
	if !ok || p.value != "1234" {
		t.Errorf("expected port=1234 from relative include")
	}
}

func TestUnmarshalIniFileIntermediate_Sections(t *testing.T) {
	contents := `
global = yes

[database]
host = dbhost

[app]
host = apphost
`
	iniFile, err := unmarshalIniStringIntermediate("test.conf", contents)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if p, ok := iniFile.defaultSection.params["global"]; !ok || p.value != "yes" {
		t.Error("expected global=yes in default section")
	}

	dbSec := iniFile.getSection("database")
	if dbSec == nil {
		t.Fatal("expected database section")
	}
	if p, ok := dbSec.params["host"]; !ok || p.value != "dbhost" {
		t.Error("expected host=dbhost in database section")
	}

	appSec := iniFile.getSection("app")
	if appSec == nil {
		t.Fatal("expected app section")
	}
	if p, ok := appSec.params["host"]; !ok || p.value != "apphost" {
		t.Error("expected host=apphost in app section")
	}
}

func TestGetSection_Nil(t *testing.T) {
	iniFile := &IniFile{
		sections:       make(map[string]*Section),
		defaultSection: &Section{params: make(map[string]*Param)},
	}
	if iniFile.getSection("nonexistent") != nil {
		t.Error("expected nil for nonexistent section")
	}
}

func TestUnmarshalIniFileIntermediate_NotFound(t *testing.T) {
	_, err := unmarshalIniFileIntermediate("/nonexistent/path.conf")
	if err == nil {
		t.Error("expected error")
	}
}

func TestUnmarshalIniFileIntermediate_IncludeWithLinesAfter(t *testing.T) {
	// Test that lines after an include directive are still processed
	dir := t.TempDir()
	mainPath := filepath.Join(dir, "main.conf")
	includePath := filepath.Join(dir, "inc.conf")

	mainContent := "a = 1\ninclude '" + includePath + "'\nc = 3\n"
	includeContent := "b = 2\n"

	os.WriteFile(mainPath, []byte(mainContent), 0o644)
	os.WriteFile(includePath, []byte(includeContent), 0o644)

	iniFile, err := unmarshalIniFileIntermediate(mainPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, key := range []string{"a", "b", "c"} {
		if _, ok := iniFile.defaultSection.params[key]; !ok {
			t.Errorf("expected param %q", key)
		}
	}
}

func TestResolvePath(t *testing.T) {
	if resolvePath("/abs/path", "/base") != "/abs/path" {
		t.Error("absolute path should not change")
	}
	result := resolvePath("relative/path", "/base")
	expected := filepath.Join("/base", "relative/path")
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestCursorString(t *testing.T) {
	c := Cursor{Line: 5, Offset: 10}
	if c.String() != "line 5, offset 10" {
		t.Errorf("unexpected: %s", c.String())
	}
}

func TestFileCursorString(t *testing.T) {
	fc := &FileCursor{Cursor: Cursor{Line: 3, Offset: 7}, Path: "foo.conf"}
	if fc.String() != "foo.conf:3:7" {
		t.Errorf("unexpected: %s", fc.String())
	}
}

func TestUnmarshalIniFileIntermediate_IncludeDirNotFound(t *testing.T) {
	dir := t.TempDir()
	mainPath := filepath.Join(dir, "main.conf")
	mainContent := "include_dir '/nonexistent/dir'\n"
	os.WriteFile(mainPath, []byte(mainContent), 0o644)

	_, err := unmarshalIniFileIntermediate(mainPath)
	if err == nil {
		t.Error("expected error for nonexistent include_dir")
	}
}

func TestUnmarshalIniFileIntermediate_IncludeIfExistsFound(t *testing.T) {
	dir := t.TempDir()
	mainPath := filepath.Join(dir, "main.conf")
	incPath := filepath.Join(dir, "extra.conf")

	os.WriteFile(incPath, []byte("port = 7777\n"), 0o644)
	mainContent := "include_if_exists '" + incPath + "'\n"
	os.WriteFile(mainPath, []byte(mainContent), 0o644)

	iniFile, err := unmarshalIniFileIntermediate(mainPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	p, ok := iniFile.defaultSection.params["port"]
	if !ok || p.value != "7777" {
		t.Error("expected port=7777 from include_if_exists")
	}
}

func TestMarshalIniFile_Error(t *testing.T) {
	cfg := &BasicConfig{Host: "localhost"}
	// Write to invalid path
	err := MarshalIniFile("/nonexistent/dir/file.conf", cfg)
	if err == nil {
		t.Error("expected error for invalid path")
	}
}

func TestUnmarshalIniStringIntermediate_DuplicateSection(t *testing.T) {
	contents := `
[section]
a = 1

[section]
b = 2
`
	iniFile, err := unmarshalIniStringIntermediate("test.conf", contents)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	sec := iniFile.getSection("section")
	if sec == nil {
		t.Fatal("expected section")
	}
	// Both params should be in the same section
	if _, ok := sec.params["a"]; !ok {
		t.Error("expected param a")
	}
	if _, ok := sec.params["b"]; !ok {
		t.Error("expected param b")
	}
}
