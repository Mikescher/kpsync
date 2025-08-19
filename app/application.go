package app

import (
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"fyne.io/systray"
	"git.blackforestbytes.com/BlackForestBytes/goext/syncext"
	"git.blackforestbytes.com/BlackForestBytes/goext/termext"
	"mikescher.com/kpsync/assets"
)

type Application struct {
	masterLock sync.Mutex

	config Config

	trayReady       *syncext.AtomicBool
	uploadRunning   *syncext.AtomicBool
	syncLoopRunning *syncext.AtomicBool
	keepassRunning  *syncext.AtomicBool

	sigStopChan chan bool  // keepass exited
	sigErrChan  chan error // fatal error

	sigSyncLoopStopChan chan bool // stop sync loop
	sigTermKeepassChan  chan bool // stop keepass

	dbFile    string
	stateFile string
}

func NewApplication() *Application {

	app := &Application{
		masterLock:          sync.Mutex{},
		uploadRunning:       syncext.NewAtomicBool(false),
		trayReady:           syncext.NewAtomicBool(false),
		syncLoopRunning:     syncext.NewAtomicBool(false),
		keepassRunning:      syncext.NewAtomicBool(false),
		sigStopChan:         make(chan bool, 128),
		sigErrChan:          make(chan error, 128),
		sigSyncLoopStopChan: make(chan bool, 128),
		sigTermKeepassChan:  make(chan bool, 128),
	}

	app.LogInfo("Starting kpsync...")
	app.LogDebug(fmt.Sprintf("SupportsColors := %v", termext.SupportsColors()))
	app.LogLine()

	return app
}

func (app *Application) Run() {
	var configPath string

	app.config, configPath = app.loadConfig()

	app.LogInfo(fmt.Sprintf("Loaded config from %s", configPath))
	app.LogDebug(fmt.Sprintf("WebDAVURL     := '%s'", app.config.WebDAVURL))
	app.LogDebug(fmt.Sprintf("WebDAVUser    := '%s'", app.config.WebDAVUser))
	app.LogDebug(fmt.Sprintf("WebDAVPass    := '%s'", app.config.WebDAVPass))
	app.LogDebug(fmt.Sprintf("LocalFallback := '%s'", app.config.LocalFallback))
	app.LogDebug(fmt.Sprintf("WorkDir       := '%s'", app.config.WorkDir))
	app.LogDebug(fmt.Sprintf("ForceColors   := %v", app.config.ForceColors))
	app.LogDebug(fmt.Sprintf("Debounce      := %d", app.config.Debounce))
	app.LogDebug(fmt.Sprintf("ForceColors   := %v", app.config.ForceColors))
	app.LogLine()

	go func() { app.initTray() }()

	go func() {
		app.syncLoopRunning = syncext.NewAtomicBool(true)
		defer app.syncLoopRunning.Set(false)

		err := app.initSync()
		if err != nil {
			app.sigErrChan <- err
			return
		}

		go func() {
			app.keepassRunning = syncext.NewAtomicBool(true)
			defer app.keepassRunning.Set(false)

			app.runKeepass()
		}()

		time.Sleep(1 * time.Second)

		app.setTrayStateDirect("Sleeping...", assets.IconDefault)

		err = app.runSyncLoop()
		if err != nil {
			app.sigErrChan <- err
			return
		}
	}()

	sigTerm := make(chan os.Signal, 1)
	signal.Notify(sigTerm, os.Interrupt, syscall.SIGTERM)

	select {
	case <-sigTerm: // kpsync received SIGTERM

		app.LogInfo("Stopping application (received SIGTERM signal)")

		app.stopBackgroundRoutines()

		// TODO try final sync

	case err := <-app.sigErrChan: // fatal error

		app.LogInfo("Stopping application (received ERROR)")

		app.stopBackgroundRoutines()

		app.LogError("Stopped due to error: "+err.Error(), nil)

		// TODO stop?

	case _ = <-app.sigStopChan: // keepass exited

		app.LogInfo("Stopping application (received STOP)")

		app.stopBackgroundRoutines()

		// TODO try final sync

	}
}

func (app *Application) stopBackgroundRoutines() {
	app.LogInfo("Stopping go-routines...")

	app.LogDebug("Stopping systray...")
	systray.Quit()
	app.trayReady.Wait(false)
	app.LogDebug("Stopped systray.")

	if app.uploadRunning.Get() {
		app.LogInfo("Waiting for active upload...")
		app.uploadRunning.Wait(false)
		app.LogInfo("Upload finished.")
	}

	app.LogDebug("Stopping sync-loop...")
	app.sigSyncLoopStopChan <- true
	app.syncLoopRunning.Wait(false)
	app.LogDebug("Stopped sync-loop.")

	app.LogDebug("Stopping keepass...")
	app.sigTermKeepassChan <- true
	app.keepassRunning.Wait(false)
	app.LogDebug("Stopped keepass.")

	app.LogLine()
}
