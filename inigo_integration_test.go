package inigo

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// helper to parse an INI string via Parse().
func mustParse(t *testing.T, input string) *Config {
	t.Helper()
	cfg, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	return cfg
}

// helper to assert Parse() returns an error.
func mustFailParse(t *testing.T, input string) {
	t.Helper()
	_, err := Parse(strings.NewReader(input))
	if err == nil {
		t.Fatal("expected Parse to fail, but it succeeded")
	}
}

func TestParseEmpty(t *testing.T) {
	cfg := mustParse(t, "")
	if names := cfg.SectionNames(); len(names) != 0 {
		t.Errorf("expected no sections, got %v", names)
	}
}

func TestParseBlankLinesAndComments(t *testing.T) {
	input := `
# comment line
   # indented comment


`
	cfg := mustParse(t, input)
	sec := cfg.Section("")
	if len(sec.AllParams()) != 0 {
		t.Errorf("expected no params, got %v", sec.AllParams())
	}
}

func TestParseDefaultSection(t *testing.T) {
	input := `
host = localhost
port = 5432
`
	cfg := mustParse(t, input)
	sec := cfg.Section("")
	if sec.GetParam("host").String() != "localhost" {
		t.Errorf("host = %q, want %q", sec.GetParam("host").String(), "localhost")
	}
	if sec.GetParam("port").String() != "5432" {
		t.Errorf("port = %q, want %q", sec.GetParam("port").String(), "5432")
	}
}

func TestParseSingleSection(t *testing.T) {
	input := `
[mydb]
host = db.example.com
port = 5432
dbname = production
`
	cfg := mustParse(t, input)

	names := cfg.SectionNames()
	if len(names) != 1 || names[0] != "mydb" {
		t.Fatalf("SectionNames() = %v, want [mydb]", names)
	}

	sec := cfg.Section("mydb")
	if sec == nil {
		t.Fatal("Section(mydb) returned nil")
	}
	if sec.GetParam("host").String() != "db.example.com" {
		t.Errorf("host = %q", sec.GetParam("host").String())
	}
	if sec.GetParam("port").String() != "5432" {
		t.Errorf("port = %q", sec.GetParam("port").String())
	}
	if sec.GetParam("dbname").String() != "production" {
		t.Errorf("dbname = %q", sec.GetParam("dbname").String())
	}
}

func TestParseMultipleSections(t *testing.T) {
	input := `
[staging]
host = staging.local
port = 5432

[production]
host = prod.example.com
port = 5433
`
	cfg := mustParse(t, input)

	names := cfg.SectionNames()
	if len(names) != 2 {
		t.Fatalf("SectionNames() = %v, want 2 sections", names)
	}

	stg := cfg.Section("staging")
	if stg.GetParam("host").String() != "staging.local" {
		t.Errorf("staging host = %q", stg.GetParam("host").String())
	}

	prd := cfg.Section("production")
	if prd.GetParam("host").String() != "prod.example.com" {
		t.Errorf("production host = %q", prd.GetParam("host").String())
	}
	if prd.GetParam("port").String() != "5433" {
		t.Errorf("production port = %q", prd.GetParam("port").String())
	}
}

func TestParseParamCaseInsensitive(t *testing.T) {
	input := `
HOST = localhost
Port = 5432
DbName = mydb
`
	cfg := mustParse(t, input)
	sec := cfg.Section("")

	// Lookup with different cases should all work
	if sec.GetParam("host").String() != "localhost" {
		t.Errorf("host lookup failed")
	}
	if sec.GetParam("HOST").String() != "localhost" {
		t.Errorf("HOST lookup failed")
	}
	if sec.GetParam("port").String() != "5432" {
		t.Errorf("port lookup failed")
	}
	if sec.GetParam("DBNAME").String() != "mydb" {
		t.Errorf("DBNAME lookup failed")
	}
}

func TestParseLastValueWins(t *testing.T) {
	input := `
host = first
host = second
host = third
`
	cfg := mustParse(t, input)
	sec := cfg.Section("")
	if sec.GetParam("host").String() != "third" {
		t.Errorf("host = %q, want %q", sec.GetParam("host").String(), "third")
	}
}

func TestParseQuotedValues(t *testing.T) {
	input := `
simple = 'hello world'
escaped_double = 'it''s fine'
escaped_backslash = 'it\'s fine'
empty_quoted = ''
with_hash = 'value # not a comment'
`
	cfg := mustParse(t, input)
	sec := cfg.Section("")

	tests := map[string]string{
		"simple":            "hello world",
		"escaped_double":    "it's fine",
		"escaped_backslash": "it's fine",
		"empty_quoted":      "",
		"with_hash":         "value # not a comment",
	}
	for key, want := range tests {
		got := sec.GetParam(key).String()
		if got != want {
			t.Errorf("%s = %q, want %q", key, got, want)
		}
	}
}

func TestParseUnquotedValues(t *testing.T) {
	input := `
host = localhost
port = 5432
timeout = 30
`
	cfg := mustParse(t, input)
	sec := cfg.Section("")

	if sec.GetParam("host").String() != "localhost" {
		t.Errorf("host = %q", sec.GetParam("host").String())
	}
	v, err := sec.GetParam("port").Int()
	if err != nil || v != 5432 {
		t.Errorf("port.Int() = %d, %v", v, err)
	}
}

func TestParseEmptyAndBareValues(t *testing.T) {
	input := `
empty_eq =
bare_param
`
	cfg := mustParse(t, input)
	sec := cfg.Section("")

	if sec.GetParam("empty_eq").String() != "" {
		t.Errorf("empty_eq = %q, want empty", sec.GetParam("empty_eq").String())
	}
	if sec.GetParam("bare_param").String() != "" {
		t.Errorf("bare_param = %q, want empty", sec.GetParam("bare_param").String())
	}
}

func TestParseInlineComments(t *testing.T) {
	input := `
host = localhost # the host
port = 5432 # the port
name = 'hello # world' # actual comment
`
	cfg := mustParse(t, input)
	sec := cfg.Section("")

	if sec.GetParam("host").String() != "localhost" {
		t.Errorf("host = %q, want %q", sec.GetParam("host").String(), "localhost")
	}
	if sec.GetParam("port").String() != "5432" {
		t.Errorf("port = %q, want %q", sec.GetParam("port").String(), "5432")
	}
	if sec.GetParam("name").String() != "hello # world" {
		t.Errorf("name = %q, want %q", sec.GetParam("name").String(), "hello # world")
	}
}

func TestParseWhitespaceInsignificant(t *testing.T) {
	input := `
  host   =   localhost
	port	=	5432
`
	cfg := mustParse(t, input)
	sec := cfg.Section("")

	if sec.GetParam("host").String() != "localhost" {
		t.Errorf("host = %q", sec.GetParam("host").String())
	}
	if sec.GetParam("port").String() != "5432" {
		t.Errorf("port = %q", sec.GetParam("port").String())
	}
}

func TestParseWhitespacePreservedInQuotes(t *testing.T) {
	input := `name = '  hello  '`
	cfg := mustParse(t, input)
	sec := cfg.Section("")

	if sec.GetParam("name").String() != "  hello  " {
		t.Errorf("name = %q, want %q", sec.GetParam("name").String(), "  hello  ")
	}
}

func TestParsePgServiceConf(t *testing.T) {
	input := `
# pg_service.conf example
[myservice]
host = db.example.com
port = 5432
dbname = appdb
user = appuser
password = 'my''secret'
sslmode = require

[devservice]
host = localhost
port = 5432
dbname = devdb
user = dev
`
	cfg := mustParse(t, input)

	names := cfg.SectionNames()
	if len(names) != 2 {
		t.Fatalf("expected 2 sections, got %v", names)
	}

	svc := cfg.Section("myservice")
	if svc.GetParam("host").String() != "db.example.com" {
		t.Errorf("myservice host = %q", svc.GetParam("host").String())
	}
	if svc.GetParam("password").String() != "my'secret" {
		t.Errorf("myservice password = %q", svc.GetParam("password").String())
	}

	ssl, err := svc.GetParam("sslmode").String(), error(nil)
	_ = err
	if ssl != "require" {
		t.Errorf("myservice sslmode = %q", ssl)
	}

	dev := cfg.Section("devservice")
	if dev.GetParam("user").String() != "dev" {
		t.Errorf("devservice user = %q", dev.GetParam("user").String())
	}
}

func TestParseReentrantSection(t *testing.T) {
	// Re-entering a section should add params to the existing section
	input := `
[svc]
host = localhost

[other]
x = 1

[svc]
port = 5432
`
	cfg := mustParse(t, input)

	svc := cfg.Section("svc")
	if !svc.HasParam("host") || !svc.HasParam("port") {
		t.Errorf("re-entered section should have both host and port, got %v", svc.AllParams())
	}
	if svc.GetParam("host").String() != "localhost" {
		t.Errorf("host = %q", svc.GetParam("host").String())
	}
	if svc.GetParam("port").String() != "5432" {
		t.Errorf("port = %q", svc.GetParam("port").String())
	}
}

func TestParseEqualsInValue(t *testing.T) {
	input := `conn_str = host=localhost port=5432`
	cfg := mustParse(t, input)
	sec := cfg.Section("")

	if sec.GetParam("conn_str").String() != "host=localhost port=5432" {
		t.Errorf("conn_str = %q", sec.GetParam("conn_str").String())
	}
}

func TestParseTypedValues(t *testing.T) {
	input := `
[types]
bool_on = on
bool_off = off
bool_yes = yes
bool_prefix = tr
int_dec = 42
int_hex = 0xFF
int_oct = 010
float = 3.14
`
	cfg := mustParse(t, input)
	sec := cfg.Section("types")

	if v, err := sec.GetParam("bool_on").Bool(); err != nil || !v {
		t.Errorf("bool_on.Bool() = %v, %v", v, err)
	}
	if v, err := sec.GetParam("bool_off").Bool(); err != nil || v {
		t.Errorf("bool_off.Bool() = %v, %v", v, err)
	}
	if v, err := sec.GetParam("bool_yes").Bool(); err != nil || !v {
		t.Errorf("bool_yes.Bool() = %v, %v", v, err)
	}
	if v, err := sec.GetParam("bool_prefix").Bool(); err != nil || !v {
		t.Errorf("bool_prefix.Bool() = %v, %v", v, err)
	}
	if v, err := sec.GetParam("int_dec").Int(); err != nil || v != 42 {
		t.Errorf("int_dec.Int() = %d, %v", v, err)
	}
	if v, err := sec.GetParam("int_hex").Int(); err != nil || v != 255 {
		t.Errorf("int_hex.Int() = %d, %v", v, err)
	}
	if v, err := sec.GetParam("int_oct").Int(); err != nil || v != 8 {
		t.Errorf("int_oct.Int() = %d, %v", v, err)
	}
	if v, err := sec.GetParam("float").Float64(); err != nil || v != 3.14 {
		t.Errorf("float.Float64() = %f, %v", v, err)
	}
}

// --- Error cases ---

func TestParseErrorUnterminatedSection(t *testing.T) {
	mustFailParse(t, "[unterminated\nhost = x\n")
}

func TestParseErrorEmptySection(t *testing.T) {
	mustFailParse(t, "[]\nhost = x\n")
}

func TestParseErrorInvalidParamName(t *testing.T) {
	mustFailParse(t, "1bad = value\n")
}

func TestParseErrorUnterminatedQuote(t *testing.T) {
	mustFailParse(t, "name = 'unterminated\n")
}

func TestParseErrorIncludeInReader(t *testing.T) {
	mustFailParse(t, "include '/etc/something.conf'\n")
}

// --- File-based integration tests for Load and includes ---

func TestLoadSimpleFile(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "test.conf", `
host = localhost
port = 5432
`)
	cfg, err := Load(filepath.Join(dir, "test.conf"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	sec := cfg.Section("")
	if sec.GetParam("host").String() != "localhost" {
		t.Errorf("host = %q", sec.GetParam("host").String())
	}
}

func TestLoadInclude(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "base.conf", `
host = localhost
include 'extra.conf'
`)
	writeTestFile(t, dir, "extra.conf", `
port = 5432
dbname = testdb
`)
	cfg, err := Load(filepath.Join(dir, "base.conf"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	sec := cfg.Section("")
	if sec.GetParam("host").String() != "localhost" {
		t.Errorf("host = %q", sec.GetParam("host").String())
	}
	if sec.GetParam("port").String() != "5432" {
		t.Errorf("port = %q", sec.GetParam("port").String())
	}
	if sec.GetParam("dbname").String() != "testdb" {
		t.Errorf("dbname = %q", sec.GetParam("dbname").String())
	}
}

func TestLoadIncludeOverride(t *testing.T) {
	// Included values should be overridable by later params in the parent
	dir := t.TempDir()
	writeTestFile(t, dir, "base.conf", `
host = original
include 'override.conf'
host = final
`)
	writeTestFile(t, dir, "override.conf", `
host = included
port = 5432
`)
	cfg, err := Load(filepath.Join(dir, "base.conf"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	sec := cfg.Section("")
	// "final" should win (last value wins, after include)
	if sec.GetParam("host").String() != "final" {
		t.Errorf("host = %q, want %q", sec.GetParam("host").String(), "final")
	}
	if sec.GetParam("port").String() != "5432" {
		t.Errorf("port = %q", sec.GetParam("port").String())
	}
}

func TestLoadIncludeIfExists(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "base.conf", `
host = localhost
include_if_exists 'optional.conf'
`)
	// optional.conf does NOT exist — should not error
	cfg, err := Load(filepath.Join(dir, "base.conf"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Section("").GetParam("host").String() != "localhost" {
		t.Errorf("host = %q", cfg.Section("").GetParam("host").String())
	}
}

func TestLoadIncludeIfExistsPresent(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "base.conf", `
host = localhost
include_if_exists 'optional.conf'
`)
	writeTestFile(t, dir, "optional.conf", `
port = 9999
`)
	cfg, err := Load(filepath.Join(dir, "base.conf"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Section("").GetParam("port").String() != "9999" {
		t.Errorf("port = %q", cfg.Section("").GetParam("port").String())
	}
}

func TestLoadIncludeDir(t *testing.T) {
	dir := t.TempDir()
	confDir := filepath.Join(dir, "conf.d")
	os.Mkdir(confDir, 0o755)

	writeTestFile(t, dir, "base.conf", `
host = localhost
include_dir 'conf.d'
`)
	// b.conf should be processed before z.conf (ASCII order)
	writeTestFile(t, confDir, "b.conf", `port = 1111`)
	writeTestFile(t, confDir, "z.conf", `port = 2222`)

	// dotfile should be excluded
	writeTestFile(t, confDir, ".hidden.conf", `port = 9999`)

	// non-.conf file should be excluded
	writeTestFile(t, confDir, "readme.txt", `port = 8888`)

	cfg, err := Load(filepath.Join(dir, "base.conf"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	sec := cfg.Section("")
	// z.conf is processed last, so port should be 2222
	if sec.GetParam("port").String() != "2222" {
		t.Errorf("port = %q, want %q (last file wins)", sec.GetParam("port").String(), "2222")
	}
	if sec.GetParam("host").String() != "localhost" {
		t.Errorf("host = %q", sec.GetParam("host").String())
	}
}

func TestLoadIncludeDirOrder(t *testing.T) {
	dir := t.TempDir()
	confDir := filepath.Join(dir, "conf.d")
	os.Mkdir(confDir, 0o755)

	writeTestFile(t, dir, "base.conf", `include_dir 'conf.d'`)
	writeTestFile(t, confDir, "01_first.conf", `order = first`)
	writeTestFile(t, confDir, "02_second.conf", `order = second`)
	writeTestFile(t, confDir, "03_third.conf", `order = third`)

	cfg, err := Load(filepath.Join(dir, "base.conf"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	// Last file wins
	if cfg.Section("").GetParam("order").String() != "third" {
		t.Errorf("order = %q, want %q", cfg.Section("").GetParam("order").String(), "third")
	}
}

func TestLoadIncludePreservesSection(t *testing.T) {
	// Include within a section should continue in the default section of the included file,
	// but section changes in included files persist.
	dir := t.TempDir()
	writeTestFile(t, dir, "base.conf", `
[svc]
host = original
include 'extra.conf'
port = 5432
`)
	writeTestFile(t, dir, "extra.conf", `
[svc]
dbname = frominclude
`)
	cfg, err := Load(filepath.Join(dir, "base.conf"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	svc := cfg.Section("svc")
	if svc.GetParam("host").String() != "original" {
		t.Errorf("host = %q", svc.GetParam("host").String())
	}
	if svc.GetParam("dbname").String() != "frominclude" {
		t.Errorf("dbname = %q", svc.GetParam("dbname").String())
	}
	if svc.GetParam("port").String() != "5432" {
		t.Errorf("port = %q", svc.GetParam("port").String())
	}
}

func TestLoadNestedInclude(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "a.conf", `
host = a
include 'b.conf'
`)
	writeTestFile(t, dir, "b.conf", `
port = b
include 'c.conf'
`)
	writeTestFile(t, dir, "c.conf", `
dbname = c
`)
	cfg, err := Load(filepath.Join(dir, "a.conf"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	sec := cfg.Section("")
	if sec.GetParam("host").String() != "a" {
		t.Errorf("host = %q", sec.GetParam("host").String())
	}
	if sec.GetParam("port").String() != "b" {
		t.Errorf("port = %q", sec.GetParam("port").String())
	}
	if sec.GetParam("dbname").String() != "c" {
		t.Errorf("dbname = %q", sec.GetParam("dbname").String())
	}
}

func TestLoadErrorCircularInclude(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "a.conf", `include 'b.conf'`)
	writeTestFile(t, dir, "b.conf", `include 'a.conf'`)

	_, err := Load(filepath.Join(dir, "a.conf"))
	if err == nil {
		t.Fatal("expected circular include error")
	}
	if !strings.Contains(err.Error(), "circular") {
		t.Errorf("error = %q, expected to mention circular", err.Error())
	}
}

func TestLoadErrorMissingInclude(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "base.conf", `include 'nonexistent.conf'`)

	_, err := Load(filepath.Join(dir, "base.conf"))
	if err == nil {
		t.Fatal("expected error for missing include file")
	}
}

func TestLoadErrorMissingFile(t *testing.T) {
	_, err := Load("/tmp/definitely_does_not_exist_alter_test.conf")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func writeTestFile(t *testing.T, dir, name, content string) {
	t.Helper()
	err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644)
	if err != nil {
		t.Fatalf("failed to write test file %s: %v", name, err)
	}
}
