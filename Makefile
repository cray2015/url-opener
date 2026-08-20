# Stamped into the binary as main.version. Overridden by the release
# workflow with the tag name; falls back to git describe, then "dev".
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

build:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="-H windowsgui -s -w -X main.version=$(VERSION)" -o url-opener.exe ./src

build-debug:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -tags debug -ldflags="-X main.version=$(VERSION)" -o url-opener-debug.exe ./src

# Linux has no system tray yet (see CLAUDE.md) — it runs headless in the
# foreground / under a service manager. CGO_ENABLED=0 is required: the tray
# library's Linux backend needs cgo + libayatana-appindicator, which this
# build path intentionally does not pull in.
build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w -X main.version=$(VERSION)" -o url-opener-linux ./src

build-linux-debug:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -tags debug -ldflags="-X main.version=$(VERSION)" -o url-opener-linux-debug ./src

changelog:
	git-cliff --config cliff.toml -o CHANGELOG.md

# Install the headless Linux build as a systemd user service. Builds first if
# url-opener-linux is missing. Override BIN_DIR / UNIT_DIR to relocate.
install-linux:
	./dist/linux/install.sh

uninstall-linux:
	./dist/linux/install.sh --uninstall
