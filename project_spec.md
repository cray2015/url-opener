# PROJECT_SPEC.md — url-opener

## Overview

A lightweight Windows desktop utility written in Go. It runs a local HTTP server on port **8765** that accepts a POST request containing a URL and opens it in the system's default browser. The process lives silently in the Windows system tray, exposing a right-click context menu with **Re-run** and **Exit** options.

---

## Goals

- Accept a `POST /open` request with a JSON body and open the URL in the default browser.
- Run headlessly after launch — no console window visible to the user.
- Appear as a tray icon in the Windows notification area.
- Be a single self-contained `.exe` with no installer or runtime dependencies.

---

## Non-Goals

- No authentication or API key on the HTTP endpoint (localhost-only, trusted caller).
- No URL reachability check — format validation only.
- No Windows autostart / registry integration.
- No cross-platform support (Windows only).
- No persistent configuration file or settings UI.

---

## Tech Stack

| Concern | Package |
|---|---|
| HTTP server | `net/http` (stdlib) |
| Tray icon | `github.com/getlantern/systray` |
| Browser open | `github.com/pkg/browser` |
| Build tooling | `go build` with `-ldflags "-H windowsgui"` |

> **Windowsgui flag**: `-H windowsgui` suppresses the console window on launch. This must be set in the build command (or `Makefile`), not in code.

---

## Project Structure

```
url-opener/
├── main.go           # Entry point: starts server + systray
├── handler.go        # HTTP handler: parse, validate, open
├── tray.go           # Systray setup and menu logic
├── icon.go           # Embedded tray icon (PNG → []byte via go:embed)
├── assets/
│   └── icon.png      # 16x16 or 32x32 PNG tray icon
├── go.mod
├── go.sum
└── Makefile          # build target with windowsgui flag
```

---

## HTTP API

### `POST /open`

Opens the provided URL in the system default browser.

**Request**

```
Content-Type: application/json

{
  "url": "https://example.com"
}
```

**Responses**

| Status | Condition |
|---|---|
| `200 OK` | URL is valid and `browser.OpenURL()` was called successfully |
| `400 Bad Request` | Body is malformed, `url` field is missing, or URL fails format validation |
| `405 Method Not Allowed` | Request method is not POST |
| `500 Internal Server Error` | `browser.OpenURL()` returned an error |

**Response body (all cases)**

```json
{ "status": "ok" }
{ "status": "error", "message": "..." }
```

---

## URL Validation Rules

Validation uses `net/url.ParseRequestURI()` from stdlib. A URL is considered valid if:

1. It parses without error.
2. Scheme is present and is one of: `http`, `https`.
3. Host is non-empty.

Any URL failing these checks returns `400` with a descriptive message. No network request is made to check reachability.

---

## Tray Icon Behaviour

- Icon appears in the Windows notification area immediately on launch.
- **Tooltip**: `"URL Opener — listening on :8765"`
- **Right-click menu**:
  - `Re-run` — restarts the HTTP server listener (stops and re-binds on `:8765`). Useful if the port was briefly in use.
  - `Exit` — calls `systray.Quit()` and `os.Exit(0)`.
- No double-click action required.
- The tray icon PNG is embedded at compile time via `//go:embed assets/icon.png`.

---

## Startup Sequence

```
main()
 ├── go startHTTPServer()      // goroutine: net/http ListenAndServe :8765
 └── systray.Run(onReady, onExit)
      └── onReady()
           ├── systray.SetIcon(iconBytes)
           ├── systray.SetTooltip(...)
           ├── mRerun := systray.AddMenuItem("Re-run", "Restart HTTP server")
           ├── mExit  := systray.AddMenuItem("Exit", "Quit url-opener")
           └── go func() {
                 for {
                   select {
                     case <-mRerun.ClickedCh: restartServer()
                     case <-mExit.ClickedCh:  systray.Quit()
                   }
                 }
               }()
```

`systray.Run` blocks the main goroutine (as required by the library). The HTTP server runs on a separate goroutine.

---

## Re-run Logic

`restartServer()` should:

1. If a server is running, close its `http.Server` via `server.Shutdown(ctx)` with a short timeout (e.g. 2 seconds).
2. Create a new `http.Server` bound to `:8765`.
3. Start `server.ListenAndServe()` in a new goroutine.
4. Store the server reference in a package-level variable protected by a `sync.Mutex`.

---

## Build

```makefile
# Makefile
build:
	go build -ldflags="-H windowsgui -s -w" -o url-opener.exe .

build-debug:
	go build -o url-opener-debug.exe .
```

`build-debug` keeps the console window for development and log output.

---

## Logging

- In debug builds (no `-H windowsgui`): use `log.Printf` to stdout.
- In release builds: logging is suppressed (console hidden). Optionally write to a rotating log file at `%APPDATA%\url-opener\app.log` — treat this as a stretch goal, not required for v1.

---

## Error Handling Summary

| Scenario | Behaviour |
|---|---|
| Port 8765 already in use on start | Log error, show systray tooltip "Port 8765 in use" — do not crash |
| `browser.OpenURL()` fails | Return `500`, log the error |
| Malformed JSON body | Return `400 Bad Request` |
| Re-run while server is healthy | Graceful shutdown + rebind, no user-visible interruption |

---

## Out of Scope (v1)

- HTTPS / TLS on the local server
- Multiple URL support in one request
- Notification popups on open success
- Windows startup registry entry
- Installer / NSIS / WiX packaging