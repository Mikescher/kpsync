package app

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"git.blackforestbytes.com/BlackForestBytes/goext/cryptext"
	"git.blackforestbytes.com/BlackForestBytes/goext/exerr"
	"git.blackforestbytes.com/BlackForestBytes/goext/timeext"
	"github.com/shirou/gopsutil/v3/process"
)

func fileExists(p string) bool {
	_, err := os.Stat(p)
	if os.IsNotExist(err) {
		return false
	}
	return true
}

type State struct {
	ETag         string    `json:"etag"`
	Size         int64     `json:"size"`
	Checksum     string    `json:"checksum"`
	LastModified time.Time `json:"lastModified"`
}

func (app *Application) readState() *State {
	app.masterLock.Lock()
	defer app.masterLock.Unlock()

	bin, err := os.ReadFile(app.stateFile)
	if err != nil {
		return nil
	}

	var state State
	err = json.Unmarshal(bin, &state)
	if err != nil {
		return nil
	}

	return &state
}

func (app *Application) saveState(eTag string, lastModified time.Time, checksum string, size int64) error {
	app.masterLock.Lock()
	defer app.masterLock.Unlock()

	obj := State{
		ETag:         eTag,
		Size:         size,
		Checksum:     checksum,
		LastModified: lastModified,
	}

	bin, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return exerr.Wrap(err, "Failed to marshal state").Build()
	}

	err = os.WriteFile(app.stateFile, bin, 0644)
	if err != nil {
		return exerr.Wrap(err, "Failed to write state file").Build()
	}

	if app.trayItemChecksum != nil {
		app.trayItemChecksum.SetTitle(fmt.Sprintf("Checksum: %s", checksum))
	}
	if app.trayItemETag != nil {
		app.trayItemETag.SetTitle(fmt.Sprintf("ETag: %s", eTag))
	}
	if app.trayItemLastModified != nil {
		app.trayItemLastModified.SetTitle(fmt.Sprintf("LastModified: %s", lastModified.In(timeext.TimezoneBerlin).Format(time.RFC3339)))
	}

	return nil
}

func (app *Application) calcLocalChecksum() (string, error) {
	bin, err := os.ReadFile(app.dbFile)
	if err != nil {
		return "", exerr.Wrap(err, "").Build()
	}

	return cryptext.BytesSha256(bin), nil
}

func (app *Application) isKeepassRunning() bool {
	proc, err := process.Processes()
	if err != nil {
		app.LogError("failed to query existing keepass process", err)
		return false
	}

	for _, p := range proc {
		name, err := p.Name()
		if err != nil {
			continue
		}

		if strings.ToLower(name) == "keepass" || strings.ToLower(name) == "keepassxc" {
			return true
		}
	}

	return false
}
