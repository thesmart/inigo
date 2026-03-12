package pgini

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeTemp creates a file in dir with the given name and content, returning
// the absolute path.
func writeTemp(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return p
}

// ---------------------------------------------------------------------------
// NewRootCursor
// ---------------------------------------------------------------------------

func TestNewRootCursor(t *testing.T) {
	dir := t.TempDir()
	p := writeTemp(t, dir, "root.conf", "[db]\nhost = localhost\n")

	rc, err := NewRootCursor(p)
	if err != nil {
		t.Fatalf("NewRootCursor: unexpected error: %v", err)
	}
	if rc.File == nil {
		t.Fatal("File should not be nil")
	}
	if rc.File.Path != p {
		t.Errorf("File.Path = %q, want %q", rc.File.Path, p)
	}
	if rc.current == nil {
		t.Fatal("current should not be nil")
	}
	if rc.current.Path != p {
		t.Errorf("current.Path = %q, want %q", rc.current.Path, p)
	}
	// The root file should already be in the visited set.
	abs, _ := filepath.Abs(p)
	if !rc.visited[abs] {
		t.Error("root file should be in visited set")
	}
}

func TestNewRootCursor_FileNotFound(t *testing.T) {
	_, err := NewRootCursor(testPath("file.conf"))
	if err == nil {
		t.Error("NewRootCursor should return error for non-existent file")
	}
}

func TestNewRootCursor_InvalidPath(t *testing.T) {
	// NewIniFile rejects a path whose base name is not a valid filename.
	// Use an empty path to trigger the filepath.Abs or os.ReadFile error.
	_, err := NewRootCursor("")
	if err == nil {
		t.Error("NewRootCursor should return error for empty path")
	}
}

// ---------------------------------------------------------------------------
// RootCursor.AddInclude
// ---------------------------------------------------------------------------

func TestRootCursor_AddInclude_NilCurrent(t *testing.T) {
	dir := t.TempDir()
	p := writeTemp(t, dir, "root.conf", "key = val\n")
	rc, err := NewRootCursor(p)
	if err != nil {
		t.Fatal(err)
	}
	rc.current = nil

	err = rc.AddInclude("other.conf")
	if err == nil {
		t.Error("AddInclude should error when current is nil")
	}
}

func TestRootCursor_AddInclude_RelativePath(t *testing.T) {
	dir := t.TempDir()
	writeTemp(t, dir, "root.conf", "key = val\n")
	child := writeTemp(t, dir, "sub/child.conf", "child_key = yes\n")

	rc, err := NewRootCursor(filepath.Join(dir, "root.conf"))
	if err != nil {
		t.Fatal(err)
	}

	// Relative path resolved against root.conf's directory.
	err = rc.AddInclude("sub/child.conf")
	if err != nil {
		t.Fatalf("AddInclude relative: %v", err)
	}
	// Verify it was pushed to the stack.
	fc := rc.NextInclude()
	if fc == nil {
		t.Fatal("NextInclude should return the added file")
	}
	abs, _ := filepath.Abs(child)
	if fc.Path != abs {
		t.Errorf("Path = %q, want %q", fc.Path, abs)
	}
}

func TestRootCursor_AddInclude_AbsolutePath(t *testing.T) {
	dir := t.TempDir()
	writeTemp(t, dir, "root.conf", "key = val\n")
	absChild := writeTemp(t, dir, "child.conf", "child_key = yes\n")

	rc, err := NewRootCursor(filepath.Join(dir, "root.conf"))
	if err != nil {
		t.Fatal(err)
	}

	err = rc.AddInclude(absChild)
	if err != nil {
		t.Fatalf("AddInclude absolute: %v", err)
	}
	fc := rc.NextInclude()
	if fc == nil {
		t.Fatal("NextInclude should return the added file")
	}
	if fc.Path != absChild {
		t.Errorf("Path = %q, want %q", fc.Path, absChild)
	}
}

func TestRootCursor_AddInclude_Circular(t *testing.T) {
	dir := t.TempDir()
	p := writeTemp(t, dir, "root.conf", "key = val\n")

	rc, err := NewRootCursor(p)
	if err != nil {
		t.Fatal(err)
	}

	// The root file is already visited; adding it again should error.
	err = rc.AddInclude(p)
	if err == nil {
		t.Error("AddInclude should detect circular include")
	}
	if err != nil && !strings.Contains(err.Error(), "circular include detected") {
		t.Errorf("error should mention circular include, got: %v", err)
	}
}

func TestRootCursor_AddInclude_FileNotFound(t *testing.T) {
	dir := t.TempDir()
	p := writeTemp(t, dir, "root.conf", "key = val\n")

	rc, err := NewRootCursor(p)
	if err != nil {
		t.Fatal(err)
	}

	err = rc.AddInclude(testPath("missing.conf"))
	if err == nil {
		t.Error("AddInclude should error for non-existent file")
	}
}

// ---------------------------------------------------------------------------
// RootCursor.NextInclude
// ---------------------------------------------------------------------------

func TestRootCursor_NextInclude_EmptyStack(t *testing.T) {
	dir := t.TempDir()
	p := writeTemp(t, dir, "root.conf", "key = val\n")

	rc, err := NewRootCursor(p)
	if err != nil {
		t.Fatal(err)
	}

	// The constructor pushes root onto the stack; pop it first.
	rc.NextInclude()

	// Now the stack is empty.
	fc := rc.NextInclude()
	if fc != nil {
		t.Errorf("NextInclude on empty stack should return nil, got %v", fc)
	}
}

func TestRootCursor_NextInclude_LIFO(t *testing.T) {
	dir := t.TempDir()
	p := writeTemp(t, dir, "root.conf", "key = val\n")
	a := writeTemp(t, dir, "a.conf", "a = 1\n")
	b := writeTemp(t, dir, "b.conf", "b = 2\n")

	rc, err := NewRootCursor(p)
	if err != nil {
		t.Fatal(err)
	}
	// Pop the root from the initial push.
	rc.NextInclude()

	rc.AddInclude(a)
	rc.AddInclude(b)

	// Stack is LIFO: b should come first.
	fc1 := rc.NextInclude()
	if fc1 == nil {
		t.Fatal("expected first pop")
	}
	absB, _ := filepath.Abs(b)
	if fc1.Path != absB {
		t.Errorf("first pop Path = %q, want %q", fc1.Path, absB)
	}

	fc2 := rc.NextInclude()
	if fc2 == nil {
		t.Fatal("expected second pop")
	}
	absA, _ := filepath.Abs(a)
	if fc2.Path != absA {
		t.Errorf("second pop Path = %q, want %q", fc2.Path, absA)
	}

	if rc.NextInclude() != nil {
		t.Error("stack should be empty")
	}
}

// ---------------------------------------------------------------------------
// RootCursor.String
// ---------------------------------------------------------------------------

func TestRootCursor_String_WithCurrent(t *testing.T) {
	dir := t.TempDir()
	p := writeTemp(t, dir, "root.conf", "line1\nline2\n")

	rc, err := NewRootCursor(p)
	if err != nil {
		t.Fatal(err)
	}

	s := rc.String()
	// Should contain the file path and position info.
	if !strings.Contains(s, "RootCursor") {
		t.Errorf("String() = %q, should contain 'RootCursor'", s)
	}
	if !strings.Contains(s, rc.current.Path) {
		t.Errorf("String() = %q, should contain path %q", s, rc.current.Path)
	}
}

func TestRootCursor_String_NilCurrent(t *testing.T) {
	dir := t.TempDir()
	p := writeTemp(t, dir, "root.conf", "key = val\n")

	rc, err := NewRootCursor(p)
	if err != nil {
		t.Fatal(err)
	}
	rc.current = nil

	s := rc.String()
	if !strings.Contains(s, "RootCursor") {
		t.Errorf("String() = %q, should contain 'RootCursor'", s)
	}
	// Without current, should still show the file name and path.
	if !strings.Contains(s, rc.File.Name) {
		t.Errorf("String() = %q, should contain file name %q", s, rc.File.Name)
	}
}

func TestRootCursor_String_NilFile(t *testing.T) {
	dir := t.TempDir()
	p := writeTemp(t, dir, "root.conf", "key = val\n")

	rc, err := NewRootCursor(p)
	if err != nil {
		t.Fatal(err)
	}
	rc.File = nil

	s := rc.String()
	if !strings.Contains(s, "RootCursor") {
		t.Errorf("String() = %q, should contain 'RootCursor'", s)
	}
}

// ---------------------------------------------------------------------------
// NewFileCursor
// ---------------------------------------------------------------------------

func TestNewFileCursor(t *testing.T) {
	dir := t.TempDir()
	p := writeTemp(t, dir, "test.conf", "line1\nline2\nline3\n")

	fc, err := NewFileCursor(p)
	if err != nil {
		t.Fatalf("NewFileCursor: unexpected error: %v", err)
	}
	abs, _ := filepath.Abs(p)
	if fc.Path != abs {
		t.Errorf("Path = %q, want %q", fc.Path, abs)
	}
	// Start-before-first: lineOffset should be -1.
	if fc.lineOffset != -1 {
		t.Errorf("lineOffset = %d, want -1", fc.lineOffset)
	}
	if fc.byteOffset != -1 {
		t.Errorf("byteOffset = %d, want -1", fc.byteOffset)
	}
}

func TestNewFileCursor_FileNotFound(t *testing.T) {
	_, err := NewFileCursor(testPath("file.conf"))
	if err == nil {
		t.Error("NewFileCursor should return error for non-existent file")
	}
}

// ---------------------------------------------------------------------------
// FileCursor.GetLine
// ---------------------------------------------------------------------------

func TestFileCursor_GetLine_Valid(t *testing.T) {
	dir := t.TempDir()
	p := writeTemp(t, dir, "test.conf", "hello\nworld\n")

	fc, err := NewFileCursor(p)
	if err != nil {
		t.Fatal(err)
	}

	// Advance to first line.
	fc.NextLine()

	line, ok := fc.GetLine()
	if !ok {
		t.Fatal("GetLine should return true for valid line")
	}
	if line != "hello" {
		t.Errorf("GetLine = %q, want %q", line, "hello")
	}
}

func TestFileCursor_GetLine_BeforeFirst(t *testing.T) {
	dir := t.TempDir()
	p := writeTemp(t, dir, "test.conf", "hello\n")

	fc, err := NewFileCursor(p)
	if err != nil {
		t.Fatal(err)
	}

	// Before NextLine, lineOffset is -1.
	line, ok := fc.GetLine()
	if ok {
		t.Errorf("GetLine before NextLine should return false, got line=%q", line)
	}
}

func TestFileCursor_GetLine_AfterExhaustion(t *testing.T) {
	dir := t.TempDir()
	// "only\n" splits into ["only", ""], so 2 lines.
	p := writeTemp(t, dir, "test.conf", "only\n")

	fc, err := NewFileCursor(p)
	if err != nil {
		t.Fatal(err)
	}

	// Advance past all lines. NextLine stops advancing when there is no
	// next line, leaving the cursor on the last valid line.
	for {
		_, ok := fc.NextLine()
		if !ok {
			break
		}
	}

	// The cursor is on the last line (trailing empty string from split).
	// GetLine should still return true for the current position.
	line, ok := fc.GetLine()
	if !ok {
		t.Error("GetLine should return true while cursor is on a valid line")
	}
	if line != "" {
		t.Errorf("GetLine = %q, want empty trailing line", line)
	}
}

// ---------------------------------------------------------------------------
// FileCursor.NextLine
// ---------------------------------------------------------------------------

func TestFileCursor_NextLine_FullTraversal(t *testing.T) {
	dir := t.TempDir()
	p := writeTemp(t, dir, "test.conf", "alpha\nbeta\ngamma\n")

	fc, err := NewFileCursor(p)
	if err != nil {
		t.Fatal(err)
	}

	// "alpha\nbeta\ngamma\n" splits into ["alpha", "beta", "gamma", ""]
	want := []string{"alpha", "beta", "gamma", ""}
	for i, expected := range want {
		line, ok := fc.NextLine()
		if !ok {
			t.Fatalf("NextLine()[%d]: unexpected false", i)
		}
		if line != expected {
			t.Errorf("NextLine()[%d] = %q, want %q", i, line, expected)
		}
	}

	// One more should return false.
	_, ok := fc.NextLine()
	if ok {
		t.Error("NextLine past end should return false")
	}
}

func TestFileCursor_NextLine_ResetsbyteOffset(t *testing.T) {
	dir := t.TempDir()
	p := writeTemp(t, dir, "test.conf", "ab\ncd\n")

	fc, err := NewFileCursor(p)
	if err != nil {
		t.Fatal(err)
	}

	fc.NextLine()
	// Simulate advancing byteOffset.
	fc.byteOffset = 5

	fc.NextLine()
	if fc.byteOffset != 0 {
		t.Errorf("byteOffset after NextLine = %d, want 0", fc.byteOffset)
	}
}

// ---------------------------------------------------------------------------
// FileCursor.NextChar
// ---------------------------------------------------------------------------

func TestFileCursor_NextChar_ASCII(t *testing.T) {
	dir := t.TempDir()
	p := writeTemp(t, dir, "test.conf", "abc\n")

	fc, err := NewFileCursor(p)
	if err != nil {
		t.Fatal(err)
	}

	fc.NextLine() // advance to "abc"

	want := []rune{'a', 'b', 'c'}
	for i, expected := range want {
		r, ok := fc.NextChar()
		if !ok {
			t.Fatalf("NextChar()[%d]: unexpected false", i)
		}
		if r != expected {
			t.Errorf("NextChar()[%d] = %q, want %q", i, r, expected)
		}
	}

	// Past end of line.
	_, ok := fc.NextChar()
	if ok {
		t.Error("NextChar past end of line should return false")
	}
}

func TestFileCursor_NextChar_UTF8(t *testing.T) {
	dir := t.TempDir()
	// UTF-8 multibyte: é is 2 bytes, 日 is 3 bytes.
	p := writeTemp(t, dir, "test.conf", "é日\n")

	fc, err := NewFileCursor(p)
	if err != nil {
		t.Fatal(err)
	}

	fc.NextLine()

	r1, ok := fc.NextChar()
	if !ok || r1 != 'é' {
		t.Errorf("NextChar()[0] = %q, ok=%v; want 'é', true", r1, ok)
	}

	r2, ok := fc.NextChar()
	if !ok || r2 != '日' {
		t.Errorf("NextChar()[1] = %q, ok=%v; want '日', true", r2, ok)
	}

	_, ok = fc.NextChar()
	if ok {
		t.Error("NextChar past end should return false")
	}
}

func TestFileCursor_NextChar_BeforeNextLine(t *testing.T) {
	dir := t.TempDir()
	p := writeTemp(t, dir, "test.conf", "abc\n")

	fc, err := NewFileCursor(p)
	if err != nil {
		t.Fatal(err)
	}

	// Without calling NextLine, lineOffset is -1, so GetLine returns false.
	_, ok := fc.NextChar()
	if ok {
		t.Error("NextChar before NextLine should return false")
	}
}

func TestFileCursor_NextChar_EmptyLine(t *testing.T) {
	dir := t.TempDir()
	p := writeTemp(t, dir, "test.conf", "\nnotempty\n")

	fc, err := NewFileCursor(p)
	if err != nil {
		t.Fatal(err)
	}

	fc.NextLine() // advance to empty line ""

	_, ok := fc.NextChar()
	if ok {
		t.Error("NextChar on empty line should return false")
	}
}

// ---------------------------------------------------------------------------
// FileCursor.String
// ---------------------------------------------------------------------------

func TestFileCursor_String(t *testing.T) {
	dir := t.TempDir()
	p := writeTemp(t, dir, "test.conf", "hello\n")

	fc, err := NewFileCursor(p)
	if err != nil {
		t.Fatal(err)
	}

	s := fc.String()
	if !strings.Contains(s, "FileCursor") {
		t.Errorf("String() = %q, should contain 'FileCursor'", s)
	}
	if !strings.Contains(s, fc.Path) {
		t.Errorf("String() = %q, should contain path %q", s, fc.Path)
	}
}

func TestFileCursor_String_AfterAdvance(t *testing.T) {
	dir := t.TempDir()
	p := writeTemp(t, dir, "test.conf", "ab\ncd\n")

	fc, err := NewFileCursor(p)
	if err != nil {
		t.Fatal(err)
	}

	fc.NextLine()
	fc.NextChar()

	s := fc.String()
	// After NextLine (lineOffset=0) and NextChar (byteOffset=1),
	// displayed as 1-indexed: line 1, col 2.
	if !strings.Contains(s, ":1:2") {
		t.Errorf("String() = %q, want position :1:2", s)
	}
}
