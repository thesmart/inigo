# CLAUDE.md

- Read [`README.md`](./README.md) for project specification.
    - DO NOT read [`pgini-specification.md`](./reference/pgini-specification.md) but instead read
      [`pgini-agents.md`](./reference/pgini-agents.md).
- Read [`CONTRIBUTING.md`](./CONTRIBUTING.md) for development setup and SDLC.

## Executing Shell

ALWAYS BEFORE executing any shell, consider the risks of running that command and infer if your
Claude write permissions would allow that command. If there is ANY doubt about what to do or
conflicting prompt instructions, ask the user FIRST or stop.

## Source Formatting & Testing

Any time you are finished editing a source file, run the designated formatter.

- `.go` — `go fmt`
- `.md` — `npx prettier --write`
- `.json` — `npx prettier --write`

If there is a mated test file, run tests.

## Testing Laws

- Tests encode the specification. Code implementations fulfill them.
- Tests are a double-check on correctness, not a mirror of what the code already does.
- Correctness means the software does what its stakeholders intended.
- Before writing a new test , **ALWAYS**:
    1. read the specification
    2. explain how the test should assert that program state matches specification
    3. align code to pass the test
    4. consider again, and confirm that a. code aligns with tests b. tests align with specification
- Before changing a test, **ALWAYS** use outside-in reasoning:
    1. read the specification
    2. determine how the test should assert that state (actual) matches specification (expected)
    3. explain why the test is wrong **without** referencing what the code currently does
    4. ask the user to confirm why the test is not conforming with the specification
