# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build Commands

This is a **Windows-only** application. All builds cross-compile from the current host targeting `GOOS=windows GOARCH=amd64`. Go is not in the WSL PATH — use the full path `/tmp/go/bin/go` (after extracting a Linux Go tarball to `/tmp`) or run from a Windows terminal.

```bash
# Release build — no console window, stripped binary
make build
# Expands to: CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="-H windowsgui -s -w" -o url-opener.exe .

# Debug build — console window visible, log output shown
make build-debug
# Expands to: CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o url-opener-debug.exe .
```

The `-H windowsgui` linker flag suppresses the console window. It must stay in the build command, not in code. `CGO_ENABLED=0` is required because the systray library uses pure-Go Win32 syscalls.

There are no tests in this project.

## Architecture

`main()` kicks off two concurrent paths and then blocks:

```
main()
 ├── go startHTTPServer()   // goroutine — net/http on :8765
 └── systray.Run(onReady, onExit)  // blocks main goroutine (library requirement)
```

**`handler.go`** owns the HTTP server lifecycle and the `POST /open` handler. The active `*http.Server` is stored in a package-level var protected by `sync.Mutex`. `restartServer()` (called from the tray menu) does a graceful `Shutdown` with a 2-second timeout then starts a fresh server in a new goroutine. If the port is unavailable on any `ListenAndServe`, the systray tooltip is updated to "Port 8765 in use" rather than crashing.

**`tray.go`** contains `onReady()` which is called by `systray.Run` on its internal thread. It sets the icon/tooltip and spawns a goroutine that selects over the Re-run and Exit menu item channels.

**`icon.go`** embeds `assets/icon.ico` at compile time via `//go:embed`. The icon **must be ICO format** — `getlantern/systray` passes the bytes directly to the Windows `CreateIconFromResourceEx` API, which rejects PNG.

## Key Constraints

- **ICO not PNG**: `assets/icon.ico` is the embedded tray icon. If you replace it, it must remain a valid `.ico` file (BMP-in-ICO format). The `assets/icon.png` file is unused at runtime.
- **No auth on the HTTP endpoint**: The `/open` endpoint is intentionally unauthenticated — it is localhost-only by design.
- **URL validation**: `validateURL` in `handler.go` uses `net/url.ParseRequestURI` and only accepts `http`/`https` schemes with a non-empty host. No network reachability check is made.
- **`systray.Run` must be on the main goroutine**: The library requires this. Do not move it to a goroutine.
