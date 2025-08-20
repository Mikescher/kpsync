package app

import (
	"fyne.io/systray"
	"mikescher.com/kpsync/assets"
)

func (app *Application) initTray() {

	sigBGStop := make(chan bool, 128)

	trayOnReady := func() {

		app.masterLock.Lock()
		defer app.masterLock.Unlock()

		systray.SetIcon(assets.IconInit)
		systray.SetTitle("KeepassXC Sync")
		app.currSysTrayTooltop = "Initializing..."
		systray.SetTooltip(app.currSysTrayTooltop)

		miSync := systray.AddMenuItem("Sync Now (checked)", "")
		miSyncForce := systray.AddMenuItem("Sync Now (forced)", "")
		miShowLog := systray.AddMenuItem("Show Log", "")
		systray.AddMenuItem("", "")
		app.trayItemChecksum = systray.AddMenuItem("Checksum: {...}", "")
		app.trayItemETag = systray.AddMenuItem("ETag: {...}", "")
		app.trayItemLastModified = systray.AddMenuItem("LastModified: {...}", "")
		systray.AddMenuItem("", "")
		miQuit := systray.AddMenuItem("Quit", "")

		app.LogDebug("SysTray initialized")
		app.LogLine()

		go func() {
			for {
				select {
				case <-miSync.ClickedCh:
					app.LogDebug("SysTray: [Sync Now (checked)] clicked")
					//TODO
				case <-miSyncForce.ClickedCh:
					app.LogDebug("SysTray: [Sync Now (forced)] clicked")
					//TODO
				case <-miShowLog.ClickedCh:
					app.LogDebug("SysTray: [Show Log] clicked")
					//TODO
				case <-miQuit.ClickedCh:
					app.LogDebug("SysTray: [Quit] clicked")
					app.sigManualStopChan <- true
				case <-sigBGStop:
					app.LogDebug("SysTray: Click-Listener goroutine stopped")
					return

				}
			}
		}()

		app.trayReady.Set(true)
	}

	systray.Run(trayOnReady, nil)

	sigBGStop <- true

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
		app.currSysTrayTooltop = "Sleeping..."
		systray.SetTooltip(app.currSysTrayTooltop)

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

func (app *Application) setTrayTooltip(txt string) {
	if !app.trayReady.Get() {
		return
	}

	app.masterLock.Lock()
	defer app.masterLock.Unlock()

	systray.SetTooltip(txt)
	app.currSysTrayTooltop = txt
	systray.SetTooltip(app.currSysTrayTooltop)
}
