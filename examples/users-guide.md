# User's Guide

The `pgini` package reads and writes INI configuration files using
[PostgreSQL's conventions](../reference/pgini-specification.md). This guide walks through the
package by example, starting simple and building up.

## The conf format

A PGINI file has sections, key-value parameters, comments, and quoted strings:

```ini
# Default section (before any [header])
app_name = myapp

[server]
host = 0.0.0.0
port = 8080

[database]
host = localhost
port = 5432
name = myapp_db       # trailing comments are allowed
connection = 'host=localhost port=5432 dbname=myapp'  ; quoted values support spaces
```

See the [full specification](../reference/pgini-specification.md) for quoting rules, escape
sequences, include directives, and the formal grammar.

## Example 01: Load into a struct

The fastest way to use pgini. Define a struct with `ini:"key"` tags and call `Load[T]` — it parses
the file and populates the struct in one step. Supported types: `string`, `bool`, `int*`, `uint*`,
`float*`.

Source: [package/01-load-struct/main.go](package/01-load-struct/main.go)

## Example 02: Parse and query

When you don't know the schema ahead of time, use `Parse` to get an `*IniFile` and navigate it
programmatically. Iterate sections with `Sections()`, iterate parameters with `Params()`, or look up
specific values with `GetSection` and `GetValue`.

Source: [package/02-parse-and-query/main.go](package/02-parse-and-query/main.go)

## Example 03: Marshal a struct

Build a conf file from scratch. Create an empty `IniFile` with `NewIniFile`, encode structs into
sections with `MarshalSection`, then serialize the whole thing to PGINI format with `MarshalIni`.

Source: [package/03-marshal-struct/main.go](package/03-marshal-struct/main.go)

## Example 04: Custom marshal and unmarshal

For non-primitive types like `time.Duration` or `[]string`, define custom methods on your struct.
pgini calls `Unmarshal<FieldName>(string) (*T, error)` when loading and
`Marshal<FieldName>(*T) (string, error)` when saving — automatically, based on the field name.

Source: [package/04-custom-marshal/main.go](package/04-custom-marshal/main.go)

## Running the examples

```sh
# Run all examples
task example

# Run a specific example
task example:01
```
