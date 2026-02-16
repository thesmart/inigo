package inigo

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type dbConfig struct {
	Host     string  `ini:"host"`
	Port     int     `ini:"port"`
	DbName   string  `ini:"dbname"`
	SslMode  string  `ini:"sslmode"`
	Timeout  float64 `ini:"connect_timeout"`
	UsePool  bool    `ini:"use_pool"`
	MaxConns uint16  `ini:"max_conns"`
}

func TestApplyIntoBasic(t *testing.T) {
	cfg := mustParse(t, `
[mydb]
host = db.example.com
port = 5432
dbname = production
sslmode = require
connect_timeout = 30.5
use_pool = on
max_conns = 100
`)
	var c dbConfig
	if err := ApplyInto(cfg, "mydb", &c); err != nil {
		t.Fatalf("ApplyInto: %v", err)
	}
	if c.Host != "db.example.com" {
		t.Errorf("Host = %q", c.Host)
	}
	if c.Port != 5432 {
		t.Errorf("Port = %d", c.Port)
	}
	if c.DbName != "production" {
		t.Errorf("DbName = %q", c.DbName)
	}
	if c.SslMode != "require" {
		t.Errorf("SslMode = %q", c.SslMode)
	}
	if c.Timeout != 30.5 {
		t.Errorf("Timeout = %f", c.Timeout)
	}
	if !c.UsePool {
		t.Error("UsePool = false, want true")
	}
	if c.MaxConns != 100 {
		t.Errorf("MaxConns = %d", c.MaxConns)
	}
}

func TestApplyIntoMissingParams(t *testing.T) {
	cfg := mustParse(t, `
[svc]
host = localhost
`)
	var c dbConfig
	if err := ApplyInto(cfg, "svc", &c); err != nil {
		t.Fatalf("ApplyInto: %v", err)
	}
	if c.Host != "localhost" {
		t.Errorf("Host = %q", c.Host)
	}
	// Unset fields should remain at zero values
	if c.Port != 0 {
		t.Errorf("Port = %d, want 0", c.Port)
	}
	if c.DbName != "" {
		t.Errorf("DbName = %q, want empty", c.DbName)
	}
}

func TestApplyIntoUntaggedFieldsSkipped(t *testing.T) {
	type simple struct {
		Host string
		Port int
	}
	cfg := mustParse(t, `
host = localhost
port = 3000
`)
	var s simple
	if err := ApplyInto(cfg, "", &s); err != nil {
		t.Fatalf("ApplyInto: %v", err)
	}
	// Fields without ini tags should remain at zero values
	if s.Host != "" {
		t.Errorf("Host = %q, want empty (untagged field should be skipped)", s.Host)
	}
	if s.Port != 0 {
		t.Errorf("Port = %d, want 0 (untagged field should be skipped)", s.Port)
	}
}

func TestApplyIntoSkipTag(t *testing.T) {
	type withSkip struct {
		Host    string `ini:"host"`
		Ignored string `ini:"-"`
	}
	cfg := mustParse(t, `
host = localhost
ignored = should_not_appear
`)
	var s withSkip
	if err := ApplyInto(cfg, "", &s); err != nil {
		t.Fatalf("ApplyInto: %v", err)
	}
	if s.Host != "localhost" {
		t.Errorf("Host = %q", s.Host)
	}
	if s.Ignored != "" {
		t.Errorf("Ignored = %q, want empty", s.Ignored)
	}
}

func TestApplyIntoUnexportedFieldsSkipped(t *testing.T) {
	type withUnexported struct {
		Host    string `ini:"host"`
		private string //nolint:unused
	}
	cfg := mustParse(t, `host = localhost`)
	var s withUnexported
	if err := ApplyInto(cfg, "", &s); err != nil {
		t.Fatalf("ApplyInto: %v", err)
	}
	if s.Host != "localhost" {
		t.Errorf("Host = %q", s.Host)
	}
}

func TestApplyIntoIntTypes(t *testing.T) {
	type intTypes struct {
		I   int   `ini:"i"`
		I8  int8  `ini:"i8"`
		I16 int16 `ini:"i16"`
		I32 int32 `ini:"i32"`
		I64 int64 `ini:"i64"`
	}
	cfg := mustParse(t, `
i = 42
i8 = 127
i16 = 32000
i32 = 100000
i64 = 9999999999
`)
	var s intTypes
	if err := ApplyInto(cfg, "", &s); err != nil {
		t.Fatalf("ApplyInto: %v", err)
	}
	if s.I != 42 || s.I8 != 127 || s.I16 != 32000 || s.I32 != 100000 || s.I64 != 9999999999 {
		t.Errorf("got %+v", s)
	}
}

func TestApplyIntoUintTypes(t *testing.T) {
	type uintTypes struct {
		U   uint   `ini:"u"`
		U8  uint8  `ini:"u8"`
		U16 uint16 `ini:"u16"`
		U32 uint32 `ini:"u32"`
		U64 uint64 `ini:"u64"`
	}
	cfg := mustParse(t, `
u = 42
u8 = 255
u16 = 65000
u32 = 100000
u64 = 9999999999
`)
	var s uintTypes
	if err := ApplyInto(cfg, "", &s); err != nil {
		t.Fatalf("ApplyInto: %v", err)
	}
	if s.U != 42 || s.U8 != 255 || s.U16 != 65000 || s.U32 != 100000 || s.U64 != 9999999999 {
		t.Errorf("got %+v", s)
	}
}

func TestApplyIntoFloatTypes(t *testing.T) {
	type floatTypes struct {
		F32 float32 `ini:"f32"`
		F64 float64 `ini:"f64"`
	}
	cfg := mustParse(t, `
f32 = 3.14
f64 = 2.718281828
`)
	var s floatTypes
	if err := ApplyInto(cfg, "", &s); err != nil {
		t.Fatalf("ApplyInto: %v", err)
	}
	if s.F64 != 2.718281828 {
		t.Errorf("F64 = %f", s.F64)
	}
}

func TestApplyIntoErrorNonPointer(t *testing.T) {
	cfg := mustParse(t, `host = x`)
	var s struct{ Host string }
	if err := ApplyInto(cfg, "", s); err == nil {
		t.Fatal("expected error for non-pointer target")
	}
}

func TestApplyIntoErrorNilPointer(t *testing.T) {
	cfg := mustParse(t, `host = x`)
	var s *struct{ Host string }
	if err := ApplyInto(cfg, "", s); err == nil {
		t.Fatal("expected error for nil pointer")
	}
}

func TestApplyIntoErrorNotStruct(t *testing.T) {
	cfg := mustParse(t, `host = x`)
	var s string
	if err := ApplyInto(cfg, "", &s); err == nil {
		t.Fatal("expected error for non-struct pointer")
	}
}

func TestApplyIntoErrorMissingSection(t *testing.T) {
	cfg := mustParse(t, `host = x`)
	var s struct{ Host string }
	if err := ApplyInto(cfg, "nonexistent", &s); err == nil {
		t.Fatal("expected error for missing section")
	}
}

func TestApplyIntoErrorBadBool(t *testing.T) {
	cfg := mustParse(t, `flag = notabool`)
	var s struct {
		Flag bool `ini:"flag"`
	}
	if err := ApplyInto(cfg, "", &s); err == nil {
		t.Fatal("expected error for invalid bool value")
	}
}

func TestApplyIntoErrorBadInt(t *testing.T) {
	cfg := mustParse(t, `count = notanumber`)
	var s struct {
		Count int `ini:"count"`
	}
	if err := ApplyInto(cfg, "", &s); err == nil {
		t.Fatal("expected error for invalid int value")
	}
}

func TestApplyIntoErrorNegativeUint(t *testing.T) {
	cfg := mustParse(t, `count = -1`)
	var s struct {
		Count uint `ini:"count"`
	}
	if err := ApplyInto(cfg, "", &s); err == nil {
		t.Fatal("expected error for negative uint")
	}
}

func TestApplyIntoErrorUnsupportedType(t *testing.T) {
	cfg := mustParse(t, `data = hello`)
	var s struct {
		Data []string `ini:"data"`
	}
	if err := ApplyInto(cfg, "", &s); err == nil {
		t.Fatal("expected error for unsupported slice type")
	}
}

func TestLoadInto(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "test.conf"), []byte(`
[mydb]
host = db.local
port = 5432
`), 0o644)

	var c dbConfig
	if err := LoadInto(filepath.Join(dir, "test.conf"), "mydb", &c); err != nil {
		t.Fatalf("LoadInto: %v", err)
	}
	if c.Host != "db.local" {
		t.Errorf("Host = %q", c.Host)
	}
	if c.Port != 5432 {
		t.Errorf("Port = %d", c.Port)
	}
}

func TestLoadIntoErrorMissingFile(t *testing.T) {
	var c dbConfig
	if err := LoadInto("/tmp/nonexistent_alter_test.conf", "", &c); err == nil {
		t.Fatal("expected error for missing file")
	}
}

// --- Marshal / SaveFrom tests ---

func TestMarshalBasic(t *testing.T) {
	c := &dbConfig{
		Host:     "db.example.com",
		Port:     5432,
		DbName:   "production",
		SslMode:  "require",
		Timeout:  30.5,
		UsePool:  true,
		MaxConns: 100,
	}
	out, err := Marshal(c, "mydb")
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	// Verify section header
	if !strings.HasPrefix(out, "[mydb]\n") {
		t.Errorf("missing section header, got:\n%s", out)
	}
	// Verify key params are present
	for _, want := range []string{
		"host = db.example.com",
		"port = 5432",
		"dbname = production",
		"sslmode = require",
		"connect_timeout = 30.5",
		"use_pool = on",
		"max_conns = 100",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q, got:\n%s", want, out)
		}
	}
}

func TestMarshalNoSection(t *testing.T) {
	c := &struct {
		Host string `ini:"host"`
	}{Host: "localhost"}
	out, err := Marshal(c, "")
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if strings.Contains(out, "[") {
		t.Errorf("expected no section header, got:\n%s", out)
	}
	if !strings.Contains(out, "host = localhost") {
		t.Errorf("missing host param, got:\n%s", out)
	}
}

func TestMarshalZeroFieldsSkipped(t *testing.T) {
	c := &dbConfig{
		Host: "localhost",
		// All other fields are zero
	}
	out, err := Marshal(c, "svc")
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if strings.Contains(out, "port") {
		t.Errorf("zero-value port should be skipped, got:\n%s", out)
	}
	if strings.Contains(out, "use_pool") {
		t.Errorf("zero-value use_pool should be skipped, got:\n%s", out)
	}
}

func TestMarshalSkipTag(t *testing.T) {
	type withSkip struct {
		Host   string `ini:"host"`
		Secret string `ini:"-"`
	}
	c := &withSkip{Host: "localhost", Secret: "hidden"}
	out, err := Marshal(c, "")
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if strings.Contains(out, "hidden") {
		t.Errorf("skipped field should not appear, got:\n%s", out)
	}
}

func TestMarshalUntaggedFieldsSkipped(t *testing.T) {
	type simple struct {
		Host string
		Port int
	}
	c := &simple{Host: "localhost", Port: 8080}
	out, err := Marshal(c, "")
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	// Untagged fields should not appear in output
	if strings.Contains(out, "host") {
		t.Errorf("untagged Host should be skipped, got:\n%s", out)
	}
	if strings.Contains(out, "port") {
		t.Errorf("untagged Port should be skipped, got:\n%s", out)
	}
}

func TestMarshalQuotesSpecialChars(t *testing.T) {
	type quoted struct {
		Desc   string `ini:"desc"`
		Path   string `ini:"path"`
		Simple string `ini:"simple"`
	}
	c := &quoted{
		Desc:   "hello world",
		Path:   "a=b#c",
		Simple: "noquotes",
	}
	out, err := Marshal(c, "")
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(out, "desc = 'hello world'") {
		t.Errorf("space value should be quoted, got:\n%s", out)
	}
	if !strings.Contains(out, "path = 'a=b#c'") {
		t.Errorf("special chars should be quoted, got:\n%s", out)
	}
	if !strings.Contains(out, "simple = noquotes") {
		t.Errorf("simple value should not be quoted, got:\n%s", out)
	}
}

func TestMarshalEmbeddedQuotes(t *testing.T) {
	type q struct {
		Val string `ini:"val"`
	}
	c := &q{Val: "it's"}
	out, err := Marshal(c, "")
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(out, "val = 'it''s'") {
		t.Errorf("embedded quote should be doubled, got:\n%s", out)
	}
}

func TestMarshalBoolValues(t *testing.T) {
	type bools struct {
		Enabled  bool `ini:"enabled"`
		Disabled bool `ini:"disabled"`
	}
	c := &bools{Enabled: true, Disabled: false}
	out, err := Marshal(c, "")
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(out, "enabled = on") {
		t.Errorf("true should be 'on', got:\n%s", out)
	}
	// false is zero value, should be skipped
	if strings.Contains(out, "disabled") {
		t.Errorf("false bool (zero value) should be skipped, got:\n%s", out)
	}
}

func TestMarshalErrorNonPointer(t *testing.T) {
	s := struct{ Host string }{Host: "x"}
	if _, err := Marshal(s, ""); err == nil {
		t.Fatal("expected error for non-pointer")
	}
}

func TestMarshalErrorNilPointer(t *testing.T) {
	var s *struct{ Host string }
	if _, err := Marshal(s, ""); err == nil {
		t.Fatal("expected error for nil pointer")
	}
}

func TestMarshalErrorNotStruct(t *testing.T) {
	s := "hello"
	if _, err := Marshal(&s, ""); err == nil {
		t.Fatal("expected error for non-struct pointer")
	}
}

func TestMarshalErrorUnsupportedType(t *testing.T) {
	type bad struct {
		Data []string `ini:"data"`
	}
	c := &bad{Data: []string{"a"}}
	if _, err := Marshal(c, ""); err == nil {
		t.Fatal("expected error for unsupported slice type")
	}
}

func TestSaveFrom(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.conf")

	c := &dbConfig{
		Host: "db.local",
		Port: 5432,
	}
	if err := SaveFrom(c, "mydb", path); err != nil {
		t.Fatalf("SaveFrom: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "[mydb]") {
		t.Errorf("missing section header")
	}
	if !strings.Contains(content, "host = db.local") {
		t.Errorf("missing host")
	}
	if !strings.Contains(content, "port = 5432") {
		t.Errorf("missing port")
	}
}

func TestSaveFromRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "roundtrip.conf")

	original := &dbConfig{
		Host:     "db.example.com",
		Port:     5432,
		DbName:   "myapp",
		SslMode:  "require",
		Timeout:  15,
		UsePool:  true,
		MaxConns: 50,
	}
	if err := SaveFrom(original, "svc", path); err != nil {
		t.Fatalf("SaveFrom: %v", err)
	}

	var loaded dbConfig
	if err := LoadInto(path, "svc", &loaded); err != nil {
		t.Fatalf("LoadInto: %v", err)
	}

	if loaded != *original {
		t.Errorf("round trip mismatch:\n  original: %+v\n  loaded:   %+v", *original, loaded)
	}
}

// helper shared with ini_integration_test.go
func mustParseStr(t *testing.T, input string) *Config {
	t.Helper()
	cfg, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	return cfg
}
