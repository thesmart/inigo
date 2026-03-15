package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

var testBinary string

func TestMain(m *testing.M) {
	tmp, err := os.MkdirTemp("", "inigo-test-*")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmp)

	testBinary = filepath.Join(tmp, "inigo")
	cmd := exec.Command("go", "build", "-o", testBinary, ".")
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		panic("failed to build test binary: " + err.Error())
	}

	os.Exit(m.Run())
}

func TestHelp(t *testing.T) {
	out, err := exec.Command(testBinary, "--help").Output()
	if err != nil {
		t.Fatalf("--help failed: %v", err)
	}
	if !strings.Contains(string(out), "env") || !strings.Contains(string(out), "json") {
		t.Errorf("expected subcommands in help, got:\n%s", out)
	}
}

func TestEnvHelp(t *testing.T) {
	out, err := exec.Command(testBinary, "env", "--help").Output()
	if err != nil {
		t.Fatalf("env --help failed: %v", err)
	}
	if !strings.Contains(string(out), "env") {
		t.Errorf("expected env usage in help, got:\n%s", out)
	}
}

func TestJsonHelp(t *testing.T) {
	out, err := exec.Command(testBinary, "json", "--help").Output()
	if err != nil {
		t.Fatalf("json --help failed: %v", err)
	}
	if !strings.Contains(string(out), "INIGO_JSON") {
		t.Errorf("expected INIGO_JSON in json help, got:\n%s", out)
	}
}

func writeIni(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.ini")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestEnvDefaultSection(t *testing.T) {
	ini := writeIni(t, "host = localhost\nport = 5432\n")
	out, err := exec.Command(testBinary, "env", ini, "--", "env").Output()
	if err != nil {
		t.Fatalf("env default section failed: %v", err)
	}
	output := string(out)
	if !strings.Contains(output, "HOST=localhost") {
		t.Errorf("expected HOST=localhost, got:\n%s", output)
	}
}

func TestEnvMissingFile(t *testing.T) {
	err := exec.Command(testBinary, "env", "/nonexistent/file.ini", "--", "env").Run()
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestEnvMissingSection(t *testing.T) {
	ini := writeIni(t, "[mydb]\nhost = localhost\n")
	err := exec.Command(testBinary, "env", ini, "nosection", "--", "env").Run()
	if err == nil {
		t.Fatal("expected error for missing section")
	}
}

func TestEnvMissingCommand(t *testing.T) {
	ini := writeIni(t, "host = localhost\n")
	err := exec.Command(testBinary, "env", ini, "--").Run()
	if err == nil {
		t.Fatal("expected error for missing command after --")
	}
}

func TestEnvCommandNotFound(t *testing.T) {
	ini := writeIni(t, "host = localhost\n")
	cmd := exec.Command(testBinary, "env", ini, "--", "nonexistent_command_xyz_999")
	err := cmd.Run()
	if err == nil {
		t.Fatal("expected error for missing command")
	}
	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("expected ExitError, got %T", err)
	}
	if exitErr.ExitCode() != 127 {
		t.Errorf("exit code = %d, want 127", exitErr.ExitCode())
	}
}

func TestJsonStdout(t *testing.T) {
	ini := writeIni(t, "[mydb]\nhost = localhost\nport = 5432\ndbname = myapp\n")
	out, err := exec.Command(testBinary, "json", ini, "mydb").Output()
	if err != nil {
		t.Fatalf("json stdout failed: %v", err)
	}
	output := strings.TrimSpace(string(out))
	if !strings.Contains(output, `"host":"localhost"`) {
		t.Errorf("expected host in JSON, got: %s", output)
	}
	if !strings.Contains(output, `"port":"5432"`) {
		t.Errorf("expected port in JSON, got: %s", output)
	}
}

func TestJsonDefaultSection(t *testing.T) {
	ini := writeIni(t, "host = localhost\n")
	out, err := exec.Command(testBinary, "json", ini).Output()
	if err != nil {
		t.Fatalf("json default section failed: %v", err)
	}
	if !strings.Contains(string(out), `"host":"localhost"`) {
		t.Errorf("expected host in JSON, got: %s", out)
	}
}

func TestJsonExec(t *testing.T) {
	ini := writeIni(t, "[mydb]\nhost = localhost\n")
	out, err := exec.Command(testBinary, "json", ini, "mydb", "--", "env").Output()
	if err != nil {
		t.Fatalf("json exec failed: %v", err)
	}
	if !strings.Contains(string(out), `INIGO_JSON=`) {
		t.Errorf("expected INIGO_JSON env var, got:\n%s", out)
	}
}

func TestJsonMissingSection(t *testing.T) {
	ini := writeIni(t, "[mydb]\nhost = localhost\n")
	err := exec.Command(testBinary, "json", ini, "nosection").Run()
	if err == nil {
		t.Fatal("expected error for missing section")
	}
}

func TestJsonCaseSnake(t *testing.T) {
	ini := writeIni(t, "[mydb]\ndb_host = localhost\ndb_port = 5432\n")
	out, err := exec.Command(testBinary, "json", "--case", "snake", ini, "mydb").Output()
	if err != nil {
		t.Fatalf("json --case snake failed: %v", err)
	}
	output := string(out)
	if !strings.Contains(output, `"db_host"`) {
		t.Errorf("expected db_host in JSON, got: %s", output)
	}
	if !strings.Contains(output, `"db_port"`) {
		t.Errorf("expected db_port in JSON, got: %s", output)
	}
}

func TestJsonCaseCamel(t *testing.T) {
	ini := writeIni(t, "[mydb]\ndb_host = localhost\ndb_port = 5432\n")
	out, err := exec.Command(testBinary, "json", "--case", "camel", ini, "mydb").Output()
	if err != nil {
		t.Fatalf("json --case camel failed: %v", err)
	}
	output := string(out)
	if !strings.Contains(output, `"dbHost"`) {
		t.Errorf("expected dbHost in JSON, got: %s", output)
	}
	if !strings.Contains(output, `"dbPort"`) {
		t.Errorf("expected dbPort in JSON, got: %s", output)
	}
}

func TestJsonCaseCamelFromUnderscore(t *testing.T) {
	ini := writeIni(t, "[mydb]\ndb_host = localhost\n")
	out, err := exec.Command(testBinary, "json", "--case", "camel", ini, "mydb").Output()
	if err != nil {
		t.Fatalf("json --case camel failed: %v", err)
	}
	if !strings.Contains(string(out), `"dbHost"`) {
		t.Errorf("expected dbHost in JSON, got: %s", out)
	}
}

func TestJsonCaseUpper(t *testing.T) {
	ini := writeIni(t, "[mydb]\nhost = localhost\n")
	out, err := exec.Command(testBinary, "json", "--case", "upper", ini, "mydb").Output()
	if err != nil {
		t.Fatalf("json --case upper failed: %v", err)
	}
	if !strings.Contains(string(out), `"HOST"`) {
		t.Errorf("expected HOST in JSON, got: %s", out)
	}
}

func TestJsonCaseLower(t *testing.T) {
	ini := writeIni(t, "[mydb]\nHOST = localhost\n")
	out, err := exec.Command(testBinary, "json", "--case", "lower", ini, "mydb").Output()
	if err != nil {
		t.Fatalf("json --case lower failed: %v", err)
	}
	if !strings.Contains(string(out), `"host"`) {
		t.Errorf("expected host in JSON, got: %s", out)
	}
}

func TestJsonCasePascal(t *testing.T) {
	ini := writeIni(t, "[mydb]\ndb_host = localhost\n")
	out, err := exec.Command(testBinary, "json", "--case", "pascal", ini, "mydb").Output()
	if err != nil {
		t.Fatalf("json --case pascal failed: %v", err)
	}
	if !strings.Contains(string(out), `"DbHost"`) {
		t.Errorf("expected DbHost in JSON, got: %s", out)
	}
}

func TestJsonCaseExampleUpperSnake(t *testing.T) {
	ini := writeIni(t, "[mydb]\ndb_host = localhost\n")
	out, err := exec.Command(testBinary, "json", "--case", "UPPER_CASE", ini, "mydb").Output()
	if err != nil {
		t.Fatalf("json --case UPPER_CASE failed: %v", err)
	}
	if !strings.Contains(string(out), `"DB_HOST"`) {
		t.Errorf("expected DB_HOST in JSON, got: %s", out)
	}
}

func TestJsonCaseExampleUpperKebab(t *testing.T) {
	ini := writeIni(t, "[mydb]\ndb_host = localhost\n")
	out, err := exec.Command(testBinary, "json", "--case", "UPPER-CASE", ini, "mydb").Output()
	if err != nil {
		t.Fatalf("json --case UPPER-CASE failed: %v", err)
	}
	if !strings.Contains(string(out), `"DB-HOST"`) {
		t.Errorf("expected DB-HOST in JSON, got: %s", out)
	}
}

func TestJsonCaseKebab(t *testing.T) {
	ini := writeIni(t, "[mydb]\ndb_host = localhost\n")
	out, err := exec.Command(testBinary, "json", "--case", "kebab", ini, "mydb").Output()
	if err != nil {
		t.Fatalf("json --case kebab failed: %v", err)
	}
	if !strings.Contains(string(out), `"db-host"`) {
		t.Errorf("expected db-host in JSON, got: %s", out)
	}
}

func TestJsonCaseInvalid(t *testing.T) {
	ini := writeIni(t, "[mydb]\nhost = localhost\n")
	// Mixed separators: both _ and - is ambiguous and should error.
	err := exec.Command(testBinary, "json", "--case", "snake_kebab-mix", ini, "mydb").Run()
	if err == nil {
		t.Fatal("expected error for invalid --case value")
	}
}

func TestJsonCaseDown(t *testing.T) {
	ini := writeIni(t, "[mydb]\nHOST = localhost\n")
	out, err := exec.Command(testBinary, "json", "--case", "down", ini, "mydb").Output()
	if err != nil {
		t.Fatalf("json --case down failed: %v", err)
	}
	if !strings.Contains(string(out), `"host"`) {
		t.Errorf("expected host in JSON, got: %s", out)
	}
}

func TestJsonCaseUp(t *testing.T) {
	ini := writeIni(t, "[mydb]\nhost = localhost\n")
	out, err := exec.Command(testBinary, "json", "--case", "up", ini, "mydb").Output()
	if err != nil {
		t.Fatalf("json --case up failed: %v", err)
	}
	if !strings.Contains(string(out), `"HOST"`) {
		t.Errorf("expected HOST in JSON, got: %s", out)
	}
}

func TestJsonCaseExampleSnake(t *testing.T) {
	ini := writeIni(t, "[mydb]\ndb_host = localhost\n")
	out, err := exec.Command(testBinary, "json", "--case", "snake_case", ini, "mydb").Output()
	if err != nil {
		t.Fatalf("json --case snake_case failed: %v", err)
	}
	if !strings.Contains(string(out), `"db_host"`) {
		t.Errorf("expected db_host in JSON, got: %s", out)
	}
}

func TestJsonCaseExampleKebab(t *testing.T) {
	ini := writeIni(t, "[mydb]\ndb_host = localhost\n")
	out, err := exec.Command(testBinary, "json", "--case", "kebab-case", ini, "mydb").Output()
	if err != nil {
		t.Fatalf("json --case kebab-case failed: %v", err)
	}
	if !strings.Contains(string(out), `"db-host"`) {
		t.Errorf("expected db-host in JSON, got: %s", out)
	}
}

func TestJsonCaseExampleCamel(t *testing.T) {
	ini := writeIni(t, "[mydb]\ndb_host = localhost\n")
	out, err := exec.Command(testBinary, "json", "--case", "camelCase", ini, "mydb").Output()
	if err != nil {
		t.Fatalf("json --case camelCase failed: %v", err)
	}
	if !strings.Contains(string(out), `"dbHost"`) {
		t.Errorf("expected dbHost in JSON, got: %s", out)
	}
}

func TestJsonCaseExamplePascal(t *testing.T) {
	ini := writeIni(t, "[mydb]\ndb_host = localhost\n")
	out, err := exec.Command(testBinary, "json", "--case", "PascalCase", ini, "mydb").Output()
	if err != nil {
		t.Fatalf("json --case PascalCase failed: %v", err)
	}
	if !strings.Contains(string(out), `"DbHost"`) {
		t.Errorf("expected DbHost in JSON, got: %s", out)
	}
}

func TestJsonCaseExampleLowercase(t *testing.T) {
	ini := writeIni(t, "[mydb]\nhost = localhost\n")
	out, err := exec.Command(testBinary, "json", "--case", "lowercase", ini, "mydb").Output()
	if err != nil {
		t.Fatalf("json --case lowercase failed: %v", err)
	}
	if !strings.Contains(string(out), `"host"`) {
		t.Errorf("expected host in JSON, got: %s", out)
	}
}

func TestJsonCaseExampleUppercase(t *testing.T) {
	ini := writeIni(t, "[mydb]\nhost = localhost\n")
	out, err := exec.Command(testBinary, "json", "--case", "UPPERCASE", ini, "mydb").Output()
	if err != nil {
		t.Fatalf("json --case UPPERCASE failed: %v", err)
	}
	if !strings.Contains(string(out), `"HOST"`) {
		t.Errorf("expected HOST in JSON, got: %s", out)
	}
}
