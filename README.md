# inigo

A Go package for parsing `.ini` configuration files, aligned with the
[PostgreSQL Connection Service File](https://www.postgresql.org/docs/current/libpq-pgservice.html)
format.

<!-- BADGES:START -->

[![Go Reference](./badges/go-reference-badge.svg)](https://pkg.go.dev/github.com/thesmart/inigo)
[![Coverage 98%](./badges/coverage-badge.svg)](https://github.com/thesmart/inigo)
[![Go Report A+](./badges/go-report-badge.svg)](https://goreportcard.com/report/github.com/thesmart/inigo)
[![License MIT](./badges/license-badge.svg)](./LICENSE)

<!-- BADGES:END -->

## Using as a library

```go
import "github.com/thesmart/inigo"

// Parse from file
cfg, err := inigo.Load("config.ini")

// Parse from reader
cfg, err := inigo.Parse(reader)

// Load directly into a struct using ini tags
type DBConfig struct {
    Host    string `ini:"host"`
    Port    int    `ini:"port"`
    Name    string `ini:"dbname"`
    SslMode string `ini:"sslmode"`
    Secret  string `ini:"-"`     // ignored
    Other   string               // ignored (no ini tag)
}

var cfg DBConfig
err := inigo.LoadInto("pg_service.conf", "mydb", &cfg)
```

## Using with CLI

The `inigo` binary loads params from an INI section, exports them as uppercase
environment variables, and execs a command with those variables set.

### Install

Install latest to `~/.local/bin`:

```sh
curl -fsSL https://raw.githubusercontent.com/thesmart/inigo/main/install.sh | sh
```

Options via environment variables:

```sh
# Install to a custom directory
INSTALL_DIR=/usr/local/bin curl -fsSL https://raw.githubusercontent.com/thesmart/inigo/main/install.sh | sh

# Install a specific version
VERSION=v0.1.2 curl -fsSL https://raw.githubusercontent.com/thesmart/inigo/main/install.sh | sh

# Install via Go:
go install github.com/thesmart/inigo/cmd/inigo@latest
```

### Example

Given a `pg_service.conf` file:

```ini
[mydb]
host = localhost
port = 5432
dbname = myapp
sslmode = require
```

Run any command with those params injected as environment variables:

```sh
# Use --prefix to prepend a string to each variable name
inigo --prefix PG pg_service.conf mydb -- psql
# psql receives: PGHOST=localhost PGPORT=5432 PGDBNAME=myapp PGSSLMODE=require

# Without --prefix, params become plain uppercase names
inigo pg_service.conf mydb -- env | grep -E 'HOST|PORT'
# HOST=localhost
# PORT=5432

# Pass additional flags to the child command
inigo --prefix PG pg_service.conf mydb -- pg_dump --format=custom -f backup.dump
```

Run `inigo --help` for full usage.

## Ini File Specification

Configuration `ini` files isn't really a specification, but is a convention that
is is implementation specific. This library follows the convention used by the
[PostgreSQL project](https://www.postgresql.org/docs/9.3/config-setting.html).

### Rules

All parameter names are case-insensitive and may contain characters `a-z`,
`A-Z`, `0-9`, `_`, and `$`. Parameter names may only start with `a-z`, `A-Z`,
and `_`.

Every parameter takes a value of one of four types:

- _Boolean:_ Values can be written as `on`, `off`, `true`, `false`, `yes`, `no`,
  `1`, `0` (all case-insensitive) or any unambiguous prefix (`t`, `f`, `y`, `n`)
  of one of these.

- _String:_ In general, enclose the value in single quotes (`'`). Escape any
  single quotes by doubling any single quotes (`''`) within the value. Quotes
  can usually be omitted if the value is a simple number or identifier.

- _Numeric (integer and floating point):_ Numeric parameters can be specified in
  the customary integer and floating-point formats; fractional values are
  rounded to the nearest integer if the parameter is of integer type. Integer
  parameters additionally accept hexadecimal input (beginning with `0x`) and
  octal input (beginning with `0`), but these formats cannot have a fraction. Do
  not use thousands separators. Quotes are not required, except for hexadecimal
  input.

One parameter is specified per line. The equal sign `=` must be between name and
value. Value is optional. Empty values are interpreted as empty-string.
Whitespace is insignificant (except within a quoted parameter value) and blank
lines are ignored. Hash marks (`#`) designate the remainder of the line as a
comment. Parameter values that are not simple identifiers or numbers must be
single-quoted. To embed a single quote in a parameter value, write either two
quotes `''` (preferred) or backslash-quote `\'`. If the file contains multiple
entries for the same parameter, all but the last one are ignored.

### Sections

Sections are declared with a `[name]` header. All parameters following a section
header belong to that section until the next header or end of file.

```ini
[mydb]
host = localhost
port = 5432
```

Sections are **optional**. A file may contain zero, one, or many sections.
Re-entering a section (declaring `[name]` again later in the file) merges
parameters into the existing section rather than replacing it.

#### Default Section

Parameters that appear _before_ any section header belong to the **default
section**. The default section has an empty name and is always present in the
parsed result, even when the file contains no parameters outside of named
sections.

```ini
# These belong to the default section
host = localhost
port = 5432

[mydb]
dbname = app
```

Access the default section with an empty string:

```go
cfg, _ := inigo.Load("config.ini")

// Named section
db := cfg.Section("mydb")

// Default section
defaults := cfg.Section("")
```

`SectionNames()` returns only named sections — it excludes the default section.
`LoadInto` and `SaveFrom` also accept an empty string to target the default
section.

### Include Directives

- `include 'filename'` - include another file
- `include_if_exists 'filename'` - include if file exists
- `include_dir 'directory'` - include all `.conf` files in directory processed
  in ascii order, files starting with `.` are excluded

Processes the referenced file "as if it were inserted into the configuration
file at that point".
