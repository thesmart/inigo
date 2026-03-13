// Marshaling encodes Go structs into IniFile sections using struct field tags.
//
// Fields are mapped via `ini:"KEY"` tags. Fields without an `ini` tag or with
// an empty tag value are skipped. For primitive types (string, bool, int*,
// uint*, float*), a default formatter is used. For other types, a custom
// Marshal<FieldName> method must exist on the struct.

package pgini

import (
	"fmt"
	"reflect"
)

// MarshalSection encodes the exported fields of `data` into the named section,
// creating the section if it does not exist. `data` must be a pointer to a struct.
// Fields are matched by their `ini:"KEY"` tag. Fields without an `ini` tag
// or with an empty tag value are skipped.
func (f *IniFile) MarshalSection(name string, data any) error {
	rv := reflect.ValueOf(data)
	if rv.Kind() != reflect.Pointer || rv.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("MarshalSection: data must be a pointer to a struct, got %T", data)
	}
	rv = rv.Elem()
	rt := rv.Type()

	s, err := f.AddSection(name)
	if err != nil {
		return fmt.Errorf("MarshalSection: %w", err)
	}

	for i := range rt.NumField() {
		sf := rt.Field(i)
		tag, ok := sf.Tag.Lookup("ini")
		if !ok || tag == "" {
			continue
		}

		fv := rv.Field(i)
		str, err := marshalField(rv, sf, fv)
		if err != nil {
			return fmt.Errorf("MarshalSection: field %s: %w", sf.Name, err)
		}

		if _, err := s.SetParam(tag, str); err != nil {
			return fmt.Errorf("MarshalSection: field %s: %w", sf.Name, err)
		}
	}

	return nil
}

// marshalField converts a struct field value to its PGINI string representation.
// If a custom Marshal<FieldName> method exists on the struct, it is used.
// Otherwise, the default formatField is used for primitive types.
func marshalField(structVal reflect.Value, sf reflect.StructField, fv reflect.Value) (string, error) {
	methodName := "Marshal" + sf.Name
	method := structVal.Addr().MethodByName(methodName)
	if method.IsValid() {
		return callCustomMarshal(method, sf, fv)
	}
	return formatField(fv)
}

// callCustomMarshal invokes a custom Marshal<FieldName> method and validates
// its signature: func(s *StructType) Marshal<FieldName>(value *FieldType) (string, error).
func callCustomMarshal(method reflect.Value, sf reflect.StructField, fv reflect.Value) (string, error) {
	mt := method.Type()

	if mt.NumIn() != 1 {
		return "", fmt.Errorf("Marshal%s: expected 1 parameter, got %d", sf.Name, mt.NumIn())
	}
	if mt.NumOut() != 2 {
		return "", fmt.Errorf("Marshal%s: expected 2 return values, got %d", sf.Name, mt.NumOut())
	}
	if mt.Out(0) != reflect.TypeFor[string]() {
		return "", fmt.Errorf("Marshal%s: first return value must be string, got %s", sf.Name, mt.Out(0))
	}
	if !mt.Out(1).Implements(reflect.TypeFor[error]()) {
		return "", fmt.Errorf("Marshal%s: second return value must be error, got %s", sf.Name, mt.Out(1))
	}

	// Parameter must be a pointer to the field type.
	expectedIn := reflect.PointerTo(sf.Type)
	if mt.In(0) != expectedIn {
		return "", fmt.Errorf("Marshal%s: parameter must be %s, got %s", sf.Name, expectedIn, mt.In(0))
	}

	// Build a pointer to the field value.
	ptr := reflect.New(sf.Type)
	ptr.Elem().Set(fv)

	results := method.Call([]reflect.Value{ptr})
	if !results[1].IsNil() {
		return "", results[1].Interface().(error)
	}
	return results[0].String(), nil
}

// formatField converts a primitive field value to its PGINI string representation.
func formatField(fv reflect.Value) (string, error) {
	switch fv.Kind() {
	case reflect.String:
		return pginiEscape(fv.String()), nil
	case reflect.Bool:
		if fv.Bool() {
			return "true", nil
		}
		return "false", nil
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
