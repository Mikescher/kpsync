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

func (app *Application) setTrayState(txt string, icon []byte) func() {
	if !app.trayReady {
		return func() {}
	}

	app.masterLock.Lock()
	defer app.masterLock.Unlock()

	systray.SetIcon(icon)
	systray.SetTooltip(txt)

	fin := func() {
		app.masterLock.Lock()
		defer app.masterLock.Unlock()

		if !app.trayReady {
			return
		}

		systray.SetIcon(assets.IconDefault)
		systray.SetTooltip("Sleeping...")
	}

	return fin
}

func (app *Application) setTrayStateDirect(txt string, icon []byte) {
	if !app.trayReady {
		return
	}

	app.masterLock.Lock()
	defer app.masterLock.Unlock()

	systray.SetIcon(icon)
	systray.SetTooltip(txt)
}
