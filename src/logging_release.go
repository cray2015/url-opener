//go:build !debug

package main

import (
	"log"
	"os"
	"path/filepath"
)

func init() {
	appData := os.Getenv("APPDATA")
	if appData == "" {
		return
	}
	dir := filepath.Join(appData, "url-opener")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return
	}
	f, err := os.OpenFile(filepath.Join(dir, "app.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	log.SetOutput(f)
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}
