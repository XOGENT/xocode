#!/bin/sh
# xocode installer — https://code.xogent.com/install
#
#   curl https://code.xogent.com/install -fsS | bash
#
# Environment overrides:
#   XOCODE_VERSION      pin a specific tag (e.g. v1.2.0); default: latest
#   XOCODE_INSTALL_DIR  install location; default: $HOME/.local/bin
set -eu

REPO="xogent/xocode"
BINARY="xocode"
INSTALL_DIR="${XOCODE_INSTALL_DIR:-$HOME/.local/bin}"
VERSION="${XOCODE_VERSION:-latest}"

log()  { printf '\033[1;35m==>\033[0m %s\n' "$1"; }
err()  { printf '\033[1;31merror:\033[0m %s\n' "$1" >&2; exit 1; }
have() { command -v "$1" >/dev/null 2>&1; }

detect_platform() {
  os="$(uname -s)"; arch="$(uname -m)"
  case "$os" in
    Darwin) os="darwin" ;;
    Linux)  os="linux" ;;
    *) err "unsupported OS: $os (xocode supports macOS and Linux)" ;;
  esac
  case "$arch" in
    x86_64|amd64) arch="amd64" ;;
    arm64|aarch64) arch="arm64" ;;
    *) err "unsupported architecture: $arch" ;;
  esac
  PLATFORM="${os}_${arch}"
}

dl() { # dl <url> <outfile>
  if have curl; then curl -fsSL "$1" -o "$2"
  elif have wget; then wget -qO "$2" "$1"
  else err "need curl or wget to download"; fi
}

resolve_version() {
  [ "$VERSION" != "latest" ] && return 0
  # Resolve the latest tag via the releases/latest redirect (no jq needed).
  if have curl; then
    VERSION="$(curl -fsSLI -o /dev/null -w '%{url_effective}' \
      "https://github.com/${REPO}/releases/latest" | sed 's#.*/tag/##')"
  else
    VERSION="$(wget -q -S -O /dev/null "https://github.com/${REPO}/releases/latest" 2>&1 \
      | sed -n 's#.*location:.*/tag/##p' | tr -d '\r' | tail -1)"
  fi
  [ -n "$VERSION" ] || err "could not resolve the latest version"
}

main() {
  detect_platform

  archive="${BINARY}_${PLATFORM}.tar.gz"
  # XOCODE_BASE_URL points at a directory containing the archives + checksums.txt
  # (used for mirrors and testing). Default: the GitHub release download URL.
  if [ -n "${XOCODE_BASE_URL:-}" ]; then
    base="${XOCODE_BASE_URL%/}"
  else
    resolve_version
    base="https://github.com/${REPO}/releases/download/${VERSION}"
  fi
  tmp="$(mktemp -d)"
  trap 'rm -rf "$tmp"' EXIT

  log "Downloading ${BINARY} ${VERSION} (${PLATFORM})"
  dl "${base}/${archive}"    "${tmp}/${archive}"
  dl "${base}/checksums.txt" "${tmp}/checksums.txt"

  log "Verifying checksum"
  (
    cd "$tmp"
    want="$(grep " ${archive}\$" checksums.txt | awk '{print $1}')"
    [ -n "$want" ] || err "no checksum found for ${archive}"
    if have sha256sum; then got="$(sha256sum "$archive" | awk '{print $1}')"
    elif have shasum; then got="$(shasum -a 256 "$archive" | awk '{print $1}')"
    else err "need sha256sum or shasum to verify"; fi
    [ "$want" = "$got" ] || err "checksum mismatch (want ${want}, got ${got})"
  )

  log "Installing to ${INSTALL_DIR}"
  tar -xzf "${tmp}/${archive}" -C "$tmp"
  mkdir -p "$INSTALL_DIR"
  install -m 0755 "${tmp}/${BINARY}" "${INSTALL_DIR}/${BINARY}"

  case ":$PATH:" in
    *":$INSTALL_DIR:"*) : ;;
    *)
      log "Adding ${INSTALL_DIR} to your PATH"
      for rc in "$HOME/.zshrc" "$HOME/.bashrc" "$HOME/.profile"; do
        [ -f "$rc" ] || continue
        grep -q '# xocode PATH' "$rc" 2>/dev/null && break
        printf '\n# xocode PATH\nexport PATH="%s:$PATH"\n' "$INSTALL_DIR" >> "$rc"
        break
      done
      printf '  Restart your shell, or run: export PATH="%s:$PATH"\n' "$INSTALL_DIR"
      ;;
  esac

  log "Installed ${BINARY} ${VERSION}."
  printf '\nRun \033[1;35mxocode\033[0m to get started. It will help you install and log in\nto the Claude Code and Cursor CLIs on first run.\n'
}

main "$@"
