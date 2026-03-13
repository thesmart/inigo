package pgini

import (
	"fmt"
	"reflect"
	"testing"
)

// --- test helper types for marshal ---

type marshalPrimitives struct {
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

type marshalCustomType struct {
	Name string `ini:"name"`
	Data nested `ini:"data"`
}

type nested struct {
	X int
	Y int
}

func (s *marshalCustomType) MarshalData(value *nested) (string, error) {
	return fmt.Sprintf("%d,%d", value.X, value.Y), nil
}

type marshalCustomError struct {
	Data nested `ini:"data"`
}

func (s *marshalCustomError) MarshalData(value *nested) (string, error) {
	return "", fmt.Errorf("marshal error")
}

type marshalNoCustom struct {
	Data nested `ini:"data"`
}

type marshalCustomOverride struct {
	Port int `ini:"port"`
}

func (s *marshalCustomOverride) MarshalPort(value *int) (string, error) {
	return fmt.Sprintf("port-%d", *value), nil
}

// --- MarshalSection tests ---

func TestMarshalSection(t *testing.T) {
	t.Run("primitive fields", func(t *testing.T) {
		f, err := NewIniFile(nonExistingPath("test.conf"))
		if err != nil {
			t.Fatalf("NewIniFile: %v", err)
		}

		v := &marshalPrimitives{
			Host:    "localhost",
			Port:    5432,
			Debug:   true,
			Rate:    3.14,
			Small:   -1,
			Medium:  1000,
			Large:   100000,
			Huge:    9999999999,
			USmall:  255,
			UMedium: 65535,
			ULarge:  4294967295,
			UHuge:   18446744073709551615,
			Half:    1.5,
			NoTag:   "skipped",
			Empty:   "skipped",
		}

		if err := f.MarshalSection("", v); err != nil {
			t.Fatalf("MarshalSection: %v", err)
		}

		s := f.GetSection("")
		if s == nil {
			t.Fatal("default section not found")
		}

		tests := []struct {
			key  string
			want string
		}{
			{"host", "localhost"},
			{"port", "5432"},
			{"debug", "true"},
			{"rate", "3.14"},
			{"small", "-1"},
			{"medium", "1000"},
			{"large", "100000"},
			{"huge", "9999999999"},
			{"usmall", "255"},
			{"umedium", "65535"},
			{"ularge", "4294967295"},
			{"uhuge", "18446744073709551615"},
			{"half", "1.5"},
		}
		for _, tt := range tests {
			t.Run(tt.key, func(t *testing.T) {
				val, ok := s.GetValue(tt.key)
				if !ok {
					t.Fatalf("param %q not found", tt.key)
				}
				if val != tt.want {
					t.Errorf("param %q = %q, want %q", tt.key, val, tt.want)
				}
			})
		}

		// Fields without ini tag or empty tag should be skipped
		if _, ok := s.GetValue("NoTag"); ok {
			t.Error("NoTag field should be skipped (no ini tag)")
		}
		if _, ok := s.GetValue(""); ok {
			t.Error("Empty tag field should be skipped")
		}
	})

	t.Run("bool false", func(t *testing.T) {
		f, err := NewIniFile(nonExistingPath("test.conf"))
		if err != nil {
			t.Fatalf("NewIniFile: %v", err)
		}

		v := &marshalPrimitives{Debug: false}
		if err := f.MarshalSection("", v); err != nil {
			t.Fatalf("MarshalSection: %v", err)
		}

		val, ok := f.GetSection("").GetValue("debug")
		if !ok {
			t.Fatal("debug param not found")
		}
		if val != "false" {
			t.Errorf("debug = %q, want %q", val, "false")
		}
	})

	t.Run("string with special chars is escaped", func(t *testing.T) {
		f, err := NewIniFile(nonExistingPath("test.conf"))
		if err != nil {
			t.Fatalf("NewIniFile: %v", err)
		}

		v := &marshalPrimitives{Host: "it's a test\n"}
		if err := f.MarshalSection("", v); err != nil {
			t.Fatalf("MarshalSection: %v", err)
		}

		val, _ := f.GetSection("").GetValue("host")
		if val != `it\'s a test\n` {
			t.Errorf("escaped value = %q, want %q", val, `it\'s a test\n`)
		}
	})

	t.Run("creates named section", func(t *testing.T) {
		f, err := NewIniFile(nonExistingPath("test.conf"))
		if err != nil {
			t.Fatalf("NewIniFile: %v", err)
		}

		v := &marshalPrimitives{Host: "example.com", Port: 80}
		if err := f.MarshalSection("production", v); err != nil {
			t.Fatalf("MarshalSection: %v", err)
		}

		s := f.GetSection("production")
		if s == nil {
			t.Fatal("production section not found")
		}
		val, _ := s.GetValue("host")
		if val != "example.com" {
			t.Errorf("host = %q, want %q", val, "example.com")
		}
	})

	t.Run("custom marshal method", func(t *testing.T) {
		f, err := NewIniFile(nonExistingPath("test.conf"))
		if err != nil {
			t.Fatalf("NewIniFile: %v", err)
		}

		v := &marshalCustomType{Name: "test", Data: nested{X: 10, Y: 20}}
		if err := f.MarshalSection("", v); err != nil {
			t.Fatalf("MarshalSection: %v", err)
		}

		val, _ := f.GetSection("").GetValue("data")
		if val != "10,20" {
			t.Errorf("data = %q, want %q", val, "10,20")
		}
	})

	t.Run("custom marshal overrides default for primitive", func(t *testing.T) {
		f, err := NewIniFile(nonExistingPath("test.conf"))
		if err != nil {
			t.Fatalf("NewIniFile: %v", err)
		}

		v := &marshalCustomOverride{Port: 8080}
		if err := f.MarshalSection("", v); err != nil {
			t.Fatalf("MarshalSection: %v", err)
		}

		val, _ := f.GetSection("").GetValue("port")
		if val != "port-8080" {
			t.Errorf("port = %q, want %q", val, "port-8080")
		}
	})

	t.Run("custom marshal returns error", func(t *testing.T) {
		f, err := NewIniFile(nonExistingPath("test.conf"))
		if err != nil {
			t.Fatalf("NewIniFile: %v", err)
		}

		v := &marshalCustomError{Data: nested{X: 1, Y: 2}}
		err = f.MarshalSection("", v)
		if err == nil {
			t.Fatal("expected error from custom marshal")
		}
	})

	t.Run("unsupported type without custom method errors", func(t *testing.T) {
		f, err := NewIniFile(nonExistingPath("test.conf"))
		if err != nil {
			t.Fatalf("NewIniFile: %v", err)
		}

		v := &marshalNoCustom{Data: nested{X: 1, Y: 2}}
		err = f.MarshalSection("", v)
		if err == nil {
			t.Fatal("expected error for unsupported type without custom method")
		}
	})

	t.Run("non-pointer arg errors", func(t *testing.T) {
		f, err := NewIniFile(nonExistingPath("test.conf"))
		if err != nil {
			t.Fatalf("NewIniFile: %v", err)
		}

		err = f.MarshalSection("", marshalPrimitives{})
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
		err = f.MarshalSection("", &s)
		if err == nil {
			t.Fatal("expected error for non-struct pointer")
		}
	})

	t.Run("invalid section name errors", func(t *testing.T) {
		f, err := NewIniFile(nonExistingPath("test.conf"))
		if err != nil {
			t.Fatalf("NewIniFile: %v", err)
		}

		v := &marshalPrimitives{Host: "test"}
		err = f.MarshalSection("123invalid", v)
		if err == nil {
			t.Fatal("expected error for invalid section name")
		}
	})
}

// --- formatField tests ---

func TestFormatField(t *testing.T) {
	tests := []struct {
		name string
		val  any
		want string
	}{
		{"string", "hello", "hello"},
		{"string with escape", "it's", `it\'s`},
		{"bool true", true, "true"},
		{"bool false", false, "false"},
		{"int", int(42), "42"},
		{"int8", int8(-1), "-1"},
		{"int16", int16(1000), "1000"},
		{"int32", int32(100000), "100000"},
		{"int64", int64(9999999999), "9999999999"},
		{"uint", uint(42), "42"},
		{"uint8", uint8(255), "255"},
		{"uint16", uint16(65535), "65535"},
		{"uint32", uint32(4294967295), "4294967295"},
		{"uint64", uint64(18446744073709551615), "18446744073709551615"},
		{"float32", float32(1.5), "1.5"},
		{"float64", float64(3.14), "3.14"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fv := reflect.ValueOf(tt.val)
			got, err := formatField(fv)
			if err != nil {
				t.Fatalf("formatField: %v", err)
			}
			if got != tt.want {
				t.Errorf("formatField = %q, want %q", got, tt.want)
			}
		})
	}

	t.Run("unsupported type", func(t *testing.T) {
		fv := reflect.ValueOf([]int{1, 2, 3})
		_, err := formatField(fv)
		if err == nil {
			t.Fatal("expected error for unsupported type")
		}
	})
}

// --- callCustomMarshal signature validation ---

type badMarshalWrongParamCount struct {
	Data nested `ini:"data"`
}

// MarshalData has wrong parameter count (0 instead of 1)
func (s *badMarshalWrongParamCount) MarshalData() (string, error) {
	return "", nil
}

type badMarshalWrongReturnCount struct {
	Data nested `ini:"data"`
}

// MarshalData has wrong return count (1 instead of 2)
func (s *badMarshalWrongReturnCount) MarshalData(value *nested) string {
	return ""
}

type badMarshalWrongReturnType struct {
	Data nested `ini:"data"`
}

// MarshalData has wrong first return type (int instead of string)
func (s *badMarshalWrongReturnType) MarshalData(value *nested) (int, error) {
	return 0, nil
}

type badMarshalWrongParamType struct {
	Data nested `ini:"data"`
}

// MarshalData has wrong param type (*int instead of *nested)
func (s *badMarshalWrongParamType) MarshalData(value *int) (string, error) {
	return "", nil
}

func TestCallCustomMarshalValidation(t *testing.T) {
	t.Run("wrong param count", func(t *testing.T) {
		f, err := NewIniFile(nonExistingPath("test.conf"))
		if err != nil {
			t.Fatalf("NewIniFile: %v", err)
		}

		v := &badMarshalWrongParamCount{Data: nested{X: 1, Y: 2}}
		err = f.MarshalSection("", v)
		if err == nil {
			t.Fatal("expected error for wrong param count")
		}
	})

	t.Run("wrong return count", func(t *testing.T) {
		f, err := NewIniFile(nonExistingPath("test.conf"))
		if err != nil {
			t.Fatalf("NewIniFile: %v", err)
		}

		v := &badMarshalWrongReturnCount{Data: nested{X: 1, Y: 2}}
		err = f.MarshalSection("", v)
		if err == nil {
			t.Fatal("expected error for wrong return count")
		}
	})

	t.Run("wrong return type", func(t *testing.T) {
		f, err := NewIniFile(nonExistingPath("test.conf"))
		if err != nil {
			t.Fatalf("NewIniFile: %v", err)
		}

		v := &badMarshalWrongReturnType{Data: nested{X: 1, Y: 2}}
		err = f.MarshalSection("", v)
		if err == nil {
			t.Fatal("expected error for wrong return type")
		}
	})

	t.Run("wrong param type", func(t *testing.T) {
		f, err := NewIniFile(nonExistingPath("test.conf"))
		if err != nil {
			t.Fatalf("NewIniFile: %v", err)
		}

		v := &badMarshalWrongParamType{Data: nested{X: 1, Y: 2}}
		err = f.MarshalSection("", v)
		if err == nil {
			t.Fatal("expected error for wrong param type")
		}
	})
}

// badMarshalWrongErrorReturn has a second return value that doesn't implement error
type badMarshalWrongErrorReturn struct {
	Data nested `ini:"data"`
}

func (s *badMarshalWrongErrorReturn) MarshalData(value *nested) (string, int) {
	return "", 0
}

func TestCallCustomMarshalWrongErrorReturn(t *testing.T) {
	f, err := NewIniFile(nonExistingPath("test.conf"))
	if err != nil {
		t.Fatalf("NewIniFile: %v", err)
	}

	v := &badMarshalWrongErrorReturn{Data: nested{X: 1, Y: 2}}
	err = f.MarshalSection("", v)
	if err == nil {
		t.Fatal("expected error for wrong error return type")
	}
}
