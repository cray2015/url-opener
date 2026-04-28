build:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="-H windowsgui -s -w" -o url-opener.exe .

build-debug:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -tags debug -o url-opener-debug.exe .

changelog:
	git-cliff --config cliff.toml -o CHANGELOG.md
