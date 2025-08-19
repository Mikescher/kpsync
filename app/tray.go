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

		app.LogDebug("SysTray initialized")
		app.LogLine()

		app.trayReady.Set(true)
	}

	systray.Run(trayOnReady, nil)

	app.LogDebug("SysTray stopped")
	app.LogLine()

	app.trayReady.Set(false)
}

func (app *Application) setTrayState(txt string, icon []byte) func() {
	if !app.trayReady.Get() {
		return func() {}
	}

	app.masterLock.Lock()
	defer app.masterLock.Unlock()

	systray.SetIcon(icon)
	systray.SetTooltip(txt)

	var finDone = false

	fin := func() {
		app.masterLock.Lock()
		defer app.masterLock.Unlock()

		if finDone {
			return
		}

		if !app.trayReady.Get() {
			return
		}

		systray.SetIcon(assets.IconDefault)
		systray.SetTooltip("Sleeping...")

		finDone = true
	}

	return fin
}

func (app *Application) setTrayStateDirect(txt string, icon []byte) {
	if !app.trayReady.Get() {
		return
	}

	app.masterLock.Lock()
	defer app.masterLock.Unlock()

	systray.SetIcon(icon)
	systray.SetTooltip(txt)
}
