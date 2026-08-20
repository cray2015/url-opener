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
# Expands to: CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -tags debug -o url-opener-debug.exe .

# Linux release/debug — headless, no tray (see Platform split)
make build-linux
make build-linux-debug

# Linux install/uninstall as a systemd *user* service
make install-linux      # -> dist/linux/install.sh
make uninstall-linux    # -> dist/linux/install.sh --uninstall
```

`dist/linux/install.sh` builds `url-opener-linux` if it's missing, installs it
to `BIN_DIR` (default `~/.local/bin`), renders `dist/linux/url-opener.service`
into `UNIT_DIR` (default `~/.config/systemd/user`) with `ExecStart` rewritten to
the real `BIN_DIR` path, then `enable --now`s it. It stops the unit before
copying, since `cp` onto a running binary fails with `ETXTBSY`. The shipped
`.service` template keeps its `%h/.local/bin/...` default for people who install
by hand — if you change that path in the template, the `sed` rewrite in
`install_unit()` must still match the `ExecStart=` line.

The `-H windowsgui` linker flag suppresses the console window. It must stay in the build command, not in code. `CGO_ENABLED=0` is required on Windows because the systray library uses pure-Go Win32 syscalls, and on Linux because this project deliberately avoids pulling in the cgo + GTK/libayatana-appindicator toolchain that the tray library's Linux backend needs — see Platform split.

All four build targets compile the `main` package in `./src`; the binaries are still written to the repo root. `go.mod`/`go.sum` stay at the root — the module root and the package directory are deliberately different.

There are no tests in this project.

## CI and releases

- **`.github/workflows/ci.yml`** — push/PR: `gofmt -l ./src`, `go vet ./src`, all
  four `make` targets, and `shellcheck dist/linux/install.sh`. Building every
  target is the only real signal here, since there are no tests: it's what
  catches build-tag breakage, because the Windows-only files never compile on
  the Linux path and vice versa.
- **`.github/workflows/release.yml`** — triggered by a `v*.*.*` tag (or manual
  dispatch with a tag name). Builds `url-opener.exe` + `url-opener-linux` **from
  the tagged tree**, uploads them with `SHA256SUMS.txt`, generates release notes
  via `git-cliff --latest`, then checks out master, regenerates `CHANGELOG.md`
  and commits it back.

### Cutting a release

Everything is tag-driven; there is no manual step and no release branch.

```bash
git tag -a v1.4.0 -m "v1.4.0" && git push origin v1.4.0
```

Bump the minor when the range contains a `feat:`, the patch when it's only
`fix:`/`chore:`/`docs:`. The tag must be an ancestor of `master` (see below).
Verified end-to-end on v1.3.0: assets published, checksums matched a
re-downloaded binary, `main.version` correct in the shipped binary, and the
`CHANGELOG.md` commit landed back on `master`. Repository **workflow
permissions are already set to read/write** — the release job needs that and
a `permissions:` block cannot raise the token above the repo default, so
don't "fix" a 403 by editing the YAML.

Three things that will bite:

- The changelog commit is skipped unless the tag is an ancestor of `master`, and
  it pushes to `master` with the default `GITHUB_TOKEN`. **If you ever enable
  branch protection on `master`, that push starts failing** and needs either a
  PAT or an explicit bypass for the actions bot.
- Release notes come from `cliff.toml`, which sets `filter_unconventional =
  true`. A commit that isn't `feat:`/`fix:`/`docs:`/`refactor:`/`chore:` is
  silently dropped from the changelog. Nothing in CI enforces the format.
  **`ci:`, `build:`, `test:`, `perf:` and `style:` are not in the parser list**
  — CI and tooling work has to be typed `chore:` or it vanishes from the notes.
- Dependabot (`.github/dependabot.yml`) bumps action versions across **both**
  workflow files, so a merged bump changes the release path too and won't be
  exercised until the next tag. `getlantern/systray` is deliberately on the
  ignore list: a bump there is never routine, since it dictates the platform
  split. Both ecosystems are capped at 3 open PRs.

## Platform split

`getlantern/systray` only has a pure-Go backend on Windows; its Linux backend requires cgo and links against `ayatana-appindicator3-0.1` (or the legacy `appindicator3-0.1`) via pkg-config, which is not installed on this host and is out of scope for now (no tray icon on Linux yet, by design). To keep the Linux build buildable with `CGO_ENABLED=0`, `main()` and tray-only code are split by build tag:

- **`src/main_windows.go`** (`//go:build windows`) — `go startHTTPServer()` then `systray.Run(onReady, onExit)`, exactly as before.
- **`src/main_other.go`** (`//go:build !windows`) — runs `startHTTPServer()` in the foreground and blocks on `SIGINT`/`SIGTERM` instead of a tray event loop. No menu, no icon; install as a service (e.g. `dist/linux/url-opener.service`) if you want it to survive logout.
- **`src/tray.go`** (`//go:build windows`) — unchanged systray setup, but the tooltip string now comes from the shared `serverAddress()` helper.
- **`src/serverinfo.go`** (no build tag) — `serverAddress()`, the hostname-based address string (`http://<hostname>.local:8765`) used by both the tray tooltip and the Linux startup log line.
- **`src/handler.go`** no longer imports `systray` directly. It calls `onListenError()`, a package-level func var that's a no-op by default and is overridden in `src/tray.go`'s `init()` to update the tray tooltip on Windows.

`browser.OpenURL` (via `pkg/browser`) already supports Linux natively (`xdg-open`/`x-www-browser`/`www-browser`, no cgo needed), so `/open` works identically on both platforms — the only thing genuinely missing on Linux is the tray icon itself.

`main()` kicks off two concurrent paths and then blocks (Windows shown; Linux is the same shape with a signal channel instead of `systray.Run`):

```
main()
 ├── go startHTTPServer()   // goroutine — net/http on :8765
 └── systray.Run(onReady, onExit)  // blocks main goroutine (library requirement)
```

**`src/handler.go`** owns the HTTP server lifecycle and the `POST /open` handler. The active `*http.Server` is stored in a package-level var protected by `sync.Mutex`. `restartServer()` (called from the tray menu, Windows only) does a graceful `Shutdown` with a 2-second timeout then starts a fresh server in a new goroutine. If the port is unavailable on any `ListenAndServe`, `onListenError()` fires rather than crashing (tray tooltip update on Windows, no-op on Linux).

**`src/tray.go`** (Windows only) contains `onReady()` which is called by `systray.Run` on its internal thread. It sets the icon/tooltip and spawns a goroutine that selects over the Re-run and Exit menu item channels.

**`src/icon.go`** embeds `assets/icon.ico` at compile time via `//go:embed`. The directive is package-relative, so on disk that file is `src/assets/icon.ico`. The icon **must be ICO format** — `getlantern/systray` passes the bytes directly to the Windows `CreateIconFromResourceEx` API, which rejects PNG. This file has no build tag (embedding is harmless cross-platform); it's simply unused on Linux until a tray/indicator icon is added there.

## Key Constraints

- **`assets/` must stay inside `src/`**: `//go:embed` cannot reference paths outside its own package directory (no `..`), so moving `src/assets/` back to the repo root breaks the build. That constraint is the only reason the icon lives under `src/`.
- **ICO not PNG**: `src/assets/icon.ico` is the embedded tray icon. If you replace it, it must remain a valid `.ico` file (BMP-in-ICO format). The `src/assets/icon.png` file is unused at runtime.
- **No auth on the HTTP endpoint**: The `/open` endpoint is intentionally unauthenticated — it is localhost-only by design.
- **The `url` field is text, not a URL**: `extractURL` in `src/handler.go` runs before `validateURL` — it URL-decodes, unescapes `\/`, tries the whole string, then regex-scans for the first `https?://\S+` token and strips trailing punctuation. That is deliberate, so a pasted share-sheet payload works; don't "simplify" it into a straight parse. `validateURL` then requires `net/url.ParseRequestURI` to succeed with an `http`/`https` scheme and a non-empty host. No network reachability check is made. See `project_spec.md` §7.2.
- **`-tags debug` selects the logging destination, and is separate from `-H windowsgui`**: `src/logging_release.go` (`//go:build !debug`) redirects `log` output to `%APPDATA%\url-opener\app.log`; `src/logging_debug.go` (`//go:build debug`) leaves it on stderr. Release logging is written to a file, not suppressed — and the file is append-only with no rotation. Dropping `-tags debug` from `build-debug` silently sends a developer's log output to a file instead of the console.
- **Version is a link-time stamp, not a constant**: `src/version.go` declares `var version = "dev"`, and the Makefile overrides it with `-ldflags "-X main.version=$(VERSION)"`, defaulting to `git describe --tags --always --dirty`. It must stay a plain package-level `var` in `main` — `const`, a different package, or renaming it silently breaks the stamp, and the build still succeeds with `version` left at `dev`. The release workflow passes the tag explicitly (`VERSION=v1.3.0`) rather than relying on `git describe`.
- **`systray.Run` must be on the main goroutine**: The library requires this on Windows. Do not move it to a goroutine.
- **No Linux tray icon yet**: intentionally deferred. If it's added later, it needs cgo re-enabled for that build path plus `libayatana-appindicator3-dev` (or the legacy appindicator) at build time — don't silently flip `CGO_ENABLED=1` on the existing `build-linux` target when doing so, since that changes its dependency footprint.
