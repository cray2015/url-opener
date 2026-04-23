# url-opener

A lightweight Windows desktop utility that runs a local HTTP server on port **8765** and opens URLs in the system default browser on demand. It lives silently in the Windows system tray with no console window.

## Companion iOS Shortcut

An Apple Shortcut is available to trigger url-opener from any Apple device on the same network:

**[Download Shortcut](https://www.icloud.com/shortcuts/ac5982d6058c4924bfa5214cc10b3bcc)**

The shortcut sends a `POST /open` request to the machine running url-opener, letting you push a URL from your iPhone/iPad/Mac directly into the Windows browser.

**After installing, update the IP address in the shortcut:**

1. Find your Windows PC's local IP — run `ipconfig` in Command Prompt and note the IPv4 address (e.g. `192.168.1.50`).
2. Open the Shortcuts app, tap the shortcut, and hit **Edit**.
3. In the URL field, replace the placeholder IP with your PC's IP: `http://192.168.1.50:8765/open`.
4. Save. Both devices must be on the same local network.

## Build

Requires Go 1.21+. All builds target Windows (`GOOS=windows GOARCH=amd64`).

```bash
# Release — no console window
make build

# Debug — console window with log output
make build-debug
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
