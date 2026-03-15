package pgini

import (
	"fmt"
	"math"
	"reflect"
	"testing"
)

// --- test helper types for unmarshal ---

type unmarshalPrimitives struct {
	Host    string  `ini:"host"`
	Port    int     `ini:"port"`
	Debug   bool    `ini:"debug"`
	Rate    float64 `ini:"rate"`
	Small   int8    `ini:"small"`
	Medium  int16   `ini:"medium"`
	Large   int32   `ini:"large"`
	Huge    int64   `ini:"huge"`
	USmall  uint8   `ini:"usmall"`
	UMedium uint16  `ini:"umedium"`
	ULarge  uint32  `ini:"ularge"`
	UHuge   uint64  `ini:"uhuge"`
	Half    float32 `ini:"half"`
	NoTag   string
	Empty   string `ini:""`
}

type unmarshalCustomType struct {
	Name string          `ini:"name"`
	Data unmarshalNested `ini:"data"`
}

type unmarshalNested struct {
	X int
	Y int
}

func (s *unmarshalCustomType) UnmarshalData(value string) (*unmarshalNested, error) {
	var x, y int
	_, err := fmt.Sscanf(value, "%d,%d", &x, &y)
	if err != nil {
		return nil, err
	}
	return &unmarshalNested{X: x, Y: y}, nil
}

type unmarshalCustomError struct {
	Data unmarshalNested `ini:"data"`
}

func (s *unmarshalCustomError) UnmarshalData(value string) (*unmarshalNested, error) {
	return nil, fmt.Errorf("unmarshal error")
}

type unmarshalNoCustom struct {
	Data unmarshalNested `ini:"data"`
}

type unmarshalCustomOverride struct {
	Port int `ini:"port"`
}

func (s *unmarshalCustomOverride) UnmarshalPort(value string) (*int, error) {
	var v int
	_, err := fmt.Sscanf(value, "port-%d", &v)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

// --- UnmarshalSection tests ---

func TestUnmarshalSection(t *testing.T) {
	t.Run("primitive fields", func(t *testing.T) {
		f, err := NewIniFile(nonExistingPath("test.conf"))
		if err != nil {
			t.Fatalf("NewIniFile: %v", err)
		}

		s := f.GetSection("")
		s.SetParam("host", "localhost")
		s.SetParam("port", "5432")
		s.SetParam("debug", "true")
		s.SetParam("rate", "3.14")
		s.SetParam("small", "-1")
		s.SetParam("medium", "1000")
		s.SetParam("large", "100000")
		s.SetParam("huge", "9999999999")
		s.SetParam("usmall", "255")
		s.SetParam("umedium", "65535")
		s.SetParam("ularge", "4294967295")
		s.SetParam("uhuge", "18446744073709551615")
		s.SetParam("half", "1.5")

		v := &unmarshalPrimitives{}
		if err := f.UnmarshalSection("", v); err != nil {
			t.Fatalf("UnmarshalSection: %v", err)
		}

		if v.Host != "localhost" {
			t.Errorf("Host = %q, want %q", v.Host, "localhost")
		}
		if v.Port != 5432 {
			t.Errorf("Port = %d, want %d", v.Port, 5432)
		}
		if v.Debug != true {
			t.Errorf("Debug = %v, want true", v.Debug)
		}
		if v.Rate != 3.14 {
			t.Errorf("Rate = %f, want 3.14", v.Rate)
		}
		if v.Small != -1 {
			t.Errorf("Small = %d, want -1", v.Small)
		}
		if v.Medium != 1000 {
			t.Errorf("Medium = %d, want 1000", v.Medium)
		}
		if v.Large != 100000 {
			t.Errorf("Large = %d, want 100000", v.Large)
		}
		if v.Huge != 9999999999 {
			t.Errorf("Huge = %d, want 9999999999", v.Huge)
		}
		if v.USmall != 255 {
			t.Errorf("USmall = %d, want 255", v.USmall)
		}
		if v.UMedium != 65535 {
			t.Errorf("UMedium = %d, want 65535", v.UMedium)
		}
		if v.ULarge != 4294967295 {
			t.Errorf("ULarge = %d, want 4294967295", v.ULarge)
		}
		if v.UHuge != 18446744073709551615 {
			t.Errorf("UHuge = %d, want 18446744073709551615", v.UHuge)
		}
		if v.Half != 1.5 {
			t.Errorf("Half = %f, want 1.5", v.Half)
		}
	})

	t.Run("missing param leaves field at zero value", func(t *testing.T) {
		f, err := NewIniFile(nonExistingPath("test.conf"))
		if err != nil {
			t.Fatalf("NewIniFile: %v", err)
		}

		s := f.GetSection("")
		s.SetParam("host", "example.com")

		v := &unmarshalPrimitives{}
		if err := f.UnmarshalSection("", v); err != nil {
			t.Fatalf("UnmarshalSection: %v", err)
		}

		if v.Host != "example.com" {
			t.Errorf("Host = %q, want %q", v.Host, "example.com")
		}
		if v.Port != 0 {
			t.Errorf("Port should be zero value, got %d", v.Port)
		}
	})

	t.Run("named section", func(t *testing.T) {
		f, err := NewIniFile(nonExistingPath("test.conf"))
		if err != nil {
			t.Fatalf("NewIniFile: %v", err)
		}

		sec, _ := f.AddSection("production")
		sec.SetParam("host", "prod.example.com")
		sec.SetParam("port", "443")

		v := &unmarshalPrimitives{}
		if err := f.UnmarshalSection("production", v); err != nil {
			t.Fatalf("UnmarshalSection: %v", err)
		}

		if v.Host != "prod.example.com" {
			t.Errorf("Host = %q, want %q", v.Host, "prod.example.com")
		}
		if v.Port != 443 {
			t.Errorf("Port = %d, want %d", v.Port, 443)
		}
	})

	t.Run("custom unmarshal method", func(t *testing.T) {
		f, err := NewIniFile(nonExistingPath("test.conf"))
		if err != nil {
			t.Fatalf("NewIniFile: %v", err)
		}

		s := f.GetSection("")
		s.SetParam("name", "test")
		s.SetParam("data", "10,20")

		v := &unmarshalCustomType{}
		if err := f.UnmarshalSection("", v); err != nil {
			t.Fatalf("UnmarshalSection: %v", err)
		}

		if v.Name != "test" {
			t.Errorf("Name = %q, want %q", v.Name, "test")
		}
		if v.Data.X != 10 || v.Data.Y != 20 {
			t.Errorf("Data = {%d,%d}, want {10,20}", v.Data.X, v.Data.Y)
		}
	})

	t.Run("custom unmarshal overrides default for primitive", func(t *testing.T) {
		f, err := NewIniFile(nonExistingPath("test.conf"))
		if err != nil {
			t.Fatalf("NewIniFile: %v", err)
		}

		s := f.GetSection("")
		s.SetParam("port", "port-8080")

		v := &unmarshalCustomOverride{}
		if err := f.UnmarshalSection("", v); err != nil {
			t.Fatalf("UnmarshalSection: %v", err)
		}

		if v.Port != 8080 {
			t.Errorf("Port = %d, want 8080", v.Port)
		}
	})

	t.Run("custom unmarshal returns error", func(t *testing.T) {
		f, err := NewIniFile(nonExistingPath("test.conf"))
		if err != nil {
			t.Fatalf("NewIniFile: %v", err)
		}

		s := f.GetSection("")
		s.SetParam("data", "bad")

		v := &unmarshalCustomError{}
		err = f.UnmarshalSection("", v)
		if err == nil {
			t.Fatal("expected error from custom unmarshal")
		}
	})

	t.Run("unsupported type without custom method errors", func(t *testing.T) {
		f, err := NewIniFile(nonExistingPath("test.conf"))
		if err != nil {
			t.Fatalf("NewIniFile: %v", err)
		}

		s := f.GetSection("")
		s.SetParam("data", "1,2")

		v := &unmarshalNoCustom{}
		err = f.UnmarshalSection("", v)
		if err == nil {
			t.Fatal("expected error for unsupported type without custom method")
		}
	})

	t.Run("non-pointer arg errors", func(t *testing.T) {
		f, err := NewIniFile(nonExistingPath("test.conf"))
		if err != nil {
			t.Fatalf("NewIniFile: %v", err)
		}

		err = f.UnmarshalSection("", unmarshalPrimitives{})
		if err == nil {
			t.Fatal("expected error for non-pointer arg")
		}
	})

	t.Run("non-struct pointer errors", func(t *testing.T) {
		f, err := NewIniFile(nonExistingPath("test.conf"))
		if err != nil {
			t.Fatalf("NewIniFile: %v", err)
		}

		s := "not a struct"
		err = f.UnmarshalSection("", &s)
		if err == nil {
			t.Fatal("expected error for non-struct pointer")
		}
	})

	t.Run("section not found errors", func(t *testing.T) {
		f, err := NewIniFile(nonExistingPath("test.conf"))
		if err != nil {
			t.Fatalf("NewIniFile: %v", err)
		}

		v := &unmarshalPrimitives{}
		err = f.UnmarshalSection("nonexistent", v)
		if err == nil {
			t.Fatal("expected error for missing section")
		}
	})
}

// --- parseBool tests ---

func TestParseBool(t *testing.T) {
	trueValues := []string{"t", "1", "true", "on", "y", "yes", "T", "TRUE", "On", "YES"}
	for _, s := range trueValues {
		t.Run("true/"+s, func(t *testing.T) {
			v, err := parseBool(s)
			if err != nil {
				t.Fatalf("parseBool(%q): %v", s, err)
			}
			if !v {
				t.Errorf("parseBool(%q) = false, want true", s)
			}
		})
	}

	falseValues := []string{"f", "0", "false", "off", "n", "no", "F", "FALSE", "Off", "NO"}
	for _, s := range falseValues {
		t.Run("false/"+s, func(t *testing.T) {
			v, err := parseBool(s)
			if err != nil {
				t.Fatalf("parseBool(%q): %v", s, err)
			}
			if v {
				t.Errorf("parseBool(%q) = true, want false", s)
			}
		})
	}

	t.Run("empty string errors", func(t *testing.T) {
		_, err := parseBool("")
		if err == nil {
			t.Fatal("expected error for empty string")
		}
	})

	t.Run("whitespace-only errors", func(t *testing.T) {
		_, err := parseBool("   ")
		if err == nil {
			t.Fatal("expected error for whitespace-only")
		}
	})

	t.Run("invalid value errors", func(t *testing.T) {
		_, err := parseBool("maybe")
		if err == nil {
			t.Fatal("expected error for invalid bool")
		}
	})

	t.Run("whitespace is trimmed", func(t *testing.T) {
		v, err := parseBool("  true  ")
		if err != nil {
			t.Fatalf("parseBool: %v", err)
		}
		if !v {
			t.Error("expected true after trimming")
		}
	})
}

// --- parseInt tests ---

func TestParseInt(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want int64
	}{
		{"decimal", "42", 42},
		{"negative", "-1", -1},
		{"hex", "0xFF", 255},
		{"octal", "0777", 511},
		{"zero", "0", 0},
		{"float fallback", "3.7", 3},
		{"negative float", "-2.9", -3},
		{"whitespace", "  42  ", 42},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := parseInt(tt.in)
			if err != nil {
				t.Fatalf("parseInt(%q): %v", tt.in, err)
			}
			if v != tt.want {
				t.Errorf("parseInt(%q) = %d, want %d", tt.in, v, tt.want)
			}
		})
	}

	t.Run("empty errors", func(t *testing.T) {
		_, err := parseInt("")
		if err == nil {
			t.Fatal("expected error for empty string")
		}
	})

	t.Run("invalid errors", func(t *testing.T) {
		_, err := parseInt("abc")
		if err == nil {
			t.Fatal("expected error for non-numeric")
		}
	})
}

// --- parseUint tests ---

func TestParseUint(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want uint64
	}{
		{"decimal", "42", 42},
		{"hex", "0xFF", 255},
		{"zero", "0", 0},
		{"float fallback", "3.7", 3},
		{"whitespace", "  42  ", 42},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := parseUint(tt.in)
			if err != nil {
				t.Fatalf("parseUint(%q): %v", tt.in, err)
			}
			if v != tt.want {
				t.Errorf("parseUint(%q) = %d, want %d", tt.in, v, tt.want)
			}
		})
	}

	t.Run("empty errors", func(t *testing.T) {
		_, err := parseUint("")
		if err == nil {
			t.Fatal("expected error for empty string")
		}
	})

	t.Run("negative float errors", func(t *testing.T) {
		_, err := parseUint("-1.5")
		if err == nil {
			t.Fatal("expected error for negative float")
		}
	})

	t.Run("invalid errors", func(t *testing.T) {
		_, err := parseUint("abc")
		if err == nil {
			t.Fatal("expected error for non-numeric")
		}
	})
}

// --- parseFloat tests ---

func TestParseFloat(t *testing.T) {
	t.Run("float64", func(t *testing.T) {
		v, err := parseFloat("3.14", 64)
		if err != nil {
			t.Fatalf("parseFloat: %v", err)
		}
		if v != 3.14 {
			t.Errorf("parseFloat = %f, want 3.14", v)
		}
	})

	t.Run("float32", func(t *testing.T) {
		v, err := parseFloat("1.5", 32)
		if err != nil {
			t.Fatalf("parseFloat: %v", err)
		}
		if float32(v) != 1.5 {
			t.Errorf("parseFloat = %f, want 1.5", v)
		}
	})

	t.Run("empty errors", func(t *testing.T) {
		_, err := parseFloat("", 64)
		if err == nil {
			t.Fatal("expected error for empty string")
		}
	})

	t.Run("invalid errors", func(t *testing.T) {
		_, err := parseFloat("abc", 64)
		if err == nil {
			t.Fatal("expected error for non-numeric")
		}
	})

	t.Run("whitespace trimmed", func(t *testing.T) {
		v, err := parseFloat("  2.5  ", 64)
		if err != nil {
			t.Fatalf("parseFloat: %v", err)
		}
		if v != 2.5 {
			t.Errorf("parseFloat = %f, want 2.5", v)
		}
	})
}

// --- intInBounds tests ---

func TestIntInBounds(t *testing.T) {
	tests := []struct {
		name string
		kind reflect.Kind
		val  int64
		want bool
	}{
		{"int8 min", reflect.Int8, math.MinInt8, true},
		{"int8 max", reflect.Int8, math.MaxInt8, true},
		{"int8 overflow", reflect.Int8, math.MaxInt8 + 1, false},
		{"int8 underflow", reflect.Int8, math.MinInt8 - 1, false},
		{"int16 min", reflect.Int16, math.MinInt16, true},
		{"int16 max", reflect.Int16, math.MaxInt16, true},
		{"int16 overflow", reflect.Int16, math.MaxInt16 + 1, false},
		{"int32 min", reflect.Int32, math.MinInt32, true},
		{"int32 max", reflect.Int32, math.MaxInt32, true},
		{"int32 overflow", reflect.Int32, math.MaxInt32 + 1, false},
		{"int64 max", reflect.Int64, math.MaxInt64, true},
		{"int max", reflect.Int, math.MaxInt64, true},
		{"default kind", reflect.Float64, 42, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := intInBounds(tt.kind, tt.val)
			if got != tt.want {
				t.Errorf("intInBounds(%v, %d) = %v, want %v", tt.kind, tt.val, got, tt.want)
			}
		})
	}
}

// --- uintInBounds tests ---

func TestUintInBounds(t *testing.T) {
	tests := []struct {
		name string
		kind reflect.Kind
		val  uint64
		want bool
	}{
		{"uint8 max", reflect.Uint8, math.MaxUint8, true},
		{"uint8 overflow", reflect.Uint8, math.MaxUint8 + 1, false},
		{"uint16 max", reflect.Uint16, math.MaxUint16, true},
		{"uint16 overflow", reflect.Uint16, math.MaxUint16 + 1, false},
		{"uint32 max", reflect.Uint32, math.MaxUint32, true},
		{"uint32 overflow", reflect.Uint32, math.MaxUint32 + 1, false},
		{"uint64 max", reflect.Uint64, math.MaxUint64, true},
		{"uint max", reflect.Uint, math.MaxUint64, true},
		{"default kind", reflect.Float64, 42, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := uintInBounds(tt.kind, tt.val)
			if got != tt.want {
				t.Errorf("uintInBounds(%v, %d) = %v, want %v", tt.kind, tt.val, got, tt.want)
			}
		})
	}
}

// --- setFieldFromParam overflow tests ---

func TestSetFieldFromParamOverflow(t *testing.T) {
	t.Run("int8 overflow", func(t *testing.T) {
		type s struct {
			V int8 `ini:"v"`
		}
		fv := reflect.ValueOf(&s{}).Elem().Field(0)
		err := setFieldFromParam(fv, &Param{Name: "v", Value: "200"})
		if err == nil {
			t.Fatal("expected overflow error")
		}
	})

	t.Run("uint8 overflow", func(t *testing.T) {
		type s struct {
			V uint8 `ini:"v"`
		}
		fv := reflect.ValueOf(&s{}).Elem().Field(0)
		err := setFieldFromParam(fv, &Param{Name: "v", Value: "300"})
		if err == nil {
			t.Fatal("expected overflow error")
		}
	})

	t.Run("bool parse error", func(t *testing.T) {
		type s struct {
			V bool `ini:"v"`
		}
		fv := reflect.ValueOf(&s{}).Elem().Field(0)
		err := setFieldFromParam(fv, &Param{Name: "v", Value: "maybe"})
		if err == nil {
			t.Fatal("expected parse error")
		}
	})

	t.Run("int parse error", func(t *testing.T) {
		type s struct {
			V int `ini:"v"`
		}
		fv := reflect.ValueOf(&s{}).Elem().Field(0)
		err := setFieldFromParam(fv, &Param{Name: "v", Value: "abc"})
		if err == nil {
			t.Fatal("expected parse error")
		}
	})

	t.Run("uint parse error", func(t *testing.T) {
		type s struct {
			V uint `ini:"v"`
		}
		fv := reflect.ValueOf(&s{}).Elem().Field(0)
		err := setFieldFromParam(fv, &Param{Name: "v", Value: "abc"})
		if err == nil {
			t.Fatal("expected parse error")
		}
	})

	t.Run("float32 parse error", func(t *testing.T) {
		type s struct {
			V float32 `ini:"v"`
		}
		fv := reflect.ValueOf(&s{}).Elem().Field(0)
		err := setFieldFromParam(fv, &Param{Name: "v", Value: "abc"})
		if err == nil {
			t.Fatal("expected parse error")
		}
	})

	t.Run("float64 parse error", func(t *testing.T) {
		type s struct {
			V float64 `ini:"v"`
		}
		fv := reflect.ValueOf(&s{}).Elem().Field(0)
		err := setFieldFromParam(fv, &Param{Name: "v", Value: "abc"})
		if err == nil {
			t.Fatal("expected parse error")
		}
	})
}

// --- callCustomUnmarshal signature validation ---

type badUnmarshalWrongParamCount struct {
	Data unmarshalNested `ini:"data"`
}

func (s *badUnmarshalWrongParamCount) UnmarshalData() (*unmarshalNested, error) {
	return nil, nil
}

type badUnmarshalWrongReturnCount struct {
	Data unmarshalNested `ini:"data"`
}

func (s *badUnmarshalWrongReturnCount) UnmarshalData(value string) *unmarshalNested {
	return nil
}

type badUnmarshalWrongParamType struct {
	Data unmarshalNested `ini:"data"`
}

func (s *badUnmarshalWrongParamType) UnmarshalData(value int) (*unmarshalNested, error) {
	return nil, nil
}

type badUnmarshalWrongReturnType struct {
	Data unmarshalNested `ini:"data"`
}

func (s *badUnmarshalWrongReturnType) UnmarshalData(value string) (*int, error) {
	return nil, nil
}

type badUnmarshalWrongErrorReturn struct {
	Data unmarshalNested `ini:"data"`
}

func (s *badUnmarshalWrongErrorReturn) UnmarshalData(value string) (*unmarshalNested, int) {
	return nil, 0
}

type unmarshalReturnsNil struct {
	Data unmarshalNested `ini:"data"`
}

func (s *unmarshalReturnsNil) UnmarshalData(value string) (*unmarshalNested, error) {
	return nil, nil
}

func TestCallCustomUnmarshalValidation(t *testing.T) {
	setup := func(t *testing.T) *IniFile {
		t.Helper()
		f, err := NewIniFile(nonExistingPath("test.conf"))
		if err != nil {
			t.Fatalf("NewIniFile: %v", err)
		}
		f.GetSection("").SetParam("data", "1,2")
		return f
	}

	t.Run("wrong param count", func(t *testing.T) {
		f := setup(t)
		err := f.UnmarshalSection("", &badUnmarshalWrongParamCount{})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("wrong return count", func(t *testing.T) {
		f := setup(t)
		err := f.UnmarshalSection("", &badUnmarshalWrongReturnCount{})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("wrong param type", func(t *testing.T) {
		f := setup(t)
		err := f.UnmarshalSection("", &badUnmarshalWrongParamType{})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("wrong return type", func(t *testing.T) {
		f := setup(t)
		err := f.UnmarshalSection("", &badUnmarshalWrongReturnType{})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("wrong error return type", func(t *testing.T) {
		f := setup(t)
		err := f.UnmarshalSection("", &badUnmarshalWrongErrorReturn{})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("returns nil pointer", func(t *testing.T) {
		f := setup(t)
		err := f.UnmarshalSection("", &unmarshalReturnsNil{})
		if err == nil {
			t.Fatal("expected error for nil pointer return")
		}
	})
}

// ---------------------------------------------------------------------------
// Unmarshal from conf files 01–13
// ---------------------------------------------------------------------------

// TestUnmarshal_01_Blank verifies that unmarshaling a blank file leaves all
// fields at zero values.
func TestUnmarshal_01_Blank(t *testing.T) {
	type cfg struct {
		Host string `ini:"host"`
		Port int    `ini:"port"`
	}
	c, err := Load[cfg](unitPath("01_blank.conf"), "")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if c.Host != "" {
		t.Errorf("Host = %q, want empty", c.Host)
	}
	if c.Port != 0 {
		t.Errorf("Port = %d, want 0", c.Port)
	}
}

// TestUnmarshal_02_Comments verifies that a comments-only file produces zero
// values (no parameters).
func TestUnmarshal_02_Comments(t *testing.T) {
	type cfg struct {
		Key string `ini:"key"`
	}
	c, err := Load[cfg](unitPath("02_comments.conf"), "")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if c.Key != "" {
		t.Errorf("Key = %q, want empty", c.Key)
	}
}

// TestUnmarshal_03_Sections verifies unmarshaling specific named sections.
func TestUnmarshal_03_Sections(t *testing.T) {
	type section struct {
		Key string `ini:"key"`
	}

	tests := []struct {
		section string
		want    string
	}{
		{"basic", "one"},
		{"upper", "two"},
		{"mixed", "three"},
		{"_private", "four"},
		{"section_2_name", "five"},
		{"x", "six"},
		{"trailing", "seven"},
		{"commented", "eight"},
		{"semicommented", "nine"},
		{"padded", "ten"},
	}
	for _, tt := range tests {
		t.Run(tt.section, func(t *testing.T) {
			c, err := Load[section](unitPath("03_sections.conf"), tt.section)
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			if c.Key != tt.want {
				t.Errorf("Key = %q, want %q", c.Key, tt.want)
			}
		})
	}
}

// TestUnmarshal_04_Identifiers verifies that case-insensitive keys unmarshal
// correctly into struct fields.
func TestUnmarshal_04_Identifiers(t *testing.T) {
	type cfg struct {
		A               string `ini:"a"`
		Z               string `ini:"z"`
		Underscore      string `ini:"_"`
		ABC             string `ini:"abc"`
		Mixed           string `ini:"mixed"`
		Leading         string `ini:"_leading"`
		Double          string `ini:"__double"`
		A1              string `ini:"a1"`
		U0              string `ini:"_0"`
		Long            string `ini:"long_identifier_name_with_many_parts"`
		LettersDigits   string `ini:"abc123def456"`
		MixedEverything string `ini:"a_b_c_1_2_3"`
	}
	c, err := Load[cfg](unitPath("04_identifiers.conf"), "")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	tests := []struct{ name, got, want string }{
		{"a", c.A, "single_letter"},
		{"z", c.Z, "uppercase_single"},
		{"_", c.Underscore, "underscore_start"},
		{"abc", c.ABC, "uppercase"},
		{"mixed", c.Mixed, "mixed_case"},
		{"_leading", c.Leading, "underscore_leading"},
		{"__double", c.Double, "double_underscore"},
		{"a1", c.A1, "letter_then_digit"},
		{"_0", c.U0, "underscore_then_digit"},
		{"long", c.Long, "long"},
		{"abc123def456", c.LettersDigits, "letters_and_digits"},
		{"a_b_c_1_2_3", c.MixedEverything, "mixed_everything"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("%s = %q, want %q", tt.name, tt.got, tt.want)
			}
		})
	}
}

// TestUnmarshal_05_Separators verifies that = and : separators (and space-only)
// all unmarshal identically.
func TestUnmarshal_05_Separators(t *testing.T) {
	type cfg struct {
		Equals      string `ini:"equals"`
		Colon       string `ini:"colon"`
		SpaceOnly   string `ini:"space_only"`
		EqualsNoSpc string `ini:"equals_no_space"`
		ColonNoSpc  string `ini:"colon_no_space"`
		ExtraSpace  string `ini:"extra_space"`
		ExtraColon  string `ini:"extra_colon"`
		TabAround   string `ini:"tab_around"`
	}
	c, err := Load[cfg](unitPath("05_separators.conf"), "")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	tests := []struct{ name, got, want string }{
		{"equals", c.Equals, "value_equals"},
		{"colon", c.Colon, "value_colon"},
		{"space_only", c.SpaceOnly, "value_space"},
		{"equals_no_space", c.EqualsNoSpc, "value_tight"},
		{"colon_no_space", c.ColonNoSpc, "value_tight_colon"},
		{"extra_space", c.ExtraSpace, "value_extra"},
		{"extra_colon", c.ExtraColon, "value_extra_colon"},
		{"tab_around", c.TabAround, "value_tab"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("%s = %q, want %q", tt.name, tt.got, tt.want)
			}
		})
	}
}

// TestUnmarshal_06_UnquotedValues verifies unquoted safe-char values unmarshal
// as strings.
func TestUnmarshal_06_UnquotedValues(t *testing.T) {
	type cfg struct {
		Alpha    string `ini:"alpha"`
		URLLike  string `ini:"url_like"`
		IPAddr   string `ini:"ip_addr"`
		Version  string `ini:"version"`
		Negative string `ini:"negative"`
		Signed   string `ini:"signed"`
	}
	c, err := Load[cfg](unitPath("06_unquoted_values.conf"), "")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	tests := []struct{ name, got, want string }{
		{"alpha", c.Alpha, "abcdef"},
		{"url_like", c.URLLike, "https://example.com:443/path"},
		{"ip_addr", c.IPAddr, "192.168.1.1"},
		{"version", c.Version, "v1.2.3-rc.1+build.42"},
		{"negative", c.Negative, "-123"},
		{"signed", c.Signed, "+456"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("%s = %q, want %q", tt.name, tt.got, tt.want)
			}
		})
	}
}

// TestUnmarshal_07_QuotedValues verifies quoted values with escapes unmarshal
// correctly as strings.
func TestUnmarshal_07_QuotedValues(t *testing.T) {
	type cfg struct {
		Empty            string `ini:"empty"`
		Simple           string `ini:"simple"`
		EscapedBackslash string `ini:"escaped_backslash"`
		EscapedN         string `ini:"escaped_n"`
		DoubledQuote     string `ini:"doubled_quote"`
		UTF8Content      string `ini:"utf8_content"`
		UTF8Emoji        string `ini:"utf8_emoji"`
		CommentChars     string `ini:"comment_chars"`
	}
	c, err := Load[cfg](unitPath("07_quoted_values.conf"), "")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	tests := []struct{ name, got, want string }{
		{"empty", c.Empty, ""},
		{"simple", c.Simple, "hello world"},
		{"escaped_backslash", c.EscapedBackslash, "back\\slash"},
		{"escaped_n", c.EscapedN, "new\nline"},
		{"doubled_quote", c.DoubledQuote, "it's doubled"},
		{"utf8_content", c.UTF8Content, "café résumé naïve"},
		{"utf8_emoji", c.UTF8Emoji, "🌍🎉"},
		{"comment_chars", c.CommentChars, "# not a comment ; also not"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("%s = %q, want %q", tt.name, tt.got, tt.want)
			}
		})
	}
}

// TestUnmarshal_08_Booleans verifies that boolean string values unmarshal into
// bool fields correctly.
func TestUnmarshal_08_Booleans(t *testing.T) {
	type cfg struct {
		TrueLower  bool `ini:"true_lower"`
		TrueUpper  bool `ini:"true_upper"`
		TrueMixed  bool `ini:"true_mixed"`
		FalseLower bool `ini:"false_lower"`
		FalseUpper bool `ini:"false_upper"`
		FalseMixed bool `ini:"false_mixed"`
		OnLower    bool `ini:"on_lower"`
		OnUpper    bool `ini:"on_upper"`
		OffLower   bool `ini:"off_lower"`
		OffUpper   bool `ini:"off_upper"`
		YesLower   bool `ini:"yes_lower"`
		YesUpper   bool `ini:"yes_upper"`
		NoLower    bool `ini:"no_lower"`
		NoUpper    bool `ini:"no_upper"`
		One        bool `ini:"one"`
		Zero       bool `ini:"zero"`
		TLower     bool `ini:"t_lower"`
		TUpper     bool `ini:"t_upper"`
		FLower     bool `ini:"f_lower"`
		FUpper     bool `ini:"f_upper"`
		YLower     bool `ini:"y_lower"`
		YUpper     bool `ini:"y_upper"`
		NLower     bool `ini:"n_lower"`
		NUpper     bool `ini:"n_upper"`
	}
	c, err := Load[cfg](unitPath("08_booleans.conf"), "")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	trueFields := []struct {
		name string
		got  bool
	}{
		{"true_lower", c.TrueLower},
		{"true_upper", c.TrueUpper},
		{"true_mixed", c.TrueMixed},
		{"on_lower", c.OnLower},
		{"on_upper", c.OnUpper},
		{"yes_lower", c.YesLower},
		{"yes_upper", c.YesUpper},
		{"one", c.One},
		{"t_lower", c.TLower},
		{"t_upper", c.TUpper},
		{"y_lower", c.YLower},
		{"y_upper", c.YUpper},
	}
	for _, tt := range trueFields {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.got {
				t.Errorf("%s = false, want true", tt.name)
			}
		})
	}

	falseFields := []struct {
		name string
		got  bool
	}{
		{"false_lower", c.FalseLower},
		{"false_upper", c.FalseUpper},
		{"false_mixed", c.FalseMixed},
		{"off_lower", c.OffLower},
		{"off_upper", c.OffUpper},
		{"no_lower", c.NoLower},
		{"no_upper", c.NoUpper},
		{"zero", c.Zero},
		{"f_lower", c.FLower},
		{"f_upper", c.FUpper},
		{"n_lower", c.NLower},
		{"n_upper", c.NUpper},
	}
	for _, tt := range falseFields {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got {
				t.Errorf("%s = true, want false", tt.name)
			}
		})
	}
}

// TestUnmarshal_09_Numbers verifies numeric values unmarshal into typed fields.
func TestUnmarshal_09_Numbers(t *testing.T) {
	type cfg struct {
		Decimal    int     `ini:"decimal"`
		Zero       int     `ini:"zero"`
		Negative   int     `ini:"negative"`
		Positive   int     `ini:"positive"`
		Large      int     `ini:"large"`
		HexLower   int     `ini:"hex_lower"`
		HexUpper   int     `ini:"hex_upper"`
		HexLong    int64   `ini:"hex_long"`
		FloatSimp  float64 `ini:"float_simple"`
		FloatSmall float64 `ini:"float_small"`
		FloatNoLd  float64 `ini:"float_no_lead"`
		FloatTrail float64 `ini:"float_trail"`
	}
	c, err := Load[cfg](unitPath("09_numbers.conf"), "")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	intTests := []struct {
		name string
		got  any
		want any
	}{
		{"decimal", c.Decimal, 100},
		{"zero", c.Zero, 0},
		{"negative", c.Negative, -1},
		{"positive", c.Positive, 1},
		{"large", c.Large, 9999999},
		{"hex_lower", c.HexLower, 0xff},
		{"hex_upper", c.HexUpper, 0xFF},
		{"hex_long", c.HexLong, int64(0xDEADBEEF)},
		{"float_simple", c.FloatSimp, 1.5},
		{"float_small", c.FloatSmall, 0.001},
		{"float_no_lead", c.FloatNoLd, 0.5},
		{"float_trail", c.FloatTrail, 1.0},
	}
	for _, tt := range intTests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("%s = %v, want %v", tt.name, tt.got, tt.want)
			}
		})
	}
}

// TestUnmarshal_10_EmptyValues verifies that missing/empty values unmarshal to
// zero values for each type.
func TestUnmarshal_10_EmptyValues(t *testing.T) {
	type cfg struct {
		NoValueNoSep   string `ini:"no_value_no_sep"`
		NoValueEq      string `ini:"no_value_eq"`
		NoValueColon   string `ini:"no_value_colon"`
		NoValueEqSpace string `ini:"no_value_eq_space"`
		NoValueComment string `ini:"no_value_comment"`
		NoValueSemi    string `ini:"no_value_semi"`
	}
	c, err := Load[cfg](unitPath("10_empty_values.conf"), "")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	fields := []struct {
		name string
		got  string
	}{
		{"no_value_no_sep", c.NoValueNoSep},
		{"no_value_eq", c.NoValueEq},
		{"no_value_colon", c.NoValueColon},
		{"no_value_eq_space", c.NoValueEqSpace},
		{"no_value_comment", c.NoValueComment},
		{"no_value_semi", c.NoValueSemi},
	}
	for _, tt := range fields {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != "" {
				t.Errorf("%s = %q, want empty", tt.name, tt.got)
			}
		})
	}
}

// TestUnmarshal_11_Duplicates verifies that duplicate keys resolve to last-wins
// when unmarshaled.
func TestUnmarshal_11_Duplicates(t *testing.T) {
	type defaultSection struct {
		Key     string `ini:"key"`
		Another string `ini:"another"`
	}
	type sectionA struct {
		Dup   string `ini:"dup"`
		Extra string `ini:"extra"`
	}

	f, err := Parse(unitPath("11_duplicates.conf"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	var def defaultSection
	if err := f.UnmarshalSection("", &def); err != nil {
		t.Fatalf("UnmarshalSection default: %v", err)
	}
	if def.Key != "third" {
		t.Errorf("Key = %q, want %q", def.Key, "third")
	}
	if def.Another != "two" {
		t.Errorf("Another = %q, want %q", def.Another, "two")
	}

	var sa sectionA
	if err := f.UnmarshalSection("section_a", &sa); err != nil {
		t.Fatalf("UnmarshalSection section_a: %v", err)
	}
	if sa.Dup != "gamma" {
		t.Errorf("Dup = %q, want %q", sa.Dup, "gamma")
	}
	if sa.Extra != "added" {
		t.Errorf("Extra = %q, want %q", sa.Extra, "added")
	}
}

// TestUnmarshal_12_Whitespace verifies that whitespace around keys, separators,
// and values is properly trimmed when unmarshaled.
func TestUnmarshal_12_Whitespace(t *testing.T) {
	type cfg struct {
		SpacedKey         string `ini:"spaced_key"`
		TabbedKey         string `ini:"tabbed_key"`
		MixedKey          string `ini:"mixed_key"`
		PaddedEq          string `ini:"padded_eq"`
		PaddedColon       string `ini:"padded_colon"`
		TabsEq            string `ini:"tabs_eq"`
		TrailingSpaces    string `ini:"trailing_spaces"`
		TrailingTabs      string `ini:"trailing_tabs"`
		TrailingAfterQuot string `ini:"trailing_after_quote"`
	}
	c, err := Load[cfg](unitPath("12_whitespace.conf"), "")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	tests := []struct{ name, got, want string }{
		{"spaced_key", c.SpacedKey, "value"},
		{"tabbed_key", c.TabbedKey, "value"},
		{"mixed_key", c.MixedKey, "value"},
		{"padded_eq", c.PaddedEq, "padded_value"},
		{"padded_colon", c.PaddedColon, "padded_value_colon"},
		{"tabs_eq", c.TabsEq, "tabbed_value"},
		{"trailing_spaces", c.TrailingSpaces, "value"},
		{"trailing_tabs", c.TrailingTabs, "value"},
		{"trailing_after_quote", c.TrailingAfterQuot, "quoted"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("%s = %q, want %q", tt.name, tt.got, tt.want)
			}
		})
	}
}

// TestUnmarshal_13_TrailingComments verifies that trailing comments are stripped
// and do not appear in unmarshaled values.
func TestUnmarshal_13_TrailingComments(t *testing.T) {
	type defaultCfg struct {
		Key1 string `ini:"key1"`
		Key2 string `ini:"key2"`
		Key3 string `ini:"key3"`
		Key4 string `ini:"key4"`
		Key5 string `ini:"key5"`
		Key6 string `ini:"key6"`
	}
	type commentedCfg struct {
		Key7 string `ini:"key7"`
		Key8 string `ini:"key8"`
		Key9 string `ini:"key9"`
	}

	f, err := Parse(unitPath("13_trailing_comments.conf"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	var def defaultCfg
	if err := f.UnmarshalSection("", &def); err != nil {
		t.Fatalf("UnmarshalSection default: %v", err)
	}

	defTests := []struct{ name, got, want string }{
		{"key1", def.Key1, "value"},
		{"key2", def.Key2, "value"},
		{"key3", def.Key3, "quoted"},
		{"key4", def.Key4, "quoted"},
		{"key5", def.Key5, ""},
		{"key6", def.Key6, ""},
	}
	for _, tt := range defTests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("%s = %q, want %q", tt.name, tt.got, tt.want)
			}
		})
	}

	var commented commentedCfg
	if err := f.UnmarshalSection("commented", &commented); err != nil {
		t.Fatalf("UnmarshalSection commented: %v", err)
	}

	commentedTests := []struct{ name, got, want string }{
		{"key7", commented.Key7, "value"},
		{"key8", commented.Key8, "has # inside"},
		{"key9", commented.Key9, "has ; inside"},
	}
	for _, tt := range commentedTests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("%s = %q, want %q", tt.name, tt.got, tt.want)
			}
		})
	}
}

// --- round-trip test ---

func TestMarshalUnmarshalRoundTrip(t *testing.T) {
	f, err := NewIniFile(nonExistingPath("test.conf"))
	if err != nil {
		t.Fatalf("NewIniFile: %v", err)
	}

	original := &unmarshalPrimitives{
		Host:    "localhost",
		Port:    5432,
		Debug:   true,
		Rate:    3.14,
		Small:   -1,
		Medium:  1000,
		Large:   100000,
		Huge:    9999999999,
		USmall:  200,
		UMedium: 50000,
		ULarge:  3000000000,
		UHuge:   12345678901234567890,
		Half:    1.5,
	}

	if err := f.MarshalSection("", original); err != nil {
		t.Fatalf("MarshalSection: %v", err)
	}

	restored := &unmarshalPrimitives{}
	if err := f.UnmarshalSection("", restored); err != nil {
		t.Fatalf("UnmarshalSection: %v", err)
	}

	if restored.Host != original.Host {
		t.Errorf("Host = %q, want %q", restored.Host, original.Host)
	}
	if restored.Port != original.Port {
		t.Errorf("Port = %d, want %d", restored.Port, original.Port)
	}
	if restored.Debug != original.Debug {
		t.Errorf("Debug = %v, want %v", restored.Debug, original.Debug)
	}
	if restored.Rate != original.Rate {
		t.Errorf("Rate = %f, want %f", restored.Rate, original.Rate)
	}
	if restored.Small != original.Small {
		t.Errorf("Small = %d, want %d", restored.Small, original.Small)
	}
	if restored.USmall != original.USmall {
		t.Errorf("USmall = %d, want %d", restored.USmall, original.USmall)
	}
	if restored.Half != original.Half {
		t.Errorf("Half = %f, want %f", restored.Half, original.Half)
	}
}
