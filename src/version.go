package main

// version is stamped at build time by the Makefile via
// -ldflags "-X main.version=...". A plain `go build ./src` leaves it "dev".
var version = "dev"
