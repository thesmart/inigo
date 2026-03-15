package pgini

import (
	"path/filepath"
	"strings"
	"testing"
)

// unitsDir is the base path for unit test data files.
var unitsDir = filepath.Join("testdata", "conf")

// unitPath returns the absolute path for a testdata/units file.
func unitPath(name string) string {
	abs, err := filepath.Abs(filepath.Join(unitsDir, name))
	if err != nil {
		panic(err)
	}
	return abs
}

// requireLoad loads a conf file and fails the test if it errors.
func requireLoad(t *testing.T, name string) *IniFile {
	t.Helper()
	f, err := Parse(unitPath(name))
	if err != nil {
		t.Fatalf("Load(%q): %v", name, err)
	}
	return f
}

// requireSection gets a section or fails the test.
func requireSection(t *testing.T, f *IniFile, name string) *Section {
	t.Helper()
	s := f.GetSection(name)
	if s == nil {
		t.Fatalf("section %q not found", name)
	}
	return s
}

// requireParam asserts a section has a key with the expected value.
func requireParam(t *testing.T, s *Section, key, want string) {
	t.Helper()
	got, ok := s.GetValue(key)
	if !ok {
		t.Errorf("section %q: param %q not found", s.Name, key)
		return
	}
	if got != want {
		t.Errorf("section %q: param %q = %q, want %q", s.Name, key, got, want)
	}
}

// requireParamMissing asserts a section does NOT have a key.
func requireParamMissing(t *testing.T, s *Section, key string) {
	t.Helper()
	if _, ok := s.GetValue(key); ok {
		t.Errorf("section %q: param %q should not exist", s.Name, key)
	}
}

// requireParamCount asserts the number of params in a section.
func requireParamCount(t *testing.T, s *Section, want int) {
	t.Helper()
	got := 0
	for range s.Params() {
		got++
	}
	if got != want {
		t.Errorf("section %q: param count = %d, want %d", s.Name, got, want)
	}
}

// requireSectionCount asserts the number of sections in a file.
func requireSectionCount(t *testing.T, f *IniFile, want int) {
	t.Helper()
	got := 0
	for range f.Sections() {
		got++
	}
	if got != want {
		t.Errorf("section count = %d, want %d", got, want)
	}
}

// ---------------------------------------------------------------------------
// 01 — blank lines
// ---------------------------------------------------------------------------

func TestLoad_01_Blank(t *testing.T) {
	f := requireLoad(t, "01_blank.conf")
	// Only the default section, with no parameters.
	requireSectionCount(t, f, 1)
	def := requireSection(t, f, "")
	requireParamCount(t, def, 0)
}

// TestLoad_01_Blank_WhitespaceOnly tests whitespace-only lines via inline
// content, since editors strip trailing whitespace from static files.
func TestLoad_01_Blank_WhitespaceOnly(t *testing.T) {
	dir := t.TempDir()
	content := "   \n\t\t\n \t \n"
	path := writeTemp(t, dir, "ws.conf", content)
	f, err := Parse(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	requireSectionCount(t, f, 1)
	def := requireSection(t, f, "")
	requireParamCount(t, def, 0)
}

// ---------------------------------------------------------------------------
// 02 — comments
// ---------------------------------------------------------------------------

func TestLoad_02_Comments(t *testing.T) {
	f := requireLoad(t, "02_comments.conf")
	// Comments produce no parameters.
	requireSectionCount(t, f, 1)
	def := requireSection(t, f, "")
	requireParamCount(t, def, 0)
}

// ---------------------------------------------------------------------------
// 03 — sections
// ---------------------------------------------------------------------------

func TestLoad_03_Sections(t *testing.T) {
	f := requireLoad(t, "03_sections.conf")

	// Expected sections: default, basic, upper, mixed, _private,
	// section_2_name, x, trailing, commented, semicommented, padded.
	// "basic" is reopened but not duplicated. "default" reopens default.
	requireSectionCount(t, f, 11)

	// basic section: reopened, should have both keys
	basic := requireSection(t, f, "basic")
	requireParam(t, basic, "key", "one")
	requireParam(t, basic, "key2", "eleven")

	// uppercase section (stored lowercase)
	upper := requireSection(t, f, "upper")
	requireParam(t, upper, "key", "two")

	// mixed case (stored lowercase)
	mixed := requireSection(t, f, "mixed")
	requireParam(t, mixed, "key", "three")

	// underscore start
	priv := requireSection(t, f, "_private")
	requireParam(t, priv, "key", "four")

	// underscores and digits
	s2 := requireSection(t, f, "section_2_name")
	requireParam(t, s2, "key", "five")

	// single letter
	x := requireSection(t, f, "x")
	requireParam(t, x, "key", "six")

	// trailing whitespace after ]
	trail := requireSection(t, f, "trailing")
	requireParam(t, trail, "key", "seven")

	// trailing comment after ]
	commented := requireSection(t, f, "commented")
	requireParam(t, commented, "key", "eight")

	// trailing semicolon comment after ]
	semi := requireSection(t, f, "semicommented")
	requireParam(t, semi, "key", "nine")

	// whitespace inside brackets
	padded := requireSection(t, f, "padded")
	requireParam(t, padded, "key", "ten")

	// [default] alias reopens the default section
	def := requireSection(t, f, "")
	requireParam(t, def, "default_key", "twelve")
}

// ---------------------------------------------------------------------------
// 04 — identifiers (keys)
// ---------------------------------------------------------------------------

func TestLoad_04_Identifiers(t *testing.T) {
	f := requireLoad(t, "04_identifiers.conf")
	def := requireSection(t, f, "")

	// a = single_letter
	requireParam(t, def, "a", "single_letter")
	// Z → z (lowercase), = uppercase_single
	requireParam(t, def, "z", "uppercase_single")
	// _ = underscore_start
	requireParam(t, def, "_", "underscore_start")
	// abc = lowercase, then ABC = uppercase (overwrites, last wins)
	requireParam(t, def, "abc", "uppercase")
	// MiXeD → mixed
	requireParam(t, def, "mixed", "mixed_case")
	// _leading
	requireParam(t, def, "_leading", "underscore_leading")
	// __double
	requireParam(t, def, "__double", "double_underscore")
	// a1
	requireParam(t, def, "a1", "letter_then_digit")
	// _0
	requireParam(t, def, "_0", "underscore_then_digit")
	// long name
	requireParam(t, def, "long_identifier_name_with_many_parts", "long")
	// abc123def456
	requireParam(t, def, "abc123def456", "letters_and_digits")
	// A_b_C_1_2_3 → a_b_c_1_2_3
	requireParam(t, def, "a_b_c_1_2_3", "mixed_everything")
}

// ---------------------------------------------------------------------------
// 05 — separators
// ---------------------------------------------------------------------------

func TestLoad_05_Separators(t *testing.T) {
	f := requireLoad(t, "05_separators.conf")
	def := requireSection(t, f, "")

	requireParam(t, def, "equals", "value_equals")
	requireParam(t, def, "colon", "value_colon")
	requireParam(t, def, "space_only", "value_space")
	requireParam(t, def, "equals_no_space", "value_tight")
	requireParam(t, def, "colon_no_space", "value_tight_colon")
	requireParam(t, def, "extra_space", "value_extra")
	requireParam(t, def, "extra_colon", "value_extra_colon")
	requireParam(t, def, "tab_around", "value_tab")
}

// ---------------------------------------------------------------------------
// 06 — unquoted values
// ---------------------------------------------------------------------------

func TestLoad_06_UnquotedValues(t *testing.T) {
	f := requireLoad(t, "06_unquoted_values.conf")
	def := requireSection(t, f, "")

	tests := []struct{ key, want string }{
		{"alpha", "abcdef"},
		{"upper", "ABCDEF"},
		{"digits", "1234567890"},
		{"underscore", "hello_world"},
		{"dot", "file.txt"},
		{"dash", "my-value"},
		{"colon", "host:port"},
		{"slash", "/usr/local/bin"},
		{"plus", "value+extra"},
		{"mixed", "abc123_DEF.foo-bar:8080/path+end"},
		{"url_like", "https://example.com:443/path"},
		{"ip_addr", "192.168.1.1"},
		{"version", "v1.2.3-rc.1+build.42"},
		{"path", "/tmp/my_dir/file.conf"},
		{"hex_like", "0xFF"},
		{"negative", "-123"},
		{"signed", "+456"},
		{"float_like", "3.14"},
	}
	for _, tt := range tests {
		requireParam(t, def, tt.key, tt.want)
	}
}

// ---------------------------------------------------------------------------
// 07 — quoted values (escaping)
// ---------------------------------------------------------------------------

func TestLoad_07_QuotedValues(t *testing.T) {
	f := requireLoad(t, "07_quoted_values.conf")
	def := requireSection(t, f, "")

	tests := []struct{ key, want string }{
		// empty quoted string
		{"empty", ""},
		// simple text
		{"simple", "hello world"},
		// preserved internal whitespace
		{"spaced", "  leading and trailing  "},
		// backslash escapes
		{"escaped_backslash", "back\\slash"},
		{"escaped_single_quote", "it's"},
		{"escaped_b", "back\bspace"},
		{"escaped_f", "form\ffeed"},
		{"escaped_n", "new\nline"},
		{"escaped_r", "carriage\rreturn"},
		{"escaped_t", "tab\there"},
		// doubled quote
		{"doubled_quote", "it's doubled"},
		// octal escapes
		{"octal_one", "\x00"},
		{"octal_null", "\x00"},
		{"octal_bell", "\x07"},
		{"octal_tab", "\t"},
		{"octal_newline", "\n"},
		{"octal_two_digit", "\x3f"},
		{"octal_one_digit", "\x07"},
		// multiple escapes in one value
		{"multi_escape", "line1\nline2\ttab\\backslash"},
		// UTF-8 content
		{"utf8_content", "café résumé naïve"},
		{"utf8_emoji", "🌍🎉"},
		{"utf8_cjk", "你好世界"},
		// delimiter chars are literal inside quotes
		{"comment_chars", "# not a comment ; also not"},
		{"bracket_chars", "[not a section]"},
		{"equals_in_value", "key=value"},
		{"colon_in_value", "key:value"},
		// value with only escapes
		{"only_escapes", "\n\t\r"},
	}
	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			requireParam(t, def, tt.key, tt.want)
		})
	}
}

// TestLoad_07_QuotedValues_RoundTrip parses the quoted-values test file,
// marshals it back to INI, re-parses, and verifies every value survives.
func TestLoad_07_QuotedValues_RoundTrip(t *testing.T) {
	f := requireLoad(t, "07_quoted_values.conf")
	def := requireSection(t, f, "")

	// Collect original values.
	type kv struct{ key, value string }
	var originals []kv
	for _, p := range def.Params() {
		originals = append(originals, kv{p.Name, p.Value})
	}

	// Marshal to bytes, write to temp file, re-parse.
	data, err := f.MarshalIni()
	if err != nil {
		t.Fatalf("MarshalIni: %v", err)
	}

	dir := t.TempDir()
	path := writeTemp(t, dir, "07_roundtrip.conf", string(data))
	f2, err := Parse(path)
	if err != nil {
		t.Fatalf("re-Parse: %v", err)
	}

	def2 := requireSection(t, f2, "")
	for _, orig := range originals {
		requireParam(t, def2, orig.key, orig.value)
	}
}

// ---------------------------------------------------------------------------
// 08 — booleans (parsed as string values)
// ---------------------------------------------------------------------------

func TestLoad_08_Booleans(t *testing.T) {
	f := requireLoad(t, "08_booleans.conf")
	def := requireSection(t, f, "")

	tests := []struct{ key, want string }{
		{"true_lower", "true"},
		{"true_upper", "TRUE"},
		{"true_mixed", "True"},
		{"false_lower", "false"},
		{"false_upper", "FALSE"},
		{"false_mixed", "False"},
		{"on_lower", "on"},
		{"on_upper", "ON"},
		{"off_lower", "off"},
		{"off_upper", "OFF"},
		{"yes_lower", "yes"},
		{"yes_upper", "YES"},
		{"no_lower", "no"},
		{"no_upper", "NO"},
		{"one", "1"},
		{"zero", "0"},
		{"t_lower", "t"},
		{"t_upper", "T"},
		{"f_lower", "f"},
		{"f_upper", "F"},
		{"y_lower", "y"},
		{"y_upper", "Y"},
		{"n_lower", "n"},
		{"n_upper", "N"},
	}
	for _, tt := range tests {
		requireParam(t, def, tt.key, tt.want)
	}
}

// ---------------------------------------------------------------------------
// 09 — numbers (parsed as string values)
// ---------------------------------------------------------------------------

func TestLoad_09_Numbers(t *testing.T) {
	f := requireLoad(t, "09_numbers.conf")
	def := requireSection(t, f, "")

	tests := []struct{ key, want string }{
		{"decimal", "100"},
		{"zero", "0"},
		{"negative", "-1"},
		{"positive", "+1"},
		{"large", "9999999"},
		{"hex_lower", "0xff"},
		{"hex_upper", "0xFF"},
		{"hex_long", "0xDEADBEEF"},
		{"float_simple", "1.5"},
		{"float_small", "0.001"},
		{"float_no_lead", ".5"},
		{"float_trail", "1."},
	}
	for _, tt := range tests {
		requireParam(t, def, tt.key, tt.want)
	}
}

// TestLoad_09_Numbers_RoundTrip parses the numbers test file, marshals it
// back to INI, re-parses, and verifies every value survives unchanged.
func TestLoad_09_Numbers_RoundTrip(t *testing.T) {
	f := requireLoad(t, "09_numbers.conf")
	def := requireSection(t, f, "")

	// Collect original values.
	type kv struct{ key, value string }
	var originals []kv
	for _, p := range def.Params() {
		originals = append(originals, kv{p.Name, p.Value})
	}

	// Marshal to bytes, write to temp file, re-parse.
	data, err := f.MarshalIni()
	if err != nil {
		t.Fatalf("MarshalIni: %v", err)
	}

	dir := t.TempDir()
	path := writeTemp(t, dir, "09_roundtrip.conf", string(data))
	f2, err := Parse(path)
	if err != nil {
		t.Fatalf("re-Parse: %v", err)
	}

	def2 := requireSection(t, f2, "")
	for _, orig := range originals {
		requireParam(t, def2, orig.key, orig.value)
	}
}

// ---------------------------------------------------------------------------
// 10 — empty / missing values
// ---------------------------------------------------------------------------

func TestLoad_10_EmptyValues(t *testing.T) {
	f := requireLoad(t, "10_empty_values.conf")
	def := requireSection(t, f, "")

	emptyKeys := []string{
		"no_value_no_sep",
		"no_value_eq",
		"no_value_colon",
		"no_value_eq_space",
		"no_value_comment",
		"no_value_semi",
	}
	for _, key := range emptyKeys {
		requireParam(t, def, key, "")
	}
}

// ---------------------------------------------------------------------------
// 11 — duplicates (last wins)
// ---------------------------------------------------------------------------

func TestLoad_11_Duplicates(t *testing.T) {
	f := requireLoad(t, "11_duplicates.conf")

	// Default section: key set 3 times, last wins.
	def := requireSection(t, f, "")
	requireParam(t, def, "key", "third")
	// "another" set twice via [default] reopen, last wins.
	requireParam(t, def, "another", "two")

	// section_a: reopened, dup set 3 times total, last wins.
	sa := requireSection(t, f, "section_a")
	requireParam(t, sa, "dup", "gamma")
	requireParam(t, sa, "extra", "added")
}

// ---------------------------------------------------------------------------
// 12 — whitespace handling
// ---------------------------------------------------------------------------

func TestLoad_12_Whitespace(t *testing.T) {
	f := requireLoad(t, "12_whitespace.conf")
	def := requireSection(t, f, "")

	// Leading whitespace before key is consumed.
	requireParam(t, def, "spaced_key", "value")
	requireParam(t, def, "tabbed_key", "value")
	requireParam(t, def, "mixed_key", "value")

	// Whitespace around separator is consumed.
	requireParam(t, def, "padded_eq", "padded_value")
	requireParam(t, def, "padded_colon", "padded_value_colon")
	requireParam(t, def, "tabs_eq", "tabbed_value")

	// Trailing whitespace after value is consumed.
	requireParam(t, def, "trailing_spaces", "value")
	requireParam(t, def, "trailing_tabs", "value")

	// Trailing whitespace after quoted value is consumed.
	requireParam(t, def, "trailing_after_quote", "quoted")

	// Section with whitespace inside brackets.
	ss := requireSection(t, f, "spaced_section")
	requireParam(t, ss, "inner", "value")
}

// ---------------------------------------------------------------------------
// 13 — trailing comments
// ---------------------------------------------------------------------------

func TestLoad_13_TrailingComments(t *testing.T) {
	f := requireLoad(t, "13_trailing_comments.conf")
	def := requireSection(t, f, "")

	// After unquoted value, comment is stripped.
	requireParam(t, def, "key1", "value")
	requireParam(t, def, "key2", "value")

	// After quoted value, comment is stripped.
	requireParam(t, def, "key3", "quoted")
	requireParam(t, def, "key4", "quoted")

	// Comment after separator makes value empty.
	requireParam(t, def, "key5", "")
	requireParam(t, def, "key6", "")

	// Section with trailing comment.
	commented := requireSection(t, f, "commented")
	requireParam(t, commented, "key7", "value")

	// Comment chars inside quotes are literal (key8, key9 are in [commented]).
	requireParam(t, commented, "key8", "has # inside")
	requireParam(t, commented, "key9", "has ; inside")
}

// ---------------------------------------------------------------------------
// 14 — include directive
// ---------------------------------------------------------------------------

func TestLoad_14_Include(t *testing.T) {
	f := requireLoad(t, "includes/14_include.conf")
	def := requireSection(t, f, "")

	// "before" is set to "original", then overridden by included file.
	requireParam(t, def, "before", "overridden")
	requireParam(t, def, "included_key", "included_value")
	requireParam(t, def, "after", "final")
}

// ---------------------------------------------------------------------------
// 15 — include_if_exists directive
// ---------------------------------------------------------------------------

func TestLoad_15_IncludeIfExists(t *testing.T) {
	f := requireLoad(t, "includes/15_include_if_exists.conf")
	def := requireSection(t, f, "")

	requireParam(t, def, "base", "value")
	// Existing file is included.
	requireParam(t, def, "exists_key", "found")
	// Missing file silently skipped — no error, and "final" still parsed.
	requireParam(t, def, "final", "done")
}

// ---------------------------------------------------------------------------
// 16 — include_dir directive
// ---------------------------------------------------------------------------

func TestLoad_16_IncludeDir(t *testing.T) {
	f := requireLoad(t, "includes/16_include_dir.conf")
	def := requireSection(t, f, "")

	requireParam(t, def, "top", "value")
	// a.conf loaded first (ascii sort), then b.conf.
	requireParam(t, def, "from_a", "alpha")
	requireParam(t, def, "from_b", "beta")
	// "order" set by a.conf then overridden by b.conf (last wins).
	requireParam(t, def, "order", "b")
	// .hidden.conf should be skipped.
	requireParamMissing(t, def, "hidden")
}

// ---------------------------------------------------------------------------
// 17 — include with trailing comment
// ---------------------------------------------------------------------------

func TestLoad_17_IncludeWithComment(t *testing.T) {
	f := requireLoad(t, "includes/17_include_with_comment.conf")
	def := requireSection(t, f, "")

	requireParam(t, def, "included_key", "included_value")
	requireParam(t, def, "before", "overridden")
}

// ---------------------------------------------------------------------------
// 18 — include respects section context
// ---------------------------------------------------------------------------

func TestLoad_18_IncludeSections(t *testing.T) {
	f := requireLoad(t, "includes/18_include_sections.conf")

	// Params before include land in [myapp].
	myapp := requireSection(t, f, "myapp")
	requireParam(t, myapp, "local", "value")
	requireParam(t, myapp, "from_include", "in_myapp")

	// The included file switches to [other]. Per PG behavior, section context
	// propagates back to the caller, so after_include lands in [other].
	other := requireSection(t, f, "other")
	requireParam(t, other, "other_key", "other_value")
	requireParam(t, other, "after_include", "still_in_myapp")
}

// ---------------------------------------------------------------------------
// Error cases — each file should fail to Load
// ---------------------------------------------------------------------------

func TestLoad_Errors(t *testing.T) {
	tests := []struct {
		name    string
		file    string
		wantErr string // substring expected in error message
	}{
		// Section errors
		{
			name:    "section_empty",
			file:    "errors/section_empty.conf",
			wantErr: "empty section name",
		},
		{
			name:    "section_unclosed",
			file:    "errors/section_unclosed.conf",
			wantErr: "expected ']'",
		},
		{
			name:    "section_digit_start",
			file:    "errors/section_digit_start.conf",
			wantErr: "invalid section name start",
		},
		{
			name:    "section_trailing_garbage",
			file:    "errors/section_trailing_garbage.conf",
			wantErr: "unexpected character",
		},
		{
			name:    "section_space_in_name",
			file:    "errors/section_space_in_name.conf",
			wantErr: "expected ']'",
		},
		// Key errors
		{
			name:    "key_digit_start",
			file:    "errors/key_digit_start.conf",
			wantErr: "unexpected character",
		},
		{
			name:    "key_special_start",
			file:    "errors/key_special_start.conf",
			wantErr: "unexpected character",
		},
		{
			name:    "line_starts_bang",
			file:    "errors/line_starts_bang.conf",
			wantErr: "unexpected character",
		},
		{
			name:    "line_starts_dash",
			file:    "errors/line_starts_dash.conf",
			wantErr: "unexpected character",
		},
		// Value errors
		{
			name:    "trailing_garbage_after_value",
			file:    "errors/trailing_garbage_after_value.conf",
			wantErr: "unexpected character",
		},
		{
			name:    "trailing_garbage_after_quoted",
			file:    "errors/trailing_garbage_after_quoted.conf",
			wantErr: "unexpected character",
		},
		{
			name:    "unterminated_quote",
			file:    "errors/unterminated_quote.conf",
			wantErr: "unterminated quoted value",
		},
		{
			name:    "unterminated_escape",
			file:    "errors/unterminated_escape.conf",
			wantErr: "unterminated",
		},
		{
			name:    "invalid_escape",
			file:    "errors/invalid_escape.conf",
			wantErr: "invalid escape sequence",
		},
		{
			name:    "double_quote_value",
			file:    "errors/double_quote_value.conf",
			wantErr: "unexpected character",
		},
		{
			name:    "double_quotes_rejected",
			file:    "errors/double_quotes_rejected.conf",
			wantErr: "unexpected character",
		},
		// Include errors
		{
			name:    "include_no_quote",
			file:    "errors/include_no_quote.conf",
			wantErr: "single-quoted path",
		},
		{
			name:    "include_empty_path",
			file:    "errors/include_empty_path.conf",
			wantErr: "must not be empty",
		},
		{
			name:    "include_missing_file",
			file:    "errors/include_missing_file.conf",
			wantErr: "no such file",
		},
		{
			name:    "include_trailing_garbage",
			file:    "errors/include_trailing_garbage.conf",
			wantErr: "unexpected character",
		},
		{
			name:    "include_dir_not_found",
			file:    "errors/include_dir_not_found.conf",
			wantErr: "include_dir",
		},
		{
			name:    "include_no_whitespace",
			file:    "errors/include_no_whitespace.conf",
			wantErr: "requires a quoted path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse(unitPath(tt.file))
			if err == nil {
				t.Fatalf("Load(%q): expected error, got nil", tt.file)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("Load(%q): error = %q, want substring %q", tt.file, err, tt.wantErr)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Error cases — control characters in include paths (inline, not static files)
// ---------------------------------------------------------------------------

func TestLoad_Errors_IncludePathControlChar(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr string
	}{
		{
			name:    "SOH in path",
			content: "include '" + nonExistingPath("bad\x01path.conf") + "'\n",
			wantErr: "invalid control character",
		},
		{
			name:    "DEL in path",
			content: "include '" + nonExistingPath("bad\x7fpath.conf") + "'\n",
			wantErr: "invalid control character",
		},
		{
			name:    "null in path",
			content: "include '" + nonExistingPath("bad\x00path.conf") + "'\n",
			wantErr: "invalid control character",
		},
		{
			name:    "tab in path",
			content: "include '" + nonExistingPath("bad\tpath.conf") + "'\n",
			wantErr: "invalid control character",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := writeTemp(t, dir, "ctrl.conf", tt.content)
			_, err := Parse(path)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want substring %q", err, tt.wantErr)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Load — generic Load[T] convenience function
// ---------------------------------------------------------------------------

func TestLoadInto_09_Numbers(t *testing.T) {
	type Numbers struct {
		Decimal   int     `ini:"decimal"`
		Zero      int     `ini:"zero"`
		Negative  int     `ini:"negative"`
		Large     int     `ini:"large"`
		FloatSimp float64 `ini:"float_simple"`
	}

	// Pre-populate with values that should be overwritten.
	cfg := &Numbers{Decimal: 999, Zero: 999, Negative: 999, Large: 999, FloatSimp: 999.0}

	err := LoadInto(unitPath("09_numbers.conf"), "", cfg)
	if err != nil {
		t.Fatalf("LoadInto: %v", err)
	}

	tests := []struct {
		name string
		got  any
		want any
	}{
		{"decimal", cfg.Decimal, 100},
		{"zero", cfg.Zero, 0},
		{"negative", cfg.Negative, -1},
		{"large", cfg.Large, 9999999},
		{"float_simple", cfg.FloatSimp, 1.5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("%s = %v, want %v", tt.name, tt.got, tt.want)
			}
		})
	}
}

func TestLoadGeneric_09_Numbers(t *testing.T) {
	type Numbers struct {
		Decimal    int     `ini:"decimal"`
		Zero       int     `ini:"zero"`
		Negative   int     `ini:"negative"`
		Positive   int     `ini:"positive"`
		Large      int     `ini:"large"`
		HexLower   int     `ini:"hex_lower"`
		HexUpper   int     `ini:"hex_upper"`
		HexLong    int64   `ini:"hex_long"`
		FloatSimp  float64 `ini:"float_simple"`
		FloatSmall float64 `ini:"float_small"`
		FloatNoLd  float64 `ini:"float_no_lead"`
		FloatTrail float64 `ini:"float_trail"`
	}

	cfg, err := Load[Numbers](unitPath("09_numbers.conf"), "")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	tests := []struct {
		name string
		got  any
		want any
	}{
		{"decimal", cfg.Decimal, 100},
		{"zero", cfg.Zero, 0},
		{"negative", cfg.Negative, -1},
		{"positive", cfg.Positive, 1},
		{"large", cfg.Large, 9999999},
		{"hex_lower", cfg.HexLower, 0xff},
		{"hex_upper", cfg.HexUpper, 0xFF},
		{"hex_long", cfg.HexLong, int64(0xDEADBEEF)},
		{"float_simple", cfg.FloatSimp, 1.5},
		{"float_small", cfg.FloatSmall, 0.001},
		{"float_no_lead", cfg.FloatNoLd, 0.5},
		{"float_trail", cfg.FloatTrail, 1.0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("%s = %v, want %v", tt.name, tt.got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Parse — empty file (zero bytes)
// ---------------------------------------------------------------------------

func TestParse_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := writeTemp(t, dir, "empty.conf", "")
	f, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse empty file: %v", err)
	}
	requireSectionCount(t, f, 1)
	def := requireSection(t, f, "")
	requireParamCount(t, def, 0)
}

// ---------------------------------------------------------------------------
// Parse — nonexistent file
// ---------------------------------------------------------------------------

func TestParse_NonexistentFile(t *testing.T) {
	_, err := Parse("/nonexistent/path/to/file.conf")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

// ---------------------------------------------------------------------------
// Load — error paths
// ---------------------------------------------------------------------------

func TestLoad_Error_BadFile(t *testing.T) {
	type cfg struct {
		Host string `ini:"host"`
	}
	_, err := Load[cfg]("/nonexistent/path/to/file.conf", "")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestLoad_Error_BadSection(t *testing.T) {
	type cfg struct {
		Host string `ini:"host"`
	}
	_, err := Load[cfg](unitPath("01_blank.conf"), "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent section")
	}
}

// ---------------------------------------------------------------------------
// LoadInto — error paths
// ---------------------------------------------------------------------------

func TestLoadInto_Error_BadFile(t *testing.T) {
	type cfg struct {
		Host string `ini:"host"`
	}
	err := LoadInto("/nonexistent/path/to/file.conf", "", &cfg{})
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestLoadInto_Error_BadSection(t *testing.T) {
	type cfg struct {
		Host string `ini:"host"`
	}
	err := LoadInto(unitPath("01_blank.conf"), "nonexistent", &cfg{})
	if err == nil {
		t.Fatal("expected error for nonexistent section")
	}
}
