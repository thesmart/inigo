package inigo

import (
	"testing"
)

func TestStripComment(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"no comment", "key = value", "key = value"},
		{"comment only", "# full line comment", ""},
		{"trailing comment", "key = value # comment", "key = value "},
		{"hash in single quotes", "key = 'val#ue'", "key = 'val#ue'"},
		{"hash in quotes then comment", "key = 'val#ue' # comment", "key = 'val#ue' "},
		{"doubled quote escape", "key = 'it''s' # comment", "key = 'it''s' "},
		{"backslash quote escape", `key = 'it\'s' # comment`, `key = 'it\'s' `},
		{"empty string", "", ""},
		{"hash at start", "#comment", ""},
		{"no hash", "key = value", "key = value"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripComment(tt.in)
			if got != tt.want {
				t.Errorf("stripComment(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestParseSectionHeader(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    string
		wantErr bool
	}{
		{"simple", "[myservice]", "myservice", false},
		{"with spaces", "[  myservice  ]", "myservice", false},
		{"unterminated", "[myservice", "", true},
		{"empty name", "[]", "", true},
		{"whitespace only name", "[   ]", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseSectionHeader(tt.in)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseSectionHeader(%q) error = %v, wantErr %v", tt.in, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("parseSectionHeader(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestIsValidParamName(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want bool
	}{
		{"alpha start", "host", true},
		{"underscore start", "_private", true},
		{"with digits", "port5432", true},
		{"with dollar", "var$1", true},
		{"with underscore", "my_var", true},
		{"uppercase", "HOST", true},
		{"mixed case", "myHost", true},
		{"digit start", "1host", false},
		{"dollar start", "$var", false},
		{"hyphen", "my-var", false},
		{"space", "my var", false},
		{"dot", "my.var", false},
		{"empty", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidParamName(tt.in)
			if got != tt.want {
				t.Errorf("isValidParamName(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestParseKeyValue(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		wantKey string
		wantVal string
		wantErr bool
	}{
		{"simple", "host = localhost", "host", "localhost", false},
		{"no spaces", "host=localhost", "host", "localhost", false},
		{"empty value", "host =", "host", "", false},
		{"bare param", "host", "host", "", false},
		{"quoted value", "name = 'hello world'", "name", "hello world", false},
		{"equals in value", "cmd = a=b", "cmd", "a=b", false},
		{"invalid name", "1bad = val", "", "", true},
		{"empty name", " = val", "", "", true},
		{"doubled quote", "msg = 'it''s'", "msg", "it's", false},
		{"backslash quote", `msg = 'it\'s'`, "msg", "it's", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, val, err := parseKeyValue(tt.in)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseKeyValue(%q) error = %v, wantErr %v", tt.in, err, tt.wantErr)
				return
			}
			if key != tt.wantKey {
				t.Errorf("parseKeyValue(%q) key = %q, want %q", tt.in, key, tt.wantKey)
			}
			if val != tt.wantVal {
				t.Errorf("parseKeyValue(%q) val = %q, want %q", tt.in, val, tt.wantVal)
			}
		})
	}
}

func TestParseQuotedValue(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    string
		wantErr bool
	}{
		{"simple", "'hello'", "hello", false},
		{"empty", "''", "", false},
		{"with spaces", "'hello world'", "hello world", false},
		{"doubled escape", "'it''s'", "it's", false},
		{"backslash escape", `'it\'s'`, "it's", false},
		{"multiple escapes", "'a''b''c'", "a'b'c", false},
		{"unterminated", "'hello", "", true},
		{"hash inside", "'val#ue'", "val#ue", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseQuotedValue(tt.in)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseQuotedValue(%q) error = %v, wantErr %v", tt.in, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("parseQuotedValue(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestParseValue(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    string
		wantErr bool
	}{
		{"empty", "", "", false},
		{"unquoted", "localhost", "localhost", false},
		{"quoted", "'hello'", "hello", false},
		{"unquoted number", "5432", "5432", false},
		{"unterminated quote", "'oops", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseValue(tt.in)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseValue(%q) error = %v, wantErr %v", tt.in, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("parseValue(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestParseBool(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    bool
		wantErr bool
	}{
		// Full words
		{"on", "on", true, false},
		{"off", "off", false, false},
		{"true", "true", true, false},
		{"false", "false", false, false},
		{"yes", "yes", true, false},
		{"no", "no", false, false},
		{"1", "1", true, false},
		{"0", "0", false, false},

		// Case insensitivity
		{"ON", "ON", true, false},
		{"TRUE", "TRUE", true, false},
		{"False", "False", false, false},
		{"YES", "YES", true, false},

		// Unambiguous prefixes
		{"t", "t", true, false},
		{"tr", "tr", true, false},
		{"tru", "tru", true, false},
		{"f", "f", false, false},
		{"fa", "fa", false, false},
		{"fal", "fal", false, false},
		{"y", "y", true, false},
		{"ye", "ye", true, false},
		{"n", "n", false, false},
		{"of", "of", false, false},

		// Ambiguous prefix: "o" matches both "on" and "off"
		{"o ambiguous", "o", false, true},

		// Whitespace handling
		{"leading space", "  true", true, false},
		{"trailing space", "true  ", true, false},

		// Errors
		{"empty", "", false, true},
		{"garbage", "maybe", false, true},
		{"number 2", "2", false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseBool(tt.in)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseBool(%q) error = %v, wantErr %v", tt.in, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("parseBool(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestParseInt(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    int64
		wantErr bool
	}{
		{"decimal", "42", 42, false},
		{"negative", "-7", -7, false},
		{"zero", "0", 0, false},
		{"hex", "0xFF", 255, false},
		{"hex lower", "0xff", 255, false},
		{"octal", "010", 8, false},
		{"float rounds down", "3.2", 3, false},
		{"float rounds up", "3.7", 4, false},
		{"float half up", "2.5", 3, false},
		{"negative float", "-1.6", -2, false},
		{"whitespace", "  42  ", 42, false},
		{"empty", "", 0, true},
		{"garbage", "abc", 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseInt(tt.in)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseInt(%q) error = %v, wantErr %v", tt.in, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("parseInt(%q) = %d, want %d", tt.in, got, tt.want)
			}
		})
	}
}

func TestParamFloat64(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    float64
		wantErr bool
	}{
		{"integer", "42", 42.0, false},
		{"decimal", "3.14", 3.14, false},
		{"negative", "-2.5", -2.5, false},
		{"whitespace", "  1.5  ", 1.5, false},
		{"empty", "", 0, true},
		{"garbage", "abc", 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Param{value: tt.value}
			got, err := p.Float64()
			if (err != nil) != tt.wantErr {
				t.Errorf("Param(%q).Float64() error = %v, wantErr %v", tt.value, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("Param(%q).Float64() = %f, want %f", tt.value, got, tt.want)
			}
		})
	}
}

func TestMatchDirective(t *testing.T) {
	tests := []struct {
		name     string
		lower    string
		original string
		dir      string
		want     bool
		wantRest string
	}{
		{"include with space", "include 'f.conf'", "include 'f.conf'", "include", true, "'f.conf'"},
		{"include with tab", "include\t'f.conf'", "include\t'f.conf'", "include", true, "'f.conf'"},
		{"include_dir", "include_dir '/etc'", "include_dir '/etc'", "include_dir", true, "'/etc'"},
		{"include_if_exists", "include_if_exists 'f'", "include_if_exists 'f'", "include_if_exists", true, "'f'"},
		{"not a directive", "included = true", "included = true", "include", false, ""},
		{"include_var param", "include_var = 5", "include_var = 5", "include", false, ""},
		{"too short", "include", "include", "include", false, ""},
		{"no match", "something else", "something else", "include", false, ""},
		{"adjacent quote", "include'f.conf'", "include'f.conf'", "include", true, "'f.conf'"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var directive, rest string
			got := matchDirective(tt.lower, tt.original, tt.dir, &directive, &rest)
			if got != tt.want {
				t.Errorf("matchDirective(%q, %q, %q) = %v, want %v", tt.lower, tt.original, tt.dir, got, tt.want)
			}
			if got && rest != tt.wantRest {
				t.Errorf("matchDirective rest = %q, want %q", rest, tt.wantRest)
			}
		})
	}
}

func TestParseIncludePath(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    string
		wantErr bool
	}{
		{"quoted", "'/etc/pg.conf'", "/etc/pg.conf", false},
		{"unquoted", "/etc/pg.conf", "/etc/pg.conf", false},
		{"unquoted with trailing", "/etc/pg.conf extra", "/etc/pg.conf", false},
		{"quoted with spaces", "'path with spaces/file.conf'", "path with spaces/file.conf", false},
		{"empty", "", "", true},
		{"whitespace only", "   ", "", true},
		{"unterminated quote", "'/etc/pg.conf", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseIncludePath(tt.in)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseIncludePath(%q) error = %v, wantErr %v", tt.in, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("parseIncludePath(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestResolvePath(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		baseDir string
		want    string
	}{
		{"absolute unchanged", "/etc/pg.conf", "/home/user", "/etc/pg.conf"},
		{"relative resolved", "pg.conf", "/etc", "/etc/pg.conf"},
		{"relative subdir", "conf.d/extra.conf", "/etc", "/etc/conf.d/extra.conf"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolvePath(tt.path, tt.baseDir)
			if got != tt.want {
				t.Errorf("resolvePath(%q, %q) = %q, want %q", tt.path, tt.baseDir, got, tt.want)
			}
		})
	}
}

func TestSectionMethods(t *testing.T) {
	s := &Section{
		name: "test",
		params: map[string]*Param{
			"host": {name: "host", value: "localhost"},
			"port": {name: "port", value: "5432"},
		},
	}

	t.Run("HasParam existing", func(t *testing.T) {
		if !s.HasParam("host") {
			t.Error("HasParam(host) = false, want true")
		}
	})
	t.Run("HasParam case insensitive", func(t *testing.T) {
		if !s.HasParam("HOST") {
			t.Error("HasParam(HOST) = false, want true")
		}
	})
	t.Run("HasParam missing", func(t *testing.T) {
		if s.HasParam("dbname") {
			t.Error("HasParam(dbname) = true, want false")
		}
	})

	t.Run("GetParam existing", func(t *testing.T) {
		p := s.GetParam("host")
		if p.String() != "localhost" {
			t.Errorf("GetParam(host).String() = %q, want %q", p.String(), "localhost")
		}
	})
	t.Run("GetParam case insensitive", func(t *testing.T) {
		p := s.GetParam("PORT")
		if p.String() != "5432" {
			t.Errorf("GetParam(PORT).String() = %q, want %q", p.String(), "5432")
		}
	})
	t.Run("GetParam missing returns empty", func(t *testing.T) {
		p := s.GetParam("dbname")
		if p.String() != "" {
			t.Errorf("GetParam(dbname).String() = %q, want empty", p.String())
		}
	})

	t.Run("AllParams sorted", func(t *testing.T) {
		names := s.AllParams()
		if len(names) != 2 || names[0] != "host" || names[1] != "port" {
			t.Errorf("AllParams() = %v, want [host port]", names)
		}
	})
}

func TestConfigMethods(t *testing.T) {
	c := &Config{
		sections: map[string]*Section{
			"":    {name: "", params: make(map[string]*Param)},
			"svc": {name: "svc", params: make(map[string]*Param)},
			"db":  {name: "db", params: make(map[string]*Param)},
		},
	}

	t.Run("Section found", func(t *testing.T) {
		if c.Section("svc") == nil {
			t.Error("Section(svc) returned nil")
		}
	})
	t.Run("Section missing", func(t *testing.T) {
		if c.Section("nope") != nil {
			t.Error("Section(nope) should return nil")
		}
	})
	t.Run("HasSection", func(t *testing.T) {
		if !c.HasSection("svc") {
			t.Error("HasSection(svc) = false, want true")
		}
		if c.HasSection("nope") {
			t.Error("HasSection(nope) = true, want false")
		}
	})

	t.Run("Section found", func(t *testing.T) {
		if c.Section("db") == nil {
			t.Error("Section(db) = nil, want non-nil")
		}
	})
	t.Run("Section missing", func(t *testing.T) {
		if c.Section("nope") != nil {
			t.Error("Section(nope) should be nil")
		}
	})

	t.Run("SectionNames excludes default", func(t *testing.T) {
		names := c.SectionNames()
		if len(names) != 2 || names[0] != "db" || names[1] != "svc" {
			t.Errorf("SectionNames() = %v, want [db svc]", names)
		}
	})
}

func TestParamTypeMethods(t *testing.T) {
	t.Run("String", func(t *testing.T) {
		p := &Param{value: "hello"}
		if p.String() != "hello" {
			t.Errorf("String() = %q, want %q", p.String(), "hello")
		}
	})
	t.Run("Bool", func(t *testing.T) {
		p := &Param{value: "yes"}
		v, err := p.Bool()
		if err != nil || !v {
			t.Errorf("Bool() = %v, %v; want true, nil", v, err)
		}
	})
	t.Run("Int", func(t *testing.T) {
		p := &Param{value: "42"}
		v, err := p.Int()
		if err != nil || v != 42 {
			t.Errorf("Int() = %d, %v; want 42, nil", v, err)
		}
	})
	t.Run("Float64", func(t *testing.T) {
		p := &Param{value: "3.14"}
		v, err := p.Float64()
		if err != nil || v != 3.14 {
			t.Errorf("Float64() = %f, %v; want 3.14, nil", v, err)
		}
	})
}
