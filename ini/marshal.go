package ini

import (
	"fmt"
	"os"
	"reflect"
	"strings"
)

// MarshalIniFile marshals a struct into ini format and writes it to a file.
func MarshalIniFile[T any](path string, data *T) error {
	content, err := MarshalIniString("", data)
	if err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

// MarshalIniString marshals a struct into ini-formatted text.
// If section is non-empty, a [section] header is prepended.
func MarshalIniString[T any](section string, data *T) (string, error) {
	rv := reflect.ValueOf(data).Elem()
	rt := rv.Type()

	if rt.Kind() != reflect.Struct {
		return "", fmt.Errorf("data must be a struct, got %s", rt.Kind())
	}

	var buf strings.Builder

	if section != "" {
		fmt.Fprintf(&buf, "[%s]\n", section)
	}

	for i := range rt.NumField() {
		field := rt.Field(i)
		fv := rv.Field(i)

		if !field.IsExported() {
			continue
		}

		tag := field.Tag.Get("ini")
		if tag == "" {
			continue
		}

		paramName := tag

		// Skip zero-value fields
		if fv.IsZero() {
			continue
		}

		// Check for custom marshal method: Marshal_<FieldName>
		methodName := "Marshal_" + field.Name
		method := reflect.ValueOf(data).MethodByName(methodName)
		if method.IsValid() {
			results := method.Call([]reflect.Value{fv.Addr()})
			if len(results) == 2 {
				if !results[1].IsNil() {
					err := results[1].Interface().(error)
					return "", fmt.Errorf("custom marshal %s: %w", methodName, err)
				}
				value := results[0].String()
				fmt.Fprintf(&buf, "%s = %s\n", paramName, value)
				continue
			}
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

// quoteValue wraps a string in single quotes if it contains special characters.
func quoteValue(s string) string {
	if s == "" {
		return "''"
	}
	if strings.ContainsAny(s, " \t#;='\"\\") {
		escaped := strings.ReplaceAll(s, "'", "''")
		return "'" + escaped + "'"
	}
	return s
}
