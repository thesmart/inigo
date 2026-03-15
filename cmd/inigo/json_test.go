package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// keyCaseFunc
// ---------------------------------------------------------------------------

func TestKeyCaseFunc_Identity(t *testing.T) {
	fn, err := keyCaseFunc("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := fn("db_host"); got != "db_host" {
		t.Errorf("got %q, want %q", got, "db_host")
	}
}

func TestKeyCaseFunc_NamedAliases(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"lower", "DB_HOST", "db_host"},
		{"down", "DB_HOST", "db_host"},
		{"upper", "db_host", "DB_HOST"},
		{"up", "db_host", "DB_HOST"},
		{"snake", "dbHost", "db_host"},
		{"kebab", "dbHost", "db-host"},
		{"camel", "db_host", "dbHost"},
		{"pascal", "db_host", "DbHost"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn, err := keyCaseFunc(tt.name)
			if err != nil {
				t.Fatalf("keyCaseFunc(%q): %v", tt.name, err)
			}
			if got := fn(tt.input); got != tt.want {
				t.Errorf("keyCaseFunc(%q)(%q) = %q, want %q", tt.name, tt.input, got, tt.want)
			}
		})
	}
}

func TestKeyCaseFunc_Examples(t *testing.T) {
	tests := []struct {
		example string
		input   string
		want    string
	}{
		{"snake_case", "dbHost", "db_host"},
		{"UPPER_CASE", "db_host", "DB_HOST"},
		{"kebab-case", "dbHost", "db-host"},
		{"UPPER-CASE", "db_host", "DB-HOST"},
		{"camelCase", "db_host", "dbHost"},
		{"PascalCase", "db_host", "DbHost"},
		{"lowercase", "DB_HOST", "db_host"},
		{"UPPERCASE", "db_host", "DB_HOST"},
	}
	for _, tt := range tests {
		t.Run(tt.example, func(t *testing.T) {
			fn, err := keyCaseFunc(tt.example)
			if err != nil {
				t.Fatalf("keyCaseFunc(%q): %v", tt.example, err)
			}
			if got := fn(tt.input); got != tt.want {
				t.Errorf("keyCaseFunc(%q)(%q) = %q, want %q", tt.example, tt.input, got, tt.want)
			}
		})
	}
}

func TestKeyCaseFunc_Invalid(t *testing.T) {
	_, err := keyCaseFunc("snake_kebab-mix")
	if err == nil {
		t.Fatal("expected error for mixed separators")
	}
}

// ---------------------------------------------------------------------------
// detectCaseExample
// ---------------------------------------------------------------------------

func TestDetectCaseExample_MixedSeparators(t *testing.T) {
	_, ok := detectCaseExample("a_b-c")
	if ok {
		t.Error("expected false for mixed separators")
	}
}

func TestDetectCaseExample_NoMatch(t *testing.T) {
	// Title_Case has underscore + mixed case but no clear pattern
	_, ok := detectCaseExample("Title_Case")
	if ok {
		t.Error("expected false for ambiguous pattern")
	}
}

// ---------------------------------------------------------------------------
// json command — Cobra stdout tests
// ---------------------------------------------------------------------------

func TestJsonCmd_Stdout(t *testing.T) {
	ini := writeIni(t, "[mydb]\nhost = localhost\nport = 5432\n")
	cmd := newTestRootCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"json", ini, "mydb"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	var result map[string]string
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("JSON unmarshal: %v\nraw: %s", err, buf.String())
	}
	if result["host"] != "localhost" {
		t.Errorf("host = %q, want %q", result["host"], "localhost")
	}
	if result["port"] != "5432" {
		t.Errorf("port = %q, want %q", result["port"], "5432")
	}
}

func TestJsonCmd_DefaultSection(t *testing.T) {
	ini := writeIni(t, "host = localhost\n")
	cmd := newTestRootCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"json", ini})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	var result map[string]string
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("JSON unmarshal: %v\nraw: %s", err, buf.String())
	}
	if result["host"] != "localhost" {
		t.Errorf("host = %q, want %q", result["host"], "localhost")
	}
}

func TestJsonCmd_CaseSnake(t *testing.T) {
	ini := writeIni(t, "db_host = localhost\n")
	cmd := newTestRootCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"json", "--case", "snake", ini})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	var result map[string]string
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("JSON unmarshal: %v\nraw: %s", err, buf.String())
	}
	if _, ok := result["db_host"]; !ok {
		t.Errorf("expected db_host key, got keys: %v", keys(result))
	}
}

func TestJsonCmd_CaseCamel(t *testing.T) {
	ini := writeIni(t, "db_host = localhost\n")
	cmd := newTestRootCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"json", "--case", "camel", ini})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	var result map[string]string
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("JSON unmarshal: %v\nraw: %s", err, buf.String())
	}
	if _, ok := result["dbHost"]; !ok {
		t.Errorf("expected dbHost key, got keys: %v", keys(result))
	}
}

func TestJsonCmd_CasePascal(t *testing.T) {
	ini := writeIni(t, "db_host = localhost\n")
	cmd := newTestRootCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"json", "--case", "pascal", ini})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	var result map[string]string
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("JSON unmarshal: %v\nraw: %s", err, buf.String())
	}
	if _, ok := result["DbHost"]; !ok {
		t.Errorf("expected DbHost key, got keys: %v", keys(result))
	}
}

func TestJsonCmd_CaseKebab(t *testing.T) {
	ini := writeIni(t, "db_host = localhost\n")
	cmd := newTestRootCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"json", "--case", "kebab", ini})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	var result map[string]string
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("JSON unmarshal: %v\nraw: %s", err, buf.String())
	}
	if _, ok := result["db-host"]; !ok {
		t.Errorf("expected db-host key, got keys: %v", keys(result))
	}
}

func TestJsonCmd_CaseInvalid(t *testing.T) {
	ini := writeIni(t, "host = localhost\n")
	cmd := newTestRootCmd()
	cmd.SetArgs([]string{"json", "--case", "snake_kebab-mix", ini})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for invalid --case")
	}
	if !strings.Contains(err.Error(), "unknown --case") {
		t.Errorf("error = %q, want substring %q", err, "unknown --case")
	}
}

func TestJsonCmd_MissingSection(t *testing.T) {
	ini := writeIni(t, "[mydb]\nhost = localhost\n")
	cmd := newTestRootCmd()
	cmd.SetArgs([]string{"json", ini, "nosection"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing section")
	}
}

func TestJsonCmd_MissingFile(t *testing.T) {
	cmd := newTestRootCmd()
	cmd.SetArgs([]string{"json", "/nonexistent/file.ini"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestJsonCmd_NoArgs(t *testing.T) {
	cmd := newTestRootCmd()
	cmd.SetArgs([]string{"json"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for no args")
	}
}

// keys returns the keys of a map for error messages.
func keys(m map[string]string) []string {
	var ks []string
	for k := range m {
		ks = append(ks, k)
	}
	return ks
}
