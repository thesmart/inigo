package ini

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

type BasicConfig struct {
	Host    string  `ini:"host"`
	Port    int     `ini:"port"`
	Debug   bool    `ini:"debug"`
	Rate    float64 `ini:"rate"`
	Name    string  `ini:"name"`
	Skipped string  // no tag, should be skipped
}

type IgnoredFieldConfig struct {
	Host   string `ini:"host"`
	Secret string `ini:""`
	Port   int    `ini:"port"`
}

func TestUnmarshalIniString_IgnoredField(t *testing.T) {
	// ini:"" should cause the field to be skipped, even when the file
	// contains a "secret" key that would match by name.
	contents := "host = localhost\nsecret = hunter2\nport = 5432\n"
	cfg, err := UnmarshalIniString[IgnoredFieldConfig]("test.conf", "", contents)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Secret != "" {
		t.Errorf("expected Secret to remain zero-value, got %q", cfg.Secret)
	}
	if cfg.Host != "localhost" {
		t.Errorf("expected host=localhost, got %q", cfg.Host)
	}
	if cfg.Port != 5432 {
		t.Errorf("expected port=5432, got %d", cfg.Port)
	}
}

func TestUnmarshalIniString_Basic(t *testing.T) {
	contents := `host = localhost
port = 5432
debug = on
rate = 1.5
name = 'my app'
`
	cfg, err := UnmarshalIniString[BasicConfig]("test.conf", "", contents)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Host != "localhost" {
		t.Errorf("expected host=localhost, got %q", cfg.Host)
	}
	if cfg.Port != 5432 {
		t.Errorf("expected port=5432, got %d", cfg.Port)
	}
	if !cfg.Debug {
		t.Error("expected debug=true")
	}
	if cfg.Rate != 1.5 {
		t.Errorf("expected rate=1.5, got %f", cfg.Rate)
	}
	if cfg.Name != "my app" {
		t.Errorf("expected name='my app', got %q", cfg.Name)
	}
}

func TestUnmarshalIniString_Section(t *testing.T) {
	contents := `
[database]
host = dbhost
port = 3306

[app]
host = apphost
port = 8080
`
	cfg, err := UnmarshalIniString[BasicConfig]("test.conf", "database", contents)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Host != "dbhost" {
		t.Errorf("expected host=dbhost, got %q", cfg.Host)
	}
	if cfg.Port != 3306 {
		t.Errorf("expected port=3306, got %d", cfg.Port)
	}
}

func TestUnmarshalIniString_SectionNotFound(t *testing.T) {
	contents := `host = localhost`
	_, err := UnmarshalIniString[BasicConfig]("test.conf", "nonexistent", contents)
	if err == nil {
		t.Error("expected error for missing section")
	}
}

func TestUnmarshalIniString_CaseInsensitive(t *testing.T) {
	// Section names and parameter names are case-insensitive
	contents := `
[DATABASE]
HOST = myhost
PORT = 1234
`
	cfg, err := UnmarshalIniString[BasicConfig]("test.conf", "database", contents)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Host != "myhost" {
		t.Errorf("expected host=myhost, got %q", cfg.Host)
	}
}

func TestUnmarshalIniString_DuplicateKeys(t *testing.T) {
	// Last occurrence wins
	contents := `port = 1111
port = 2222
`
	cfg, err := UnmarshalIniString[BasicConfig]("test.conf", "", contents)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Port != 2222 {
		t.Errorf("expected port=2222, got %d", cfg.Port)
	}
}

type IntTypes struct {
	I   int    `ini:"i"`
	I8  int8   `ini:"i8"`
	I16 int16  `ini:"i16"`
	I32 int32  `ini:"i32"`
	I64 int64  `ini:"i64"`
	U   uint   `ini:"u"`
	U8  uint8  `ini:"u8"`
	U16 uint16 `ini:"u16"`
	U32 uint32 `ini:"u32"`
	U64 uint64 `ini:"u64"`
}

func TestUnmarshalIniString_IntTypes(t *testing.T) {
	contents := `i = 42
i8 = 127
i16 = 32000
i32 = 100000
i64 = 999999999
u = 10
u8 = 255
u16 = 65000
u32 = 100000
u64 = 999999999
`
	cfg, err := UnmarshalIniString[IntTypes]("test.conf", "", contents)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.I != 42 {
		t.Errorf("expected i=42, got %d", cfg.I)
	}
	if cfg.U8 != 255 {
		t.Errorf("expected u8=255, got %d", cfg.U8)
	}
}

func TestUnmarshalIniString_IntBoundaryValues(t *testing.T) {
	// Test that exact min and max values are accepted for every integer type
	contents := fmt.Sprintf(`i8 = %d
i16 = %d
i32 = %d
i64 = %d
`, math.MinInt8, math.MinInt16, math.MinInt32, math.MinInt64)
	cfg, err := UnmarshalIniString[IntTypes]("test.conf", "", contents)
	if err != nil {
		t.Fatalf("unexpected error for min signed values: %v", err)
	}
	if cfg.I8 != math.MinInt8 {
		t.Errorf("expected i8=%d, got %d", math.MinInt8, cfg.I8)
	}
	if cfg.I16 != math.MinInt16 {
		t.Errorf("expected i16=%d, got %d", math.MinInt16, cfg.I16)
	}
	if cfg.I32 != math.MinInt32 {
		t.Errorf("expected i32=%d, got %d", math.MinInt32, cfg.I32)
	}
	if cfg.I64 != math.MinInt64 {
		t.Errorf("expected i64=%d, got %d", math.MinInt64, cfg.I64)
	}

	contents = fmt.Sprintf(`i8 = %d
i16 = %d
i32 = %d
i64 = %d
`, math.MaxInt8, math.MaxInt16, math.MaxInt32, math.MaxInt64)
	cfg, err = UnmarshalIniString[IntTypes]("test.conf", "", contents)
	if err != nil {
		t.Fatalf("unexpected error for max signed values: %v", err)
	}
	if cfg.I8 != math.MaxInt8 {
		t.Errorf("expected i8=%d, got %d", math.MaxInt8, cfg.I8)
	}
	if cfg.I16 != math.MaxInt16 {
		t.Errorf("expected i16=%d, got %d", math.MaxInt16, cfg.I16)
	}
	if cfg.I32 != math.MaxInt32 {
		t.Errorf("expected i32=%d, got %d", math.MaxInt32, cfg.I32)
	}
	if cfg.I64 != math.MaxInt64 {
		t.Errorf("expected i64=%d, got %d", math.MaxInt64, cfg.I64)
	}
}

func TestUnmarshalIniString_UintBoundaryValues(t *testing.T) {
	contents := fmt.Sprintf(`u8 = %d
u16 = %d
u32 = %d
u64 = %d
`, math.MaxUint8, math.MaxUint16, math.MaxUint32, uint64(math.MaxUint64))
	cfg, err := UnmarshalIniString[IntTypes]("test.conf", "", contents)
	if err != nil {
		t.Fatalf("unexpected error for max unsigned values: %v", err)
	}
	if cfg.U8 != math.MaxUint8 {
		t.Errorf("expected u8=%d, got %d", math.MaxUint8, cfg.U8)
	}
	if cfg.U16 != math.MaxUint16 {
		t.Errorf("expected u16=%d, got %d", math.MaxUint16, cfg.U16)
	}
	if cfg.U32 != math.MaxUint32 {
		t.Errorf("expected u32=%d, got %d", math.MaxUint32, cfg.U32)
	}
	if cfg.U64 != math.MaxUint64 {
		t.Errorf("expected u64=%d, got %d", uint64(math.MaxUint64), cfg.U64)
	}
}

func TestUnmarshalIniString_FloatBoundaryValues(t *testing.T) {
	type S struct {
		Val float32 `ini:"val"`
	}
	contents := "val = " + strconv.FormatFloat(math.MaxFloat32, 'e', -1, 32)
	cfg, err := UnmarshalIniString[S]("test.conf", "", contents)
	if err != nil {
		t.Fatalf("unexpected error for float32 max: %v", err)
	}
	if cfg.Val != math.MaxFloat32 {
		t.Errorf("expected val=%v, got %v", math.MaxFloat32, cfg.Val)
	}

	contents = "val = " + strconv.FormatFloat(-math.MaxFloat32, 'e', -1, 32)
	cfg, err = UnmarshalIniString[S]("test.conf", "", contents)
	if err != nil {
		t.Fatalf("unexpected error for float32 min: %v", err)
	}
	if cfg.Val != -math.MaxFloat32 {
		t.Errorf("expected val=%v, got %v", -math.MaxFloat32, cfg.Val)
	}

	type S64 struct {
		Val float64 `ini:"val"`
	}
	contents = "val = " + strconv.FormatFloat(math.MaxFloat64, 'e', -1, 64)
	cfg64, err := UnmarshalIniString[S64]("test.conf", "", contents)
	if err != nil {
		t.Fatalf("unexpected error for float64 max: %v", err)
	}
	if cfg64.Val != math.MaxFloat64 {
		t.Errorf("expected val=%v, got %v", math.MaxFloat64, cfg64.Val)
	}

	contents = "val = " + strconv.FormatFloat(-math.MaxFloat64, 'e', -1, 64)
	cfg64, err = UnmarshalIniString[S64]("test.conf", "", contents)
	if err != nil {
		t.Fatalf("unexpected error for float64 min: %v", err)
	}
	if cfg64.Val != -math.MaxFloat64 {
		t.Errorf("expected val=%v, got %v", -math.MaxFloat64, cfg64.Val)
	}
}

func TestUnmarshalIniString_HexOctal(t *testing.T) {
	contents := `i = 0xFF
i8 = 077
`
	cfg, err := UnmarshalIniString[IntTypes]("test.conf", "", contents)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.I != 255 {
		t.Errorf("expected i=255 (0xFF), got %d", cfg.I)
	}
	if cfg.I8 != 63 {
		t.Errorf("expected i8=63 (077 octal), got %d", cfg.I8)
	}
}

func TestUnmarshalIniString_FloatFloor(t *testing.T) {
	// Integer field with float value should floor
	contents := `i = 3.7`
	cfg, err := UnmarshalIniString[IntTypes]("test.conf", "", contents)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.I != 3 {
		t.Errorf("expected i=3 (floored from 3.7), got %d", cfg.I)
	}
}

func TestUnmarshalIniString_NegativeUint(t *testing.T) {
	contents := `u = -1`
	_, err := UnmarshalIniString[IntTypes]("test.conf", "", contents)
	if err == nil {
		t.Error("expected error for negative uint")
	}
}

type BoolConfig struct {
	A bool `ini:"a"`
	B bool `ini:"b"`
	C bool `ini:"c"`
	D bool `ini:"d"`
	E bool `ini:"e"`
	F bool `ini:"f"`
}

func TestUnmarshalIniString_Booleans(t *testing.T) {
	contents := `a = on
b = off
c = true
d = false
e = 1
f = 0
`
	cfg, err := UnmarshalIniString[BoolConfig]("test.conf", "", contents)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.A {
		t.Error("expected a=true")
	}
	if cfg.B {
		t.Error("expected b=false")
	}
	if !cfg.C {
		t.Error("expected c=true")
	}
	if cfg.D {
		t.Error("expected d=false")
	}
	if !cfg.E {
		t.Error("expected e=true")
	}
	if cfg.F {
		t.Error("expected f=false")
	}
}

func TestParseBool_Prefixes(t *testing.T) {
	// Unambiguous prefixes
	tests := []struct {
		input    string
		expected bool
	}{
		{"ye", true},
		{"yes", true},
		{"tr", true},
		{"tru", true},
		{"on", true},
		{"of", false},
		{"off", false},
		{"fal", false},
		{"no", false},
		{"n", false},
	}
	for _, tt := range tests {
		v, err := parseBool(tt.input)
		if err != nil {
			t.Fatalf("unexpected error for %q: %v", tt.input, err)
		}
		if v != tt.expected {
			t.Errorf("expected %v for %q, got %v", tt.expected, tt.input, v)
		}
	}
}

func TestParseBool_Errors(t *testing.T) {
	tests := []string{"", "maybe", "o"} // "o" is ambiguous (on/off)
	for _, s := range tests {
		_, err := parseBool(s)
		if err == nil {
			t.Errorf("expected error for %q", s)
		}
	}
}

func TestParseInt_Errors(t *testing.T) {
	tests := []string{"", "abc", "not_a_number"}
	for _, s := range tests {
		_, err := parseInt(s)
		if err == nil {
			t.Errorf("expected error for %q", s)
		}
	}
}

func TestParseFloat_Errors(t *testing.T) {
	tests := []string{"", "abc"}
	for _, s := range tests {
		_, err := parseFloat(s, 64)
		if err == nil {
			t.Errorf("expected error for %q", s)
		}
	}
}

type FloatConfig struct {
	F32 float32 `ini:"f32"`
	F64 float64 `ini:"f64"`
}

func TestUnmarshalIniString_Floats(t *testing.T) {
	contents := `f32 = 3.14
f64 = 0.001
`
	cfg, err := UnmarshalIniString[FloatConfig]("test.conf", "", contents)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.F32 < 3.13 || cfg.F32 > 3.15 {
		t.Errorf("expected f32~3.14, got %f", cfg.F32)
	}
	if cfg.F64 != 0.001 {
		t.Errorf("expected f64=0.001, got %f", cfg.F64)
	}
}

func TestUnmarshalIniString_Comments(t *testing.T) {
	contents := `# this is a comment
; this too
host = localhost # inline comment
port = 5432 ; inline semicolon
`
	cfg, err := UnmarshalIniString[BasicConfig]("test.conf", "", contents)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Host != "localhost" {
		t.Errorf("expected host=localhost, got %q", cfg.Host)
	}
	if cfg.Port != 5432 {
		t.Errorf("expected port=5432, got %d", cfg.Port)
	}
}

func TestUnmarshalIniString_BlankLines(t *testing.T) {
	contents := `

host = localhost

port = 5432

`
	cfg, err := UnmarshalIniString[BasicConfig]("test.conf", "", contents)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Host != "localhost" || cfg.Port != 5432 {
		t.Errorf("unexpected values: host=%q port=%d", cfg.Host, cfg.Port)
	}
}

func TestUnmarshalIniString_QuotedValues(t *testing.T) {
	contents := `host = 'my host with spaces'
name = 'it''s a name'
`
	cfg, err := UnmarshalIniString[BasicConfig]("test.conf", "", contents)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Host != "my host with spaces" {
		t.Errorf("expected host='my host with spaces', got %q", cfg.Host)
	}
	if cfg.Name != "it's a name" {
		t.Errorf("expected name=\"it's a name\", got %q", cfg.Name)
	}
}

func TestUnmarshalIniString_ColonSeparator(t *testing.T) {
	contents := `host : localhost
port : 5432
`
	cfg, err := UnmarshalIniString[BasicConfig]("test.conf", "", contents)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Host != "localhost" || cfg.Port != 5432 {
		t.Errorf("unexpected values: host=%q port=%d", cfg.Host, cfg.Port)
	}
}

func TestUnmarshalIniString_NoSeparator(t *testing.T) {
	// Bare key with no value
	type Bare struct {
		Key string `ini:"key"`
	}
	contents := `key`
	cfg, err := UnmarshalIniString[Bare]("test.conf", "", contents)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Key != "" {
		t.Errorf("expected empty value, got %q", cfg.Key)
	}
}

// Custom unmarshal test
type CustomConfig struct {
	Port int `ini:"port"`
}

func (c *CustomConfig) Unmarshal_Port(s string) (int, error) {
	if s == "default" {
		return 5432, nil
	}
	return 0, fmt.Errorf("unknown port alias: %s", s)
}

func TestUnmarshalIniString_CustomUnmarshal(t *testing.T) {
	contents := `port = default`
	cfg, err := UnmarshalIniString[CustomConfig]("test.conf", "", contents)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Port != 5432 {
		t.Errorf("expected port=5432, got %d", cfg.Port)
	}
}

func TestUnmarshalIniString_CustomUnmarshalError(t *testing.T) {
	contents := `port = unknown`
	_, err := UnmarshalIniString[CustomConfig]("test.conf", "", contents)
	if err == nil {
		t.Error("expected error from custom unmarshal")
	}
}

// File-based tests
func TestUnmarshalIniFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.conf")
	err := os.WriteFile(path, []byte("host = filehost\nport = 9999\n"), 0o644)
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := UnmarshalIniFile[BasicConfig](path, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Host != "filehost" || cfg.Port != 9999 {
		t.Errorf("unexpected values: host=%q port=%d", cfg.Host, cfg.Port)
	}
}

func TestUnmarshalIniFile_NotFound(t *testing.T) {
	_, err := UnmarshalIniFile[BasicConfig]("/nonexistent/path.conf", "")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

type UnsupportedConfig struct {
	Data []string `ini:"data"`
}

func TestUnmarshalIniString_UnsupportedType(t *testing.T) {
	contents := `data = something`
	_, err := UnmarshalIniString[UnsupportedConfig]("test.conf", "", contents)
	if err == nil {
		t.Error("expected error for unsupported type")
	}
}

func TestUnmarshalIniString_EmptySections(t *testing.T) {
	contents := `[empty]
[notempty]
host = val
`
	cfg, err := UnmarshalIniString[BasicConfig]("test.conf", "empty", contents)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Host != "" {
		t.Errorf("expected empty host in empty section, got %q", cfg.Host)
	}
}

func TestUnmarshalIniString_InvalidBoolField(t *testing.T) {
	contents := `debug = maybe`
	_, err := UnmarshalIniString[BasicConfig]("test.conf", "", contents)
	if err == nil {
		t.Error("expected error for invalid bool")
	}
}

func TestUnmarshalIniString_InvalidIntField(t *testing.T) {
	contents := `port = notanumber`
	_, err := UnmarshalIniString[BasicConfig]("test.conf", "", contents)
	if err == nil {
		t.Error("expected error for invalid int")
	}
}

func TestUnmarshalIniString_InvalidFloatField(t *testing.T) {
	contents := `f64 = notafloat`
	_, err := UnmarshalIniString[FloatConfig]("test.conf", "", contents)
	if err == nil {
		t.Error("expected error for invalid float")
	}
}

func TestUnmarshalIniString_EmptyBool(t *testing.T) {
	type B struct {
		Val bool `ini:"val"`
	}
	contents := `val =`
	_, err := UnmarshalIniString[B]("test.conf", "", contents)
	if err == nil {
		t.Error("expected error for empty bool value")
	}
}

func TestUnmarshalIniString_EmptyInt(t *testing.T) {
	type I struct {
		Val int `ini:"val"`
	}
	contents := `val =`
	_, err := UnmarshalIniString[I]("test.conf", "", contents)
	if err == nil {
		t.Error("expected error for empty int value")
	}
}

func TestUnmarshalIniString_EmptyFloat(t *testing.T) {
	type F struct {
		Val float64 `ini:"val"`
	}
	contents := `val =`
	_, err := UnmarshalIniString[F]("test.conf", "", contents)
	if err == nil {
		t.Error("expected error for empty float value")
	}
}

func TestUnmarshalIniString_IntOverflow(t *testing.T) {
	type S struct {
		Val int8 `ini:"val"`
	}
	contents := `val = 200`
	_, err := UnmarshalIniString[S]("test.conf", "", contents)
	if err == nil {
		t.Error("expected overflow error for int8=200")
	}

	contents = `val = -200`
	_, err = UnmarshalIniString[S]("test.conf", "", contents)
	if err == nil {
		t.Error("expected overflow error for int8=-200")
	}
}

func TestUnmarshalIniString_Int16Overflow(t *testing.T) {
	type S struct {
		Val int16 `ini:"val"`
	}
	contents := `val = 40000`
	_, err := UnmarshalIniString[S]("test.conf", "", contents)
	if err == nil {
		t.Error("expected overflow error for int16=40000")
	}
}

func TestUnmarshalIniString_Int32Overflow(t *testing.T) {
	type S struct {
		Val int32 `ini:"val"`
	}
	contents := `val = 3000000000`
	_, err := UnmarshalIniString[S]("test.conf", "", contents)
	if err == nil {
		t.Error("expected overflow error for int32=3000000000")
	}
}

func TestUnmarshalIniString_UintOverflow(t *testing.T) {
	type S struct {
		Val uint8 `ini:"val"`
	}
	contents := `val = 300`
	_, err := UnmarshalIniString[S]("test.conf", "", contents)
	if err == nil {
		t.Error("expected overflow error for uint8=300")
	}
}

func TestUnmarshalIniString_Uint16Overflow(t *testing.T) {
	type S struct {
		Val uint16 `ini:"val"`
	}
	contents := `val = 70000`
	_, err := UnmarshalIniString[S]("test.conf", "", contents)
	if err == nil {
		t.Error("expected overflow error for uint16=70000")
	}
}

func TestUnmarshalIniString_Uint32Overflow(t *testing.T) {
	type S struct {
		Val uint32 `ini:"val"`
	}
	contents := `val = 5000000000`
	_, err := UnmarshalIniString[S]("test.conf", "", contents)
	if err == nil {
		t.Error("expected overflow error for uint32=5000000000")
	}
}

func TestUnmarshalIniString_Float32Overflow(t *testing.T) {
	type S struct {
		Val float32 `ini:"val"`
	}
	contents := `val = 3.5e+38`
	_, err := UnmarshalIniString[S]("test.conf", "", contents)
	if err == nil {
		t.Error("expected overflow error for float32=3.5e+38")
	}

	contents = `val = -3.5e+38`
	_, err = UnmarshalIniString[S]("test.conf", "", contents)
	if err == nil {
		t.Error("expected overflow error for float32=-3.5e+38")
	}
}
