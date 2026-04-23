package main

import "github.com/getlantern/systray"

func main() {
	go startHTTPServer()
	systray.Run(onReady, onExit)
}

func onExit() {}
