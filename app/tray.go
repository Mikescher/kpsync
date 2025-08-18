package app

import (
	"fyne.io/systray"
	"mikescher.com/kpsync/assets"
)

func (app *Application) initTray() {

	trayOnReady := func() {

		systray.SetIcon(assets.IconInit)
		systray.SetTitle("KeepassXC Sync")
		systray.SetTooltip("Initializing...")

		app.trayReady = true
	}

	systray.Run(trayOnReady, nil)
}
