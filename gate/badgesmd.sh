#!/bin/sh
# badgesmd.sh — Generate a markdown snippet with badge images and links.
#
# Produces markdown lines referencing repo-relative badge SVGs from the
# badges directory. Alt-text includes the report grade and coverage
# percentage for accessibility and text-mode rendering.
#
# Output is written to --output or stdout.
#
# Usage:
#   ./badgesmd.sh [flags]
#
# Example:
#   ./badgesmd.sh -d badges -p 96 -g A+ -l MIT -o badges.md

set +x

# Resolve the directory this script lives in.
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# Load getoptions library functions.
eval "$(sh "${SCRIPT_DIR}/getoptions/getoptions.sh" -)"

parser_definition() {
  setup REST help:usage -- \
    "Usage: $0 [flags]" \
    '' \
    'Generate a markdown snippet with badge images and links.' \
    'Badges are referenced as repo-relative paths from the badges directory.' \
    'Alt-text includes report grade and coverage percentage.' \
    '' \
    'Example:' \
    "  $0 -d badges -p 96 -g A+ -l MIT -o badges.md"
  msg -- '' 'Options:'
  param BADGES_DIR     -d --badges-dir      -- "repo-relative path to badges directory (e.g. badges)"
  param COVERAGE_PCT   -p --coverage-percent -- "coverage percentage without % sign (e.g. 96)"
  param REPORT_GRADE   -g --report-grade     pattern:"A+|A|A-|B+|B|B-|C+|C|C-|D+|D|D-|F" -- "go report grade (e.g. A+, B, C-, F)"
  param LICENSE        -l --license          -- "license identifier (e.g. MIT, Apache-2.0)"
  param OUTPUT         -o --output           -- "output file path (default: stdout)"
  disp :usage -h --help
}

# Check for --help first, before any other parsing.
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

if [ -z "${BADGES_DIR}" ]; then
  echo "error: --badges-dir is required" >&2
  _err=1
fi
if [ -z "${COVERAGE_PCT}" ]; then
  echo "error: --coverage-percent is required" >&2
  _err=1
fi
if [ -z "${REPORT_GRADE}" ]; then
  echo "error: --report-grade is required" >&2
  _err=1
fi
if [ -z "${LICENSE}" ]; then
  echo "error: --license is required" >&2
  _err=1
fi

if [ "${_err}" -ne 0 ]; then
  echo "" >&2
  eval "$(getoptions_help parser_definition usage "$0")"
  usage >&2
  exit 1
fi

case "${COVERAGE_PCT}" in
  *[!0-9]*)
    echo "error: --coverage-percent must be a non-negative integer (e.g. 96)" >&2
    exit 1
    ;;
esac

# --- Generate markdown ---

_d="${BADGES_DIR}"
_p="${COVERAGE_PCT}"
_g="${REPORT_GRADE}"
_l="${LICENSE}"

_md="
[![Go Reference](${_d}/go-reference-badge.svg)](https://pkg.go.dev/github.com/thesmart/inigo)
[![Coverage ${_p}%](${_d}/coverage-badge.svg)](https://github.com/thesmart/inigo)
[![Go Report ${_g}](${_d}/go-report-badge.svg)](https://goreportcard.com/report/github.com/thesmart/inigo)
[![License ${_l}](${_d}/license-badge.svg)](./LICENSE)
"

# --- Output ---

if [ -n "${OUTPUT}" ]; then
  echo "${_md}" > "${OUTPUT}" || {
    echo "error: failed to write to output file: ${OUTPUT}" >&2
    exit 1
  }
  echo "badges markdown written to ${OUTPUT}" >&2
else
  echo "${_md}"
fi

exit 0
