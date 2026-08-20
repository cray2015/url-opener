# url-opener

[![CI](https://github.com/cray2015/url-opener/actions/workflows/ci.yml/badge.svg)](https://github.com/cray2015/url-opener/actions/workflows/ci.yml)

A lightweight desktop utility that runs a local HTTP server on port **8765** and opens URLs in the system default browser on demand.

Runs on **Windows** (silent, in the system tray) and **Linux** (headless, as a systemd user service — no tray icon yet). Both share the same HTTP endpoint, so anything that can POST a URL works identically against either.

## Why this exists

**Built specifically for the iOS → Windows gap** — the one combination every existing tool either ignores or routes through the cloud.

Sending a link from your phone to your PC browser is a solved problem — but every existing solution comes with a catch:

| Tool | Problem |
|---|---|
| **Pushbullet** | Requires an account; routes your URLs through their cloud; largely abandoned |
| **KDE Connect** | Linux + Android focused; Windows port is unreliable; no iOS support |
| **Join (joaoapps)** | Needs a Google account; cloud-dependent |
| **Microsoft Phone Link** | Windows 11 only; Android only; heavyweight |
| **Apple Handoff** | Apple ecosystem only; does not reach Windows |

url-opener is different:

- **No account, no cloud** — traffic never leaves your local network
- **Triggered by iOS Shortcuts** — plugs into the full Shortcuts ecosystem: share sheets, automations, NFC tags, Focus modes, and more
- **Open HTTP endpoint** — any device on your network (Android, tablet, scripts) can trigger it, not just one paired app
- **Single binary** — drop the Windows `.exe` anywhere and add it to startup; on Linux one script installs it as a service

## Download

Prebuilt `url-opener.exe` (Windows) and `url-opener-linux` binaries are attached
to every [release](https://github.com/cray2015/url-opener/releases), along with
`SHA256SUMS.txt`. Verify with `sha256sum -c SHA256SUMS.txt`.

> **Windows SmartScreen:** the `.exe` is unsigned, so Windows will show a
> "Windows protected your PC" prompt on first run — **More info → Run anyway**.
> Code signing needs a paid certificate; see the release notes if that changes.

## Windows

Run `url-opener.exe`. It appears in the notification area (system tray) with no
console window.

**Right-click the tray icon** for two options:
- **Re-run** — restarts the HTTP listener if the port was briefly unavailable
- **Exit** — quits the application

### Run at startup

**Option 1 — Startup folder (simplest)**

1. Press `Win + R`, type `shell:startup`, and hit Enter. This opens your personal startup folder.
2. Right-click inside the folder → **New → Shortcut**.
3. Point it to the full path of `url-opener.exe` and finish the wizard.

url-opener will now launch silently at every login.

**Option 2 — Task Scheduler (recommended if you want it to start before login or with a delay)**

1. Open **Task Scheduler** (`taskschd.msc`).
2. Click **Create Basic Task** → give it a name (e.g. `url-opener`).
3. Set trigger to **When I log on**.
4. Set action to **Start a program** → browse to `url-opener.exe`.
5. On the final screen tick **Open the Properties dialog** → in the **General** tab enable **Run with highest privileges** if needed, and set **Configure for: Windows 10** (or your version).
6. Optionally add a delay under **Triggers → Edit → Delay task for** (e.g. 10 seconds) to let the network come up first.

To remove autostart, delete the shortcut from the startup folder (Option 1) or disable/delete the task in Task Scheduler (Option 2).

## Linux

`url-opener-linux` runs the same HTTP server and opens URLs via
`xdg-open`/`x-www-browser`/`www-browser` (whichever is found on `PATH`), so
`/open` behaves identically to Windows.

**There is no tray icon on Linux yet.** The practical differences: no icon, no
right-click menu, and therefore no **Re-run** — restart the service instead
(see [Managing the service](#managing-the-service)).

### Install

```bash
make install-linux
# or directly: ./dist/linux/install.sh
```

That installs it as a **systemd user service**, so it starts at login, survives
closing your terminal, and restarts on crash.

> Run it from a clone of this repo — the script needs `dist/linux/url-opener.service`.
> If you only downloaded the release binary, follow the manual steps below instead.

The script builds `url-opener-linux` if it isn't built yet, copies it to
`~/.local/bin`, writes the unit to `~/.config/systemd/user/` with `ExecStart`
pointed at the real install path, then enables and starts it. Re-run it any
time to upgrade in place — it stops the running service first, so the binary
isn't in use. Override the destinations with `BIN_DIR` / `UNIT_DIR`:

```bash
BIN_DIR=/opt/bin ./dist/linux/install.sh
```

To remove the binary, the unit, and the enablement symlink:

```bash
make uninstall-linux
# or directly: ./dist/linux/install.sh --uninstall
```

<details>
<summary>Manual install, if you'd rather not run the script</summary>

```bash
mkdir -p ~/.local/bin ~/.config/systemd/user
cp url-opener-linux ~/.local/bin/
cp dist/linux/url-opener.service ~/.config/systemd/user/
systemctl --user daemon-reload
systemctl --user enable --now url-opener.service
```

The shipped unit's `ExecStart` is `%h/.local/bin/url-opener-linux`; edit it if
you put the binary elsewhere.

</details>

### Managing the service

```bash
systemctl --user status  url-opener      # is it running?
systemctl --user restart url-opener      # the Linux equivalent of tray "Re-run"
journalctl --user -u url-opener -f       # live logs, including the version at startup
```

To run it in the foreground instead, without installing anything:

```bash
make build-linux && ./url-opener-linux
```

> **Note:** `xdg-open` needs a graphical session (`DISPLAY`/`WAYLAND_DISPLAY` and `DBUS_SESSION_BUS_ADDRESS`). A `systemd --user` service normally inherits these once you're logged into a desktop session; if URLs stop opening after the service starts before login (or over SSH), re-run `systemctl --user daemon-reload` after logging into the desktop, or add `WantedBy=graphical-session.target` bindings appropriate to your distro.

## Companion iOS Shortcut

An Apple Shortcut is available to trigger url-opener from any Apple device on the same network:

**[Download Shortcut](https://www.icloud.com/shortcuts/ac5982d6058c4924bfa5214cc10b3bcc)**

The shortcut sends a `POST /open` request to the machine running url-opener, letting you push a URL from your iPhone/iPad/Mac straight into that machine's browser.

**After installing, update the address in the shortcut:**

1. Find the address of the machine running url-opener:
   - **Windows** — hover the tray icon; the tooltip shows it, e.g. `http://mymachine.local:8765`.
   - **Linux** — there's no tray, so read it from the startup log
     (`journalctl --user -u url-opener | tail -1`), or build it from `hostname`.
2. Open the Shortcuts app, tap the shortcut, and hit **Edit**.
3. In the URL field, set the address to that hostname: `http://mymachine.local:8765/open`.
4. Save. Both devices must be on the same local network.

> **Tip:** the `.local` hostname is stable across reboots and easier to copy than an IP address. If mDNS is not available on your network, fall back to the IPv4 address (`ipconfig` on Windows, `ip addr` on Linux).

## API

```
POST http://localhost:8765/open
Content-Type: application/json

{ "url": "https://example.com" }
```

| Status | Meaning |
|---|---|
| `200 OK` | URL opened successfully |
| `400 Bad Request` | Missing/malformed body or invalid URL (must be `http`/`https`) |
| `405 Method Not Allowed` | Non-POST request |
| `500 Internal Server Error` | Browser failed to open |

All responses return JSON: `{ "status": "ok" }` or `{ "status": "error", "message": "..." }`.

## Build

Requires Go 1.21+. All source lives in `src/`; binaries are written to the repo root.

The build stamps a version into the binary — it defaults to `git describe`, and
shows up in the tray tooltip on Windows and the startup log line on Linux:

```bash
make build                    # -> "v1.2.0-2-g80472aa-dirty"
make build VERSION=v1.3.0     # -> "v1.3.0"
```

```bash
# Windows — release (no console window) / debug
make build
make build-debug

# Linux — release / debug (headless, no tray icon)
make build-linux
make build-linux-debug
```

## Releasing

Releases are cut by pushing a tag; `.github/workflows/release.yml` does the rest.

```bash
git tag -a v1.4.0 -m "v1.4.0"
git push origin v1.4.0
```

That builds both release binaries from the tagged tree, generates the release
notes for that tag with git-cliff, publishes the GitHub Release with the
binaries plus `SHA256SUMS.txt`, and then regenerates `CHANGELOG.md` and commits
it back to `master`. Commit messages must follow
[Conventional Commits](https://www.conventionalcommits.org/) (`feat:`, `fix:`,
`docs:`, `refactor:`, `chore:`) — `cliff.toml` drops anything that doesn't.

## License

[MIT](LICENSE) © vibhanshu
