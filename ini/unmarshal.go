package ini

import (
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
)

// UnmarshalIniFile reads an ini file from disk and unmarshals the specified section into a new T.
func UnmarshalIniFile[T any](path string, section string) (*T, error) {
	iniFile, err := unmarshalIniFileIntermediate(path)
	if err != nil {
		return nil, err
	}
	target := new(T)
	if err := unmarshalIniFileStruct(iniFile, section, target); err != nil {
		return nil, err
	}
	return target, nil
}

// UnmarshalIniString parses ini content from a string and unmarshals the specified section into a new T.
// The path is used for include directive resolution and error messages.
func UnmarshalIniString[T any](path string, section string, contents string) (*T, error) {
	iniFile, err := unmarshalIniStringIntermediate(path, contents)
	if err != nil {
		return nil, err
	}
	target := new(T)
	if err := unmarshalIniFileStruct(iniFile, section, target); err != nil {
		return nil, err
	}
	return target, nil
}

// unmarshalIniFileStruct fills a struct from parsed IniFile data (Pass 2).
func unmarshalIniFileStruct[T any](iniFile *IniFile, section string, target *T) error {
	sec := iniFile.getSection(section)
	if sec == nil {
		return fmt.Errorf("section %q not found", section)
	}

	rv := reflect.ValueOf(target).Elem()
	rt := rv.Type()

	if rt.Kind() != reflect.Struct {
		return fmt.Errorf("target must be a struct, got %s", rt.Kind())
	}

	for i := range rt.NumField() {
		field := rt.Field(i)
		fv := rv.Field(i)

		if !fv.CanSet() {
			continue
		}

		tag := field.Tag.Get("ini")
		if tag == "" {
			continue
		}

		paramName := tag
		paramKey := strings.ToLower(paramName)

		param, ok := sec.params[paramKey]
		if !ok {
			continue
		}

		// Check for custom unmarshal method: Unmarshal_<FieldName>
		methodName := "Unmarshal_" + field.Name
		method := reflect.ValueOf(target).MethodByName(methodName)
		if method.IsValid() {
			results := method.Call([]reflect.Value{reflect.ValueOf(param.value)})
			if len(results) == 2 {
				if !results[1].IsNil() {
					err := results[1].Interface().(error)
					return fmt.Errorf("%s:%d:%d: custom unmarshal %s: %w",
						iniFile.path, param.cursor.Line, param.cursor.Offset, methodName, err)
				}
				fv.Set(results[0])
				continue
			}
		}

		if err := setFieldFromParam(fv, param); err != nil {
			return fmt.Errorf("%s:%d:%d: field %s (param %q): %w",
				iniFile.path, param.cursor.Line, param.cursor.Offset,
				field.Name, paramName, err)
		}
	}

	return nil
}

// setFieldFromParam assigns a Param value to a struct field based on the field's type.
func setFieldFromParam(fv reflect.Value, p *Param) error {
	switch fv.Kind() {
	case reflect.String:
		fv.SetString(p.value)
	case reflect.Bool:
		v, err := parseBool(p.value)
		if err != nil {
			return err
		}
		fv.SetBool(v)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		v, err := parseInt(p.value)
		if err != nil {
			return err
		}
		fv.SetInt(v)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		v, err := parseInt(p.value)
		if err != nil {
			return err
		}
		if v < 0 {
			return fmt.Errorf("negative value %d for unsigned field", v)
		}
		fv.SetUint(uint64(v))
	case reflect.Float32, reflect.Float64:
		v, err := parseFloat(p.value)
		if err != nil {
			return err
		}
		fv.SetFloat(v)
	default:
		return fmt.Errorf("unsupported field type: %s", fv.Kind())
	}
	return nil
}

// parseBool interprets a string as a boolean value.
// Accepts: on/off, true/false, yes/no, 1/0 (case-insensitive),
// or any unambiguous prefix of these words.
func parseBool(s string) (bool, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return false, fmt.Errorf("empty boolean value")
	}
	if s == "1" {
		return true, nil
	}
	if s == "0" {
		return false, nil
	}

	trueWords := []string{"on", "true", "yes"}
	falseWords := []string{"off", "false", "no"}

	matchesTrue := false
	matchesFalse := false
	for _, w := range trueWords {
		if strings.HasPrefix(w, s) {
			matchesTrue = true
		}
	}
	for _, w := range falseWords {
		if strings.HasPrefix(w, s) {
			matchesFalse = true
		}
	}

	if matchesTrue && !matchesFalse {
		return true, nil
	}
	if matchesFalse && !matchesTrue {
		return false, nil
	}
	if matchesTrue && matchesFalse {
		return false, fmt.Errorf("ambiguous boolean value: %q", s)
	}
	return false, fmt.Errorf("invalid boolean value: %q", s)
}

// parseInt interprets a string as an integer.
// Supports decimal, hexadecimal (0x prefix), and octal (0 prefix).
func parseInt(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty integer value")
	}

	if n, err := strconv.ParseInt(s, 0, 64); err == nil {
		return n, nil
	}

	// Fall back to float parsing and round
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return int64(math.Round(f)), nil
	}

	return 0, fmt.Errorf("invalid integer value: %q", s)
}

// parseFloat interprets a string as a float64.
func parseFloat(s string) (float64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty float value")
	}
	return strconv.ParseFloat(s, 64)
}
