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
