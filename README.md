# url-opener

A lightweight desktop utility that runs a local HTTP server on port **8765** and opens URLs in the system default browser on demand. On Windows it lives silently in the system tray with no console window; a headless Linux build is also available (no tray icon yet — see [Linux](#linux)).

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
- **Single exe, no installer** — drop it anywhere, add it to startup, done

## Companion iOS Shortcut

An Apple Shortcut is available to trigger url-opener from any Apple device on the same network:

**[Download Shortcut](https://www.icloud.com/shortcuts/ac5982d6058c4924bfa5214cc10b3bcc)**

The shortcut sends a `POST /open` request to the machine running url-opener, letting you push a URL from your iPhone/iPad/Mac directly into the Windows browser.

**After installing, update the address in the shortcut:**

1. Check the tray icon tooltip — it shows the address to use, e.g. `http://mymachine.local:8765`.
2. Open the Shortcuts app, tap the shortcut, and hit **Edit**.
3. In the URL field, set the address to your PC's hostname: `http://mymachine.local:8765/open`.
4. Save. Both devices must be on the same local network.

> **Tip:** the `.local` hostname is stable across reboots and easier to copy than an IP address. If mDNS is not available on your network, fall back to the IPv4 address from `ipconfig`.

## Build

Requires Go 1.21+.

```bash
# Windows — release (no console window) / debug
make build
make build-debug

# Linux — release / debug (headless, no tray icon)
make build-linux
make build-linux-debug
```

## Usage

Run `url-opener.exe` on Windows. It will appear in the notification area (system tray).

**Right-click the tray icon** for two options:
- **Re-run** — restarts the HTTP listener if the port was briefly unavailable
- **Exit** — quits the application

### Run at Startup

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

`url-opener-linux` runs the same HTTP server and opens URLs via `xdg-open`/`x-www-browser`/`www-browser` (whichever is found on `PATH`), so `/open` behaves identically to Windows. There is **no tray icon on Linux yet** — the process just runs in the foreground and logs to stdout until it receives `SIGINT`/`SIGTERM`; there's no "Re-run" menu, so if the port is briefly unavailable you restart the process yourself.

**Run it directly:**

```bash
make build-linux
./url-opener-linux
```

**Run it as a systemd user service** (so it survives terminal close and restarts on crash):

```bash
mkdir -p ~/.local/bin ~/.config/systemd/user
cp url-opener-linux ~/.local/bin/
cp dist/linux/url-opener.service ~/.config/systemd/user/
systemctl --user daemon-reload
systemctl --user enable --now url-opener.service
```

> **Note:** `xdg-open` needs a graphical session (`DISPLAY`/`WAYLAND_DISPLAY` and `DBUS_SESSION_BUS_ADDRESS`). A `systemd --user` service normally inherits these once you're logged into a desktop session; if URLs stop opening after the service starts before login (or over SSH), re-run `systemctl --user daemon-reload` after logging into the desktop, or add `WantedBy=graphical-session.target` bindings appropriate to your distro.

To stop/remove: `systemctl --user disable --now url-opener.service`.

### API

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
