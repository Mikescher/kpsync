package app

import (
	"encoding/json"
	"os"
	"strings"
	"time"

	"git.blackforestbytes.com/BlackForestBytes/goext/cryptext"
	"git.blackforestbytes.com/BlackForestBytes/goext/exerr"
	"github.com/shirou/gopsutil/v3/process"
	"mikescher.com/kpsync/log"
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

	return nil
}

func (app *Application) calcLocalChecksum() (string, error) {
	bin, err := os.ReadFile(app.dbFile)
	if err != nil {
		return "", exerr.Wrap(err, "").Build()
	}

	return cryptext.BytesSha256(bin), nil
}

func isKeepassRunning() bool {
	proc, err := process.Processes()
	if err != nil {
		log.LogError("failed to query existing keepass process", err)
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
