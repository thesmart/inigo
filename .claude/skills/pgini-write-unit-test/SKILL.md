---
name: pgini-write-unit-test
description: "Activate when the user asks to write, create, or add tests for the pgini package. Triggers: write test, add test, test coverage, pgini test. Do NOT activate for running existing tests, fixing non-test code, or tests outside the pgini package."
argument-hint: '[test-description]'
license: MIT
compatibility: Designed for Claude Code (and compatible)
metadata:
    author: thesmart
    version: '1.0'
---

# pgini-write-test

Write tests for the `pgini` Go package targeting 100% function coverage (public and private) and at
least 95% line coverage.

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

## Existing Tests

| Test file                | Source under test   | Reference                                      |
| ------------------------ | ------------------- | ---------------------------------------------- |
| `pgini/ini_file_test.go` | `pgini/ini_file.go` | [ini_file_test](./references/ini_file_test.md) |
| `pgini/cursors_test.go`  | `pgini/cursors.go`  | [cursors_test](./references/cursors_test.md)   |

## Instructions

1. Read the reference file for the test target from the table above.
2. Read the source file under test to identify all public and private functions, methods, and code
   paths.
3. Read the [specification](../../../reference/pgini-agents.md) to understand intended behavior.
4. Read the existing test file (if any) to understand current coverage and patterns.
5. Write tests that:
    - Cover every public and private function/method (100% function coverage).
    - Exercise branches, error paths, and edge cases to reach at least 95% line coverage.
    - Follow the Testing Laws above — derive assertions from the specification, not from current
      code behavior.
    - Use table-driven tests where appropriate.
    - Use `testing.T` helpers (`t.Run`, `t.Helper`, `t.Fatalf`, `t.Errorf`).
6. Run `go fmt` on the test file.
7. Run `go test -v -coverprofile=cover.out ./pgini/` and verify coverage targets are met.
8. Run `go tool cover -func=cover.out` to check per-function coverage. Address any gaps.
