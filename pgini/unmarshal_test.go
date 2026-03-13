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
