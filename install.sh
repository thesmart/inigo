#!/bin/sh
# install.sh — download and install the inigo binary
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/thesmart/inigo/main/install.sh | sh
#
# Environment variables:
#   VERSION      Version tag to install (default: latest release)
#   INSTALL_DIR  Directory to install into (default: ~/.local/bin)

set -eu

REPO="thesmart/inigo"
BINARY="inigo"
INSTALL_DIR="${INSTALL_DIR:-${HOME}/.local/bin}"

# --- helpers ---

log() {
  printf '%s\n' "$@"
}

err() {
  printf 'error: %s\n' "$@" >&2
  exit 1
}

need() {
  command -v "$1" >/dev/null 2>&1 || err "$1 is required but not found"
}

# download URL to FILE using curl or wget
download() {
  url="$1"
  file="$2"
  if command -v curl >/dev/null 2>&1; then
    curl --proto '=https' --tlsv1.2 -fsSL -o "$file" "$url"
  elif command -v wget >/dev/null 2>&1; then
    wget -qO "$file" "$url"
  else
    err "curl or wget is required"
  fi
}

# --- detect platform ---

detect_os() {
  case "$(uname -s)" in
    Linux*)  echo "linux" ;;
    Darwin*) echo "darwin" ;;
    FreeBSD*) echo "freebsd" ;;
    *) err "unsupported OS: $(uname -s)" ;;
  esac
}

detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64) echo "amd64" ;;
    aarch64|arm64) echo "arm64" ;;
    *) err "unsupported architecture: $(uname -m)" ;;
  esac
}

# --- resolve version ---

resolve_version() {
  if [ -n "${VERSION:-}" ]; then
    echo "$VERSION"
    return
  fi

  tmpfile=$(mktemp)
  trap 'rm -f "$tmpfile"' EXIT

  download "https://api.github.com/repos/${REPO}/releases/latest" "$tmpfile"

  tag=$(grep '"tag_name"' "$tmpfile" | sed 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/')
  rm -f "$tmpfile"
  trap - EXIT

  if [ -z "$tag" ]; then
    err "could not determine latest version"
  fi

  echo "$tag"
}

# --- main ---

main() {
  os=$(detect_os)
  arch=$(detect_arch)
  version=$(resolve_version)
  asset="${BINARY}-${os}-${arch}"
  url="https://github.com/${REPO}/releases/download/${version}/${asset}"

  log "installing ${BINARY} ${version} (${os}/${arch})..."

  tmpfile=$(mktemp)
  trap 'rm -f "$tmpfile"' EXIT

  download "$url" "$tmpfile" || err "download failed — check that ${version} has a release asset for ${os}/${arch}"

  mkdir -p "$INSTALL_DIR"
  dest="${INSTALL_DIR}/${BINARY}"
  mv "$tmpfile" "$dest"
  chmod +x "$dest"
  trap - EXIT

  log ""
  log "installed ${BINARY} to ${dest}"

  # hint about PATH if the install directory isn't already on PATH
  case ":${PATH}:" in
    *":${INSTALL_DIR}:"*) ;;
    *)
      log ""
      log "add to your PATH if needed:"
      log "  export PATH=\"${INSTALL_DIR}:\$PATH\""
      ;;
  esac
}

main
