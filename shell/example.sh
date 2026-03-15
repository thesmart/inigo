#!/bin/sh
# example.sh — Build and run an example from ./examples/package/.
#
# Resolves an example by number (e.g. 01) or full directory name
# (e.g. 01-unmarshal-struct), builds it to ./build/, and runs it
# with CWD set to the example directory so fixtures resolve.
#
# Usage:
#   ./example.sh [flags] [example]
#
# Examples:
#   ./example.sh 01
#   ./example.sh 01-unmarshal-struct
#   ./example.sh --all

set +x

# Resolve the directory this script lives in.
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
EXAMPLES_DIR="${ROOT_DIR}/examples/package"
BUILD_DIR="${ROOT_DIR}/build"

# Load getoptions library functions.
eval "$(sh "${SCRIPT_DIR}/getoptions/getoptions.sh" -)"

parser_definition() {
  setup REST help:usage -- \
    "Usage: $0 [flags] [example]" \
    '' \
    'Build and run a library example.' \
    '' \
    'The example argument can be a number (01) or full name (01-unmarshal-struct).' \
    '' \
    'Examples:' \
    "  $0 01" \
    "  $0 01-unmarshal-struct" \
    "  $0 --all"
  msg -- '' 'Options:'
  flag ALL -a --all -- "run all examples"
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

# --- Resolve example directory ---

resolve_example() {
  _input="$1"
  _match=""

  # Try exact directory name first
  if [ -d "${EXAMPLES_DIR}/${_input}" ]; then
    _match="${_input}"
  else
    # Try matching by number prefix (e.g. "01" matches "01-unmarshal-struct")
    for _dir in "${EXAMPLES_DIR}"/"${_input}"-*; do
      if [ -d "${_dir}" ]; then
        _match="$(basename "${_dir}")"
        break
      fi
    done
  fi

  if [ -z "${_match}" ]; then
    echo "error: no example found matching: ${_input}" >&2
    echo "available examples:" >&2
    ls -1 "${EXAMPLES_DIR}" >&2
    exit 1
  fi

  echo "${_match}"
}

# --- Build and run a single example ---

run_example() {
  _name="$1"
  _example_dir="${EXAMPLES_DIR}/${_name}"
  _num="$(echo "${_name}" | cut -d- -f1)"
  _binary="${BUILD_DIR}/example-${_num}"

  echo "=== ${_name} ==="

  # Build
  go build -o "${_binary}" "${_example_dir}" || {
    echo "error: build failed for ${_name}" >&2
    return 1
  }

  # Run with CWD set to example dir
  (cd "${_example_dir}" && "${_binary}") || {
    echo "error: run failed for ${_name}" >&2
    return 1
  }

  echo ""
}

# --- Main ---

mkdir -p "${BUILD_DIR}"

if [ -n "${ALL}" ]; then
  _failed=0
  for _dir in "${EXAMPLES_DIR}"/*/; do
    _name="$(basename "${_dir}")"
    run_example "${_name}" || _failed=1
  done
  if [ "${_failed}" -ne 0 ]; then
    echo "error: one or more examples failed" >&2
    exit 1
  fi
  exit 0
fi

if [ $# -eq 0 ]; then
  echo "error: provide an example number or name, or use --all" >&2
  echo "" >&2
  eval "$(getoptions_help parser_definition usage "$0")"
  usage >&2
  exit 1
fi

for _arg in "$@"; do
  _name="$(resolve_example "${_arg}")"
  run_example "${_name}"
done

exit 0
