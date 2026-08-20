//go:build windows

package main

import (
	"fmt"

	"github.com/getlantern/systray"
)

func trayTooltip() string {
	return fmt.Sprintf("URL Opener %s — listening on %s", version, serverAddress())
}

func init() {
	onListenError = func() {
		systray.SetTooltip("Port 8765 in use")
	}
}

func onReady() {
	systray.SetIcon(iconBytes)
	systray.SetTooltip(trayTooltip())

	mRerun := systray.AddMenuItem("Re-run", "Restart HTTP server")
	mExit := systray.AddMenuItem("Exit", "Quit url-opener")

	go func() {
		for {
			select {
			case <-mRerun.ClickedCh:
				restartServer()
			case <-mExit.ClickedCh:
				systray.Quit()
			}
		}
	}()
}
