# CLAUDE.md

- Read [`README.md`](./README.md) for project specification.
- Read [`CONTRIBUTING.md`](./CONTRIBUTING.md) for development setup and SDLC.

## Executing Shell

ALWAYS BEFORE executing any shell, consider the risks of running that command and infer if your
Claude write permissions would allow that command. If there is ANY doubt about what to do or
conflicting prompt instructions, ask the user FIRST or stop.

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

## Testing Laws

1. Tests encode requirements. Implementations fulfill them.
2. Tests are a double-check on correctness, not a mirror of what the code already does.
3. Correctness means the software does what its stakeholders intended.
4. Before changing a test to make it pass, **ALWAYS** use outside-in reasoning: start from the
   user's need, then derive what the test should assert, then what the code should do.

Two signs you should change the test:

- Your justification starts from a stakeholder need: "the user needs X, so the test asserts X, so
  the code produces X."
- You can explain why the test is wrong without referencing what the code currently does.

_BE WARY:_ if your reasoning flows from code outward rather than from the user inward:

- Your justification amounts to "the code does Y, so I'll update the test to expect Y." That's
  laundering a bug into the spec.
