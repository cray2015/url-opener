# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build Commands

The primary target is **Windows**, with a headless (no tray icon yet) **Linux** build as a secondary, compatibility-layer target — see "Platform split" below. Go is on PATH on this host (`go version` to check); if it ever isn't, extract a Linux Go tarball to `/tmp` and use `/tmp/go/bin/go`, or run from a Windows terminal.

```bash
# Release build — no console window, stripped binary
make build
# Expands to: CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="-H windowsgui -s -w" -o url-opener.exe .

# Debug build — console window visible, log output shown
make build-debug
# Expands to: CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o url-opener-debug.exe .

# Linux release/debug — headless, no tray (see Platform split)
make build-linux
make build-linux-debug
```

The `-H windowsgui` linker flag suppresses the console window. It must stay in the build command, not in code. `CGO_ENABLED=0` is required on Windows because the systray library uses pure-Go Win32 syscalls, and on Linux because this project deliberately avoids pulling in the cgo + GTK/libayatana-appindicator toolchain that the tray library's Linux backend needs — see Platform split.

There are no tests in this project.

## Platform split

`getlantern/systray` only has a pure-Go backend on Windows; its Linux backend requires cgo and links against `ayatana-appindicator3-0.1` (or the legacy `appindicator3-0.1`) via pkg-config, which is not installed on this host and is out of scope for now (no tray icon on Linux yet, by design). To keep the Linux build buildable with `CGO_ENABLED=0`, `main()` and tray-only code are split by build tag:

- **`main_windows.go`** (`//go:build windows`) — `go startHTTPServer()` then `systray.Run(onReady, onExit)`, exactly as before.
- **`main_other.go`** (`//go:build !windows`) — runs `startHTTPServer()` in the foreground and blocks on `SIGINT`/`SIGTERM` instead of a tray event loop. No menu, no icon; install as a service (e.g. `dist/linux/url-opener.service`) if you want it to survive logout.
- **`tray.go`** (`//go:build windows`) — unchanged systray setup, but the tooltip string now comes from the shared `serverAddress()` helper.
- **`serverinfo.go`** (no build tag) — `serverAddress()`, the hostname-based address string (`http://<hostname>.local:8765`) used by both the tray tooltip and the Linux startup log line.
- **`handler.go`** no longer imports `systray` directly. It calls `onListenError()`, a package-level func var that's a no-op by default and is overridden in `tray.go`'s `init()` to update the tray tooltip on Windows.

`browser.OpenURL` (via `pkg/browser`) already supports Linux natively (`xdg-open`/`x-www-browser`/`www-browser`, no cgo needed), so `/open` works identically on both platforms — the only thing genuinely missing on Linux is the tray icon itself.

`main()` kicks off two concurrent paths and then blocks (Windows shown; Linux is the same shape with a signal channel instead of `systray.Run`):

```
main()
 ├── go startHTTPServer()   // goroutine — net/http on :8765
 └── systray.Run(onReady, onExit)  // blocks main goroutine (library requirement)
```

**`handler.go`** owns the HTTP server lifecycle and the `POST /open` handler. The active `*http.Server` is stored in a package-level var protected by `sync.Mutex`. `restartServer()` (called from the tray menu, Windows only) does a graceful `Shutdown` with a 2-second timeout then starts a fresh server in a new goroutine. If the port is unavailable on any `ListenAndServe`, `onListenError()` fires rather than crashing (tray tooltip update on Windows, no-op on Linux).

**`tray.go`** (Windows only) contains `onReady()` which is called by `systray.Run` on its internal thread. It sets the icon/tooltip and spawns a goroutine that selects over the Re-run and Exit menu item channels.

**`icon.go`** embeds `assets/icon.ico` at compile time via `//go:embed`. The icon **must be ICO format** — `getlantern/systray` passes the bytes directly to the Windows `CreateIconFromResourceEx` API, which rejects PNG. This file has no build tag (embedding is harmless cross-platform); it's simply unused on Linux until a tray/indicator icon is added there.

## Key Constraints

- **ICO not PNG**: `assets/icon.ico` is the embedded tray icon. If you replace it, it must remain a valid `.ico` file (BMP-in-ICO format). The `assets/icon.png` file is unused at runtime.
- **No auth on the HTTP endpoint**: The `/open` endpoint is intentionally unauthenticated — it is localhost-only by design.
- **URL validation**: `validateURL` in `handler.go` uses `net/url.ParseRequestURI` and only accepts `http`/`https` schemes with a non-empty host. No network reachability check is made.
- **`systray.Run` must be on the main goroutine**: The library requires this on Windows. Do not move it to a goroutine.
- **No Linux tray icon yet**: intentionally deferred. If it's added later, it needs cgo re-enabled for that build path plus `libayatana-appindicator3-dev` (or the legacy appindicator) at build time — don't silently flip `CGO_ENABLED=1` on the existing `build-linux` target when doing so, since that changes its dependency footprint.
