package app

import (
	"fmt"
	"time"

	"git.blackforestbytes.com/BlackForestBytes/goext/exerr"
	"github.com/fsnotify/fsnotify"
	"mikescher.com/kpsync/assets"
)

func (app *Application) runSyncWatcher() error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return exerr.Wrap(err, "failed to init file-watcher").Build()
	}
	defer func() { _ = watcher.Close() }()

	err = watcher.Add(app.config.WorkDir)
	if err != nil {
		return exerr.Wrap(err, "").Build()
	}

	for {
		select {
		case <-app.sigSyncLoopStopChan:
			app.LogInfo("Stopping sync loop (received signal)")
			return nil

		case event := <-watcher.Events:
			if event.Name != app.dbFile {
				continue // no log!! otherwise we end in an endless log-loop
			}
			if !event.Has(fsnotify.Write) && !event.Has(fsnotify.Create) {
				continue // no log!! otherwise we end in an endless log-loop
			}

			app.LogDebug(fmt.Sprintf("Received inotify event: [%s] %s", event.Op.String(), event.Name))

			localCS, err := app.calcLocalChecksum()
			if err != nil {
				app.LogError("Failed to calculate local database checksum", err)
				app.showErrorNotification("KeePassSync: Error", "Failed to calculate checksum")
				continue
			}

			ignoreFN := func() bool {
				app.masterLock.Lock()
				defer app.masterLock.Unlock()

				for _, ign := range app.fileWatcherIgnore {
					if ign.V2 == localCS && time.Since(ign.V1) < 10*time.Second {
						return true
					}
				}

				return false
			}

			if ignoreFN() {
				app.LogDebug("Ignoring file-change - event is explicitly ignored")
				continue
			}

			app.uploadWaiting.Set(true)
			app.setTrayStateDirect("Uploading database (waiting)", assets.IconUpload)
			app.LogInfo(fmt.Sprintf("Database file was modified - requesting upload (currently %d pending requests)", app.uploadDCI.CountPendingRequests()))
			app.uploadDCI.Request()

		case err := <-watcher.Errors:
			app.LogError("Filewatcher reported an error", err)
		}
	}
}
