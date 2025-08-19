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
	"mikescher.com/kpsync/log"
)

func (app *Application) initSync() error {

	err := os.MkdirAll(app.config.WorkDir, os.ModePerm)
	if err != nil {
		return exerr.Wrap(err, "").Build()
	}

	app.dbFile = path.Join(app.config.WorkDir, path.Base(app.config.LocalFallback))
	app.stateFile = path.Join(app.config.WorkDir, "kpsync.state")

	if isKeepassRunning() {
		log.LogError("keepassxc is already running!", nil)
		return exerr.New(exerr.TypeInternal, "keepassxc is already running").Build()
	}

	state := app.readState()

	needsDownload := true

	if state != nil && fileExists(app.dbFile) {
		localCS, err := app.calcLocalChecksum()
		if err != nil {
			log.LogError("Failed to calculate local database checksum", err)
		} else if localCS == state.Checksum {
			remoteETag, remoteLM, err := app.getRemoteETag()
			if err != nil {
				log.LogError("Failed to get remote ETag", err)
			} else if remoteETag == state.ETag {

				log.LogInfo(fmt.Sprintf("Found local database matching remote database - skip initial download"))
				log.LogInfo(fmt.Sprintf("Checksum (cached)     := %s", state.Checksum))
				log.LogInfo(fmt.Sprintf("Checksum (local)      := %s", localCS))
				log.LogInfo(fmt.Sprintf("ETag (cached)         := %s", state.ETag))
				log.LogInfo(fmt.Sprintf("ETag (remote)         := %s", remoteETag))
				log.LogInfo(fmt.Sprintf("LastModified (cached) := %s", state.LastModified.Format(time.RFC3339)))
				log.LogInfo(fmt.Sprintf("LastModified (remote) := %s", remoteLM.Format(time.RFC3339)))
				needsDownload = false

			}
		}
	}

	if needsDownload {
		err = func() error {
			fin := app.setTrayState("Downloading database", assets.IconDownload)
			defer fin()

			log.LogInfo(fmt.Sprintf("Downloading remote database to %s", app.dbFile))

			etag, lm, sha, sz, err := app.downloadDatabase()
			if err != nil {
				log.LogError("Failed to download remote database", err)
				return exerr.Wrap(err, "Failed to download remote database").Build()
			}

			log.LogInfo(fmt.Sprintf("Downloaded remote database to %s", app.dbFile))
			log.LogInfo(fmt.Sprintf("Checksum     := %s", sha))
			log.LogInfo(fmt.Sprintf("ETag         := %s", etag))
			log.LogInfo(fmt.Sprintf("Size         := %s", langext.FormatBytes(sz)))
			log.LogInfo(fmt.Sprintf("LastModified := %s", lm.Format(time.RFC3339)))

			err = app.saveState(etag, lm, sha, sz)
			if err != nil {
				log.LogError("Failed to save state", err)
				return exerr.Wrap(err, "Failed to save state").Build()
			}

			return nil
		}()
		if err != nil {
			return exerr.Wrap(err, "").Build()
		}
	} else {
		log.LogInfo(fmt.Sprintf("Skip download - use existing local database %s", app.dbFile))
	}

	go func() {

		log.LogInfo("Starting keepassxc...")

		cmd := exec.Command("keepassxc", app.dbFile)

		go func() {
			select {
			case <-app.sigTermKeepassChan:
				log.LogInfo("Received signal to terminate keepassxc")
				if cmd != nil && cmd.Process != nil {
					log.LogInfo(fmt.Sprintf("Terminating keepassxc %d", cmd.Process.Pid))
					err := cmd.Process.Signal(syscall.SIGTERM)
					if err != nil {
						log.LogError("Failed to terminate keepassxc", err)
					} else {
						log.LogInfo("keepassxc terminated successfully")
					}
				} else {
					log.LogInfo("No keepassxc process to terminate")
				}
			}
		}()

		err := cmd.Start()
		if err != nil {
			log.LogError("Failed to start keepassxc", err)
			app.sigErrChan <- exerr.Wrap(err, "Failed to start keepassxc").Build()
			return
		}

		log.LogInfo(fmt.Sprintf("keepassxc started with PID %d", cmd.Process.Pid))

		err = cmd.Wait()

		exitErr := &exec.ExitError{}
		if errors.As(err, &exitErr) {

			log.LogInfo(fmt.Sprintf("keepass exited with code %d", exitErr.ExitCode()))
			app.sigStopChan <- true
			return

		}

		if err != nil {
			log.LogError("Failed to run keepassxc", err)
			app.sigErrChan <- exerr.Wrap(err, "Failed to run keepassxc").Build()
			return
		}

		log.LogInfo("keepassxc exited successfully")
		app.sigStopChan <- true
		return

	}()

	return nil
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
			log.LogInfo("Stopping sync loop (received signal)")
			return nil

		case event := <-watcher.Events:
			log.LogInfo(fmt.Sprintf("inotify event: [%s] %s", event.Op.String(), event.Name))

			if event.Has(fsnotify.Write) && event.Name == app.dbFile {
				func() {
					app.masterLock.Lock()
					app.uploadRunning.Wait(false)
					app.uploadRunning.Set(true)
					app.masterLock.Unlock()

					defer app.uploadRunning.Set(false)

					log.LogInfo("Database file was modified")
					log.LogInfo(fmt.Sprintf("Sleeping for %d seconds", app.config.Debounce))

					time.Sleep(timeext.FromSeconds(app.config.Debounce))

					state := app.readState()
					localCS, err := app.calcLocalChecksum()
					if err != nil {
						log.LogError("Failed to calculate local database checksum", err)
						return
					}

					if localCS == state.Checksum {
						log.LogInfo("Local database still matches remote (via checksum) - no need to upload")
						log.LogInfo(fmt.Sprintf("Checksum (remote/cached) := %s", state.Checksum))
						log.LogInfo(fmt.Sprintf("Checksum (local)         := %s", localCS))
						return
					}

					//TODO upload with IfMatch
				}()
			}
		case err := <-watcher.Errors:
			log.LogError("Filewatcher reported an error", err)
		}
	}
}
