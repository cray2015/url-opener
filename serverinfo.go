package main

import (
	"fmt"
	"os"
	"strings"
)

// serverAddress returns the local network address other devices should use
// to reach this instance, e.g. "http://mymachine.local:8765".
func serverAddress() string {
	host, err := os.Hostname()
	if err != nil || host == "" {
		host = "localhost"
	}
	return fmt.Sprintf("http://%s.local:8765", strings.ToLower(host))
}
