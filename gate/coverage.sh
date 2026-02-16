#!/bin/sh
#
# coverage.sh - Run Go tests with coverage analysis
#
# Runs go tests and coverage analysis for a Go repository. Outputs the
# percentage of statements covered (without percent sign) to stdout as a
# single line. Emits the full coverage report to stderr.
#
# No artifacts are left behind in the repo folder.
#

set +x

# Resolve script directory and load getoptions library.
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
eval "$(sh "${SCRIPT_DIR}/getoptions/getoptions.sh" -)"

parser_definition() {
  setup   REST help:usage -- \
    "Usage: $0 [flags]" \
    '' \
    'Run Go tests with coverage analysis. Outputs the coverage percentage' \
    'to stdout (without the percent sign). The full coverage report and' \
    'test output are emitted to stderr.' \
    '' \
    'No artifacts are left behind in the repo folder.'
  msg -- '' 'flags:'
  param   REPO   -r --repo init:='.' -- "path to the Go repository"
  disp    :usage -h --help
}

eval "$(getoptions parser_definition) exit 1"

main() {
  repo="${REPO}"

  # Resolve to absolute path.
  repo="$(cd "${repo}" 2>/dev/null && pwd)" || {
    echo "error: directory '${REPO}' does not exist" >&2
    return 1
  }

  # Validate it looks like a Go module.
  if [ ! -f "${repo}/go.mod" ]; then
    echo "error: no go.mod found in '${repo}'" >&2
    return 1
  fi

  # Use a temp file outside the repo for the coverage profile.
  coverprofile="$(mktemp)"
  trap 'rm -f "${coverprofile}"' EXIT

  # Run tests with coverage, redirecting test output to stderr.
  go test -C "${repo}" -coverprofile="${coverprofile}" ./... >&2
  if [ ! -s "${coverprofile}" ]; then
    echo "error: go test failed or produced no coverage data" >&2
    return 1
  fi

  # Generate the full coverage report to stderr.
  report="$(go tool cover -func="${coverprofile}")"
  echo "${report}" >&2

  # Extract the total coverage percentage.
  total_line="$(echo "${report}" | grep '^total:')"
  if [ -z "${total_line}" ]; then
    echo "error: could not find total coverage in report" >&2
    return 1
  fi

  # Pull the percentage value and strip the percent sign.
  pct="$(echo "${total_line}" | awk '{print $NF}' | tr -d '%')"
  echo "${pct}"
}

main "$@"
