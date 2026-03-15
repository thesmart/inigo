#!/bin/sh
# release.sh — Tag and publish a release of the current Go module.
#
# Validates the version, verifies the working tree is clean and on main,
# tags, pushes, triggers pkg.go.dev indexing, builds cross-platform
# binaries, and creates a GitHub release.
#
# Usage:
#   ./release.sh [flags] VERSION
#
# Examples:
#   ./release.sh v0.2.0
#   ./release.sh --dry-run v0.2.0

set +x

# Resolve the directory this script lives in.
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# Load getoptions library functions.
eval "$(sh "${SCRIPT_DIR}/getoptions/getoptions.sh" -)"

parser_definition() {
  setup REST help:usage -- \
    "Usage: $0 [flags] VERSION" \
    '' \
    'Tag and publish a release of the current Go module.' \
    '' \
    'VERSION must match vMAJOR.MINOR.PATCH (e.g. v0.2.0).' \
    '' \
    'Steps performed:' \
    '  1. Validate version format' \
    '  2. Verify working tree is clean' \
    '  3. Verify on main branch' \
    '  4. Verify tag does not already exist' \
    '  5. Verify gh CLI is authenticated' \
    '  6. Push main to origin' \
    '  7. Create and push version tag' \
    '  8. Trigger pkg.go.dev indexing' \
    '  9. Cross-compile release binaries' \
    '  10. Create GitHub release with binaries' \
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

MODULE_PATH=$(sh "${SCRIPT_DIR}/gomod.sh") || exit 1
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

# Skip strict checks in dry-run mode.
if [ "${DRY_RUN}" != "1" ]; then
  # Verify working tree is clean.
  _dirty=$(git status --porcelain)
  if [ -n "${_dirty}" ]; then
    echo "error: working tree is dirty — commit or stash changes first" >&2
    echo "${_dirty}" >&2
    exit 1
  fi

  # Verify on main branch.
  _branch=$(git rev-parse --abbrev-ref HEAD)
  if [ "${_branch}" != "main" ]; then
    echo "error: releases must be from the main branch (currently on ${_branch})" >&2
    exit 1
  fi

  # Verify tag does not already exist.
  if git rev-parse "${VERSION}" >/dev/null 2>&1; then
    echo "error: tag ${VERSION} already exists" >&2
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
fi

# --- Execute or dry-run ---

if [ "${DRY_RUN}" = "1" ]; then
  echo "release: dry-run for ${VERSION}"
  echo "  would run: git push origin main"
  echo "  would run: git tag ${VERSION}"
  echo "  would run: git push origin ${VERSION}"
  echo "  would run: curl https://proxy.golang.org/${MODULE_PATH}/@v/${VERSION}.info"
  echo "  would run: task build:all"
  echo "  would run: gh release create ${VERSION} build/*-* --generate-notes"
  exit 0
fi

echo "--- pushing main to origin ---"
git push origin main

echo "--- tagging ${VERSION} ---"
git tag "${VERSION}"

echo "--- pushing tag to origin ---"
git push origin "${VERSION}"

echo "--- triggering pkg.go.dev indexing ---"
curl -sS "https://proxy.golang.org/${MODULE_PATH}/@v/${VERSION}.info"
echo ""

echo "--- building release binaries ---"
task build:all

echo "--- creating GitHub release ---"
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
echo "release: ${VERSION} published"
echo "  https://pkg.go.dev/${MODULE_PATH}@${VERSION}"
echo "  https://github.com/${GH_REPO}/releases/tag/${VERSION}"

exit 0
