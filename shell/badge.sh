#!/bin/sh
# badge.sh — Download a single static badge SVG from shields.io.
#
# Downloads one badge as an SVG file to the specified output path.
# All badge content (label, message, color) is supplied via parameters.
#
# Usage:
#   ./badge.sh [flags]
#
# Examples:
#   ./badge.sh -o ./badges/license-badge.svg -l license -m MIT -k blue
#   ./badge.sh -o ./badges/coverage-badge.svg -m 85% -k green
#   ./badge.sh -o ./badges/build-badge.svg -l build -m passing -k brightgreen --logo github

set +x

# Resolve the directory this script lives in.
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# Load getoptions library functions.
eval "$(sh "${SCRIPT_DIR}/getoptions/getoptions.sh" -)"

parser_definition() {
  setup REST help:usage -- \
    "Usage: $0 [flags]" \
    '' \
    'Download a single static badge SVG from shields.io.' \
    '' \
    'The badge URL is built from the provided label, message, and color.' \
    'If --label is omitted, the badge shows only the message and color.' \
    '' \
    'Optional query parameters (--style, --logo, --logo-color, --label-color)' \
    'are appended to the shields.io URL when provided.' \
    '' \
    'Examples:' \
    "  $0 -o ./badges/license.svg -l license -m MIT -k blue" \
    "  $0 -o ./badges/coverage.svg -m 85%25 -k green" \
    "  $0 -o ./badges/go-ref.svg -m Reference -k 91DFFB --logo go --logo-color white --label-color white"
  msg -- '' 'Options:'
  param OUTPUT      -o --output      -- "output file path for the badge SVG"
  param LABEL       -l --label       -- "left-side label text (optional)"
  param MESSAGE     -m --message     -- "right-side message text (required)"
  param COLOR       -k --color       -- "right-side background color (required)"
  param STYLE       -s --style       pattern:"flat|flat-square|plastic|for-the-badge|social" \
                                     -- "badge style (default: flat)"
  param LOGO           --logo        -- "simple-icons slug for left-side icon"
  param LOGO_COLOR     --logo-color  -- "icon tint color"
  param LABEL_COLOR    --label-color -- "left-side background color"
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
if [ -z "${MESSAGE}" ]; then
  echo "error: --message is required" >&2
  _err=1
fi
if [ -z "${COLOR}" ]; then
  echo "error: --color is required" >&2
  _err=1
fi

if [ "${_err}" -ne 0 ]; then
  echo "" >&2
  eval "$(getoptions_help parser_definition usage "$0")"
  usage >&2
  exit 1
fi

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

_outdir="$(dirname "${OUTPUT}")"
if [ ! -d "${_outdir}" ]; then
  mkdir -p "${_outdir}" || {
    echo "error: failed to create output directory: ${_outdir}" >&2
    exit 1
  }
fi

# --- Helper functions ---

# URL-encode characters that appear in badge values.
url_encode() {
  echo "$1" | sed -e 's/%/%25/g' -e 's/+/%2B/g' -e 's/ /%20/g' -e 's/#/%23/g' -e 's/&/%26/g' -e 's/?/%3F/g' -e 's/=/%3D/g'
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

# --- Build URL ---

_encoded_message=$(url_encode "${MESSAGE}")

if [ -n "${LABEL}" ]; then
  _encoded_label=$(url_encode "${LABEL}")
  _badge_path="${_encoded_label}-${_encoded_message}-${COLOR}"
else
  _badge_path="${_encoded_message}-${COLOR}"
fi

_url="https://img.shields.io/badge/${_badge_path}"

# Build query string from optional parameters.
_qs=""
_append_qs() {
  if [ -z "${_qs}" ]; then
    _qs="?${1}=${2}"
  else
    _qs="${_qs}&${1}=${2}"
  fi
}

[ -n "${STYLE}" ]       && _append_qs "style"      "${STYLE}"
[ -n "${LOGO}" ]        && _append_qs "logo"        "${LOGO}"
[ -n "${LOGO_COLOR}" ]  && _append_qs "logoColor"   "$(url_encode "${LOGO_COLOR}")"
[ -n "${LABEL_COLOR}" ] && _append_qs "labelColor"  "$(url_encode "${LABEL_COLOR}")"

_url="${_url}${_qs}"

# --- Download badge ---

download_badge "${_url}" "${OUTPUT}" || {
  echo "error: badge download failed" >&2
  exit 1
}

echo "badge downloaded to ${OUTPUT}"
exit 0
