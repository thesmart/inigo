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
		if !intInBounds(fv.Kind(), v) {
			return fmt.Errorf("value %d overflows %s", v, fv.Kind())
		}
		fv.SetInt(v)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		v, err := parseUint(p.value)
		if err != nil {
			return err
		}
		if !uintInBounds(fv.Kind(), v) {
			return fmt.Errorf("value %d overflows %s", v, fv.Kind())
		}
		fv.SetUint(v)
	case reflect.Float32:
		v, err := parseFloat(p.value, 32)
		if err != nil {
			return err
		}
		fv.SetFloat(v)
	case reflect.Float64:
		v, err := parseFloat(p.value, 64)
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

	trueWords := []string{"true", "on", "yes"}
	falseWords := []string{"false", "off", "no"}

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

	// Fall back to float parsing and floor
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return int64(math.Floor(f)), nil
	}

	return 0, fmt.Errorf("invalid integer value: %q", s)
}

// parseUint interprets a string as an unsigned integer.
// Supports decimal, hexadecimal (0x prefix), and octal (0 prefix).
func parseUint(s string) (uint64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty unsigned integer value")
	}

	if n, err := strconv.ParseUint(s, 0, 64); err == nil {
		return n, nil
	}

	// Fall back to float parsing and floor
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		if f < 0 {
			return 0, fmt.Errorf("negative value for unsigned field: %q", s)
		}
		return uint64(math.Floor(f)), nil
	}

	return 0, fmt.Errorf("invalid unsigned integer value: %q", s)
}

// parseFloat interprets a string as a floating-point number.
// bitSize specifies the precision: 32 for float32, 64 for float64.
func parseFloat(s string, bitSize int) (float64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty float value")
	}
	return strconv.ParseFloat(s, bitSize)
}

// intInBounds checks whether v is within the min and max for a signed integer kind.
func intInBounds(k reflect.Kind, v int64) bool {
	switch k {
	case reflect.Int8:
		return v >= math.MinInt8 && v <= math.MaxInt8
	case reflect.Int16:
		return v >= math.MinInt16 && v <= math.MaxInt16
	case reflect.Int32:
		return v >= math.MinInt32 && v <= math.MaxInt32
	case reflect.Int64, reflect.Int:
		return v >= math.MinInt64 && v <= math.MaxInt64
	default:
		return v >= math.MinInt64 && v <= math.MaxInt64
	}
}

// uintInBounds checks whether v is within the max for an unsigned integer kind.
func uintInBounds(k reflect.Kind, v uint64) bool {
	switch k {
	case reflect.Uint8:
		return v <= math.MaxUint8
	case reflect.Uint16:
		return v <= math.MaxUint16
	case reflect.Uint32:
		return v <= math.MaxUint32
	case reflect.Uint64, reflect.Uint:
		return v <= math.MaxUint64
	default:
		return v <= math.MaxUint64
	}
}
