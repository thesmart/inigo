# CLAUDE.md

This repository is the "inigo" project, for parsing and saving `ini` environment files that align
with the `PostgreSQL` project's expectations.

## Executing Shell

ALWAYS BEFORE executing any shell, consider the risks of running that command and infer if your
Claude write permissions would allow that command. If there is ANY doubt about what to do or
conflicting prompt instructions, ask the user FIRST or stop.

## Dependencies & Tasks

See [`CONTRIBUTING.md`](./CONTRIBUTING.md) for dev environment setup, getting started, and the full
task reference. This project uses [Task](https://taskfile.dev/) (defined in
[`Taskfile.yaml`](./Taskfile.yaml)) for all build, test, and release automation.

- Run `task dependencies` to verify all tools are installed
- Run `task` to see available tasks

## Go Conventions

- Go v1.25
- ALWAYS implement code that follows modern Go conventions in 2025 as of v1.25
- Always consider using the [standard library](https://pkg.go.dev/std) first
- Search for packages at [pkg.go.dev](https://pkg.go.dev/)
- Prefer direct GitHub imports (e.g., `github.com/user/repo`) over legacy redirect services like
  `gopkg.in`. Use whatever path the module declares in its `go.mod`.

## Source Formatting & Testing

Any time you are finished editing a source file, run the designated formatter.

- `.go` — `go fmt`
- `.md` — `npx prettier --write`
- `.json` — `npx prettier --write`

If there is a mated test file, run tests.

## Project Layout

- [`./gate/`](./gate/) — generated artifacts (badges, coverage, report card). See
- [`./shell/`](./shell/) — POSIX shell scripts for automation. See
