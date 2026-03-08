# Contributing

## Dev Environment

- Go 1.25+
- [Task](https://taskfile.dev/) (task runner, installed via `go tool` directive)

## Getting Started

```sh
git clone git@github.com:thesmart/inigo.git
cd inigo
# validate you have all dev time dependencies
task dependencies
task build
./build/inigo --help
```

## Tasks

Everything you need to run as a developer is provided by `task` in the `Taskfile.yaml`.

```sh
task --list
```

If you need to modify or understanding the Task system, read
[`reference/taskfile-conventions.md`](./reference/taskfile-conventions.md).

## Go Conventions

- Always implement code that follows modern Go conventions in 2025 as of v1.25
- Always consider using the [standard library](https://pkg.go.dev/std) first
- Consider popular, well tested packages w/ low dependencies via [pkg.go.dev](https://pkg.go.dev/)
- Prefer direct GitHub imports (e.g., `github.com/user/repo`) over legacy redirect services like
  `gopkg.in`. Use whatever path the module declares in its `go.mod`.
