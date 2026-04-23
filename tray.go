package main

import "github.com/getlantern/systray"

func onReady() {
	systray.SetIcon(iconBytes)
	systray.SetTooltip("URL Opener — listening on :8765")

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
