# `ini` package specification

Write a new reference file that discusses how the program shall (un)marshal Go structs to ini conf
files using struct tags tags. This file should be complementary with the file
reference/ini-configuration-file-format.md. That file covers ini specification, while this file
discusses (un)marshaling between that format and the Go types and tagged structs.

**ALWAYS** use terminology from the `ini` specification when describing types, functions, methods,
etc. to define in go lang.

**DO NOT** read any `.go` code files for writing this file, but **DO** write pseudo-code as an
example if helpful.

Supported struct field types:

- string, bool, int/int8/int16/int32/int64, uint/uint8/uint16/uint32/uint64, float32/float64.

Supported `ini` parameter value types:

- boolean strings for `bool` fields
- integer values for int types, incl. hex and octal
- decimal values for float types
- unit strings are not supported

Struct tags `ini`:

- `ini:"<PARAMETER_KEY>"` maps the struct field to ini parameter key
- the struct field's type designates how to (un)marshal
- fields missing tags are skipped during unmarshaling
- empty (e.g. `ini:""`) tags are skipped during (un)marshaling
- zero-value fields are skipped during marshaling

Customized (un)marshaling:

- a struct method named `Unmarshal_<FieldName>`, where `FieldName` matches a struct field name (not
  the tag value) with an `ini` tag, shall be called during unmarshaling. The method is looked up via
  reflection on the target struct pointer.
- a struct method named `Marshal_<FieldName>`, where `FieldName` matches a struct field name with an
  `ini` tag, will be called during marshaling.

## Unmarshaling: (`ini` file contents → Go)

The goal of unmarshaling is to take an ini file and a section within that file and return a typed
struct pointer filled with data.

```go pseudo-code
func UnmarshalIniFile[T any](path string, section string) (*T, error)
func UnmarshalIniString[T any](path string, section string, contents string) (*T, error)
```

Unmashaling occurs in passes:

1. **file structure pass:** file structure parsing
1. **parameter value pass:** parameter value unmarshaling (i.e. deserialization)

Both of these passes should follow the
[ini file BNF grammar](./ini-configuration-file-format.md#bnf-grammar)).

### Pass 1: file structure

First, an ini file is read from disk and its structure loaded into an intermediary `IniFile`:

```go pseudo-code
func unmarshalIniFileIntermediate(path string, contents string) (*IniFile, error)
```

Essentially, this takes a path `string` and ini file contents `string` and returns an intermediary
`IniFile` pointer or a parsing `error`.

This function should parse the file contents line-by-line, character-by-character, following the
[ini file BNF grammar](./ini-configuration-file-format.md#bnf-grammar). If the file doesn't parse as
expected, the function **shall** return an explanatory error message including line number, offset
position, and name of the BNF node. The purpose of this error is to help users debug their ini
files.

During parsing, a simple `IniFile` struct pointer intermediate is loaded incrimentally:

```go pseudo-code
// IniFile is an intermediate structural representation of the ini configuration file.
type IniFile struct {
  // ini file path on disk (must end in `.conf`)
	path           string
  // current cursor for inline parsing
  cursor:        *FileCursor
  // a stack of inactive cursors for following include directives
  stack:         []*FileCursor
  // tracks file paths to prevent circular includes
  visited        map[string]bool
  // unnamed section in the file
	defaultSection Section
  // named sections in the file (map keys shall be downcased b/c section names are case-insensitive)
	sections       map[string]Section
}

// Section represents a named group of key-value parameters.
type Section struct {
  // zero-value if default section
	name string
	params map[string]Param
  // current cursor for inline parsing
  cursor: Cursor
}

// Param represents a single parameter with its raw string value.
type Param struct {
  // shall not be zero-value
	name string
  // zero-value if empty
	value string
  // line and offset of this param
  cursor: Cursor
}

// Parser position
type Cursor struct {
  // current line
  line: int32
  // current offset in line
  offset: int32
}

// Parser position within a file
type FileCursor struct {
  Cursor
  // ini file path on disk (must end in `.conf`)
  path: string
  // file contents
  contents: string
}
```

Here is roughly how parsing should happen:

1. read path and construct `IniFile` and a `FileCursor`, push into `IniFile#stack`
1. loop until `IniFile#stack` is empty
1. pop the `IniFile#stack` and set `IniFile#cursor` to that
1. parse line-by-line, offset-by-offset following the
   [ini file BNF grammar](./ini-configuration-file-format.md#bnf-grammar))
    - if a valid include directive is reached:
        1. push the current cursor onto the `IniFile#stack`
        1. push all the new `FileCursor` pointers onto the stack in reverse order
        1. set `IniFile#cursor` to `nil`
        1. go back to the beginning of the loop
1. before returning `*IniFile`, set `IniFile#cursor` to `nil`

**IMPORTANT:** use `visited map[string]bool` for circular include prevention

> [!NOTE] Whenever a valid include directive is reached, the referenced file(s) shall be parsed
> inline, as if all the linked ini files represent one file. Therefore, the current `IniFile`
> reference does not change. However, for parsing, debugging, and error reporting purposes, new
> `FileCursor` instance(s) is(are) constructed for the(all) included file(s). Whenever a parsing
> error is encountered, those positions and paths can be used in error messaging.

### Pass 2: parameter value pass

```go pseudo-code
func unmarshalIniFileStruct[T any](iniFile *IniFile, section string, target *T) error
```

After the full `IniFile` tree is parsed, some number of `Param` leaves exist. You shall now parse
and unmarshal their string values in insertion order. You shall implement a parser for every
supported struct field types. Use the attached `Param#cursor` instances to enhance error messaging
and debugging.

After parsing data strings, built-in methods for string unmarshaling integers, floats, etc.

#### Numeric parsing behavior

**Integers (`int`, `int8`, `int16`, `int32`, `int64`):**

- Parsed via `strconv.ParseInt` with base 0 (auto-detecting decimal, hex `0x`, octal `0` prefixes)
- **Float fallback:** if the value is not a valid integer string but is a valid float string (e.g.
  `3.7`), the value is parsed as float64 and **floored** (truncated toward negative infinity) to
  produce the integer. This matches C-style cast semantics.
- After parsing, the value is bounds-checked against the target type's range (e.g. `-128..127` for
  `int8`). Boundary values are inclusive — `math.MinInt8` and `math.MaxInt8` are both valid for
  `int8`.

**Unsigned integers (`uint`, `uint8`, `uint16`, `uint32`, `uint64`):**

- Parsed via `strconv.ParseUint` with base 0, supporting the full unsigned range up to
  `math.MaxUint64`.
- **Float fallback:** same as signed integers (floor), but negative float values are rejected.
- Bounds-checked against the target type's max (e.g. `0..255` for `uint8`). Boundary values are
  inclusive.

**Floats (`float32`, `float64`):**

- Parsed via `strconv.ParseFloat` using the **target type's bit size** (32 or 64). This ensures that
  exact boundary values like `math.MaxFloat32` are correctly accepted — parsing at 64-bit precision
  would produce a float64 slightly above `MaxFloat32`, falsely triggering an overflow.
- Out-of-range values are rejected by `strconv.ParseFloat` itself (returns `+Inf`/`-Inf` with an
  error), so no separate bounds check is needed.

**Special string types:**

The ini spec mentions special string types like boolean literals and unit strings. Boolean literals
shall be supported automatically depending if the tagged struct property is a `bool` type or
`string` type. You shall implement this.

For special string types (like `KB` or `min`), we do not support this out of the box. Users may
define their own custom (un)marshaling methods to handle whatever they want.

**Custom (un)marshaling methods:**

- `Unmarshal_` must have the signature:
    - `func (k *<STRUCT_TYPE>) Unmarshal_<FieldName>(string) (<FIELD_TYPE>, error)`
    - methods are matched using the struct field name (not the `ini` tag value)
    - method param is `Param#value`

Errors return from the functions should be wrapped with positional debugging information from the
`Param#cursor`.

## Marshaling: (`ini` file contents → Go)

```go pseudo-code
func MarshalIniFile[T any](path string, data *T) error
func MarshalIniString[T any](contents string, data *T) error
```

Iterate through the struct using reflection in the order it is defined. Use `ini:` tags and
`marshal_` functions as expected. Use consistent string types for `bool`. Be sure to single-quote
and quote-escape strings.

- `Marshal_` must have the signature:
    - `func (k *<STRUCT_TYPE>) Marshal_<FieldName>(*<FIELD_TYPE>) (string, error)`
