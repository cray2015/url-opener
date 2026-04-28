package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/getlantern/systray"
)

func trayTooltip() string {
	host, err := os.Hostname()
	if err != nil || host == "" {
		host = "localhost"
	}
	return fmt.Sprintf("URL Opener — listening on http://%s.local:8765", strings.ToLower(host))
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
