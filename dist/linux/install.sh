#!/usr/bin/env bash
#
# install.sh — install url-opener as a systemd user service on Linux.
#
#   ./dist/linux/install.sh              # build if needed, install, enable, start
#   ./dist/linux/install.sh --uninstall  # stop, disable, remove unit + binary
#
# Environment overrides:
#   BIN_DIR   where the binary goes      (default: ~/.local/bin)
#   UNIT_DIR  where the unit file goes   (default: ~/.config/systemd/user)
#
# There is no tray icon on Linux (see CLAUDE.md "Platform split"); this
# installs the headless build that listens on :8765 and shells out to
# xdg-open. That needs a graphical session (the unit starts After=
# graphical-session.target).

set -euo pipefail

readonly UNIT_NAME="url-opener.service"
readonly BINARY="url-opener-linux"

script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
repo_root=$(cd -- "$script_dir/../.." && pwd)

BIN_DIR=${BIN_DIR:-$HOME/.local/bin}
UNIT_DIR=${UNIT_DIR:-${XDG_CONFIG_HOME:-$HOME/.config}/systemd/user}

info()  { printf '\033[1;34m==>\033[0m %s\n' "$*"; }
warn()  { printf '\033[1;33mwarning:\033[0m %s\n' "$*" >&2; }
die()   { printf '\033[1;31merror:\033[0m %s\n' "$*" >&2; exit 1; }

require_systemd() {
  command -v systemctl >/dev/null 2>&1 \
    || die "systemctl not found — install manually, or run $BIN_DIR/$BINARY in the foreground."
  systemctl --user show-environment >/dev/null 2>&1 \
    || die "no systemd user instance for this session (systemctl --user is unavailable)."
}

uninstall() {
  require_systemd
  if systemctl --user list-unit-files "$UNIT_NAME" >/dev/null 2>&1; then
    info "Stopping and disabling $UNIT_NAME"
    systemctl --user disable --now "$UNIT_NAME" >/dev/null 2>&1 || true
  fi
  rm -f -- "$UNIT_DIR/$UNIT_NAME"
  rm -f -- "$BIN_DIR/$BINARY"
  systemctl --user daemon-reload
  systemctl --user reset-failed "$UNIT_NAME" >/dev/null 2>&1 || true
  info "Removed $UNIT_DIR/$UNIT_NAME and $BIN_DIR/$BINARY"
}

build_if_needed() {
  local src="$repo_root/$BINARY"
  if [[ -x $src ]]; then
    info "Using existing build: $src"
    return
  fi
  command -v go >/dev/null 2>&1 \
    || die "$BINARY not found and Go is not on PATH — run 'make build-linux' first."
  info "Building $BINARY"
  make -C "$repo_root" build-linux
}

install_binary() {
  mkdir -p -- "$BIN_DIR"
  # Copy to a temp name first: cp onto a running binary fails with ETXTBSY.
  install -m 0755 -- "$repo_root/$BINARY" "$BIN_DIR/$BINARY.new"
  mv -f -- "$BIN_DIR/$BINARY.new" "$BIN_DIR/$BINARY"
  info "Installed $BIN_DIR/$BINARY"

  case ":$PATH:" in
    *":$BIN_DIR:"*) ;;
    *) warn "$BIN_DIR is not on your PATH (only matters if you want to run it by name)." ;;
  esac
}

install_unit() {
  mkdir -p -- "$UNIT_DIR"
  # Point ExecStart at the real BIN_DIR rather than the unit's %h default.
  sed -e "s|^ExecStart=.*|ExecStart=$BIN_DIR/$BINARY|" \
      -e '/^# Update this path/d' \
      "$script_dir/$UNIT_NAME" > "$UNIT_DIR/$UNIT_NAME"
  info "Installed $UNIT_DIR/$UNIT_NAME"
}

main() {
  case "${1:-}" in
    --uninstall|-u) uninstall; exit 0 ;;
    --help|-h)      sed -n '3,15p' "${BASH_SOURCE[0]}" | sed 's/^# \?//'; exit 0 ;;
    "")             ;;
    *)              die "unknown argument: $1 (try --help)" ;;
  esac

  require_systemd
  build_if_needed

  # Stop first so the running process isn't holding the binary or the port.
  systemctl --user stop "$UNIT_NAME" >/dev/null 2>&1 || true

  install_binary
  install_unit

  info "Enabling and starting $UNIT_NAME"
  systemctl --user daemon-reload
  systemctl --user enable --now "$UNIT_NAME"

  if systemctl --user is-active --quiet "$UNIT_NAME"; then
    info "Running at http://$(hostname).local:8765 — try:"
    printf "      curl -X POST http://localhost:8765/open -d '{\"url\":\"https://example.com\"}'\n"
  else
    warn "Service did not come up. Check: systemctl --user status $UNIT_NAME"
    exit 1
  fi

  if [[ -z ${DISPLAY:-}${WAYLAND_DISPLAY:-} ]]; then
    warn "No DISPLAY/WAYLAND_DISPLAY in this shell — xdg-open needs a graphical session to open URLs."
  fi
}

main "$@"
