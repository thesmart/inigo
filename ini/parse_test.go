package ini

import (
	"testing"
)

func testFC() *FileCursor {
	return &FileCursor{
		Cursor: Cursor{Line: 1, Offset: 0},
		Path:   "test.conf",
	}
}

func TestParseLine_Blank(t *testing.T) {
	tests := []string{"", "   ", "\t\t", "  \t  "}
	for _, line := range tests {
		result, err := parseLine(line, testFC())
		if err != nil {
			t.Fatalf("unexpected error for blank line %q: %v", line, err)
		}
		if result.kind != lineBlank {
			t.Errorf("expected lineBlank for %q, got %d", line, result.kind)
		}
	}
}

func TestParseLine_Comment(t *testing.T) {
	tests := []string{"# comment", "; comment", "  # indented comment", "\t; tab comment"}
	for _, line := range tests {
		result, err := parseLine(line, testFC())
		if err != nil {
			t.Fatalf("unexpected error for comment %q: %v", line, err)
		}
		if result.kind != lineComment {
			t.Errorf("expected lineComment for %q, got %d", line, result.kind)
		}
	}
}

func TestParseLine_Section(t *testing.T) {
	tests := []struct {
		line     string
		expected string
	}{
		{"[section]", "section"},
		{"[my_section]", "my_section"},
		{"[sec-tion]", "sec-tion"},
		{"[sec.tion]", "sec.tion"},
		{"[sec$tion]", "sec$tion"},
		{"[section]   ", "section"},
		{"[section] # comment", "section"},
		{"[section] ; comment", "section"},
		{"  [section]", "section"},
	}
	for _, tt := range tests {
		result, err := parseLine(tt.line, testFC())
		if err != nil {
			t.Fatalf("unexpected error for %q: %v", tt.line, err)
		}
		if result.kind != lineSection {
			t.Errorf("expected lineSection for %q, got %d", tt.line, result.kind)
		}
		if result.section != tt.expected {
			t.Errorf("expected section %q for %q, got %q", tt.expected, tt.line, result.section)
		}
	}
}

func TestParseLine_Section_Errors(t *testing.T) {
	tests := []string{
		"[123]",       // starts with digit
		"[]",          // empty
		"[section",    // no closing bracket
		"[section] x", // trailing garbage
	}
	for _, line := range tests {
		_, err := parseLine(line, testFC())
		if err == nil {
			t.Errorf("expected error for %q, got nil", line)
		}
	}
}

func TestParseLine_Parameter(t *testing.T) {
	tests := []struct {
		line  string
		key   string
		value string
	}{
		{"key = value", "key", "value"},
		{"key=value", "key", "value"},
		{"key : value", "key", "value"},
		{"key:value", "key", "value"},
		{"key = ", "key", ""},
		{"key =", "key", ""},
		{"key", "key", ""},
		{"my_key = hello", "my_key", "hello"},
		{"KEY = val", "KEY", "val"},
		{"key = 'quoted value'", "key", "quoted value"},
		{"key = 'it''s'", "key", "it's"},
		{"key = 'back\\'slash'", "key", "back'slash"},
		{"key = 'back\\\\slash'", "key", "back\\slash"},
		{"key = value # comment", "key", "value"},
		{"key = value ; comment", "key", "value"},
		{"key = 100", "key", "100"},
		{"key = 0xFF", "key", "0xFF"},
		{"key = 'has spaces'", "key", "has spaces"},
		{"key = '\"sub\" quote'", "key", "\"sub\" quote"},
		{"a1 = b2", "a1", "b2"},
		{"_key = val", "_key", "val"},
		{"key-name = val", "key-name", "val"},
		{"key.name = val", "key.name", "val"},
		{"key$name = val", "key$name", "val"},
	}
	for _, tt := range tests {
		result, err := parseLine(tt.line, testFC())
		if err != nil {
			t.Fatalf("unexpected error for %q: %v", tt.line, err)
		}
		if result.kind != lineParameter {
			t.Errorf("expected lineParameter for %q, got %d", tt.line, result.kind)
			continue
		}
		if result.param.Name != tt.key {
			t.Errorf("expected key %q for %q, got %q", tt.key, tt.line, result.param.Name)
		}
		if result.param.Value != tt.value {
			t.Errorf("expected value %q for %q, got %q", tt.value, tt.line, result.param.Value)
		}
	}
}

func TestParseLine_Parameter_Errors(t *testing.T) {
	tests := []string{
		"123 = value",         // key starts with digit
		"key = 'unterminated", // unterminated quote
	}
	for _, line := range tests {
		_, err := parseLine(line, testFC())
		if err == nil {
			t.Errorf("expected error for %q, got nil", line)
		}
	}
}

func TestParseLine_Include(t *testing.T) {
	tests := []struct {
		line        string
		includeKind string
		includePath string
	}{
		{"include 'file.conf'", "include", "file.conf"},
		{"include file.conf", "include", "file.conf"},
		{"include_if_exists 'file.conf'", "include_if_exists", "file.conf"},
		{"include_if_exists file.conf", "include_if_exists", "file.conf"},
		{"include_dir '/some/dir'", "include_dir", "/some/dir"},
		{"include_dir /some/dir", "include_dir", "/some/dir"},
		{"  include 'file.conf'", "include", "file.conf"},
		{"INCLUDE 'file.conf'", "include", "file.conf"},
		{"Include_Dir 'dir'", "include_dir", "dir"},
	}
	for _, tt := range tests {
		result, err := parseLine(tt.line, testFC())
		if err != nil {
			t.Fatalf("unexpected error for %q: %v", tt.line, err)
		}
		if result.kind != lineInclude {
			t.Errorf("expected lineInclude for %q, got %d", tt.line, result.kind)
			continue
		}
		if result.includeKind != tt.includeKind {
			t.Errorf("expected includeKind %q for %q, got %q", tt.includeKind, tt.line, result.includeKind)
		}
		if result.includePath != tt.includePath {
			t.Errorf("expected includePath %q for %q, got %q", tt.includePath, tt.line, result.includePath)
		}
	}
}

func TestParseLine_Include_Errors(t *testing.T) {
	tests := []string{
		"include   ", // empty path after whitespace
		"include ''", // empty quoted path
	}
	for _, line := range tests {
		_, err := parseLine(line, testFC())
		if err == nil {
			t.Errorf("expected error for %q, got nil", line)
		}
	}
}

func TestParseLine_BareIncludeAsParam(t *testing.T) {
	// Bare "include" with no value should be a parameter
	result, err := parseLine("include", testFC())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.kind != lineParameter {
		t.Errorf("expected lineParameter for bare 'include', got %d", result.kind)
	}
}

func TestParseLine_IncludeAsParam(t *testing.T) {
	// "include = 'value'" should be a parameter, not an include directive
	result, err := parseLine("include = 'some_file'", testFC())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.kind != lineParameter {
		t.Errorf("expected lineParameter, got %d", result.kind)
	}
	if result.param.Name != "include" {
		t.Errorf("expected key 'include', got %q", result.param.Name)
	}
}

func TestParseQuotedValue(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"'hello'", "hello"},
		{"'it''s'", "it's"},
		{"'back\\'slash'", "back'slash"},
		{"'double\\\\back'", "double\\back"},
		{"''", ""},
		{"'has \"double\" quotes'", "has \"double\" quotes"},
	}
	for _, tt := range tests {
		val, _, err := parseQuotedValue(tt.input, 0, testFC())
		if err != nil {
			t.Fatalf("unexpected error for %q: %v", tt.input, err)
		}
		if val != tt.expected {
			t.Errorf("expected %q for %q, got %q", tt.expected, tt.input, val)
		}
	}
}

func TestParseQuotedValue_Unterminated(t *testing.T) {
	_, _, err := parseQuotedValue("'unterminated", 0, testFC())
	if err == nil {
		t.Error("expected error for unterminated quote")
	}
}

func TestIsIdentStart(t *testing.T) {
	for _, ch := range []byte("abcABC_") {
		if !isIdentStart(ch) {
			t.Errorf("expected isIdentStart(%c) to be true", ch)
		}
	}
	for _, ch := range []byte("0123-.$") {
		if isIdentStart(ch) {
			t.Errorf("expected isIdentStart(%c) to be false", ch)
		}
	}
}

func TestIsIdentContinue(t *testing.T) {
	for _, ch := range []byte("abcABC_0123-.$") {
		if !isIdentContinue(ch) {
			t.Errorf("expected isIdentContinue(%c) to be true", ch)
		}
	}
	for _, ch := range []byte(" \t#;=:") {
		if isIdentContinue(ch) {
			t.Errorf("expected isIdentContinue(%c) to be false", ch)
		}
	}
}

func TestSkipWhitespace(t *testing.T) {
	if skipWhitespace("  hello", 0) != 2 {
		t.Error("expected 2")
	}
	if skipWhitespace("\thello", 0) != 1 {
		t.Error("expected 1")
	}
	if skipWhitespace("hello", 0) != 0 {
		t.Error("expected 0")
	}
}

func TestParseError(t *testing.T) {
	err := parseError(testFC(), 5, "key", "bad stuff")
	expected := "test.conf:1:5: <key> bad stuff"
	if err.Error() != expected {
		t.Errorf("expected %q, got %q", expected, err.Error())
	}
}

func TestParseUnquotedValue(t *testing.T) {
	tests := []struct {
		line     string
		start    int
		expected string
	}{
		{"value # comment", 0, "value "},
		{"value ; comment", 0, "value "},
		{"value", 0, "value"},
		{"  value  ", 2, "value  "},
	}
	for _, tt := range tests {
		val, _ := parseUnquotedValue(tt.line, tt.start)
		if val != tt.expected {
			t.Errorf("expected %q for line=%q start=%d, got %q", tt.expected, tt.line, tt.start, val)
		}
	}
}

func TestParseDirectiveOrParam_IncludeVariant(t *testing.T) {
	// "include_extra = val" should be a param, not an include directive
	result, err := parseLine("include_extra = val", testFC())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.kind != lineParameter {
		t.Errorf("expected lineParameter, got %d", result.kind)
	}
}

func TestParseIncludeLine_TrailingGarbage(t *testing.T) {
	_, err := parseLine("include 'file.conf' garbage", testFC())
	if err == nil {
		t.Error("expected error for trailing garbage after include path")
	}
}

func TestParseValueAt_Empty(t *testing.T) {
	val, i, err := parseValueAt("", 0, testFC())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "" || i != 0 {
		t.Errorf("expected empty value at 0, got %q at %d", val, i)
	}
}
