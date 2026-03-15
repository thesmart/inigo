#!/bin/sh
# rccolor.sh — Determine a color from a report card grade letter.
#
# Reads a file containing a grade letter (e.g. A+, B, C-) and outputs
# a color name (green, orange, or red).
#
# Usage:
#   ./rccolor.sh [flags]
#
# Example:
#   ./rccolor.sh -s ./gate/goreportcard.grade.txt -o ./gate/goreportcard.color.txt

set +x

# Resolve the directory this script lives in.
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# Load getoptions library functions.
eval "$(sh "${SCRIPT_DIR}/getoptions/getoptions.sh" -)"

parser_definition() {
  setup REST help:usage -- \
    "Usage: $0 [flags]" \
    '' \
    'Determine a color from a Go Report Card grade letter.' \
    '' \
    'Grade to color mapping:' \
    '  A+, A, A-  -> green' \
    '  B+, B, B-  -> orange' \
    '  all others -> red' \
    '' \
    'Example:' \
    "  $0 -s ./gate/goreportcard.grade.txt -o ./gate/goreportcard.color.txt"
  msg -- '' 'Options:'
  param SOURCE -s --source -- "input file containing a grade letter (required)"
  param OUTPUT -o --output -- "output file for the color (default: stdout)"
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

# --- Read grade ---

_grade=$(cat "${SOURCE}" | tr -d '[:space:]')

if [ -z "${_grade}" ]; then
  echo "error: source file is empty: ${SOURCE}" >&2
  exit 1
fi

# --- Determine color ---

case "${_grade}" in
  A+|A|A-) _color="green" ;;
  B+|B|B-) _color="orange" ;;
  *)       _color="red" ;;
esac

# --- Output ---

if [ -n "${OUTPUT}" ]; then
  echo "${_color}" > "${OUTPUT}" || {
    echo "error: failed to write to output file: ${OUTPUT}" >&2
    exit 1
  }
  echo "goreportcard color: ${_color}" >&2
else
  echo "${_color}"
fi

exit 0
