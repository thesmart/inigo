package main

import (
	"strings"
	"testing"

	"github.com/thesmart/inigo"
)

func TestParseArgs(t *testing.T) {
	tests := []struct {
		name    string
		argv    []string
		want    args
		wantErr bool
	}{
		{
			"basic",
			[]string{"config.ini", "mydb", "--", "psql"},
			args{iniFile: "config.ini", section: "mydb", command: []string{"psql"}},
			false,
		},
		{
			"with prefix long flag",
			[]string{"--prefix", "PG", "config.ini", "mydb", "--", "psql"},
			args{prefix: "PG", iniFile: "config.ini", section: "mydb", command: []string{"psql"}},
			false,
		},
		{
			"with prefix short flag",
			[]string{"-p", "PG", "config.ini", "mydb", "--", "psql"},
			args{prefix: "PG", iniFile: "config.ini", section: "mydb", command: []string{"psql"}},
			false,
		},
		{
			"prefix lowercase normalized to upper",
			[]string{"-p", "pg", "config.ini", "mydb", "--", "psql"},
			args{prefix: "PG", iniFile: "config.ini", section: "mydb", command: []string{"psql"}},
			false,
		},
		{
			"command with args",
			[]string{"config.ini", "mydb", "--", "psql", "-U", "admin", "-d", "myapp"},
			args{iniFile: "config.ini", section: "mydb", command: []string{"psql", "-U", "admin", "-d", "myapp"}},
			false,
		},
		{
			"flags after positional",
			[]string{"config.ini", "mydb", "--prefix", "PG", "--", "psql"},
			args{prefix: "PG", iniFile: "config.ini", section: "mydb", command: []string{"psql"}},
			false,
		},
		{
			"missing separator",
			[]string{"config.ini", "mydb", "psql"},
			args{},
			true,
		},
		{
			"missing command after separator",
			[]string{"config.ini", "mydb", "--"},
			args{},
			true,
		},
		{
			"too few positional args",
			[]string{"config.ini", "--", "psql"},
			args{},
			true,
		},
		{
			"too many positional args",
			[]string{"config.ini", "mydb", "extra", "--", "psql"},
			args{},
			true,
		},
		{
			"unknown flag",
			[]string{"--verbose", "config.ini", "mydb", "--", "psql"},
			args{},
			true,
		},
		{
			"prefix missing value",
			[]string{"--prefix", "--", "psql"},
			args{},
			true,
		},
		{
			"no args at all",
			[]string{},
			args{},
			true,
		},
		{
			"only separator",
			[]string{"--"},
			args{},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseArgs(tt.argv)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseArgs(%v) error = %v, wantErr %v", tt.argv, err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if got.prefix != tt.want.prefix {
				t.Errorf("prefix = %q, want %q", got.prefix, tt.want.prefix)
			}
			if got.iniFile != tt.want.iniFile {
				t.Errorf("iniFile = %q, want %q", got.iniFile, tt.want.iniFile)
			}
			if got.section != tt.want.section {
				t.Errorf("section = %q, want %q", got.section, tt.want.section)
			}
			if len(got.command) != len(tt.want.command) {
				t.Errorf("command = %v, want %v", got.command, tt.want.command)
				return
			}
			for i := range got.command {
				if got.command[i] != tt.want.command[i] {
					t.Errorf("command[%d] = %q, want %q", i, got.command[i], tt.want.command[i])
				}
			}
		})
	}
}

func TestBuildEnv(t *testing.T) {
	cfg, err := inigo.Parse(strings.NewReader("[mydb]\nhost = localhost\nport = 5432\ndbname = myapp\n"))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	sec := cfg.Section("mydb")

	t.Run("no prefix", func(t *testing.T) {
		env := buildEnv(sec, "")
		want := map[string]string{
			"DBNAME": "myapp",
			"HOST":   "localhost",
			"PORT":   "5432",
		}
		if len(env) != len(want) {
			t.Fatalf("got %d entries, want %d", len(env), len(want))
		}
		for _, entry := range env {
			key, val, _ := strings.Cut(entry, "=")
			if w, ok := want[key]; !ok {
				t.Errorf("unexpected key %q", key)
			} else if val != w {
				t.Errorf("%s = %q, want %q", key, val, w)
			}
		}
	})

	t.Run("with prefix", func(t *testing.T) {
		env := buildEnv(sec, "PG")
		want := map[string]string{
			"PGDBNAME": "myapp",
			"PGHOST":   "localhost",
			"PGPORT":   "5432",
		}
		if len(env) != len(want) {
			t.Fatalf("got %d entries, want %d", len(env), len(want))
		}
		for _, entry := range env {
			key, val, _ := strings.Cut(entry, "=")
			if w, ok := want[key]; !ok {
				t.Errorf("unexpected key %q", key)
			} else if val != w {
				t.Errorf("%s = %q, want %q", key, val, w)
			}
		}
	})

	t.Run("empty section", func(t *testing.T) {
		cfg2, _ := inigo.Parse(strings.NewReader("[empty]\n"))
		sec2 := cfg2.Section("empty")
		env := buildEnv(sec2, "PG")
		if len(env) != 0 {
			t.Errorf("got %d entries, want 0", len(env))
		}
	})
}

func TestMergeEnv(t *testing.T) {
	t.Run("override existing key", func(t *testing.T) {
		current := []string{"HOME=/home/user", "HOST=oldvalue", "PATH=/usr/bin"}
		overlay := []string{"HOST=newvalue"}
		result := mergeEnv(current, overlay)

		if len(result) != 3 {
			t.Fatalf("got %d entries, want 3", len(result))
		}
		// HOST should be replaced in-place at index 1
		if result[0] != "HOME=/home/user" {
			t.Errorf("result[0] = %q, want %q", result[0], "HOME=/home/user")
		}
		if result[1] != "HOST=newvalue" {
			t.Errorf("result[1] = %q, want %q", result[1], "HOST=newvalue")
		}
		if result[2] != "PATH=/usr/bin" {
			t.Errorf("result[2] = %q, want %q", result[2], "PATH=/usr/bin")
		}
	})

	t.Run("append new keys", func(t *testing.T) {
		current := []string{"HOME=/home/user"}
		overlay := []string{"PGHOST=localhost", "PGPORT=5432"}
		result := mergeEnv(current, overlay)

		if len(result) != 3 {
			t.Fatalf("got %d entries, want 3", len(result))
		}
		if result[0] != "HOME=/home/user" {
			t.Errorf("result[0] = %q, want %q", result[0], "HOME=/home/user")
		}
		if result[1] != "PGHOST=localhost" {
			t.Errorf("result[1] = %q, want %q", result[1], "PGHOST=localhost")
		}
		if result[2] != "PGPORT=5432" {
			t.Errorf("result[2] = %q, want %q", result[2], "PGPORT=5432")
		}
	})

	t.Run("empty overlay", func(t *testing.T) {
		current := []string{"HOME=/home/user", "PATH=/usr/bin"}
		result := mergeEnv(current, nil)

		if len(result) != 2 {
			t.Fatalf("got %d entries, want 2", len(result))
		}
	})

	t.Run("empty current", func(t *testing.T) {
		overlay := []string{"PGHOST=localhost"}
		result := mergeEnv(nil, overlay)

		if len(result) != 1 {
			t.Fatalf("got %d entries, want 1", len(result))
		}
		if result[0] != "PGHOST=localhost" {
			t.Errorf("result[0] = %q, want %q", result[0], "PGHOST=localhost")
		}
	})
}
