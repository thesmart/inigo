# marshaling_test.go and unmarshaling_test.go

Tests for both these files:

1. [`./pgini/marshal.go`](../pgini/marshal.go)
2. [`./pgini/unmarshal.go`](../pgini/unmarshal.go)

Both of these files enable an end-user to write a "schema struct" using field tags, and uses
reflection of the struct's typed fields to automatically handle (un)marshaling.

## End-User Example

Example of how an end-user could use the pgini package to accomplish this:

```go
// end-user-defined "schema struct"
type MyConfigStruct struct {
  Host    string `ini:"HOST"`
  Port    int    `ini:"PORT"`
  Debug   bool   `ini:"DEBUG"`
  Owner   User   `ini:"DEFAULT_USER"`
}

// end-user-defined
type User struct {
    FirstName string
    LastName  string
    Age       int
}

// end-user-defined, called during marshaling
func (s *MyConfigStruct) MarshalOwner(value *User) (string, error) {
  // using JSON string, but could be any sub-encoding
  return json.Marshal(value)
}

// end-user-defined, called by unmarshaling
func (s *MyConfigStruct) UnmarshalOwner(value string) (*User, error) {
  var u User
  err := json.Unmarshal([]byte(input), &u)
  return &u, err
}

// ... elsewhere in code

// whatever data in MyConfigStruct instance
conf := &MyConfigStruct{...}
// a new blank ini file
iniFile := pgini.NewIniFile(path)
// encode the struct to the default section
iniFile.MarshalSection("", conf)
// encode the struct to the production section
iniFile.MarshalSection("production", conf)
// save INI file to disk w/ mod 0644
err := iniFile.WriteFile(0644)
```

Then they can unmarshal data from `.conf` files, vibe as follows:

```go
// empty MyConfigStruct
conf := &MyConfigStruct{}
// open a new ini file
iniFile := pgini.NewIniFile(path)
err := iniFile.ReadFile()
// decode from the ini file, load into the struct
myStruct = iniFile.UnmarshalSection("", conf)
```

### Supported Field Types

- string
- bool
- int/int8/int16/int32/int64
- uint/uint8/uint16/uint32/uint64
- float32/float64
- for other types, the end-user may define custom (un)marshaling methods

### Custom (un)marshal methods

The end-user may define a `Marshal<FIELD_NAME>` and/or `Unmarshal<FIELD_NAME>` method on the schema
struct. For any type, if a custom marshaling method exists, it should **ALWAYS** be used in place of
the default (un)marshaling function.

For non-primitive types that do not have a default (un)marshaling function, the existence of the
custom method is required during (un)marshaling respectively by using reflection. If the
corresponding method does not exist, then an error should be returned.

The signatures are:

```go pseudo-code
func (s *<STRUCT_TYPE>) Marshal<FIELD_NAME>(value *<FIELD_TYPE>) (string, error)
func (s *<STRUCT_TYPE>) Unmarshal<FIELD_NAME>(value string) (*<FIELD_TYPE>, error)
```

During unmarshaling, the non-error return value's type from `Unmarshal<FIELD_NAME>` should be
verified to match the field type, or else result in an error.

Consider other opportunities at runtime for validation of these signatures if the custom methods
exist.
