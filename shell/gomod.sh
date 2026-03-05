#!/bin/sh
# gomod.sh — Print the Go module path from go.mod to stdout.
#
# Searches upward from the current directory (or --dir) for go.mod
# and extracts the module path from the first "module" directive.
#
# Usage:
#   ./gomod.sh [flags]
#
# Examples:
#   ./gomod.sh
#   ./gomod.sh --dir /path/to/project

set +x

# Resolve the directory this script lives in.
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# Load getoptions library functions.
eval "$(sh "${SCRIPT_DIR}/getoptions/getoptions.sh" -)"

parser_definition() {
  setup REST help:usage -- \
    "Usage: $0 [flags]" \
    '' \
    'Print the Go module path from go.mod to stdout.' \
    '' \
    'Searches upward from --dir (default: working directory) for go.mod' \
    'and extracts the module path.' \
    '' \
    'Examples:' \
    "  $0" \
    "  $0 --dir /path/to/project"
  msg -- '' 'Options:'
  param SEARCH_DIR -d --dir -- "directory to search from (default: .)"
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

# Default to current directory.
_dir="${SEARCH_DIR:-.}"

# Resolve to absolute path.
_dir="$(cd "${_dir}" && pwd)" || {
  echo "error: directory does not exist: ${SEARCH_DIR}" >&2
  exit 1
}

# Walk upward to find go.mod.
_gomod=""
_cur="${_dir}"
while true; do
  if [ -f "${_cur}/go.mod" ]; then
    _gomod="${_cur}/go.mod"
    break
  fi
  _parent="$(dirname "${_cur}")"
  if [ "${_parent}" = "${_cur}" ]; then
    break
  fi
  _cur="${_parent}"
done

if [ -z "${_gomod}" ]; then
  echo "error: go.mod not found in ${_dir} or any parent directory" >&2
  exit 1
fi

# Extract module path: first line matching "module <path>".
_module=$(sed -n 's/^module[[:space:]]\{1,\}\(.*\)/\1/p' "${_gomod}" | head -1)

if [ -z "${_module}" ]; then
  echo "error: no module directive found in ${_gomod}" >&2
  exit 1
fi

printf '%s\n' "${_module}"

exit 0
