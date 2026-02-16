#!/bin/sh
# mdreplace.sh — Replace tagged sections in a markdown file.
#
# Finds the first occurrence of a tag pair like:
#   <!-- TAG:START -->
#   ...
#   <!-- TAG:END -->
# and replaces the content between them with the contents of a file.
#
# Usage:
#   ./mdreplace.sh --tag badges --source README.md --content badges.md
#   ./mdreplace.sh --tag badges --source README.md --content badges.md --output out.md
#
# If --output is omitted, the result is printed to stdout.
# If deno is available, the markdown is formatted with deno fmt before output.

set +x

# Resolve the directory this script lives in.
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# Load getoptions library functions.
eval "$(sh "${SCRIPT_DIR}/getoptions/getoptions.sh" -)"

parser_definition() {
  setup REST help:usage -- \
    "Usage: $0 [flags]" \
    '' \
    'Replace a tagged section in a markdown file with content from another file.' \
    '' \
    'Tags are matched as HTML comments (case-insensitive):' \
    '  <!-- TAG:START -->' \
    '  ...' \
    '  <!-- TAG:END -->' \
    '' \
    'Only the first matching tag pair is replaced.' \
    '' \
    'If --output is omitted, the result is printed to stdout.' \
    'If deno is available, deno fmt is run on the result before output.' \
    '' \
    'Example:' \
    "  $0 --tag badges --source README.md --content badges.md" \
    "  $0 -t badges -s README.md -c badges.md -o out.md"
  msg -- '' 'Options:'
  param TAG     -t --tag     -- "tag name to match (case-insensitive)"
  param SOURCE  -s --source  -- "source markdown file containing the tag pair"
  param CONTENT -c --content -- "file whose contents replace the tagged section"
  param OUTPUT  -o --output  -- "output file path (default: stdout)"
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

if [ -z "${TAG}" ]; then
  echo "error: --tag is required" >&2
  _err=1
fi
if [ -z "${SOURCE}" ]; then
  echo "error: --source is required" >&2
  _err=1
fi
if [ -z "${CONTENT}" ]; then
  echo "error: --content is required" >&2
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

if [ ! -f "${CONTENT}" ]; then
  echo "error: content file not found: ${CONTENT}" >&2
  exit 1
fi

# --- Build case-insensitive tag markers ---

# Convert tag to uppercase for matching.
TAG_UPPER="$(echo "${TAG}" | tr '[:lower:]' '[:upper:]')"

START_MARKER="<!-- ${TAG_UPPER}:START -->"
END_MARKER="<!-- ${TAG_UPPER}:END -->"

# --- Process the file ---

# Read the source file.
source_text="$(cat "${SOURCE}")"

# Convert source to uppercase for case-insensitive searching.
source_upper="$(echo "${source_text}" | tr '[:lower:]' '[:upper:]')"

# Find line numbers of the first START and END markers.
start_line=""
end_line=""
_line_num=0

# Use a temp file to avoid subshell variable scoping issues.
_tmplines="$(mktemp)"
trap 'rm -f "${_tmplines}"' EXIT

echo "${source_upper}" | while IFS= read -r _line; do
  _line_num=$((_line_num + 1))
  # Strip leading/trailing whitespace for comparison.
  _trimmed="$(echo "${_line}" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')"
  if [ -z "${start_line}" ] && [ "${_trimmed}" = "${START_MARKER}" ]; then
    echo "start=${_line_num}" >> "${_tmplines}"
    start_line="${_line_num}"
  elif [ -n "${start_line}" ] && [ -z "${end_line}" ] && [ "${_trimmed}" = "${END_MARKER}" ]; then
    echo "end=${_line_num}" >> "${_tmplines}"
    end_line="${_line_num}"
  fi
done

# Read back the line numbers.
start_line=""
end_line=""
while IFS= read -r _entry; do
  case "${_entry}" in
    start=*) [ -z "${start_line}" ] && start_line="${_entry#start=}" ;;
    end=*)   [ -z "${end_line}" ]   && end_line="${_entry#end=}" ;;
  esac
done < "${_tmplines}"

if [ -z "${start_line}" ]; then
  echo "error: start marker not found: ${START_MARKER}" >&2
  exit 1
fi

if [ -z "${end_line}" ]; then
  echo "error: end marker not found: ${END_MARKER}" >&2
  exit 1
fi

# Count total lines in source.
total_lines="$(echo "${source_text}" | wc -l | tr -d ' ')"

# Build the result:
#   1. Lines up to and including the START marker
#   2. A blank line, the content, and a blank line
#   3. Lines from the END marker to EOF
content_text="$(cat "${CONTENT}")"

result="$(echo "${source_text}" | head -n "${start_line}")
${content_text}
$(echo "${source_text}" | tail -n "+${end_line}")"

# --- Output ---

if [ -n "${OUTPUT}" ]; then
  # Write to output file.
  echo "${result}" > "${OUTPUT}" || {
    echo "error: failed to write to output file: ${OUTPUT}" >&2
    exit 1
  }

  # Format with deno if available.
  if command -v deno >/dev/null 2>&1; then
    deno fmt --ext md --config "${SCRIPT_DIR}/deno.json" "${OUTPUT}" 2>/dev/null
  fi

  echo "replaced tag '${TAG}' in ${SOURCE} -> ${OUTPUT}" >&2
else
  # Format with deno if available (use a temp file).
  if command -v deno >/dev/null 2>&1; then
    _tmpout="$(mktemp)"
    # Update trap to clean up both temp files.
    trap 'rm -f "${_tmplines}" "${_tmpout}"' EXIT
    echo "${result}" > "${_tmpout}"
    deno fmt --ext md --config "${SCRIPT_DIR}/deno.json" "${_tmpout}" 2>/dev/null
    cat "${_tmpout}"
  else
    echo "${result}"
  fi
fi

exit 0
