package app

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"syscall"
	"time"

	"git.blackforestbytes.com/BlackForestBytes/goext/exerr"
	"git.blackforestbytes.com/BlackForestBytes/goext/langext"
	"git.blackforestbytes.com/BlackForestBytes/goext/timeext"
	"github.com/fsnotify/fsnotify"
	"mikescher.com/kpsync/assets"
)

func (app *Application) initSync() error {

	err := os.MkdirAll(app.config.WorkDir, os.ModePerm)
	if err != nil {
		return exerr.Wrap(err, "").Build()
	}

	app.dbFile = path.Join(app.config.WorkDir, path.Base(app.config.LocalFallback))
	app.stateFile = path.Join(app.config.WorkDir, "kpsync.state")

	if app.isKeepassRunning() {
		app.LogError("keepassxc is already running!", nil)
		return exerr.New(exerr.TypeInternal, "keepassxc is already running").Build()
	}

	state := app.readState()

	needsDownload := true

	if state != nil && fileExists(app.dbFile) {
		localCS, err := app.calcLocalChecksum()
		if err != nil {
			app.LogError("Failed to calculate local database checksum", err)
		} else if localCS == state.Checksum {
			remoteETag, remoteLM, err := app.getRemoteETag()
			if err != nil {
				app.LogError("Failed to get remote ETag", err)
			} else if remoteETag == state.ETag {

				app.LogInfo(fmt.Sprintf("Found local database matching remote database - skip initial download"))
				app.LogDebug(fmt.Sprintf("Checksum (cached)     := %s", state.Checksum))
				app.LogDebug(fmt.Sprintf("Checksum (local)      := %s", localCS))
				app.LogDebug(fmt.Sprintf("ETag (cached)         := %s", state.ETag))
				app.LogDebug(fmt.Sprintf("ETag (remote)         := %s", remoteETag))
				app.LogDebug(fmt.Sprintf("LastModified (cached) := %s", state.LastModified.Format(time.RFC3339)))
				app.LogDebug(fmt.Sprintf("LastModified (remote) := %s", remoteLM.Format(time.RFC3339)))
				app.LogLine()
				needsDownload = false

			}
		}
	}

	if needsDownload {
		err = func() error {
			fin := app.setTrayState("Downloading database", assets.IconDownload)
			defer fin()

			app.LogInfo(fmt.Sprintf("Downloading remote database to %s", app.dbFile))

			etag, lm, sha, sz, err := app.downloadDatabase()
			if err != nil {
				app.LogError("Failed to download remote database", err)
				return exerr.Wrap(err, "Failed to download remote database").Build()
			}

			app.LogInfo(fmt.Sprintf("Downloaded remote database to %s", app.dbFile))
			app.LogInfo(fmt.Sprintf("Checksum     := %s", sha))
			app.LogInfo(fmt.Sprintf("ETag         := %s", etag))
			app.LogInfo(fmt.Sprintf("Size         := %s (%d)", langext.FormatBytes(sz), sz))
			app.LogInfo(fmt.Sprintf("LastModified := %s", lm.Format(time.RFC3339)))

			err = app.saveState(etag, lm, sha, sz)
			if err != nil {
				app.LogError("Failed to save state", err)
				return exerr.Wrap(err, "Failed to save state").Build()
			}

			app.LogLine()

			return nil
		}()
		if err != nil {
			return exerr.Wrap(err, "").Build()
		}
	} else {
		app.LogInfo(fmt.Sprintf("Skip download - use existing local database %s", app.dbFile))
		app.LogLine()
	}

	return nil
}

func (app *Application) runKeepass() {
	app.LogInfo("Starting keepassxc...")

	cmd := exec.Command("keepassxc", app.dbFile)

	go func() {
		select {
		case <-app.sigTermKeepassChan:
			app.LogInfo("Received signal to terminate keepassxc")
			if cmd != nil && cmd.Process != nil {
				app.LogInfo(fmt.Sprintf("Terminating keepassxc %d", cmd.Process.Pid))
				err := cmd.Process.Signal(syscall.SIGTERM)
				if err != nil {
					app.LogError("Failed to terminate keepassxc", err)
				} else {
					app.LogInfo("keepassxc terminated successfully")
				}
			} else {
				app.LogInfo("No keepassxc process to terminate")
			}
		}
	}()

	err := cmd.Start()
	if err != nil {
		app.LogError("Failed to start keepassxc", err)
		app.sigErrChan <- exerr.Wrap(err, "Failed to start keepassxc").Build()
		return
	}

	app.LogInfo(fmt.Sprintf("keepassxc started with PID %d", cmd.Process.Pid))
	app.LogLine()

	err = cmd.Wait()

	exitErr := &exec.ExitError{}
	if errors.As(err, &exitErr) {

		app.LogInfo(fmt.Sprintf("keepass exited with code %d", exitErr.ExitCode()))
		app.sigStopChan <- true
		return

	}

	if err != nil {
		app.LogError("Failed to run keepassxc", err)
		app.sigErrChan <- exerr.Wrap(err, "Failed to run keepassxc").Build()
		return
	}

	app.LogInfo("keepassxc exited successfully")
	app.LogLine()

	app.sigStopChan <- true
	return

}

func (app *Application) runSyncLoop() error {
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
			app.LogDebug(fmt.Sprintf("Received inotify event: [%s] %s", event.Op.String(), event.Name))

			if !event.Has(fsnotify.Write) {
				app.LogDebug("Ignoring event - not a write event")
				app.LogLine()
				continue
			}

			if event.Name != app.dbFile {
				app.LogDebug(fmt.Sprintf("Ignoring event - not the database file (%s)", app.dbFile))
				app.LogLine()
				continue
			}

			func() {
				app.masterLock.Lock()
				app.uploadRunning.Wait(false)
				app.uploadRunning.Set(true)
				app.masterLock.Unlock()

				defer app.uploadRunning.Set(false)

				app.LogInfo("Database file was modified")
				app.LogInfo(fmt.Sprintf("Sleeping for %d seconds", app.config.Debounce))

				time.Sleep(timeext.FromSeconds(app.config.Debounce))

				state := app.readState()
				localCS, err := app.calcLocalChecksum()
				if err != nil {
					app.LogError("Failed to calculate local database checksum", err)
					app.showErrorNotification("Failed to calculate local database checksum")
					return
				}

				if localCS == state.Checksum {
					app.LogInfo("Local database still matches remote (via checksum) - no need to upload")
					app.LogInfo(fmt.Sprintf("Checksum (remote/cached) := %s", state.Checksum))
					app.LogInfo(fmt.Sprintf("Checksum (local)         := %s", localCS))
					return
				}

				etag, lm, sha, sz, err := app.uploadDatabase(langext.Ptr(state.ETag))
				if errors.Is(err, ETagConflictError) {

					//TODO - choice notification

				} else if err != nil {
					app.LogError("Failed to upload remote database", err)
					app.showErrorNotification("Failed to upload remote database")
					return
				}

				app.LogInfo(fmt.Sprintf("Uploaded database to remote"))
				app.LogDebug(fmt.Sprintf("Checksum     := %s", sha))
				app.LogDebug(fmt.Sprintf("ETag         := %s", etag))
				app.LogDebug(fmt.Sprintf("Size         := %s (%d)", langext.FormatBytes(sz), sz))
				app.LogDebug(fmt.Sprintf("LastModified := %s", lm.Format(time.RFC3339)))

				err = app.saveState(etag, lm, sha, sz)
				if err != nil {
					app.LogError("Failed to save state", err)
					app.showErrorNotification("Failed to save state")
					return
				}

				app.showSuccessNotification("Uploaded database successfully")

				app.LogLine()
			}()

		case err := <-watcher.Errors:
			app.LogError("Filewatcher reported an error", err)
		}
	}
}
