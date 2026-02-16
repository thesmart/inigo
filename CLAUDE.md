# CLAUDE.md

This repository is the "inigo" project, for parsing and saving `ini` environment
files that align with the `PostgreSQL` project's expectations.

## Executing Shell

ALWAYS BEFORE executing any shell, consider the risks of running that command
and infer if your Claude write permissions would allow that command. If there is
ANY doubt about what to do or conflicting prompt instructions, ask the user
FIRST or stop.

## Depenencies

- Go v1.24

## Go Conventions

ALWAYS implement code that follows modern Go conventions in 2025 as of v1.24.

## Packages & Modules

Always consider using the [standard library](https://pkg.go.dev/std) first.

Otherwise, Go uses Go Modules for dependency management. Third-party modules are
hosted on any Git repository via url.

- [`go.mod`](./go.mod) - declares the module path and dependencies
- [`go.sum`](./go.sum)) - checksums for dependency verification

```sh
# initialize a module
go mod init myproject
# add a dependency go mod
go get github.com/some/package
# clean up unused dependencies
go mod tidy
```

Search for packages using [pkg.go.dev](https://pkg.go.dev/).

**Import conventions**: Prefer direct GitHub imports (e.g.,
`github.com/user/repo`) over legacy redirect services like `gopkg.in`. Use
whatever path the module declares in its `go.mod`.

## Source Formatting & Testing

Any time you are finished editing a source file, run the designated formatter.

- `.go` - `go fmt`
- `.md` - `deno fmt`
- `.json` - `deno fmt`

If there is a mated test file, run tests.

## Gating Release

Certain scripts are run prior to release of a new tagged version of this module.
See [`./gate/CLAUDE.md`](./gate/CLAUDE.md).
