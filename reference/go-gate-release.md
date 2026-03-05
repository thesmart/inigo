# Go Project Gate & Release Setup

Generic guide for adding quality-gate and release automation to any Go project using
[Task](https://taskfile.dev/) and POSIX shell scripts.

## Concepts

- **Gate**: sequential checks that must all pass before a release is allowed. Ensures code quality,
  test coverage, formatting, and a clean working tree.
- **Release**: tag a semver version, push, index on pkg.go.dev, cross-compile binaries, publish a
  GitHub release.

## Project Layout

```
.
├── Taskfile.yaml          # all build/test/gate/release tasks
├── gate/                  # generated artifacts (coverage, report card, badges)
│   └── .gitignore         # ignore ephemeral files (*.profile), track text outputs
├── shell/                 # POSIX sh scripts for automation
│   ├── getoptions/        # vendored getoptions parser (CC0 license)
│   │   └── getoptions.sh
│   ├── covpct.sh          # extract coverage % from go tool cover report
│   ├── covcolor.sh        # map coverage % → color (green/orange/red)
│   ├── rcgrade.sh         # extract grade from goreportcard-cli output
│   ├── rccolor.sh         # map grade → color (green/orange/red)
│   ├── badge.sh           # download one shields.io badge SVG
│   ├── badges.sh          # download all project badges (batch)
│   ├── badgesmd.sh        # generate markdown snippet with badge images
│   ├── mdreplace.sh       # replace tagged sections in a markdown file
│   ├── release.sh         # full release: validate, tag, push, build, gh release
│   └── ghrelease.sh       # create GitHub release for an existing tag
├── go.mod
└── README.md              # contains <!-- BADGES:START/END --> markers
```

## Dependencies

| Tool               | Purpose                         | Install                                                                            |
| ------------------ | ------------------------------- | ---------------------------------------------------------------------------------- |
| `go`               | compile, test, vet, fmt         | https://go.dev/dl/                                                                 |
| `task`             | task runner                     | `go install github.com/go-task/task/v3/cmd/task@latest` or `go.mod` tool directive |
| `goreportcard-cli` | code quality grading            | `go.mod` tool directive                                                            |
| `curl`             | download badge SVGs             | usually pre-installed                                                              |
| `npx` (Node.js)    | prettier for md/json formatting | https://nodejs.org/                                                                |
| `typos`            | spell checking                  | https://crates.io/crates/typos-cli                                                 |
| `git`              | version control                 | https://git-scm.com/                                                               |
| `gh`               | GitHub CLI (releases, PRs)      | https://cli.github.com                                                             |

Use Go 1.21+ `tool` directive in `go.mod` to vendor `task` and `goreportcard-cli`:

```
tool (
    github.com/go-task/task/v3/cmd/task
    github.com/gojp/goreportcard/cmd/goreportcard-cli
)
```

## Taskfile.yaml Structure

### Dependency Checks (internal tasks)

Each tool gets an internal precondition task:

```yaml
dependencies:go:
    internal: true
    silent: true
    preconditions:
        - sh: command -v go >/dev/null 2>&1
          msg: 'go not found. Install Go: https://go.dev/dl/'
    cmds:
        - echo "go — Go compiler and toolchain"

dependencies:
    desc: Check that all tool dependencies are installed
    deps:
        [
            dependencies:go,
            dependencies:git,
            dependencies:gh,
            dependencies:curl,
            dependencies:npx,
            dependencies:typos,
            dependencies:goreportcard,
        ]
    cmds:
        - echo "all dependencies satisfied"
```

### Core Tasks

```yaml
vars:
    GATE_DIR: ./gate
    SHELL_DIR: ./shell
    README: ./README.md
    LICENSE_ID: MIT # or Apache-2.0, etc.

tasks:
    tidy:
        desc: Run go mod tidy
        deps: [dependencies:go]
        cmds: [go mod tidy]

    test:
        desc: Run all Go tests
        deps: [dependencies:go]
        cmds: [go test -v ./...]

    test:coverage:
        desc: Run tests with coverage and extract metrics
        deps: [dependencies:go]
        cmds:
            - mkdir -p {{.GATE_DIR}}
            - go test -coverprofile={{.GATE_DIR}}/coverage.profile ./...
            - go tool cover -func={{.GATE_DIR}}/coverage.profile > {{.GATE_DIR}}/coverage.report.txt
            - sh {{.SHELL_DIR}}/covpct.sh -s {{.GATE_DIR}}/coverage.report.txt -o
              {{.GATE_DIR}}/coverage.pct.txt
            - sh {{.SHELL_DIR}}/covcolor.sh -s {{.GATE_DIR}}/coverage.pct.txt -o
              {{.GATE_DIR}}/coverage.color.txt

    goreportcard:
        desc: Run goreportcard-cli and extract grade
        deps: [dependencies:goreportcard]
        cmds:
            - mkdir -p {{.GATE_DIR}}
            - go tool goreportcard-cli > {{.GATE_DIR}}/goreportcard.report.txt 2>&1 || true
            - sh {{.SHELL_DIR}}/rcgrade.sh -s {{.GATE_DIR}}/goreportcard.report.txt -o
              {{.GATE_DIR}}/goreportcard.grade.txt
            - sh {{.SHELL_DIR}}/rccolor.sh -s {{.GATE_DIR}}/goreportcard.grade.txt -o
              {{.GATE_DIR}}/goreportcard.color.txt

    check:
        desc: Lint, vet, and format check (read-only)
        deps: [dependencies:go, dependencies:npx]
        cmds:
            - go vet ./...
            - |
                unformatted=$(gofmt -l .)
                if [ -n "$unformatted" ]; then
                  echo "error: unformatted Go files:" >&2
                  echo "$unformatted" >&2
                  exit 1
                fi
            - npx prettier --check "**/*.md" "**/*.json" --ignore-path .gitignore

    format:
        desc: Format all source files
        deps: [dependencies:go, dependencies:npx, dependencies:typos]
        cmds:
            - go fmt ./...
            - npx prettier --write "**/*.md" "**/*.json" --ignore-path .gitignore
            - typos
```

### Badge Generation

The `readme:badges` task chains coverage + report card, downloads SVGs, generates a markdown
snippet, and replaces a tagged section in the README:

```yaml
readme:badges:
    desc: Download badges and update README
    deps: [test:coverage, goreportcard, dependencies:curl, dependencies:npx]
    vars:
        COV_PCT:
            sh: cat {{.GATE_DIR}}/coverage.pct.txt | cut -d. -f1
        COV_COLOR:
            sh: cat {{.GATE_DIR}}/coverage.color.txt
        GRADE:
            sh: cat {{.GATE_DIR}}/goreportcard.grade.txt
        GRADE_COLOR:
            sh: cat {{.GATE_DIR}}/goreportcard.color.txt
    cmds:
        - mkdir -p {{.GATE_DIR}}
        # Download individual badges via badge.sh
        - sh {{.SHELL_DIR}}/badge.sh -o {{.GATE_DIR}}/go-reference-badge.svg -m "Reference" -k
          "91DFFB" --logo go --logo-color white --label-color white
        - sh {{.SHELL_DIR}}/badge.sh -o {{.GATE_DIR}}/coverage-badge.svg -l "coverage" -m
          "{{.COV_PCT}}%25" -k "{{.COV_COLOR}}"
        - sh {{.SHELL_DIR}}/badge.sh -o {{.GATE_DIR}}/go-report-badge.svg -l "go report" -m
          "{{.GRADE}}" -k "{{.GRADE_COLOR}}"
        - sh {{.SHELL_DIR}}/badge.sh -o {{.GATE_DIR}}/license-badge.svg -l "license" -m
          "{{.LICENSE_ID}}" -k "blue"
        # Generate markdown and replace in README
        - |
            badges_md=$(mktemp)
            trap 'rm -f "$badges_md"' EXIT
            sh {{.SHELL_DIR}}/badgesmd.sh -d {{.GATE_DIR}} -p "{{.COV_PCT}}" -g "{{.GRADE}}" -l "{{.LICENSE_ID}}" -o "$badges_md"
            sh {{.SHELL_DIR}}/mdreplace.sh -t badges -s {{.README}} -c "$badges_md" -o {{.README}}
            rm -f "$badges_md"
        - npx prettier --write {{.README}}
```

README must contain markers:

```markdown
<!-- BADGES:START -->

(badge content is auto-replaced here)

<!-- BADGES:END -->
```

### Gate Task

Runs the full pipeline sequentially, then enforces quality thresholds and a clean working tree:

```yaml
gate:
    desc: Run full release gate
    cmds:
        - task: tidy
        - task: format
        - task: check
        - task: readme:badges
        - |
            color=$(cat {{.GATE_DIR}}/coverage.color.txt)
            pct=$(cat {{.GATE_DIR}}/coverage.pct.txt)
            if [ "$color" != "green" ]; then
              echo "error: coverage ${pct}% is not green (got ${color})" >&2
              exit 1
            fi
        - |
            grade=$(cat {{.GATE_DIR}}/goreportcard.grade.txt | tr -d '[:space:]')
            case "$grade" in
              A+|A|A-) ;;
              *) echo "error: go report card grade ${grade} is below A" >&2; exit 1 ;;
            esac
        - |
            dirty=$(git status --porcelain)
            if [ -n "$dirty" ]; then
              echo "error: working tree is dirty after gating" >&2
              echo "$dirty" >&2
              exit 1
            fi
            echo "gate: clean — ready to tag a release"
```

### Release Task

```yaml
release:
    desc: Tag and publish a release (requires VERSION, e.g. task release VERSION=v1.0.0)
    deps: [dependencies:go, dependencies:git, dependencies:gh, dependencies:curl]
    cmds:
        - sh {{.SHELL_DIR}}/release.sh {{.VERSION}}

release:dry-run:
    desc: Dry-run release
    cmds:
        - sh {{.SHELL_DIR}}/release.sh --dry-run {{.VERSION}}
```

### Build Tasks

```yaml
build:
    desc: Build the CLI binary
    deps: [dependencies:go]
    cmds:
        - go build -o build/myapp ./cmd/myapp

build:all:
    desc: Cross-compile for all POSIX platforms
    deps: [dependencies:go]
    vars:
        PLATFORMS: linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 freebsd/amd64
    cmds:
        - |
            for platform in {{.PLATFORMS}}; do
              os=$(echo "$platform" | cut -d/ -f1)
              arch=$(echo "$platform" | cut -d/ -f2)
              output="build/myapp-${os}-${arch}"
              GOOS=$os GOARCH=$arch go build -o "$output" ./cmd/myapp || exit 1
            done
```

## Shell Script Conventions

All scripts in `shell/` follow these rules:

- **POSIX `sh`** — no bash/zsh-isms
- **Idempotent** — safe to run multiple times
- **`getoptions`** for argument parsing (vendored in `shell/getoptions/`)
- **`--help` checked first** before validation, so help always works
- **Exit 0 on success**, non-zero on failure
- **`SCRIPT_DIR` resolution** at top of every script: `SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"`
- **Path safety** — reject `..` in output paths, validate inputs exist
- **`set +x`** at top (disable trace)

### Script Template

```sh
#!/bin/sh
# myscript.sh — One-line description.

set +x

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
eval "$(sh "${SCRIPT_DIR}/getoptions/getoptions.sh" -)"

parser_definition() {
  setup REST help:usage -- \
    "Usage: $0 [flags]" '' 'Description.'
  msg -- '' 'Options:'
  param OUTPUT -o --output -- "output file path"
  param SOURCE -s --source -- "input file path"
  disp :usage -h --help
}

# Check --help first
for _arg in "$@"; do
  case "${_arg}" in
    -h|--help) eval "$(getoptions_help parser_definition usage "$0")"; usage; exit 0 ;;
  esac
done

eval "$(getoptions parser_definition parse "$0")"
parse "$@"
eval "set -- $REST"

# --- Validation ---
_err=0
[ -z "${OUTPUT}" ] && echo "error: --output is required" >&2 && _err=1
[ -z "${SOURCE}" ] && echo "error: --source is required" >&2 && _err=1
if [ "${_err}" -ne 0 ]; then
  eval "$(getoptions_help parser_definition usage "$0")"; usage >&2; exit 1
fi
[ ! -f "${SOURCE}" ] && echo "error: source not found: ${SOURCE}" >&2 && exit 1

# --- Main logic ---
# ...

exit 0
```

### Coverage Pipeline Scripts

| Script        | Input                        | Output                 | Purpose                               |
| ------------- | ---------------------------- | ---------------------- | ------------------------------------- |
| `covpct.sh`   | `go tool cover -func` report | percentage (e.g. 98.5) | Extract `total:` line number          |
| `covcolor.sh` | percentage file              | color name             | `>=80` green, `>=60` orange, else red |
| `rcgrade.sh`  | `goreportcard-cli` output    | grade (e.g. A+)        | Extract `Grade` line                  |
| `rccolor.sh`  | grade file                   | color name             | A-tier green, B-tier orange, else red |

### Badge Scripts

| Script         | Purpose                                                                      |
| -------------- | ---------------------------------------------------------------------------- |
| `badge.sh`     | Download one shields.io SVG with label/message/color params                  |
| `badges.sh`    | Batch download all project badges                                            |
| `badgesmd.sh`  | Generate markdown snippet with badge image refs                              |
| `mdreplace.sh` | Replace `<!-- TAG:START -->...<!-- TAG:END -->` section in any markdown file |

### Release Script

`release.sh` performs these steps in order:

1. Validate version format (`vMAJOR.MINOR.PATCH`)
2. Verify working tree is clean (`git status --porcelain`)
3. Verify on `main` branch
4. Verify tag does not exist
5. Verify `gh auth status`
6. `git push origin main`
7. `git tag VERSION && git push origin VERSION`
8. Trigger pkg.go.dev indexing via `curl https://proxy.golang.org/MODULE/@v/VERSION.info`
9. Cross-compile binaries (`task build:all`)
10. `gh release create VERSION build/binaries --generate-notes`

Supports `--dry-run` flag to print actions without executing.

`ghrelease.sh` is a companion for creating a GitHub release from an existing tag (skips steps 1-8).

## Gate Directory

`gate/.gitignore` should track text outputs but ignore ephemeral binary files:

```gitignore
# Generated gate artifacts
coverage.profile
```

Tracked files (committed to repo):

- `coverage.report.txt` — full `go tool cover -func` output
- `coverage.pct.txt` — single number (e.g. `98.5`)
- `coverage.color.txt` — single word (`green`, `orange`, `red`)
- `goreportcard.report.txt` — full report card output
- `goreportcard.grade.txt` — single grade (e.g. `A+`)
- `goreportcard.color.txt` — single word
- `*.svg` — downloaded badge images

## Setup Checklist

1. **Create directories**: `mkdir -p gate shell shell/getoptions`
2. **Vendor getoptions**: copy `getoptions.sh` into `shell/getoptions/`
3. **Copy shell scripts** into `shell/` and adapt `badgesmd.sh` links to your repo
4. **Create `gate/.gitignore`** to exclude `coverage.profile`
5. **Add `<!-- BADGES:START -->` / `<!-- BADGES:END -->`** markers to README
6. **Create `Taskfile.yaml`** with tasks above, updating vars for your project
7. **Add `tool` directives** to `go.mod` for `task` and `goreportcard-cli`
8. **Run `task dependencies`** to verify setup
9. **Run `task gate`** to verify full pipeline
10. **Run `task release:dry-run VERSION=v0.1.0`** to verify release flow

## Workflow Summary

```
task tidy            → sync go.mod
task format          → auto-format all files
task check           → read-only lint/vet/format check
task test            → run tests
task test:coverage   → tests + extract coverage metrics
task goreportcard    → code quality grade
task readme:badges   → generate + embed badges in README
task gate            → full pipeline: tidy → format → check → badges → thresholds → clean tree
task release VERSION=v1.0.0   → tag + push + build + gh release
task clean           → remove generated artifacts
```
