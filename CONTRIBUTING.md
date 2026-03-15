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
- Within a `.go` file, keep a struct and its code together and ordered:
    - struct definition
        1.  struct's constructor(s)
        2.  struct's method(s)
        3.  methods implementing built-in interfaces: `error`, `fmt.Stringer`, `fmt.GoStringer`,
            `sort.Interface`, `io.Reader` / `io.Writer`, marshalers / unmarshalers, `io.Closer`
- Always nil-check pointer and interface fields before dereferencing, even if current constructors
  initialize them. The type system is the contract.
