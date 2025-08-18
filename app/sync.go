package app

import (
	"fmt"
	"os"
	"path"

	"git.blackforestbytes.com/BlackForestBytes/goext/exerr"
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

	state := app.readState()

	needsDownload := true

	if state != nil && fileExists(app.dbFile) {
		localCS, err := app.calcLocalChecksum()
		if err != nil {
			log.LogError("Failed to calculate local database checksum", err)
		} else if localCS == state.Checksum {
			remoteETag, err := app.getRemoteETag()
			if err != nil {
				log.LogError("Failed to get remote ETag", err)
			} else if remoteETag == state.ETag {

				log.LogInfo(fmt.Sprintf("Found local database matching remote database - skip initial download"))
				log.LogInfo(fmt.Sprintf("Checksum (cached)  := %s", state.Checksum))
				log.LogInfo(fmt.Sprintf("Checksum (local)   := %s", localCS))
				log.LogInfo(fmt.Sprintf("ETag (cached)      := %s", state.ETag))
				log.LogInfo(fmt.Sprintf("ETag (remote)      := %s", remoteETag))
				needsDownload = false

			}
		}
	}

	if needsDownload {
		func() {
			fin := app.setTrayState("Downloading database", assets.IconDefault)
			defer fin()

			log.LogInfo(fmt.Sprintf("Downloading remote database to %s", app.dbFile))

			etag, err := app.downloadDatabase()
			if err != nil {
				log.LogError("Failed to download remote database", err)
				app.sigErrChan <- exerr.Wrap(err, "Failed to download remote database").Build()
				return
			}
		}()
	}
}

func (app *Application) runSyncLoop() error {

}
