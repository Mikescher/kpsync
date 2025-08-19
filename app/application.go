package app

import (
	"os"
	"os/signal"
	"sync"
	"syscall"

	"fyne.io/systray"
	"git.blackforestbytes.com/BlackForestBytes/goext/syncext"
	"mikescher.com/kpsync"
	"mikescher.com/kpsync/assets"
	"mikescher.com/kpsync/log"
)

type Application struct {
	masterLock sync.Mutex

	config kpsync.Config

	trayReady     bool
	uploadRunning *syncext.AtomicBool

	sigStopChan chan bool  // keepass exited
	sigErrChan  chan error // fatal error

	sigSyncLoopStopChan chan bool // stop sync loop
	sigTermKeepassChan  chan bool // stop keepass

	dbFile    string
	stateFile string
}

func NewApplication() *Application {

	cfg := kpsync.LoadConfig()

	return &Application{
		masterLock:          sync.Mutex{},
		config:              cfg,
		uploadRunning:       syncext.NewAtomicBool(false),
		trayReady:           false,
		sigStopChan:         make(chan bool, 128),
		sigErrChan:          make(chan error, 128),
		sigSyncLoopStopChan: make(chan bool, 128),
		sigTermKeepassChan:  make(chan bool, 128),
	}
}

func (app *Application) Run() {

	go func() { app.initTray() }()

	go func() {
		err := app.initSync()
		if err != nil {
			app.sigErrChan <- err
			return
		}

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
	case <-sigTerm:

		app.sigSyncLoopStopChan <- true
		app.sigTermKeepassChan <- true
		log.LogInfo("Stopping application (received SIGTERM signal)")

		// TODO term

	case err := <-app.sigErrChan:

		app.sigSyncLoopStopChan <- true
		app.sigTermKeepassChan <- true
		log.LogInfo("Stopping application (received ERROR)")
		log.LogError(err.Error(), err)

		// TODO stop

	case _ = <-app.sigStopChan:

		app.sigSyncLoopStopChan <- true
		app.sigTermKeepassChan <- true
		log.LogInfo("Stopping application (received STOP)")

		// TODO stop
	}

	if app.trayReady {
		systray.Quit()
	}

}
