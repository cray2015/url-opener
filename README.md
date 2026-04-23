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
