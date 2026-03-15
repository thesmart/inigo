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

// MarshalSection encodes the exported fields of structPtr into the named section,
// creating the section if it does not exist. structPtr must be a pointer to a struct.
// Fields are matched by their `ini:"KEY"` tag. Fields without an `ini` tag
// or with an empty tag value are skipped.
func (f *IniFile) MarshalSection(name string, structPtr any) error {
	// Unwrap the pointer to get the underlying struct value and its type descriptor.
	structValue := reflect.ValueOf(structPtr)
	if structValue.Kind() != reflect.Pointer || structValue.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("MarshalSection: data must be a pointer to a struct, got %T", structPtr)
	}
	structValue = structValue.Elem()
	structType := structValue.Type()

	section, err := f.AddSection(name)
	if err != nil {
		return fmt.Errorf("MarshalSection: %w", err)
	}

	// Iterate over each exported struct field, looking for `ini` tags.
	for i := range structType.NumField() {
		fieldDef := structType.Field(i) // field metadata (name, type, tags)
		tag, ok := fieldDef.Tag.Lookup("ini")
		if !ok || tag == "" {
			continue
		}

		fieldValue := structValue.Field(i) // the runtime value of this field
		str, err := marshalField(structValue, fieldDef, fieldValue)
		if err != nil {
			return fmt.Errorf("MarshalSection: field %s: %w", fieldDef.Name, err)
		}

		if _, err := section.SetParam(tag, str); err != nil {
			return fmt.Errorf("MarshalSection: field %s: %w", fieldDef.Name, err)
		}
	}

	return nil
}

// marshalField converts a struct field value to its PGINI string representation.
// If a custom Marshal<FieldName> method exists on the struct, it is used.
// Otherwise, the default formatField is used for primitive types.
//
// Parameters:
//   - structValue: the reflect.Value of the dereferenced struct instance
//   - fieldDef: metadata for the struct field (name, type, tags)
//   - fieldValue: the runtime value of the struct field being marshaled
func marshalField(structValue reflect.Value, fieldDef reflect.StructField, fieldValue reflect.Value) (string, error) {
	// Look for a custom marshal method named Marshal<FieldName> on the struct's pointer receiver.
	methodName := "Marshal" + fieldDef.Name
	method := structValue.Addr().MethodByName(methodName)
	if method.IsValid() {
		return callCustomMarshal(method, fieldDef, fieldValue)
	}
	return formatField(fieldValue)
}

// callCustomMarshal invokes a custom Marshal<FieldName> method and validates
// its signature: func(s *StructType) Marshal<FieldName>(value *FieldType) (string, error).
//
// Parameters:
//   - method: the reflected method value (already resolved via MethodByName)
//   - fieldDef: metadata for the struct field (used to derive expected parameter type)
//   - fieldValue: the runtime value of the struct field to pass to the method
func callCustomMarshal(method reflect.Value, fieldDef reflect.StructField, fieldValue reflect.Value) (string, error) {
	methodSig := method.Type() // the method's function signature

	// Validate the method signature: exactly 1 input, 2 outputs (string, error).
	if methodSig.NumIn() != 1 {
		return "", fmt.Errorf("Marshal%s: expected 1 parameter, got %d", fieldDef.Name, methodSig.NumIn())
	}
	if methodSig.NumOut() != 2 {
		return "", fmt.Errorf("Marshal%s: expected 2 return values, got %d", fieldDef.Name, methodSig.NumOut())
	}
	if methodSig.Out(0) != reflect.TypeFor[string]() {
		return "", fmt.Errorf("Marshal%s: first return value must be string, got %s", fieldDef.Name, methodSig.Out(0))
	}
	if !methodSig.Out(1).Implements(reflect.TypeFor[error]()) {
		return "", fmt.Errorf("Marshal%s: second return value must be error, got %s", fieldDef.Name, methodSig.Out(1))
	}

	// The input parameter must be a pointer to the field's declared type.
	expectedParamType := reflect.PointerTo(fieldDef.Type)
	if methodSig.In(0) != expectedParamType {
		return "", fmt.Errorf("Marshal%s: parameter must be %s, got %s", fieldDef.Name, expectedParamType, methodSig.In(0))
	}

	// Allocate a new pointer to the field type and copy the field value into it.
	fieldPtr := reflect.New(fieldDef.Type)
	fieldPtr.Elem().Set(fieldValue)

	results := method.Call([]reflect.Value{fieldPtr})
	if !results[1].IsNil() {
		return "", results[1].Interface().(error)
	}
	return results[0].String(), nil
}

// formatField converts a primitive field value to its PGINI string representation.
func formatField(fieldValue reflect.Value) (string, error) {
	switch fieldValue.Kind() {
	case reflect.String:
		return pginiEscape(fieldValue.String()), nil
	case reflect.Bool:
		if fieldValue.Bool() {
			return "true", nil
		}
		return "false", nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return fmt.Sprintf("%d", fieldValue.Int()), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return fmt.Sprintf("%d", fieldValue.Uint()), nil
	case reflect.Float32, reflect.Float64:
		return fmt.Sprintf("%g", fieldValue.Float()), nil
	default:
		return "", fmt.Errorf("unsupported field type: %s", fieldValue.Kind())
	}
}
