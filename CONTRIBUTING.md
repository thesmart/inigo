# Contributing

## Dev Environment

- Go 1.24+
- [Deno](https://deno.land/) (for markdown/JSON formatting)

```sh
git clone git@github.com:thesmart/inigo.git
cd inigo
make check
```

## Make Targets

| Command                                 | Description                                         |
| --------------------------------------- | --------------------------------------------------- |
| `make check`                            | Run `go fmt`, `go vet`, and tests                   |
| `make gate`                             | Run checks + coverage + badges + README update      |
| `make gate DRY_RUN=1`                   | Dry-run gate (temp dir artifacts, README untouched) |
| `make release VERSION=v0.1.0`           | Run gate + tag + push + pkg.go.dev indexing         |
| `make release VERSION=v0.1.0 DRY_RUN=1` | Dry-run release (prints actions only)               |
| `make clean`                            | Remove generated badge SVGs                         |
