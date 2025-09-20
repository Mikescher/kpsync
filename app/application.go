package app

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path"
	"sync"
	"syscall"
	"time"

	"fyne.io/systray"
	"git.blackforestbytes.com/BlackForestBytes/goext/dataext"
	"git.blackforestbytes.com/BlackForestBytes/goext/langext"
	"git.blackforestbytes.com/BlackForestBytes/goext/mathext"
	"git.blackforestbytes.com/BlackForestBytes/goext/syncext"
	"git.blackforestbytes.com/BlackForestBytes/goext/termext"
	"git.blackforestbytes.com/BlackForestBytes/goext/timeext"
	"mikescher.com/kpsync/assets"
)

type Application struct {
	masterLock sync.Mutex

	logLock        sync.Mutex
	logFile        *os.File // file to write logs to, if set
	logList        []LogMessage
	logBroadcaster *dataext.PubSub[string, LogMessage]

	config Config

	trayReady       *syncext.AtomicBool
	uploadWaiting   *syncext.AtomicBool
	uploadActive    *syncext.AtomicBool
	syncLoopRunning *syncext.AtomicBool
	keepassRunning  *syncext.AtomicBool

	fileWatcherIgnore []dataext.Tuple[time.Time, string]

	sigKPExitChan     chan bool  // keepass exited
	sigManualStopChan chan bool  // manual stop
	sigErrChan        chan error // fatal error

	sigSyncLoopStopChan chan bool // stop sync loop
	sigTermKeepassChan  chan bool // stop keepass

	dbFile    string
	stateFile string

	currSysTrayTooltip string

	uploadDCI *dataext.DelayedCombiningInvoker

	trayItemChecksum     *systray.MenuItem
	trayItemETag         *systray.MenuItem
	trayItemLastModified *systray.MenuItem
}

func NewApplication() *Application {

	app := &Application{
		masterLock:          sync.Mutex{},
		logLock:             sync.Mutex{},
		logList:             make([]LogMessage, 0, 1024),
		logBroadcaster:      dataext.NewPubSub[string, LogMessage](128),
		uploadWaiting:       syncext.NewAtomicBool(false),
		uploadActive:        syncext.NewAtomicBool(false),
		trayReady:           syncext.NewAtomicBool(false),
		syncLoopRunning:     syncext.NewAtomicBool(false),
		keepassRunning:      syncext.NewAtomicBool(false),
		fileWatcherIgnore:   make([]dataext.Tuple[time.Time, string], 0, 128),
		sigKPExitChan:       make(chan bool, 128),
		sigManualStopChan:   make(chan bool, 128),
		sigErrChan:          make(chan error, 128),
		sigSyncLoopStopChan: make(chan bool, 128),
		sigTermKeepassChan:  make(chan bool, 128),
	}

	app.LogInfo(fmt.Sprintf("Starting kpsync {%s} ...", time.Now().In(timeext.TimezoneBerlin).Format(time.RFC3339)))
	app.LogLine()

	app.LogDebug(fmt.Sprintf("SupportsColors := %v", termext.SupportsColors()))
	app.LogLine()

	return app
}

func (app *Application) Run() {
	var configPath string
	var err error

	app.config, configPath = app.loadConfig()

	app.LogInfo(fmt.Sprintf("Loaded config from %s", configPath))
	app.LogDebug(fmt.Sprintf("WebDAVURL     := '%s'", app.config.WebDAVURL))
	app.LogDebug(fmt.Sprintf("WebDAVUser    := '%s'", app.config.WebDAVUser))
	app.LogDebug(fmt.Sprintf("WebDAVPass    := '%s'", app.config.WebDAVPass))
	app.LogDebug(fmt.Sprintf("LocalFallback := '%s'", langext.Coalesce(app.config.LocalFallback, "<null>")))
	app.LogDebug(fmt.Sprintf("WorkDir       := '%s'", app.config.WorkDir))
	app.LogDebug(fmt.Sprintf("Debounce      := %d ms", app.config.Debounce))
	app.LogDebug(fmt.Sprintf("ForceColors   := %v", app.config.ForceColors))
	app.LogLine()

	app.logFile, err = os.OpenFile(path.Join(app.config.WorkDir, "kpsync.log"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		app.LogFatalErr("Failed to open log file", err)
	}
	defer func() {
		if err := app.logFile.Close(); err != nil {
			app.fallbackLog("Failed to close log file: " + err.Error())
		}
	}()
	app.writeOutStartupLogs()

	go func() { app.initTray() }()

	if app.config.LocalFallback != nil {
		if _, err := os.Stat(*app.config.LocalFallback); errors.Is(err, os.ErrNotExist) {
			app.config.LocalFallback = nil

			app.LogError(fmt.Sprintf("Configured local-fallback '%s' not found - disabling.", *app.config.LocalFallback), nil)

			app.showErrorNotification("Local fallback database not found", fmt.Sprintf("Configured local-fallback '%s' not found - fallback option won't be available.", *app.config.LocalFallback))
		}
	}

	debounce := timeext.FromMilliseconds(app.config.Debounce)
	app.uploadDCI = dataext.NewDelayedCombiningInvoker(app.runDBUpload, debounce, mathext.Max(45*time.Second, debounce*3))

	app.uploadDCI.RegisterOnRequest(func(_ int, _ bool) { app.uploadWaiting.Set(app.uploadDCI.HasPendingRequests()) })
	app.uploadDCI.RegisterOnExecutionDone(func() { app.uploadWaiting.Set(app.uploadDCI.HasPendingRequests()) })

	go func() {
		app.syncLoopRunning.Set(true)
		defer app.syncLoopRunning.Set(false)

		isr, err := app.initSync()
		if err != nil {
			app.sigErrChan <- err
			return
		}

		if isr == InitSyncResponseAbort {
			app.sigManualStopChan <- true
			return
		} else if isr == InitSyncResponseOkay {

			go func() {
				app.keepassRunning.Set(true)
				defer app.keepassRunning.Set(false)

				app.runKeepass(false)
			}()

			time.Sleep(1 * time.Second)

			app.setTrayStateDirect("Sleeping...", assets.IconDefault)

			err = app.runSyncWatcher()
			if err != nil {
				app.sigErrChan <- err
				return
			}

		} else if isr == InitSyncResponseFallback && app.config.LocalFallback != nil {

			app.LogInfo(fmt.Sprintf("Starting KeepassXC with local fallback database (without sync loop!)"))
			app.LogDebug(fmt.Sprintf("DB-Path := '%s'", *app.config.LocalFallback))

			go func() {
				app.keepassRunning.Set(true)
				defer app.keepassRunning.Set(false)

				app.runKeepass(true)
			}()

			app.setTrayStateDirect("Sleeping...", assets.IconDefault)

		} else {
			app.LogError("Unknown InitSyncResponse: "+string(isr), nil)
			app.sigErrChan <- fmt.Errorf("unknown InitSyncResponse: %s", isr)
			return
		}

	}()

	sigTerm := make(chan os.Signal, 1)
	signal.Notify(sigTerm, os.Interrupt, syscall.SIGTERM)

	select {
	case <-sigTerm: // kpsync received SIGTERM

		app.LogInfo("Stopping application (received SIGTERM signal)")

		app.stopBackgroundRoutines()

		app.runFinalSync()

		return

	case err := <-app.sigErrChan: // fatal error

		app.LogInfo("Stopping application (received ERROR)")

		app.stopBackgroundRoutines()

		app.LogError("Stopped due to error: "+err.Error(), nil)

		return

	case <-app.sigManualStopChan: // manual

		app.LogInfo("Stopping application (manual)")

		app.stopBackgroundRoutines()

		return

	case _ = <-app.sigKPExitChan: // keepass exited

		app.LogInfo("Stopping application (received STOP)")

		app.stopBackgroundRoutines()

		app.runFinalSync()

		return

	}
}

func (app *Application) stopBackgroundRoutines() {
	app.LogInfo("Stopping go-routines...")

	app.LogDebug("Stopping systray...")
	systray.Quit()
	app.trayReady.Wait(false)
	app.LogDebug("Stopped systray.")

	if app.uploadWaiting.Get() {
		app.LogInfo("Triggering pending upload immediately...")
		app.uploadDCI.ExecuteNow()
	}

	if app.uploadActive.Get() {
		app.LogInfo("Waiting for active upload...")
		app.uploadActive.Wait(false)
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
