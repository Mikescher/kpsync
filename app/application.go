package app

import (
	"os"
	"os/signal"
	"syscall"

	"fyne.io/systray"
	"mikescher.com/kpsync"
)

type Application struct {
	config kpsync.Config

	trayReady   bool
	sigStopChan chan bool
	sigErrChan  chan error

	dbFile    string
	stateFile string
}

func NewApplication() *Application {

	cfg := kpsync.LoadConfig()

	return &Application{
		config:      cfg,
		trayReady:   false,
		sigStopChan: make(chan bool, 128),
		sigErrChan:  make(chan error, 128),
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

		// TODO term

	case _ = <-app.sigErrChan:

		// TODO stop

	case _ = <-app.sigStopChan:

		// TODO stop
	}

	if app.trayReady {
		systray.Quit()
	}

}
