#!/usr/bin/env bash
# InfraGraph installer — downloads the latest release binary for the
# current platform and installs it to /usr/local/bin (or a custom path).
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/timkrebs/infragraph/main/scripts/install.sh | bash
#   curl -fsSL ... | INSTALL_DIR=~/.local/bin bash
#
# Inspired by the HashiCorp Vault installer pattern.

set -euo pipefail

REPO="timkrebs/infragraph"
BINARY="infragraph"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

# ─── Detect platform ────────────────────────────────────────────────────────

detect_os() {
  local os
  os="$(uname -s | tr '[:upper:]' '[:lower:]')"
  case "$os" in
    linux*)  echo "linux" ;;
    darwin*) echo "darwin" ;;
    mingw*|msys*|cygwin*) echo "windows" ;;
    freebsd*) echo "freebsd" ;;
    *)
      echo "Unsupported OS: $os" >&2
      exit 1
      ;;
  esac
}

detect_arch() {
  local arch
  arch="$(uname -m)"
  case "$arch" in
    x86_64|amd64) echo "amd64" ;;
    aarch64|arm64) echo "arm64" ;;
    armv7*|armhf)  echo "armv7" ;;
    *)
      echo "Unsupported architecture: $arch" >&2
      exit 1
      ;;
  esac
}

# ─── Resolve latest version ─────────────────────────────────────────────────

latest_version() {
  local url="https://api.github.com/repos/${REPO}/releases/latest"
  if command -v curl &>/dev/null; then
    curl -fsSL "$url" | grep '"tag_name"' | sed -E 's/.*"v([^"]+)".*/\1/'
  elif command -v wget &>/dev/null; then
    wget -qO- "$url" | grep '"tag_name"' | sed -E 's/.*"v([^"]+)".*/\1/'
  else
    echo "Error: curl or wget is required." >&2
    exit 1
  fi
}

# ─── Download & install ─────────────────────────────────────────────────────

main() {
  local os arch version archive_name url checksum_url

  os="$(detect_os)"
  arch="$(detect_arch)"
  version="${VERSION:-$(latest_version)}"

  if [ -z "$version" ]; then
    echo "Error: could not determine latest version." >&2
    echo "Set VERSION=x.y.z manually or check https://github.com/${REPO}/releases" >&2
    exit 1
  fi

  echo "Installing InfraGraph v${version} (${os}/${arch})..."

  local ext="tar.gz"
  if [ "$os" = "windows" ]; then
    ext="zip"
  fi

  archive_name="${BINARY}_${version}_${os}_${arch}.${ext}"
  url="https://github.com/${REPO}/releases/download/v${version}/${archive_name}"
  checksum_url="https://github.com/${REPO}/releases/download/v${version}/${BINARY}_${version}_SHA256SUMS"

  local tmpdir
  tmpdir="$(mktemp -d)"
  trap 'rm -rf "$tmpdir"' EXIT

  echo "  Downloading ${archive_name}..."
  curl -fsSL -o "${tmpdir}/${archive_name}" "$url"

  echo "  Downloading checksums..."
  curl -fsSL -o "${tmpdir}/SHA256SUMS" "$checksum_url"

  echo "  Verifying checksum..."
  (cd "$tmpdir" && sha256sum -c SHA256SUMS --ignore-missing 2>/dev/null) || \
  (cd "$tmpdir" && shasum -a 256 -c SHA256SUMS --ignore-missing 2>/dev/null) || {
    echo "Error: checksum verification failed!" >&2
    exit 1
  }

  echo "  Extracting..."
  if [ "$ext" = "tar.gz" ]; then
    tar xzf "${tmpdir}/${archive_name}" -C "$tmpdir"
  else
    unzip -qo "${tmpdir}/${archive_name}" -d "$tmpdir"
  fi

  # Install the binary.
  if [ -w "$INSTALL_DIR" ]; then
    mv "${tmpdir}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
  else
    echo "  Installing to ${INSTALL_DIR} (requires sudo)..."
    sudo mv "${tmpdir}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
  fi
  chmod +x "${INSTALL_DIR}/${BINARY}"

  echo ""
  echo "InfraGraph v${version} installed to ${INSTALL_DIR}/${BINARY}"
  echo ""
  "${INSTALL_DIR}/${BINARY}" version 2>/dev/null || true
}

main "$@"
