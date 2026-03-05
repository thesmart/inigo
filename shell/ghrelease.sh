#!/bin/sh
# ghrelease.sh — Create a GitHub release for an existing tag.
#
# Verifies gh authentication and push permissions, builds binaries
# if needed, and creates a GitHub release with auto-generated notes.
#
# Usage:
#   ./ghrelease.sh [flags] VERSION
#
# Examples:
#   ./ghrelease.sh v0.2.0
#   ./ghrelease.sh --dry-run v0.2.0

set +x

# Resolve the directory this script lives in.
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# Load getoptions library functions.
eval "$(sh "${SCRIPT_DIR}/getoptions/getoptions.sh" -)"

parser_definition() {
  setup REST help:usage -- \
    "Usage: $0 [flags] VERSION" \
    '' \
    'Create a GitHub release for an existing tag.' \
    '' \
    'VERSION must match vMAJOR.MINOR.PATCH (e.g. v0.2.0).' \
    '' \
    'Steps performed:' \
    '  1. Validate version format' \
    '  2. Verify tag exists' \
    '  3. Verify gh CLI is authenticated with push access' \
    '  4. Build cross-platform binaries (if not already built)' \
    '  5. Create GitHub release with binaries and auto-generated notes' \
    '' \
    'Examples:' \
    "  $0 v0.2.0" \
    "  $0 --dry-run v0.2.0"
  msg -- '' 'Options:'
  flag DRY_RUN -n --dry-run -- "print what would happen without executing"
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

# --- Resolve project identity ---

GH_REPO=$(sh "${SCRIPT_DIR}/ghrepo.sh") || exit 1

# --- Validation ---

VERSION="${1:-}"

if [ -z "${VERSION}" ]; then
  echo "error: VERSION is required (e.g. $0 v0.2.0)" >&2
  echo "" >&2
  eval "$(getoptions_help parser_definition usage "$0")"
  usage >&2
  exit 1
fi

# Validate version format.
case "${VERSION}" in
  v[0-9]*.[0-9]*.[0-9]*) ;;
  *)
    echo "error: VERSION must match vMAJOR.MINOR.PATCH (e.g. v0.2.0)" >&2
    exit 1
    ;;
esac

# Verify tag exists.
if ! git rev-parse "${VERSION}" >/dev/null 2>&1; then
  echo "error: tag ${VERSION} does not exist — create it first with release.sh" >&2
  exit 1
fi

# Verify gh CLI is available and authenticated.
if ! command -v gh >/dev/null 2>&1; then
  echo "error: gh (GitHub CLI) is required — install from: https://cli.github.com" >&2
  exit 1
fi

if ! gh auth status >/dev/null 2>&1; then
  echo "error: gh is not authenticated — run: gh auth login" >&2
  exit 1
fi

# Verify push access to the repository.
if ! gh api "repos/${GH_REPO}" --jq .permissions.push 2>/dev/null | grep -q true; then
  echo "error: gh lacks push access to ${GH_REPO}" >&2
  echo "" >&2
  echo "  try: gh auth refresh -s repo" >&2
  echo "   or: gh auth login" >&2
  echo "" >&2
  echo "  fine-grained tokens need: Contents -> Read and write" >&2
  echo "  classic tokens need:      repo scope" >&2
  exit 1
fi

# --- Execute or dry-run ---

if [ "${DRY_RUN}" = "1" ]; then
  echo "gh-release: dry-run for ${VERSION}"
  echo "  would run: task build:all (if no binaries in build/)"
  echo "  would run: gh release create ${VERSION} build/*-* --generate-notes"
  exit 0
fi

# Build binaries if not already present.
if ! ls build/*-* >/dev/null 2>&1; then
  echo "--- no binaries found in build/ — building ---"
  task build:all || {
    echo "error: build:all failed" >&2
    exit 1
  }
fi

echo "--- creating GitHub release ${VERSION} ---"
gh release create "${VERSION}" build/*-* --generate-notes || {
  echo "" >&2
  echo "error: gh release create failed" >&2
  echo "" >&2
  echo "  if the release already exists, delete it first:" >&2
  echo "    gh release delete ${VERSION} --yes" >&2
  echo "" >&2
  echo "  if this is a permissions issue:" >&2
  echo "    gh auth refresh -s repo" >&2
  exit 1
}

echo ""
echo "gh-release: ${VERSION} published"
echo "  https://github.com/${GH_REPO}/releases/tag/${VERSION}"

exit 0
