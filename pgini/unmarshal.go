// Unmarshaling decodes PGINI sections into Go structs using struct field tags.
//
// Fields are mapped via `ini:"KEY"` tags. Fields without an `ini` tag or with
// an empty tag value are skipped. For primitive types (string, bool, int*,
// uint*, float*), a default parser is used. For other types, a custom
// Unmarshal<FieldName> method must exist on the struct.

package pgini

import (
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
)

// UnmarshalSection decodes the named section's parameters into the exported
// fields of s. s must be a pointer to a struct. Fields are matched by their
// `ini:"KEY"` tag. Fields without an `ini` tag or with an empty tag value are
// skipped. Parameters that do not match any field are ignored.
func (f *IniFile) UnmarshalSection(name string, v any) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Pointer || rv.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("UnmarshalSection: v must be a pointer to a struct, got %T", v)
	}
	rv = rv.Elem()
	rt := rv.Type()

	s := f.GetSection(name)
	if s == nil {
		return fmt.Errorf("UnmarshalSection: section %q not found", name)
	}

	for i := range rt.NumField() {
		sf := rt.Field(i)
		tag, ok := sf.Tag.Lookup("ini")
		if !ok || tag == "" {
			continue
		}

		p, found := s.GetParam(tag)
		if !found {
			continue
		}

		fv := rv.Field(i)
		if err := unmarshalField(rv, sf, fv, p); err != nil {
			return fmt.Errorf("UnmarshalSection: field %s: %w", sf.Name, err)
		}
	}

	return nil
}

// unmarshalField sets a struct field from a Param value. If a custom
// Unmarshal<FieldName> method exists on the struct, it is used. Otherwise,
// the default setFieldFromParam is used for primitive types.
func unmarshalField(structVal reflect.Value, sf reflect.StructField, fv reflect.Value, p *Param) error {
	methodName := "Unmarshal" + sf.Name
	method := structVal.Addr().MethodByName(methodName)
	if method.IsValid() {
		return callCustomUnmarshal(method, sf, fv, p)
	}
	return setFieldFromParam(fv, p)
}

// callCustomUnmarshal invokes a custom Unmarshal<FieldName> method and validates
// its signature: func(s *StructType) Unmarshal<FieldName>(value string) (*FieldType, error).
func callCustomUnmarshal(method reflect.Value, sf reflect.StructField, fv reflect.Value, p *Param) error {
	mt := method.Type()

	if mt.NumIn() != 1 {
		return fmt.Errorf("Unmarshal%s: expected 1 parameter, got %d", sf.Name, mt.NumIn())
	}
	if mt.In(0) != reflect.TypeFor[string]() {
		return fmt.Errorf("Unmarshal%s: parameter must be string, got %s", sf.Name, mt.In(0))
	}
	if mt.NumOut() != 2 {
		return fmt.Errorf("Unmarshal%s: expected 2 return values, got %d", sf.Name, mt.NumOut())
	}
	if !mt.Out(1).Implements(reflect.TypeFor[error]()) {
		return fmt.Errorf("Unmarshal%s: second return value must be error, got %s", sf.Name, mt.Out(1))
	}

	// First return value must be a pointer to the field type.
	expectedOut := reflect.PointerTo(sf.Type)
	if mt.Out(0) != expectedOut {
		return fmt.Errorf("Unmarshal%s: first return value must be %s, got %s", sf.Name, expectedOut, mt.Out(0))
	}

	results := method.Call([]reflect.Value{reflect.ValueOf(p.Value)})
	if !results[1].IsNil() {
		return results[1].Interface().(error)
	}

	// Dereference the returned pointer and set the field.
	retVal := results[0]
	if retVal.IsNil() {
		return fmt.Errorf("Unmarshal%s: returned nil pointer", sf.Name)
	}
	fv.Set(retVal.Elem())
	return nil
}

// setFieldFromParam assigns a Param value to a struct field based on the field's type.
func setFieldFromParam(fv reflect.Value, p *Param) error {
	switch fv.Kind() {
	case reflect.String:
		fv.SetString(p.Value)
	case reflect.Bool:
		v, err := parseBool(p.Value)
		if err != nil {
			return err
		}
		fv.SetBool(v)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		v, err := parseInt(p.Value)
		if err != nil {
			return err
		}
		if !intInBounds(fv.Kind(), v) {
			return fmt.Errorf("value %d overflows %s", v, fv.Kind())
		}
		fv.SetInt(v)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		v, err := parseUint(p.Value)
		if err != nil {
			return err
		}
		if !uintInBounds(fv.Kind(), v) {
			return fmt.Errorf("value %d overflows %s", v, fv.Kind())
		}
		fv.SetUint(v)
	case reflect.Float32:
		v, err := parseFloat(p.Value, 32)
		if err != nil {
			return err
		}
		fv.SetFloat(v)
	case reflect.Float64:
		v, err := parseFloat(p.Value, 64)
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
func parseBool(s string) (bool, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return false, fmt.Errorf("empty boolean value")
	}

	trueWords := []string{"t", "1", "true", "on", "y", "yes"}
	falseWords := []string{"f", "0", "false", "off", "n", "no"}

	matchesTrue := false
	matchesFalse := false
	for _, w := range trueWords {
		if w == s {
			matchesTrue = true
		}
	}
	for _, w := range falseWords {
		if w == s {
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
		return true
	default:
		return true
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
		return true
	default:
		return true
	}
}
