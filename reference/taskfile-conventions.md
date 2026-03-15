# Taskfile Conventions Guide

A set of conventions for writing maintainable, reusable [Taskfiles](https://taskfile.dev/)
(go-task), derived from real-world usage.

---

## 1. Organize tasks with section headers

Use comment banners to visually group tasks into logical categories. This makes the file scannable
even at several hundred lines.

```yaml
# ---------------------------------------------------------------------------
# Test
# ---------------------------------------------------------------------------

test:
    desc: Run all tests
    cmds:
        - go test -v ./...
```

## 2. Guard external tools with dependency checks

Create an `internal: true` task for every external tool the project needs. Each one should:

- Use a `precondition` with `command -v` to verify the tool exists.
- Provide a `msg` with a human-readable install URL on failure.
- Echo a short description of what the tool does.

```yaml
dependencies:go:
    internal: true
    silent: true
    preconditions:
        - sh: command -v go >/dev/null 2>&1
          msg: 'go not found. Install Go: https://go.dev/dl/'
    cmds:
        - echo "go — Go compiler and toolchain (build, test, vet, fmt)"
```

Add a top-level `dependencies:` task that fans out to all individual checks so contributors can run
a single command to verify their environment.

```yaml
dependencies:
    desc: Check that all tool dependencies are installed
    deps:
        - dependencies:go
        - dependencies:curl
        - dependencies:npx
    cmds:
        - echo "all dependencies satisfied"
```

## 3. Declare deps on every task that needs them

Every task that requires an external tool should list it in `deps:`. This makes each task
self-contained — you can run any task in isolation and get a clear error if a tool is missing.

```yaml
format:
    desc: Format all source files
    deps: [dependencies:go, dependencies:npx]
    cmds:
        - go fmt ./...
        - npx prettier --write "**/*.md"
```

## 4. Centralize paths as top-level vars

Declare output directories, script directories, and key file paths once in `vars:` and reference
them everywhere with `{{.VAR}}`. This prevents path duplication and makes relocation trivial.

```yaml
vars:
    GATE_DIR: ./gate
    SHELL_DIR: ./shell
    README: ./README.md
```

## 5. Separate mutating tasks from read-only checks

For every action that writes files, provide a corresponding read-only check that only verifies
correctness (exits non-zero on failure). This lets CI run checks without side effects.

| Mutating        | Read-only        |
| --------------- | ---------------- |
| `format`        | `check:format`   |
| `test:coverage` | `check:coverage` |
| `gate:readme`   | `check:repo`     |

```yaml
format:
    desc: Format all source files
    cmds:
        - go fmt ./...

check:format:
    desc: Formatting check (read-only)
    cmds:
        - |
            unformatted=$(gofmt -l .)
            if [ -n "$unformatted" ]; then
              echo "error: unformatted files:" >&2
              echo "$unformatted" >&2
              exit 1
            fi
```

## 6. Use hierarchical colon-separated naming

Name tasks as `namespace:subtask` to create natural groupings. This produces clean output from
`task --list` and communicates scope at a glance.

```
test
test:coverage
check
check:lint
check:format
check:coverage
check:repo
```

## 7. Compose a gate for release readiness

Chain sub-tasks in sequence to enforce a full quality bar before release. Run repo-cleanliness
checks both before and after artifact generation to catch anything the gate itself produces.

```yaml
gate:check:
    desc: Run full release gate checks
    internal: true
    cmds:
        - task: clean
        - task: check
        - task: check:repo
        - task: check:coverage
        - task: gate:readme
        - task: check:repo
```

## 8. Send generated artifacts to a dedicated directory

All generated files (coverage profiles, reports, badges) should land in a single directory (e.g.
`./gate/`). This makes them easy to `.gitignore`, clean, and reference from other tasks.

```yaml
test:coverage:
    desc: Run tests with coverage
    cmds:
        - mkdir -p {{.GATE_DIR}}
        - go test -coverprofile={{.GATE_DIR}}/coverage.profile ./...
```

## 9. Keep complex logic in standalone shell scripts

When a task body grows beyond a few lines, move the logic into a dedicated script directory (e.g.
`./shell/`). This keeps task definitions declarative and makes the scripts independently testable
and reusable.

```yaml
cmds:
    - sh {{.SHELL_DIR}}/covpct.sh -s {{.GATE_DIR}}/coverage.report.txt -o
      {{.GATE_DIR}}/coverage.pct.txt
```

## 10. Make the default task informative

Running `task` with no arguments should list available tasks rather than doing something surprising
or destructive.

```yaml
default:
    desc: List available tasks
    silent: true
    cmds:
        - echo "Run 'task --list' to see all available tasks."
        - task --list
```

## 11. Clean only known artifacts

Target specific file patterns rather than nuking entire directory trees. Only `rm -rf` directories
that are entirely generated (like `build/`).

```yaml
clean:
    desc: Remove generated artifacts
    cmds:
        - rm -f {{.GATE_DIR}}/*.txt
        - rm -f {{.GATE_DIR}}/*.profile
        - rm -f {{.GATE_DIR}}/*.svg
        - rm -rf build/
```

## 12. Give every user-facing task a description

All public tasks must include `desc:` so `task --list` produces useful output. Internal helper tasks
should use `internal: true` to hide from the listing.
