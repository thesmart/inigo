package inigo

import (
	"fmt"
	"os"
	"reflect"
	"strings"
)

// LoadInto parses an INI file and populates the target struct using `ini` struct tags.
// The section parameter selects which INI section to read from (empty string for default).
// The target must be a non-nil pointer to a struct.
//
// Struct tags use the format `ini:"param_name"`. Only fields with an explicit ini
// tag are populated; untagged fields are skipped. A tag of "-" also skips the field.
// Params not present in the config leave the field at its zero value.
//
// Supported field types: string, bool, int/int8/int16/int32/int64,
// uint/uint8/uint16/uint32/uint64, float32/float64.
func LoadInto(path, section string, target any) error {
	cfg, err := Load(path)
	if err != nil {
		return err
	}
	return ApplyInto(cfg, section, target)
}

// ApplyInto fills the target struct from an already-parsed Config.
// See LoadInto for struct tag conventions.
func ApplyInto(cfg *Config, section string, target any) error {
	rv := reflect.ValueOf(target)
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return fmt.Errorf("target must be a non-nil pointer to a struct")
	}
	rv = rv.Elem()
	if rv.Kind() != reflect.Struct {
		return fmt.Errorf("target must point to a struct, got %s", rv.Kind())
	}

	sec := cfg.Section(section)
	if sec == nil {
		return fmt.Errorf("section %q not found", section)
	}

	rt := rv.Type()
	for i := range rt.NumField() {
		field := rt.Field(i)
		fv := rv.Field(i)

		// Skip unexported fields
		if !fv.CanSet() {
			continue
		}

		// Resolve param name from struct tag; fields without an ini tag are skipped
		tag := field.Tag.Get("ini")
		if tag == "" || tag == "-" {
			continue
		}
		paramName, _, _ := strings.Cut(tag, ",")

		// Skip params not present in the config
		if !sec.HasParam(paramName) {
			continue
		}

		param := sec.GetParam(paramName)
		if err := setField(fv, param); err != nil {
			return fmt.Errorf("field %s (param %q): %w", field.Name, paramName, err)
		}
	}

	return nil
}

// SaveFrom writes struct fields to an INI file using `ini` struct tags.
// The source must be a non-nil pointer to a struct. The section parameter sets
// the INI section header (empty string writes to the default/global section).
// The file is created or truncated at path.
//
// Struct tag conventions match LoadInto: only fields with an explicit `ini:"param_name"`
// tag are written. Untagged fields and `ini:"-"` fields are skipped.
// Zero-value fields are also skipped to keep the output minimal.
func SaveFrom(source any, section, path string) error {
	content, err := Marshal(source, section)
	if err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

// Marshal serializes a struct into INI-formatted text.
// See SaveFrom for struct tag conventions.
func Marshal(source any, section string) (string, error) {
	rv := reflect.ValueOf(source)
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return "", fmt.Errorf("source must be a non-nil pointer to a struct")
	}
	rv = rv.Elem()
	if rv.Kind() != reflect.Struct {
		return "", fmt.Errorf("source must point to a struct, got %s", rv.Kind())
	}

	var buf strings.Builder

	if section != "" {
		fmt.Fprintf(&buf, "[%s]\n", section)
	}

	rt := rv.Type()
	for i := range rt.NumField() {
		field := rt.Field(i)
		fv := rv.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// Resolve param name from struct tag; fields without an ini tag are skipped
		tag := field.Tag.Get("ini")
		if tag == "" || tag == "-" {
			continue
		}
		paramName, _, _ := strings.Cut(tag, ",")

		// Skip zero-value fields
		if fv.IsZero() {
			continue
		}

		value, err := formatField(fv)
		if err != nil {
			return "", fmt.Errorf("field %s (param %q): %w", field.Name, paramName, err)
		}

		fmt.Fprintf(&buf, "%s = %s\n", paramName, value)
	}

	return buf.String(), nil
}

// formatField converts a struct field value to its INI string representation.
func formatField(fv reflect.Value) (string, error) {
	switch fv.Kind() {
	case reflect.String:
		return quoteValue(fv.String()), nil
	case reflect.Bool:
		if fv.Bool() {
			return "on", nil
		}
		return "off", nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return fmt.Sprintf("%d", fv.Int()), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return fmt.Sprintf("%d", fv.Uint()), nil
	case reflect.Float32, reflect.Float64:
		return fmt.Sprintf("%g", fv.Float()), nil
	default:
		return "", fmt.Errorf("unsupported field type: %s", fv.Kind())
	}
}

// quoteValue wraps a string in single quotes if it contains characters that
// require quoting (spaces, #, =, quotes). Simple identifiers and numbers
// are returned unquoted.
func quoteValue(s string) string {
	if s == "" {
		return "''"
	}
	needsQuote := false
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if ch == ' ' || ch == '\t' || ch == '#' || ch == '=' || ch == '\'' {
			needsQuote = true
			break
		}
	}
	if !needsQuote {
		return s
	}
	// Escape embedded single quotes by doubling them
	escaped := strings.ReplaceAll(s, "'", "''")
	return "'" + escaped + "'"
}

// setField assigns a Param value to a struct field based on the field's type.
func setField(fv reflect.Value, p *Param) error {
	switch fv.Kind() {
	case reflect.String:
		fv.SetString(p.String())
	case reflect.Bool:
		v, err := p.Bool()
		if err != nil {
			return err
		}
		fv.SetBool(v)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		v, err := p.Int()
		if err != nil {
			return err
		}
		fv.SetInt(v)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		v, err := p.Int()
		if err != nil {
			return err
		}
		if v < 0 {
			return fmt.Errorf("negative value %d for unsigned field", v)
		}
		fv.SetUint(uint64(v))
	case reflect.Float32, reflect.Float64:
		v, err := p.Float64()
		if err != nil {
			return err
		}
		fv.SetFloat(v)
	default:
		return fmt.Errorf("unsupported field type: %s", fv.Kind())
	}
	return nil
}
