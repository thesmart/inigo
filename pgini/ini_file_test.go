package pgini

import (
	"testing"
)

// ---------------------------------------------------------------------------
// NewIniFile
// ---------------------------------------------------------------------------

func TestNewIniFile(t *testing.T) {
	p := nonExistingPath("app.conf")
	f, err := NewIniFile(p)
	if err != nil {
		t.Fatalf("NewIniFile: unexpected error: %v", err)
	}
	if f.Path != p {
		t.Errorf("Path = %q, want %q", f.Path, p)
	}
	if f.Name != "app.conf" {
		t.Errorf("Name = %q, want %q", f.Name, "app.conf")
	}
	// Default section is created automatically.
	def := f.GetSection("")
	if def == nil {
		t.Fatal("default section should exist after NewIniFile")
	}
	if def.Name != "" {
		t.Errorf("default section Name = %q, want empty string", def.Name)
	}
}

// ---------------------------------------------------------------------------
// IniFile.GetSection
// ---------------------------------------------------------------------------

func TestIniFile_GetSection_CaseInsensitive(t *testing.T) {
	f, err := NewIniFile(nonExistingPath("f.conf"))
	if err != nil {
		t.Fatal(err)
	}
	f.AddSection("Database")

	tests := []struct {
		lookup string
		want   string
	}{
		{"database", "database"},
		{"DATABASE", "database"},
		{"Database", "database"},
	}
	for _, tt := range tests {
		s := f.GetSection(tt.lookup)
		if s == nil {
			t.Errorf("GetSection(%q) returned nil", tt.lookup)
			continue
		}
		if s.Name != tt.want {
			t.Errorf("GetSection(%q).Name = %q, want %q", tt.lookup, s.Name, tt.want)
		}
	}
}

func TestIniFile_GetSection_DefaultAlias(t *testing.T) {
	f, err := NewIniFile(nonExistingPath("f.conf"))
	if err != nil {
		t.Fatal(err)
	}
	// "default" should resolve to the default section (empty name).
	s := f.GetSection("default")
	if s == nil {
		t.Fatal("GetSection(\"default\") should return the default section")
	}
	if s.Name != "" {
		t.Errorf("Name = %q, want empty string", s.Name)
	}
}

func TestIniFile_GetSection_NotFound(t *testing.T) {
	f, err := NewIniFile(nonExistingPath("f.conf"))
	if err != nil {
		t.Fatal(err)
	}
	if s := f.GetSection("nonexistent"); s != nil {
		t.Errorf("GetSection(\"nonexistent\") = %v, want nil", s)
	}
}

// ---------------------------------------------------------------------------
// IniFile.AddSection
// ---------------------------------------------------------------------------

func TestIniFile_AddSection_ReopensExisting(t *testing.T) {
	f, err := NewIniFile(nonExistingPath("f.conf"))
	if err != nil {
		t.Fatal(err)
	}
	s1, err := f.AddSection("app")
	if err != nil {
		t.Fatal(err)
	}
	s1.SetParam("key", "val")

	s2, err := f.AddSection("APP") // case-insensitive reopen
	if err != nil {
		t.Fatal(err)
	}
	if s1 != s2 {
		t.Error("AddSection should return the same *Section when reopening")
	}
	if _, ok := s2.GetParam("key"); !ok {
		t.Error("reopened section should retain existing params")
	}
}

func TestIniFile_AddSection_DefaultAlias(t *testing.T) {
	f, err := NewIniFile(nonExistingPath("f.conf"))
	if err != nil {
		t.Fatal(err)
	}
	// "default" should reopen the default section.
	s, err := f.AddSection("default")
	if err != nil {
		t.Fatal(err)
	}
	if s.Name != "" {
		t.Errorf("Name = %q, want empty string", s.Name)
	}
}

func TestIniFile_AddSection_InvalidName(t *testing.T) {
	f, err := NewIniFile(nonExistingPath("f.conf"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.AddSection("123bad")
	if err == nil {
		t.Error("AddSection with invalid identifier should return error")
	}
}

// ---------------------------------------------------------------------------
// IniFile.RemoveSection
// ---------------------------------------------------------------------------

func TestIniFile_RemoveSection(t *testing.T) {
	f, err := NewIniFile(nonExistingPath("f.conf"))
	if err != nil {
		t.Fatal(err)
	}
	f.AddSection("db")

	if !f.RemoveSection("DB") {
		t.Error("RemoveSection should return true for existing section")
	}
	if f.GetSection("db") != nil {
		t.Error("section should be gone after RemoveSection")
	}
}

func TestIniFile_RemoveSection_DefaultAlias(t *testing.T) {
	f, err := NewIniFile(nonExistingPath("f.conf"))
	if err != nil {
		t.Fatal(err)
	}
	if !f.RemoveSection("default") {
		t.Error("RemoveSection(\"default\") should remove the default section")
	}
	if f.GetSection("") != nil {
		t.Error("default section should be gone")
	}
}

func TestIniFile_RemoveSection_NotFound(t *testing.T) {
	f, err := NewIniFile(nonExistingPath("f.conf"))
	if err != nil {
		t.Fatal(err)
	}
	if f.RemoveSection("nope") {
		t.Error("RemoveSection should return false for non-existent section")
	}
}

// ---------------------------------------------------------------------------
// IniFile.Sections — insertion order
// ---------------------------------------------------------------------------

func TestIniFile_Sections_InsertionOrder(t *testing.T) {
	f, err := NewIniFile(nonExistingPath("f.conf"))
	if err != nil {
		t.Fatal(err)
	}
	f.AddSection("beta")
	f.AddSection("alpha")

	want := []string{"", "beta", "alpha"}
	var got []string
	for _, s := range f.Sections() {
		got = append(got, s.Name)
	}
	if len(got) != len(want) {
		t.Fatalf("Sections count = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("Sections()[%d].Name = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestIniFile_Sections_EarlyBreak(t *testing.T) {
	f, err := NewIniFile(nonExistingPath("f.conf"))
	if err != nil {
		t.Fatal(err)
	}
	f.AddSection("a")
	f.AddSection("b")

	count := 0
	for _, _ = range f.Sections() {
		count++
		if count == 2 {
			break
		}
	}
	if count != 2 {
		t.Errorf("early break: count = %d, want 2", count)
	}
}

// ---------------------------------------------------------------------------
// IniFile.String
// ---------------------------------------------------------------------------

func TestIniFile_String(t *testing.T) {
	f, err := NewIniFile(nonExistingPath("app.conf"))
	if err != nil {
		t.Fatal(err)
	}
	f.AddSection("db")
	got := f.String()
	want := `IniFile("app.conf", 2 sections)`
	if got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}

// ---------------------------------------------------------------------------
// IniFile.MarshalIni
// ---------------------------------------------------------------------------

func TestIniFile_MarshalIni_DefaultOnly(t *testing.T) {
	f, err := NewIniFile(nonExistingPath("f.conf"))
	if err != nil {
		t.Fatal(err)
	}
	def := f.GetSection("")
	def.SetParam("host", "localhost")
	def.SetParam("port", "5432")

	out, err := f.MarshalIni()
	if err != nil {
		t.Fatal(err)
	}
	want := "host = localhost\nport = 5432\n"
	if string(out) != want {
		t.Errorf("MarshalIni =\n%s\nwant:\n%s", out, want)
	}
}

func TestIniFile_MarshalIni_MultipleSections(t *testing.T) {
	f, err := NewIniFile(nonExistingPath("f.conf"))
	if err != nil {
		t.Fatal(err)
	}
	def := f.GetSection("")
	def.SetParam("global", "true")

	db, _ := f.AddSection("db")
	db.SetParam("host", "localhost")

	out, err := f.MarshalIni()
	if err != nil {
		t.Fatal(err)
	}
	want := "global = true\n\n[db]\nhost = localhost\n"
	if string(out) != want {
		t.Errorf("MarshalIni =\n%s\nwant:\n%s", out, want)
	}
}

// ---------------------------------------------------------------------------
// NewSection
// ---------------------------------------------------------------------------

func TestNewSection_Valid(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"app", "app"},
		{"APP", "app"},
		{"_private", "_private"},
		{"section123", "section123"},
		{"default", ""},
		{"DEFAULT", ""},
		{"", ""},
	}
	for _, tt := range tests {
		s, err := NewSection(tt.input)
		if err != nil {
			t.Errorf("NewSection(%q): unexpected error: %v", tt.input, err)
			continue
		}
		if s.Name != tt.want {
			t.Errorf("NewSection(%q).Name = %q, want %q", tt.input, s.Name, tt.want)
		}
	}
}

func TestNewSection_Invalid(t *testing.T) {
	invalid := []string{"123abc", "no spaces", "no-dash", "has.dot", ""}
	// Remove empty string — it's valid (default section).
	invalid = []string{"123abc", "no spaces", "no-dash", "has.dot"}
	for _, name := range invalid {
		_, err := NewSection(name)
		if err == nil {
			t.Errorf("NewSection(%q): expected error for invalid identifier", name)
		}
	}
}

// ---------------------------------------------------------------------------
// Section.SetParam
// ---------------------------------------------------------------------------

func TestSection_SetParam_New(t *testing.T) {
	s, _ := NewSection("app")
	p, err := s.SetParam("Host", "localhost")
	if err != nil {
		t.Fatal(err)
	}
	if p.Name != "host" {
		t.Errorf("Name = %q, want %q", p.Name, "host")
	}
	if p.Value != "localhost" {
		t.Errorf("Value = %q, want %q", p.Value, "localhost")
	}
}

func TestSection_SetParam_Overwrite(t *testing.T) {
	s, _ := NewSection("app")
	s.SetParam("key", "old")
	p, err := s.SetParam("KEY", "new")
	if err != nil {
		t.Fatal(err)
	}
	if p.Value != "new" {
		t.Errorf("Value = %q, want %q (last wins)", p.Value, "new")
	}
	// Should not duplicate in paramOrder.
	count := 0
	for range s.Params() {
		count++
	}
	if count != 1 {
		t.Errorf("param count = %d, want 1 (no duplicate)", count)
	}
}

func TestSection_SetParam_InvalidKey(t *testing.T) {
	s, _ := NewSection("app")
	_, err := s.SetParam("123bad", "val")
	if err == nil {
		t.Error("SetParam with invalid key should return error")
	}
}

// ---------------------------------------------------------------------------
// Section.GetParam / Section.GetValue
// ---------------------------------------------------------------------------

func TestSection_GetParam_CaseInsensitive(t *testing.T) {
	s, _ := NewSection("app")
	s.SetParam("host", "localhost")

	p, ok := s.GetParam("HOST")
	if !ok || p == nil {
		t.Fatal("GetParam(\"HOST\") should find param set as \"host\"")
	}
	if p.Value != "localhost" {
		t.Errorf("Value = %q, want %q", p.Value, "localhost")
	}
}

func TestSection_GetParam_NotFound(t *testing.T) {
	s, _ := NewSection("app")
	p, ok := s.GetParam("nope")
	if ok || p != nil {
		t.Error("GetParam should return nil, false for missing key")
	}
}

func TestSection_GetValue(t *testing.T) {
	s, _ := NewSection("app")
	s.SetParam("port", "5432")

	val, ok := s.GetValue("PORT")
	if !ok {
		t.Fatal("GetValue should find param")
	}
	if val != "5432" {
		t.Errorf("GetValue = %q, want %q", val, "5432")
	}
}

func TestSection_GetValue_NotFound(t *testing.T) {
	s, _ := NewSection("app")
	val, ok := s.GetValue("nope")
	if ok {
		t.Error("GetValue should return false for missing key")
	}
	if val != "" {
		t.Errorf("GetValue = %q, want empty string", val)
	}
}

// ---------------------------------------------------------------------------
// Section.RemoveParam
// ---------------------------------------------------------------------------

func TestSection_RemoveParam(t *testing.T) {
	s, _ := NewSection("app")
	s.SetParam("host", "localhost")
	s.SetParam("port", "5432")

	if !s.RemoveParam("HOST") {
		t.Error("RemoveParam should return true for existing param")
	}
	if _, ok := s.GetParam("host"); ok {
		t.Error("param should be gone after RemoveParam")
	}
	// Remaining param should still be accessible.
	if _, ok := s.GetParam("port"); !ok {
		t.Error("other params should be unaffected")
	}
}

func TestSection_RemoveParam_NotFound(t *testing.T) {
	s, _ := NewSection("app")
	if s.RemoveParam("nope") {
		t.Error("RemoveParam should return false for non-existent param")
	}
}

// ---------------------------------------------------------------------------
// Section.Params — insertion order
// ---------------------------------------------------------------------------

func TestSection_Params_InsertionOrder(t *testing.T) {
	s, _ := NewSection("app")
	s.SetParam("zeta", "3")
	s.SetParam("alpha", "1")
	s.SetParam("mu", "2")

	want := []string{"zeta", "alpha", "mu"}
	var got []string
	for _, p := range s.Params() {
		got = append(got, p.Name)
	}
	if len(got) != len(want) {
		t.Fatalf("Params count = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("Params()[%d].Name = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestSection_Params_EarlyBreak(t *testing.T) {
	s, _ := NewSection("app")
	s.SetParam("a", "1")
	s.SetParam("b", "2")
	s.SetParam("c", "3")

	count := 0
	for _, _ = range s.Params() {
		count++
		if count == 1 {
			break
		}
	}
	if count != 1 {
		t.Errorf("early break: count = %d, want 1", count)
	}
}

// ---------------------------------------------------------------------------
// Section.String
// ---------------------------------------------------------------------------

func TestSection_String(t *testing.T) {
	s, _ := NewSection("db")
	s.SetParam("host", "localhost")
	got := s.String()
	want := `Section("db", 1 params)`
	if got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}

func TestSection_String_Default(t *testing.T) {
	s, _ := NewSection("")
	got := s.String()
	want := `Section("(default)", 0 params)`
	if got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}

// ---------------------------------------------------------------------------
// Section.MarshalIni
// ---------------------------------------------------------------------------

func TestSection_MarshalIni_Default(t *testing.T) {
	s, _ := NewSection("")
	s.SetParam("host", "localhost")
	out, err := s.MarshalIni()
	if err != nil {
		t.Fatal(err)
	}
	// Default section omits the header.
	want := "host = localhost\n"
	if string(out) != want {
		t.Errorf("MarshalIni =\n%s\nwant:\n%s", out, want)
	}
}

func TestSection_MarshalIni_Named(t *testing.T) {
	s, _ := NewSection("db")
	s.SetParam("host", "localhost")
	s.SetParam("port", "5432")
	out, err := s.MarshalIni()
	if err != nil {
		t.Fatal(err)
	}
	want := "[db]\nhost = localhost\nport = 5432\n"
	if string(out) != want {
		t.Errorf("MarshalIni =\n%s\nwant:\n%s", out, want)
	}
}

func TestSection_MarshalIni_InvalidName(t *testing.T) {
	// Manually construct a Section with an invalid name to test the guard.
	s := &Section{
		Name:       "123bad",
		params:     make(map[string]*Param),
		paramOrder: nil,
	}
	_, err := s.MarshalIni()
	if err == nil {
		t.Error("MarshalIni should return error for invalid section name")
	}
}

// ---------------------------------------------------------------------------
// NewParam
// ---------------------------------------------------------------------------

func TestNewParam_Valid(t *testing.T) {
	p, err := NewParam("Host", "localhost")
	if err != nil {
		t.Fatal(err)
	}
	if p.Name != "host" {
		t.Errorf("Name = %q, want %q", p.Name, "host")
	}
	if p.Value != "localhost" {
		t.Errorf("Value = %q, want %q", p.Value, "localhost")
	}
}

func TestNewParam_Invalid(t *testing.T) {
	_, err := NewParam("123bad", "val")
	if err == nil {
		t.Error("NewParam with invalid key should return error")
	}
}

// ---------------------------------------------------------------------------
// Param.String
// ---------------------------------------------------------------------------

func TestParam_String(t *testing.T) {
	p := &Param{Name: "host", Value: "localhost"}
	got := p.String()
	want := `Param("host", "localhost")`
	if got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}

// ---------------------------------------------------------------------------
// Param.MarshalIni — unquoted vs quoted
// ---------------------------------------------------------------------------

func TestParam_MarshalIni_Unquoted(t *testing.T) {
	// Per spec: safe-char = letter | digit | [_.\-:/+]
	tests := []struct {
		name  string
		key   string
		value string
		want  string
	}{
		{"boolean true", "enabled", "true", "enabled = true"},
		{"boolean off", "debug", "off", "debug = off"},
		{"integer", "port", "5432", "port = 5432"},
		{"hex integer", "color", "0xFF", "color = 0xFF"},
		{"signed integer with unit", "mem", "+8kB", "mem = +8kB"},
		{"negative integer", "offset", "-1", "offset = -1"},
		{"float", "ratio", "1.5", "ratio = 1.5"},
		{"path-like", "dir", "/usr/local/bin", "dir = /usr/local/bin"},
		{"url-like", "endpoint", "http://localhost:8080", "endpoint = http://localhost:8080"},
		{"simple identifier", "mode", "production", "mode = production"},
		{"underscore", "log_level", "debug", "log_level = debug"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Param{Name: tt.key, Value: tt.value}
			out, err := p.MarshalIni()
			if err != nil {
				t.Fatal(err)
			}
			if string(out) != tt.want {
				t.Errorf("MarshalIni = %q, want %q", out, tt.want)
			}
		})
	}
}

func TestParam_MarshalIni_Quoted(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		value string
		want  string
	}{
		{"space in value", "greeting", "hello world", "greeting = 'hello world'"},
		{"empty value", "empty", "", "empty = ''"},
		{"backslash", "path", `c:\dir`, `path = 'c:\\dir'`},
		{"single quote", "msg", "it's", `msg = 'it\'s'`},
		{"newline", "multi", "line1\nline2", `multi = 'line1\nline2'`},
		{"tab", "tsv", "a\tb", `tsv = 'a\tb'`},
		{"carriage return", "cr", "a\rb", `cr = 'a\rb'`},
		{"backspace", "bs", "a\bb", `bs = 'a\bb'`},
		{"form feed", "ff", "a\fb", `ff = 'a\fb'`},
		{"null byte", "nul", "a\x00b", `nul = 'a\000b'`},
		{"DEL char", "del", "a\x7fb", `del = 'a\177b'`},
		{"UTF-8", "label", "café", "label = 'café'"},
		{"control char 0x01", "ctrl", "a\x01b", `ctrl = 'a\001b'`},
		{"control char 0x1F", "ctrlend", "a\x1fb", `ctrlend = 'a\037b'`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Param{Name: tt.key, Value: tt.value}
			out, err := p.MarshalIni()
			if err != nil {
				t.Fatal(err)
			}
			if string(out) != tt.want {
				t.Errorf("MarshalIni = %q, want %q", out, tt.want)
			}
		})
	}
}

func TestParam_MarshalIni_InvalidKey(t *testing.T) {
	p := &Param{Name: "123bad", Value: "val"}
	_, err := p.MarshalIni()
	if err == nil {
		t.Error("MarshalIni should return error for invalid key")
	}
}

// ---------------------------------------------------------------------------
// pginiEscape — private helper
// ---------------------------------------------------------------------------

func TestPginiEscape(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"empty", "", ""},
		{"plain text", "hello", "hello"},
		{"backslash", `a\b`, `a\\b`},
		{"single quote", "it's", `it\'s`},
		{"backspace", "\b", `\b`},
		{"form feed", "\f", `\f`},
		{"newline", "\n", `\n`},
		{"carriage return", "\r", `\r`},
		{"tab", "\t", `\t`},
		{"null", "\x00", `\000`},
		{"control 0x01", "\x01", `\001`},
		{"control 0x1F", "\x1f", `\037`},
		{"DEL 0x7F", "\x7f", `\177`},
		{"UTF-8 passthrough", "日本語", "日本語"},
		{"mixed", "line1\nline2\\end", `line1\nline2\\end`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pginiEscape(tt.input)
			if got != tt.want {
				t.Errorf("pginiEscape(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// identifierRe — regex coverage
// ---------------------------------------------------------------------------

func TestIdentifierRe(t *testing.T) {
	valid := []string{"a", "Z", "_", "_a1", "abc123", "A_B_C"}
	for _, s := range valid {
		if !identifierRe.MatchString(s) {
			t.Errorf("identifierRe should match %q", s)
		}
	}
	invalid := []string{"", "1abc", "-x", "a b", "a.b", "a-b"}
	for _, s := range invalid {
		if identifierRe.MatchString(s) {
			t.Errorf("identifierRe should not match %q", s)
		}
	}
}

// ---------------------------------------------------------------------------
// IniFile.MarshalIni — error propagation from Section
// ---------------------------------------------------------------------------

func TestIniFile_MarshalIni_ErrorPropagation(t *testing.T) {
	// Construct a file with an invalid section name to trigger marshal error.
	f, err := NewIniFile(nonExistingPath("f.conf"))
	if err != nil {
		t.Fatal(err)
	}
	// Inject an invalid section directly.
	f.sections["bad!"] = &Section{
		Name:       "bad!",
		params:     make(map[string]*Param),
		paramOrder: nil,
	}
	f.sectionOrder = append(f.sectionOrder, "bad!")

	_, err = f.MarshalIni()
	if err == nil {
		t.Error("MarshalIni should propagate Section marshal errors")
	}
}

// ---------------------------------------------------------------------------
// Section.MarshalIni — error propagation from Param
// ---------------------------------------------------------------------------

func TestSection_MarshalIni_ParamErrorPropagation(t *testing.T) {
	s, _ := NewSection("app")
	// Inject a param with an invalid key directly.
	s.params["bad!"] = &Param{Name: "bad!", Value: "val"}
	s.paramOrder = append(s.paramOrder, "bad!")

	_, err := s.MarshalIni()
	if err == nil {
		t.Error("MarshalIni should propagate Param marshal errors")
	}
}
