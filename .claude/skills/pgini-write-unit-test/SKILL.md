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


## Instructions

1. List all `./**/*_test.go` files to understand what tests there are already.
2. Unit test files will match file name with what they test, e.g. `foo_bar_test.go` -> `foo_bar.go`
    - Test files with no match are either integration tests or test-related libraries
3. Read the [specification](../../../reference/pgini-agents.md) to understand intended behavior.
4. Read the existing test file (if any) to understand current coverage and patterns.
5. Write tests that:
    - Cover every public and private function/method (100% function coverage).
    - Cover all reasonably testable code branches, error paths, and edge cases
    - Follow the Testing Laws above: derive assertions from the specification, probable end-user expectation, not from current code behavior.
    - Use table-driven tests where appropriate.
    - Use `testing.T` helpers (`t.Run`, `t.Helper`, `t.Fatalf`, `t.Errorf`).
6. Run `go fmt` on the test file.
7. Run `task test:coverage` and verify coverage color is green.
8. Run `cat ./gate/coverage.report.txt` to check per-function coverage. Address any gaps.
