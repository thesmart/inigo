#!/bin/sh
# badges.sh — Download static badge SVGs from shields.io.
#
# Downloads go-reference, coverage, go-report, and license badges
# as SVG files to the specified output directory.
#
# Usage:
#   ./badges.sh [flags]
#
# Example:
#   ./badges.sh -o ./badges -p 85 -c green -g A+ -r green -l MIT

set +x

# Resolve the directory this script lives in.
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# Load getoptions library functions.
eval "$(sh "${SCRIPT_DIR}/getoptions/getoptions.sh" -)"

parser_definition() {
  setup REST help:usage -- \
    "Usage: $0 [flags]" \
    '' \
    'Download static badge SVGs from shields.io to a local directory.' \
    'Fetches four badges: go-reference, code coverage, go report, and license.' \
    'The go-reference badge is fixed and always downloaded.' \
    'All other badge parameters are required.' \
    '' \
    'Output files:' \
    '  go-reference-badge.svg    Go package reference badge' \
    '  coverage-badge.svg        Code coverage percentage badge' \
    '  go-report-badge.svg       Go report card grade badge' \
    '  license-badge.svg         Project license badge' \
    '' \
    'Example:' \
    "  $0 -o ./badges -p 85 -c green -g A+ -r green -l MIT"
  msg -- '' 'Options:'
  param OUTPUT         -o --output           -- "output directory for badge SVG files"
  param COVERAGE_PCT   -p --coverage-percent -- "coverage percentage without % sign (e.g. 85)"
  param COVERAGE_COLOR -c --coverage-color   pattern:"red|orange|green" -- "coverage badge color: red, orange, or green"
  param REPORT_GRADE   -g --report-grade     pattern:"A+|A|A-|B+|B|B-|C+|C|C-|D+|D|D-|F" -- "go report grade (e.g. A+, B, C-, F)"
  param REPORT_COLOR   -r --report-color     pattern:"red|orange|green" -- "go report badge color: red, orange, or green"
  param LICENSE        -l --license          -- "license identifier (e.g. MIT, Apache-2.0)"
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

if [ -z "${OUTPUT}" ]; then
  echo "error: --output is required" >&2
  _err=1
fi
if [ -z "${COVERAGE_PCT}" ]; then
  echo "error: --coverage-percent is required" >&2
  _err=1
fi
if [ -z "${COVERAGE_COLOR}" ]; then
  echo "error: --coverage-color is required" >&2
  _err=1
fi
if [ -z "${REPORT_GRADE}" ]; then
  echo "error: --report-grade is required" >&2
  _err=1
fi
if [ -z "${REPORT_COLOR}" ]; then
  echo "error: --report-color is required" >&2
  _err=1
fi
if [ -z "${LICENSE}" ]; then
  echo "error: --license is required" >&2
  _err=1
fi

if [ "${_err}" -ne 0 ]; then
  echo "" >&2
  usage >&2
  exit 1
fi

case "${COVERAGE_PCT}" in
  *[!0-9]*)
    echo "error: --coverage-percent must be a non-negative integer (e.g. 85)" >&2
    exit 1
    ;;
esac

case "${OUTPUT}" in
  *..*)
    echo "error: --output path must not contain '..'" >&2
    exit 1
    ;;
esac

if ! command -v curl >/dev/null 2>&1; then
  echo "error: curl is required but not found in PATH" >&2
  exit 1
fi

# --- Setup output directory ---

if [ ! -d "${OUTPUT}" ]; then
  mkdir -p "${OUTPUT}" || {
    echo "error: failed to create output directory: ${OUTPUT}" >&2
    exit 1
  }
fi

# --- Helper functions ---

# URL-encode characters that appear in badge values.
url_encode() {
  echo "$1" | sed -e 's/+/%2B/g' -e 's/ /%20/g'
}

# Download a badge SVG from a URL.
#   $1: URL to download
#   $2: output file path
download_badge() {
  _url="$1"
  _dest="$2"
  curl -fsSL -o "${_dest}" "${_url}" || {
    echo "error: failed to download badge: ${_url}" >&2
    return 1
  }
}

# --- Download badges ---

_fail=0

_go_ref_url="https://img.shields.io/badge/Reference-white?logo=go&labelColor=white&color=91DFFB"
download_badge "${_go_ref_url}" "${OUTPUT}/go-reference-badge.svg" || _fail=1

_cov_url="https://img.shields.io/badge/coverage-${COVERAGE_PCT}%25-${COVERAGE_COLOR}"
download_badge "${_cov_url}" "${OUTPUT}/coverage-badge.svg" || _fail=1

_encoded_grade=$(url_encode "${REPORT_GRADE}")
_report_url="https://img.shields.io/badge/go%20report-${_encoded_grade}-${REPORT_COLOR}"
download_badge "${_report_url}" "${OUTPUT}/go-report-badge.svg" || _fail=1

_license_url="https://img.shields.io/badge/license-${LICENSE}-blue"
download_badge "${_license_url}" "${OUTPUT}/license-badge.svg" || _fail=1

if [ "${_fail}" -ne 0 ]; then
  echo "error: one or more badge downloads failed" >&2
  exit 1
fi

echo "badges downloaded to ${OUTPUT}"
exit 0
