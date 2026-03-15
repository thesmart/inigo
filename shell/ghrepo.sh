#!/bin/sh
# ghrepo.sh — Print the GitHub owner/repo from the git remote to stdout.
#
# Parses the "origin" remote URL (or --remote) and extracts the
# owner/repo pair (e.g. "torvalds/linux").
#
# Usage:
#   ./ghrepo.sh [flags]
#
# Examples:
#   ./ghrepo.sh
#   ./ghrepo.sh --remote upstream

set +x

# Resolve the directory this script lives in.
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# Load getoptions library functions.
eval "$(sh "${SCRIPT_DIR}/getoptions/getoptions.sh" -)"

parser_definition() {
  setup REST help:usage -- \
    "Usage: $0 [flags]" \
    '' \
    'Print the GitHub owner/repo from the git remote to stdout.' \
    '' \
    'Parses the remote URL and extracts the owner/repo pair.' \
    '' \
    'Examples:' \
    "  $0" \
    "  $0 --remote upstream"
  msg -- '' 'Options:'
  param REMOTE -r --remote -- "git remote name (default: origin)"
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

_remote="${REMOTE:-origin}"

# Get the remote URL.
_url=$(git remote get-url "${_remote}" 2>/dev/null) || {
  echo "error: git remote '${_remote}' not found" >&2
  exit 1
}

# Extract owner/repo from SSH or HTTPS URLs:
#   git@github.com:owner/repo.git
#   https://github.com/owner/repo.git
#   https://github.com/owner/repo
_repo=$(printf '%s\n' "${_url}" | sed -e 's|.*github\.com[:/]||' -e 's|\.git$||')

# Validate we got owner/repo format.
case "${_repo}" in
  */*)
    printf '%s\n' "${_repo}"
    ;;
  *)
    echo "error: could not extract owner/repo from remote URL: ${_url}" >&2
    exit 1
    ;;
esac

exit 0
