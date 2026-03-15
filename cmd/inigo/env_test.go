package main

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/thesmart/inigo/pgini"
)

// ---------------------------------------------------------------------------
// splitDashArgs
// ---------------------------------------------------------------------------

func TestSplitDashArgs_FileOnly(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetArgs([]string{"config.ini"})
	cmd.ParseFlags([]string{"config.ini"})

	iniFile, section, command, err := splitDashArgs(cmd, []string{"config.ini"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if iniFile != "config.ini" {
		t.Errorf("iniFile = %q, want %q", iniFile, "config.ini")
	}
	if section != "" {
		t.Errorf("section = %q, want empty", section)
	}
	if len(command) != 0 {
		t.Errorf("command = %v, want empty", command)
	}
}

func TestSplitDashArgs_FileAndSection(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetArgs([]string{"config.ini", "mydb"})
	cmd.ParseFlags([]string{"config.ini", "mydb"})

	iniFile, section, command, err := splitDashArgs(cmd, []string{"config.ini", "mydb"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if iniFile != "config.ini" {
		t.Errorf("iniFile = %q, want %q", iniFile, "config.ini")
	}
	if section != "mydb" {
		t.Errorf("section = %q, want %q", section, "mydb")
	}
	if len(command) != 0 {
		t.Errorf("command = %v, want empty", command)
	}
}

func TestSplitDashArgs_WithCommand(t *testing.T) {
	cmd := &cobra.Command{}
	// Simulate: config.ini mydb -- echo hello
	cmd.SetArgs([]string{"config.ini", "mydb", "--", "echo", "hello"})
	cmd.ParseFlags([]string{"config.ini", "mydb", "--", "echo", "hello"})

	iniFile, section, command, err := splitDashArgs(cmd, cmd.Flags().Args())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if iniFile != "config.ini" {
		t.Errorf("iniFile = %q, want %q", iniFile, "config.ini")
	}
	if section != "mydb" {
		t.Errorf("section = %q, want %q", section, "mydb")
	}
	if len(command) != 2 || command[0] != "echo" || command[1] != "hello" {
		t.Errorf("command = %v, want [echo hello]", command)
	}
}

func TestSplitDashArgs_TooManyPositional(t *testing.T) {
	cmd := &cobra.Command{}
	_, _, _, err := splitDashArgs(cmd, []string{"a", "b", "c"})
	if err == nil {
		t.Fatal("expected error for 3 positional args")
	}
}

func TestSplitDashArgs_ZeroPositional(t *testing.T) {
	cmd := &cobra.Command{}
	_, _, _, err := splitDashArgs(cmd, []string{})
	if err == nil {
		t.Fatal("expected error for 0 positional args")
	}
}

// ---------------------------------------------------------------------------
// buildEnvVars
// ---------------------------------------------------------------------------

func TestBuildEnvVars(t *testing.T) {
	s, _ := pgini.NewSection("")
	s.SetParam("host", "localhost")
	s.SetParam("port", "5432")

	vars := buildEnvVars(s)

	want := map[string]bool{
		"HOST=localhost": true,
		"PORT=5432":      true,
	}
	if len(vars) != len(want) {
		t.Fatalf("got %d vars, want %d", len(vars), len(want))
	}
	for _, v := range vars {
		if !want[v] {
			t.Errorf("unexpected var: %s", v)
		}
	}
}

func TestBuildEnvVars_UppercasesKeys(t *testing.T) {
	s, _ := pgini.NewSection("")
	s.SetParam("db_host", "localhost")

	vars := buildEnvVars(s)
	if len(vars) != 1 || vars[0] != "DB_HOST=localhost" {
		t.Errorf("got %v, want [DB_HOST=localhost]", vars)
	}
}

// ---------------------------------------------------------------------------
// mergeEnv
// ---------------------------------------------------------------------------

func TestMergeEnv_Appends(t *testing.T) {
	current := []string{"A=1", "B=2"}
	overlay := []string{"C=3"}
	result := mergeEnv(current, overlay)
	if len(result) != 3 {
		t.Fatalf("len = %d, want 3", len(result))
	}
	if result[2] != "C=3" {
		t.Errorf("result[2] = %q, want %q", result[2], "C=3")
	}
}

func TestMergeEnv_Overrides(t *testing.T) {
	current := []string{"A=1", "B=2"}
	overlay := []string{"A=override"}
	result := mergeEnv(current, overlay)
	if len(result) != 2 {
		t.Fatalf("len = %d, want 2", len(result))
	}
	if result[0] != "A=override" {
		t.Errorf("result[0] = %q, want %q", result[0], "A=override")
	}
	if result[1] != "B=2" {
		t.Errorf("result[1] = %q, want %q", result[1], "B=2")
	}
}

func TestMergeEnv_EmptyCurrent(t *testing.T) {
	result := mergeEnv(nil, []string{"A=1"})
	if len(result) != 1 || result[0] != "A=1" {
		t.Errorf("got %v, want [A=1]", result)
	}
}

func TestMergeEnv_EmptyOverlay(t *testing.T) {
	result := mergeEnv([]string{"A=1"}, nil)
	if len(result) != 1 || result[0] != "A=1" {
		t.Errorf("got %v, want [A=1]", result)
	}
}

// ---------------------------------------------------------------------------
// env command — Cobra error paths
// ---------------------------------------------------------------------------

func TestEnvCmd_MissingFile(t *testing.T) {
	cmd := newTestRootCmd()
	cmd.SetArgs([]string{"env", "/nonexistent/file.ini", "--", "echo"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestEnvCmd_MissingSection(t *testing.T) {
	ini := writeIni(t, "[mydb]\nhost = localhost\n")
	cmd := newTestRootCmd()
	cmd.SetArgs([]string{"env", ini, "nosection", "--", "echo"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing section")
	}
}

func TestEnvCmd_MissingCommand(t *testing.T) {
	ini := writeIni(t, "host = localhost\n")
	cmd := newTestRootCmd()
	cmd.SetArgs([]string{"env", ini, "--"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing command after --")
	}
}

func TestEnvCmd_NoArgs(t *testing.T) {
	cmd := newTestRootCmd()
	cmd.SetArgs([]string{"env"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for no args")
	}
}

// newTestRootCmd creates a fresh root command tree for in-process testing.
// Each test needs its own instance to avoid shared state between tests.
func newTestRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "inigo",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.PersistentFlags().BoolVarP(&silent, "silent", "s", false, "suppress error messages on stderr")
	root.AddCommand(envCmd)
	root.AddCommand(jsonCmd)
	return root
}
