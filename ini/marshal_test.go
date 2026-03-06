package ini

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMarshalIniString_Basic(t *testing.T) {
	cfg := &BasicConfig{
		Host:  "localhost",
		Port:  5432,
		Debug: true,
		Rate:  1.5,
		Name:  "myapp",
	}
	out, err := MarshalIniString("", cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "host = localhost\n") {
		t.Errorf("expected host=localhost in output, got:\n%s", out)
	}
	if !strings.Contains(out, "port = 5432\n") {
		t.Errorf("expected port=5432 in output, got:\n%s", out)
	}
	if !strings.Contains(out, "debug = on\n") {
		t.Errorf("expected debug=on in output, got:\n%s", out)
	}
	if !strings.Contains(out, "rate = 1.5\n") {
		t.Errorf("expected rate=1.5 in output, got:\n%s", out)
	}
	if !strings.Contains(out, "name = myapp\n") {
		t.Errorf("expected name=myapp in output, got:\n%s", out)
	}
}

func TestMarshalIniString_WithSection(t *testing.T) {
	cfg := &BasicConfig{Host: "localhost"}
	out, err := MarshalIniString("database", cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(out, "[database]\n") {
		t.Errorf("expected [database] header, got:\n%s", out)
	}
}

func TestMarshalIniString_ZeroValues(t *testing.T) {
	cfg := &BasicConfig{Host: "localhost"} // Port=0, Debug=false, etc.
	out, err := MarshalIniString("", cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(out, "port") {
		t.Errorf("zero-value port should be skipped, got:\n%s", out)
	}
	if strings.Contains(out, "debug") {
		t.Errorf("zero-value debug should be skipped, got:\n%s", out)
	}
}

func TestMarshalIniString_QuotedStrings(t *testing.T) {
	cfg := &BasicConfig{Host: "my host"}
	out, err := MarshalIniString("", cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "host = 'my host'\n") {
		t.Errorf("expected quoted host, got:\n%s", out)
	}
}

func TestMarshalIniString_QuoteEscaping(t *testing.T) {
	cfg := &BasicConfig{Host: "it's"}
	out, err := MarshalIniString("", cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "'it''s'") {
		t.Errorf("expected escaped quotes, got:\n%s", out)
	}
}

func TestMarshalIniString_EmptyString(t *testing.T) {
	cfg := &BasicConfig{Name: ""}
	out, err := MarshalIniString("", cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// empty string is zero value, should be skipped
	if strings.Contains(out, "name") {
		t.Errorf("empty string should be skipped, got:\n%s", out)
	}
}

func TestMarshalIniString_BoolOff(t *testing.T) {
	// Debug is false (zero value), should be skipped
	cfg := &BasicConfig{Host: "x", Debug: false}
	out, err := MarshalIniString("", cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(out, "debug") {
		t.Errorf("false bool should be skipped (zero value), got:\n%s", out)
	}
}

func TestMarshalIniString_AllIntTypes(t *testing.T) {
	cfg := &IntTypes{I: 1, I8: 2, I16: 3, I32: 4, I64: 5, U: 6, U8: 7, U16: 8, U32: 9, U64: 10}
	out, err := MarshalIniString("", cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "i = 1\n") {
		t.Errorf("missing i=1 in:\n%s", out)
	}
	if !strings.Contains(out, "u64 = 10\n") {
		t.Errorf("missing u64=10 in:\n%s", out)
	}
}

func TestMarshalIniString_Floats(t *testing.T) {
	cfg := &FloatConfig{F32: 3.14, F64: 0.001}
	out, err := MarshalIniString("", cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "f64 = 0.001") {
		t.Errorf("expected f64=0.001, got:\n%s", out)
	}
}

func TestMarshalIniFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "output.conf")
	cfg := &BasicConfig{Host: "localhost", Port: 5432}
	err := MarshalIniFile(path, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "host = localhost") {
		t.Errorf("expected host in file, got:\n%s", content)
	}
}

func TestMarshalUnmarshal_Roundtrip(t *testing.T) {
	original := &BasicConfig{
		Host:  "localhost",
		Port:  5432,
		Debug: true,
		Rate:  1.5,
		Name:  "myapp",
	}
	out, err := MarshalIniString("", original)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	roundtripped, err := UnmarshalIniString[BasicConfig]("test.conf", "", out)
	if err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if roundtripped.Host != original.Host {
		t.Errorf("host mismatch: %q vs %q", roundtripped.Host, original.Host)
	}
	if roundtripped.Port != original.Port {
		t.Errorf("port mismatch: %d vs %d", roundtripped.Port, original.Port)
	}
	if roundtripped.Debug != original.Debug {
		t.Errorf("debug mismatch: %v vs %v", roundtripped.Debug, original.Debug)
	}
	if roundtripped.Rate != original.Rate {
		t.Errorf("rate mismatch: %f vs %f", roundtripped.Rate, original.Rate)
	}
}

// Custom marshal test
type CustomMarshalConfig struct {
	Port int `ini:"port"`
}

func (c *CustomMarshalConfig) Marshal_Port(v *int) (string, error) {
	if *v == 5432 {
		return "'default'", nil
	}
	return "", fmt.Errorf("unknown port: %d", *v)
}

func TestMarshalIniString_CustomMarshal(t *testing.T) {
	cfg := &CustomMarshalConfig{Port: 5432}
	out, err := MarshalIniString("", cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "port = 'default'") {
		t.Errorf("expected custom marshaled value, got:\n%s", out)
	}
}

func TestMarshalIniString_CustomMarshalError(t *testing.T) {
	cfg := &CustomMarshalConfig{Port: 9999}
	_, err := MarshalIniString("", cfg)
	if err == nil {
		t.Error("expected error from custom marshal")
	}
}

func TestMarshalIniString_UnsupportedType(t *testing.T) {
	cfg := &UnsupportedConfig{Data: []string{"a"}}
	_, err := MarshalIniString("", cfg)
	if err == nil {
		t.Error("expected error for unsupported type")
	}
}

func TestQuoteValue(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"simple", "simple"},
		{"", "''"},
		{"has space", "'has space'"},
		{"has#hash", "'has#hash'"},
		{"has;semi", "'has;semi'"},
		{"has=eq", "'has=eq'"},
		{"it's", "'it''s'"},
		{"has\"dq", "'has\"dq'"},
		{"back\\slash", "'back\\slash'"},
	}
	for _, tt := range tests {
		result := quoteValue(tt.input)
		if result != tt.expected {
			t.Errorf("quoteValue(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}
