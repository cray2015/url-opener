build:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="-H windowsgui -s -w" -o url-opener.exe .

build-debug:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -tags debug -o url-opener-debug.exe .

# Linux has no system tray yet (see CLAUDE.md) — it runs headless in the
# foreground / under a service manager. CGO_ENABLED=0 is required: the tray
# library's Linux backend needs cgo + libayatana-appindicator, which this
# build path intentionally does not pull in.
build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o url-opener-linux .

build-linux-debug:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -tags debug -o url-opener-linux-debug .

changelog:
	git-cliff --config cliff.toml -o CHANGELOG.md
