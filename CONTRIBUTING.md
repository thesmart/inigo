# Contributing

## Dev Environment

- Go 1.25+
- [Task](https://taskfile.dev/) (task runner, installed via `go tool` directive)
- [Node.js](https://nodejs.org/) (for prettier via npx)
- Run `task dependencies` for all other dev time dependencies

## Getting Started

```sh
git clone git@github.com:thesmart/inigo.git
cd inigo
task build
./build/inigo --help
```

## Tasks

| Command                | Description                             |
| ---------------------- | --------------------------------------- |
| `task tidy`            | Run go mod tidy to sync dependencies    |
| `task test`            | Run all Go tests                        |
| `task test:coverage`   | Run tests with coverage analysis        |
| `task check`           | Lint, vet, and format check             |
| `task format`          | Format Go, markdown, and JSON files     |
| `task goreportcard`    | Run Go Report Card analysis             |
| `task readme:badges`   | Download badges and update README       |
| `task gate`            | Run full release gate and verify clean  |
| `task release`         | Tag and publish a release (VERSION=...) |
| `task release:dry-run` | Dry-run release (VERSION=...)           |
| `task build`           | Build the inigo CLI binary              |
| `task build:all`       | Cross-compile for all POSIX platforms   |
| `task clean`           | Remove generated artifacts              |
| `task dependencies`    | Check all tool dependencies             |
| `task help`            | Print usage help                        |
