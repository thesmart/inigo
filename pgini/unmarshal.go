// Unmarshaling decodes IniFile instances into Go structs using struct field tags.
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
// fields of structPtr. structPtr must be a pointer to a struct. Fields are matched
// by their `ini:"KEY"` tag. Fields without an `ini` tag or with an empty tag value
// are skipped. Parameters that do not match any field are ignored.
func (f *IniFile) UnmarshalSection(name string, structPtr any) error {
	// Unwrap the pointer to get the underlying struct value and its type descriptor.
	structValue := reflect.ValueOf(structPtr)
	if structValue.Kind() != reflect.Pointer || structValue.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("UnmarshalSection: data must be a pointer to a struct, got %T", structPtr)
	}
	structValue = structValue.Elem()
	structType := structValue.Type()

	section := f.GetSection(name)
	if section == nil {
		return fmt.Errorf("UnmarshalSection: section %q not found", name)
	}

	// Iterate over each exported struct field, looking for `ini` tags.
	for i := range structType.NumField() {
		fieldDef := structType.Field(i) // field metadata (name, type, tags)
		tag, ok := fieldDef.Tag.Lookup("ini")
		if !ok || tag == "" {
			continue
		}

		param, found := section.GetParam(tag)
		if !found {
			continue
		}

		fieldValue := structValue.Field(i) // the runtime value of this field
		if err := unmarshalField(structValue, fieldDef, fieldValue, param); err != nil {
			return fmt.Errorf("UnmarshalSection: field %s: %w", fieldDef.Name, err)
		}
	}

	return nil
}

// unmarshalField sets a struct field from a Param value. If a custom
// Unmarshal<FieldName> method exists on the struct, it is used. Otherwise,
// the default setFieldFromParam is used for primitive types.
//
// Parameters:
//   - structValue: the reflect.Value of the dereferenced struct instance
//   - fieldDef: metadata for the struct field (name, type, tags)
//   - fieldValue: the runtime value of the struct field to populate
//   - param: the INI parameter whose value will be decoded into fieldValue
func unmarshalField(structValue reflect.Value, fieldDef reflect.StructField, fieldValue reflect.Value, param *Param) error {
	// Look for a custom unmarshal method named Unmarshal<FieldName> on the struct's pointer receiver.
	methodName := "Unmarshal" + fieldDef.Name
	method := structValue.Addr().MethodByName(methodName)
	if method.IsValid() {
		return callCustomUnmarshal(method, fieldDef, fieldValue, param)
	}
	return setFieldFromParam(fieldValue, param)
}

// callCustomUnmarshal invokes a custom Unmarshal<FieldName> method and validates
// its signature: func(s *StructType) Unmarshal<FieldName>(value string) (*FieldType, error).
//
// Parameters:
//   - method: the reflected method value (already resolved via MethodByName)
//   - fieldDef: metadata for the struct field (used to derive expected return type)
//   - fieldValue: the runtime value of the struct field to populate with the result
//   - param: the INI parameter whose string value is passed to the method
func callCustomUnmarshal(method reflect.Value, fieldDef reflect.StructField, fieldValue reflect.Value, param *Param) error {
	methodSig := method.Type() // the method's function signature

	// Validate the method signature: exactly 1 string input, 2 outputs (*FieldType, error).
	if methodSig.NumIn() != 1 {
		return fmt.Errorf("Unmarshal%s: expected 1 parameter, got %d", fieldDef.Name, methodSig.NumIn())
	}
	if methodSig.In(0) != reflect.TypeFor[string]() {
		return fmt.Errorf("Unmarshal%s: parameter must be string, got %s", fieldDef.Name, methodSig.In(0))
	}
	if methodSig.NumOut() != 2 {
		return fmt.Errorf("Unmarshal%s: expected 2 return values, got %d", fieldDef.Name, methodSig.NumOut())
	}
	if !methodSig.Out(1).Implements(reflect.TypeFor[error]()) {
		return fmt.Errorf("Unmarshal%s: second return value must be error, got %s", fieldDef.Name, methodSig.Out(1))
	}

	// The first return value must be a pointer to the field's declared type.
	expectedReturnType := reflect.PointerTo(fieldDef.Type)
	if methodSig.Out(0) != expectedReturnType {
		return fmt.Errorf("Unmarshal%s: first return value must be %s, got %s", fieldDef.Name, expectedReturnType, methodSig.Out(0))
	}

	results := method.Call([]reflect.Value{reflect.ValueOf(param.Value)})
	if !results[1].IsNil() {
		return results[1].Interface().(error)
	}

	// Dereference the returned pointer and set the field.
	returnedPtr := results[0]
	if returnedPtr.IsNil() {
		return fmt.Errorf("Unmarshal%s: returned nil pointer", fieldDef.Name)
	}
	fieldValue.Set(returnedPtr.Elem())
	return nil
}

// setFieldFromParam assigns a Param value to a struct field based on the field's type.
//
// Parameters:
//   - fieldValue: the settable reflect.Value of the target struct field
//   - param: the INI parameter whose string value will be parsed and assigned
func setFieldFromParam(fieldValue reflect.Value, param *Param) error {
	switch fieldValue.Kind() {
	case reflect.String:
		fieldValue.SetString(param.Value)
	case reflect.Bool:
		v, err := parseBool(param.Value)
		if err != nil {
			return err
		}
		fieldValue.SetBool(v)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		v, err := parseInt(param.Value)
		if err != nil {
			return err
		}
		if !intInBounds(fieldValue.Kind(), v) {
			return fmt.Errorf("value %d overflows %s", v, fieldValue.Kind())
		}
		fieldValue.SetInt(v)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		v, err := parseUint(param.Value)
		if err != nil {
			return err
		}
		if !uintInBounds(fieldValue.Kind(), v) {
			return fmt.Errorf("value %d overflows %s", v, fieldValue.Kind())
		}
		fieldValue.SetUint(v)
	case reflect.Float32:
		v, err := parseFloat(param.Value, 32)
		if err != nil {
			return err
		}
		fieldValue.SetFloat(v)
	case reflect.Float64:
		v, err := parseFloat(param.Value, 64)
		if err != nil {
			return err
		}
		fieldValue.SetFloat(v)
	default:
		return fmt.Errorf("unsupported field type: %s", fieldValue.Kind())
	}
	return nil
}

// parseBool interprets a string as a boolean value.
func parseBool(raw string) (bool, error) {
	raw = strings.ToLower(strings.TrimSpace(raw))
	if raw == "" {
		return false, fmt.Errorf("empty boolean value")
	}

	trueWords := []string{"t", "1", "true", "on", "y", "yes"}
	falseWords := []string{"f", "0", "false", "off", "n", "no"}

	matchesTrue := false
	matchesFalse := false
	for _, w := range trueWords {
		if w == raw {
			matchesTrue = true
		}
	}
	for _, w := range falseWords {
		if w == raw {
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
		return false, fmt.Errorf("ambiguous boolean value: %q", raw)
	}
	return false, fmt.Errorf("invalid boolean value: %q", raw)
}

// parseInt interprets a string as an integer.
// Supports decimal, hexadecimal (0x prefix), and octal (0 prefix).
func parseInt(raw string) (int64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, fmt.Errorf("empty integer value")
	}

	if n, err := strconv.ParseInt(raw, 0, 64); err == nil {
		return n, nil
	}

	// Fall back to float parsing and floor
	if f, err := strconv.ParseFloat(raw, 64); err == nil {
		return int64(math.Floor(f)), nil
	}

	return 0, fmt.Errorf("invalid integer value: %q", raw)
}

// parseUint interprets a string as an unsigned integer.
// Supports decimal, hexadecimal (0x prefix), and octal (0 prefix).
func parseUint(raw string) (uint64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, fmt.Errorf("empty unsigned integer value")
	}

	if n, err := strconv.ParseUint(raw, 0, 64); err == nil {
		return n, nil
	}

	// Fall back to float parsing and floor
	if f, err := strconv.ParseFloat(raw, 64); err == nil {
		if f < 0 {
			return 0, fmt.Errorf("negative value for unsigned field: %q", raw)
		}
		return uint64(math.Floor(f)), nil
	}

	return 0, fmt.Errorf("invalid unsigned integer value: %q", raw)
}

// parseFloat interprets a string as a floating-point number.
// bitSize specifies the precision: 32 for float32, 64 for float64.
func parseFloat(raw string, bitSize int) (float64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, fmt.Errorf("empty float value")
	}
	return strconv.ParseFloat(raw, bitSize)
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
