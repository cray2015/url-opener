# url-opener — project spec

## 1. Overview

A lightweight desktop utility written in Go. It runs a local HTTP server on
port **8765** that accepts a POST containing a URL and opens it in the system's
default browser. On Windows — the primary target — the process lives silently
in the system tray with a right-click menu offering **Re-run** and **Exit**. A
headless Linux build exists as a compatibility layer, sharing the same HTTP
handler with no tray.

## 2. Goals

- Accept a `POST /open` request with a JSON body and open the URL in the
  default browser.
- Run headlessly after launch — no console window visible to the user.
- Appear as a tray icon in the Windows notification area.
- Be a single self-contained `.exe` with no installer or runtime dependencies.

## 3. Non-Goals

<!-- Tag names when the item could come back, not what release excluded it:
     [never]       — permanently out of scope, architectural or by principle
     [v2] / [v3]   — deferred to that release
     [unplanned]   — not ruled out, not scheduled
     [superseded]  — reversed; annotate in place, never delete -->

- **[never]** Authentication or an API key on the HTTP endpoint — localhost/LAN
  only, trusted caller by design.
- **[never]** URL reachability checks — format validation only, no network
  request is made to see whether the target resolves.
- **[superseded → `CLAUDE.md` → "Platform split"]** No cross-platform support
  (Windows only) — a headless Linux build now exists as a compatibility layer.
  It reuses the same HTTP handler and `pkg/browser`, and deliberately has no
  tray icon: `getlantern/systray`'s Linux backend needs cgo plus
  GTK/libayatana-appindicator, which this project otherwise doesn't depend on.
- **[unplanned]** Windows autostart / registry integration, and any
  installer/NSIS/WiX packaging. `README.md` documents the manual Startup-folder
  step instead. Linux is the exception: `dist/linux/install.sh` (wrapped by
  `make install-linux`) installs the headless binary as a systemd *user*
  service, because that path is five manual commands rather than one shortcut.
- **[unplanned]** A persistent configuration file or settings UI. Port 8765 is
  hardcoded in `newServer()` and `serverAddress()`; changing it means a rebuild.
- **[unplanned]** HTTPS/TLS on the local server.
- **[unplanned]** Multiple URLs in one request.
- **[unplanned]** Notification popups on successful open.
- **[unplanned]** A Linux tray/indicator icon. Adding one needs cgo re-enabled
  for that build path plus `libayatana-appindicator3-dev`; see `CLAUDE.md`.

## 4. Design rationale and constraints

- **The tray library dictates the platform split.** `getlantern/systray` has a
  pure-Go backend only on Windows, so Linux support means building without it
  rather than porting it. That is why `main()` and all tray code sit behind
  build tags, and why `CGO_ENABLED=0` holds on both targets.
- **`-H windowsgui` lives in the build command, not in code.** It is a linker
  flag; there is no source-level equivalent.
- **Unauthenticated on purpose.** The endpoint is reachable only from the local
  machine or LAN, and adding auth would defeat the point — the caller is a
  phone shortcut with no good place to keep a secret.
- **Input is treated as messy text, not a clean URL.** Callers paste share
  sheets and article snippets, so the handler extracts a URL from surrounding
  content rather than rejecting anything that isn't already exact. See §7.2.
- **Re-run exists because binding can fail transiently.** Rather than crash on
  a busy port, the server surfaces the failure and offers a manual rebind.

## 5. Architecture

A single Go binary. `main()` starts the HTTP server on a goroutine, then blocks
— on Windows in `systray.Run`, on Linux on a signal channel. Both paths serve
the identical `POST /open` handler.

Diagram source: `docs/architecture.d2`. Render on demand with
`d2 docs/architecture.d2 docs/architecture.svg` — the `.d2` is the source of
truth; the rendered SVG is not committed.

**Data flow:** LAN client POSTs JSON to `/open` → handler reads and unmarshals
the body → `extractURL` decodes and locates a URL in the text → `validateURL`
checks scheme and host → `browser.OpenURL` launches the default browser.

### 5.1 Startup sequence

```
main()
 ├── go startHTTPServer()      // goroutine: net/http ListenAndServe :8765
 └── systray.Run(onReady, onExit)      // Windows; blocks the main goroutine
      └── onReady()
           ├── systray.SetIcon(iconBytes)
           ├── systray.SetTooltip(serverAddress())
           ├── mRerun := systray.AddMenuItem("Re-run", "Restart HTTP server")
           ├── mExit  := systray.AddMenuItem("Exit", "Quit url-opener")
           └── go func() { select on mRerun.ClickedCh / mExit.ClickedCh }()
```

`systray.Run` must stay on the main goroutine — a library requirement on
Windows. On Linux the same shape holds with a `SIGINT`/`SIGTERM` channel in
place of `systray.Run`.

### 5.2 Platform split

The build-tag layout — which file carries which tag, and why `src/handler.go` no
longer imports `systray` — is documented in `CLAUDE.md` → "Platform split" and
is not duplicated here.

### 5.3 Re-run logic

`restartServer()` shuts the current `*http.Server` down with a 2-second
`context` timeout, builds a fresh one bound to `:8765`, and starts it on a new
goroutine. The active server is held in a package-level var guarded by a
`sync.Mutex`. Triggered from the tray menu; Windows only.

## 6. Components

| Path | Role |
|---|---|
| `src/main_windows.go` | Windows entry point — starts the server, then `systray.Run` |
| `src/main_other.go` | Non-Windows entry point — starts the server, blocks on SIGINT/SIGTERM |
| `src/handler.go` | HTTP server lifecycle, `POST /open`, `extractURL`, `validateURL`, `restartServer` |
| `src/tray.go` | Systray setup and menu loop (Windows only) |
| `src/version.go` | `var version = "dev"`, overridden at link time via `-X main.version=` |
| `src/serverinfo.go` | `serverAddress()` — the `http://<hostname>.local:8765` string, shared by tray tooltip and Linux startup log |
| `src/icon.go` | Embeds `assets/icon.ico` (package-relative) via `go:embed` |
| `src/logging_debug.go` | `//go:build debug` — log flags only, output stays on stderr |
| `src/logging_release.go` | `//go:build !debug` — redirects the log to `%APPDATA%\url-opener\app.log` |
| `src/assets/icon.ico` | The embedded tray icon. `src/assets/icon.png` is unused at runtime |
| `Makefile` | Windows and Linux build targets, plus `changelog` via git-cliff |

## 7. HTTP API

### 7.1 `POST /open`

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
| `200 OK` | A URL was extracted and validated and `browser.OpenURL()` succeeded |
| `400 Bad Request` | Body unreadable, malformed JSON, `url` field missing or empty, or no valid URL found in the text |
| `405 Method Not Allowed` | Request method is not POST |
| `500 Internal Server Error` | `browser.OpenURL()` returned an error |

**Response body**

```json
{ "status": "ok" }
{ "status": "error", "message": "..." }
```

### 7.2 URL extraction and validation

The `url` field is treated as text that *contains* a URL, not as a URL. This is
what makes pasting a share-sheet payload work. `extractURL` runs first:

1. `url.QueryUnescape` the input, if it decodes cleanly.
2. Replace backslash-escaped slashes (`\/` → `/`).
3. Try the whole trimmed string — if it validates, use it.
4. Otherwise regex-scan for the first `https?://\S+` token, strip trailing
   punctuation (`.,;:!?)"'`), and validate that.
5. If nothing validates, return `400` with "no valid URL found in body".

So `{"url": "Source: ZDNET\nhttps://example.com/article."}` opens
`https://example.com/article`.

`validateURL` then uses `net/url.ParseRequestURI` and requires:

1. It parses without error.
2. Scheme is present and is `http` or `https`.
3. Host is non-empty.

No network request is made to check reachability.

## 8. Runtime behaviour

### 8.1 Tray icon (Windows)

- Icon appears in the notification area immediately on launch.
- **Tooltip**: `"URL Opener — listening on http://<hostname>.local:8765"`,
  hostname lowercased, falling back to `localhost` if resolution fails.
- **Right-click menu**:
  - `Re-run` — restarts the listener (see §5.3). Useful if the port was
    briefly in use.
  - `Exit` — calls `systray.Quit()`.
- No double-click action.
- The icon is embedded at compile time via `//go:embed assets/icon.ico` — the
  directive is package-relative, so on disk that is `src/assets/icon.ico`.

### 8.2 Logging

- **Debug builds** (`-tags debug`): `log` output goes to stderr with a visible
  console window, since `-H windowsgui` is also omitted from that target.
- **Release builds** (`!debug`): `src/logging_release.go` redirects the log to
  `%APPDATA%\url-opener\app.log`, creating the directory if needed, and
  silently does nothing if `APPDATA` is unset or the file can't be opened. The
  file is append-only — there is no rotation.

### 8.3 Error handling

| Scenario | Behaviour |
|---|---|
| Port 8765 already in use on start | Log the error and call `onListenError()` — tray tooltip update on Windows, no-op on Linux. Does not crash |
| `browser.OpenURL()` fails | Return `500` with the error message, log it |
| Malformed JSON body | Return `400 Bad Request` |
| No URL found in the body text | Return `400`, "no valid URL found in body" |
| Re-run while the server is healthy | Graceful shutdown then rebind, no user-visible interruption |

## 9. Build

Targets live in the `Makefile`; the expansions and the reason each flag is
required are in `CLAUDE.md` → "Build Commands".

```
make build              # Windows release — windowsgui, stripped
make build-debug        # Windows debug — console + stderr logging
make build-linux        # Linux release — headless, no tray
make build-linux-debug  # Linux debug
make changelog          # regenerate CHANGELOG.md via git-cliff
```

## 10. Milestones

- [x] v1.0.0 — Initial release: `POST /open`, tray icon, Re-run/Exit menu
- [x] v1.1.0 — Logging and URL sanitisation (`extractURL`); project spec added; README Windows startup instructions
- [x] v1.2.0 — Tray tooltip shows `http://<hostname>.local:8765`; git-cliff changelog generation
- [x] Unreleased — Linux compatibility build: build-tag split, `src/serverinfo.go`, `onListenError` indirection, headless `src/main_other.go`

## 11. Acceptance criteria

Not yet defined. §8.3 is the closest thing to a pass/fail list and could be
turned into one directly; nothing has been written as an explicit checklist.

## 12. Open questions

- **Log rotation.** `app.log` grows without bound in release builds. Nobody has
  hit a problem with it, and no policy has been chosen.
- **Linux tray.** Deferred rather than rejected — see §3. The cost is cgo plus
  an appindicator dependency on that build path.

## 13. Files

- `project_spec.md` — this file
- `CLAUDE.md` — build commands, platform split, and the invariants that fail silently
- `README.md` — install, usage, motivation, and the Windows startup step
- `CHANGELOG.md` — generated by git-cliff from commit messages; don't hand-edit
- `docs/architecture.d2` — architecture diagram source
- `cliff.toml` — git-cliff configuration
- `Makefile` — build, install, and changelog targets
- `src/` — the entire `main` package; `go.mod`/`go.sum` stay at the repo root,
  so the module root and the package directory differ and every build target
  names `./src`. `src/assets/` has to sit inside the package because
  `//go:embed` cannot reach outside it
- `dist/linux/url-opener.service` — systemd user unit template
- `dist/linux/install.sh` — Linux installer/uninstaller for that unit
- `.github/workflows/ci.yml` — gofmt, vet, all four build targets, shellcheck
- `.github/workflows/release.yml` — tag-triggered release: binaries, notes,
  and the `CHANGELOG.md` commit back to master
- `.github/dependabot.yml` — monthly action and Go module bumps
- `LICENSE` — MIT
- `src/version.go` — `var version`, stamped at link time by the Makefile
