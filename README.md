# inigo

<img align="right" src="./gate/inigo-logo.png" width="100" alt="inigo logo" />

Go package and CLI tool for INI configuration files.

1. **The package**: parse and mutate INI files, loads into structs
1. **The CLI (inigo)**: injects INI config into a command's environment

<!-- BADGES:START -->

[![Go Reference](./gate/go-reference-badge.svg)](https://pkg.go.dev/github.com/thesmart/inigo)
[![Coverage 94%](./gate/coverage-badge.svg)](https://github.com/thesmart/inigo)
[![Go Report A+](./gate/go-report-badge.svg)](https://goreportcard.com/report/github.com/thesmart/inigo)
[![License MIT](./gate/license-badge.svg)](./LICENSE)

<!-- BADGES:END -->

### What is an INI config file?

```ini
# example ini configuration
name = myapp
host = localhost
port = 1337
```

> There is no standard. Every app decides its own INI conventions, so you might say we haven't ini
> standard... 😅 sorry

This app settles on [PostgreSQL's INI conventions](./reference/pgini-specification.md), and now you
can too.

## Using the package

You can also use it as a library to set structs:

```sh
go get github.com/thesmart/inigo
```

```go
import "github.com/thesmart/inigo/pgini"

// Load directly into a struct using ini tags
type AppConfig struct {
    Host    string `ini:"host"`
    Port    int    `ini:"port"`
    Other   string // ignored (no ini tag)
}

// load `$PWD/.env`
cfg, err := pgini.Load[AppConfig]("example.conf", "")
```

Check out more examples in the [User's Guide](./examples/users-guide.md).

## Using the CLI tool

Inject parameters into any command:

```sh
# contents of pg.conf:
# [mydb]
# PGHOST=localhost
# PGPORT=5432
inigo env pg.conf mydb -- psql
# psql receives: PGHOST=localhost PGPORT=5432 ...
inigo json --case=camelCase pg.conf mydb | jq
```

```json
{
    "pghost": "localhost",
    "pgport": "5432"
}
```

#### Installing the `inigo` CLI tool

```sh
# Install latest to `~/.local/bin`:
curl -fsSL https://raw.githubusercontent.com/thesmart/inigo/main/install.sh | sh
```

or:

```sh
# Install to a custom directory
INSTALL_DIR=/usr/local/bin curl -fsSL https://raw.githubusercontent.com/thesmart/inigo/main/install.sh | sh
# Install a specific version
VERSION=v0.1.2 curl -fsSL https://raw.githubusercontent.com/thesmart/inigo/main/install.sh | sh
# Install via Go:
go install github.com/thesmart/inigo/cmd/inigo@latest
```
