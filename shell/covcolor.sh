#!/bin/sh
# covcolor.sh — Determine a color grade from a coverage percentage.
#
# Reads a file containing a coverage percentage number and outputs
# a color name (green, orange, or red) based on configurable thresholds.
#
# Usage:
#   ./covcolor.sh [flags]
#
# Example:
#   ./covcolor.sh -s ./gate/coverage.pct.txt -o ./gate/coverage.color.txt

set +x

# Resolve the directory this script lives in.
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# Load getoptions library functions.
eval "$(sh "${SCRIPT_DIR}/getoptions/getoptions.sh" -)"

parser_definition() {
  setup REST help:usage -- \
    "Usage: $0 [flags]" \
    '' \
    'Determine a color grade from a coverage percentage.' \
    'Reads a percentage number from a file and outputs green, orange, or red.' \
    '' \
    'Thresholds (integer comparison):' \
    '  >= green threshold  -> green' \
    '  >= orange threshold -> orange' \
    '  < orange threshold  -> red' \
    '' \
    'Example:' \
    "  $0 -s ./gate/coverage.pct.txt -o ./gate/coverage.color.txt"
  msg -- '' 'Options:'
  param SOURCE -s --source -- "input file containing a percentage number (required)"
  param OUTPUT -o --output -- "output file for the color (default: stdout)"
  param GREEN     --green  init:="80" -- "threshold for green (default: 80)"
  param ORANGE    --orange init:="60" -- "threshold for orange (default: 60)"
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

if [ -z "${SOURCE}" ]; then
  echo "error: --source is required" >&2
  _err=1
fi

if [ "${_err}" -ne 0 ]; then
  echo "" >&2
  eval "$(getoptions_help parser_definition usage "$0")"
  usage >&2
  exit 1
fi

if [ ! -f "${SOURCE}" ]; then
  echo "error: source file not found: ${SOURCE}" >&2
  exit 1
fi

# --- Read percentage ---

_pct=$(cat "${SOURCE}" | tr -d '[:space:]')

if [ -z "${_pct}" ]; then
  echo "error: source file is empty: ${SOURCE}" >&2
  exit 1
fi

# Integer part for comparison.
_pct_int=$(echo "${_pct}" | cut -d. -f1)

# --- Determine color ---

if [ "${_pct_int}" -ge "${GREEN}" ]; then
  _color="green"
elif [ "${_pct_int}" -ge "${ORANGE}" ]; then
  _color="orange"
else
  _color="red"
fi

# --- Output ---

if [ -n "${OUTPUT}" ]; then
  echo "${_color}" > "${OUTPUT}" || {
    echo "error: failed to write to output file: ${OUTPUT}" >&2
    exit 1
  }
  echo "coverage color: ${_color}" >&2
else
  echo "${_color}"
fi

exit 0
