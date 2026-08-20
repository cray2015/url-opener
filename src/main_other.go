//go:build !windows

package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
)

// No system tray on this platform yet — the server runs in the foreground
// (or under a service manager) until it receives an interrupt/terminate signal.
func main() {
	log.Printf("url-opener starting, listening on %s", serverAddress())
	go startHTTPServer()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig
	log.Printf("shutting down")
}
