package app

import (
	"encoding/json"
	"os"

	"git.blackforestbytes.com/BlackForestBytes/goext/cryptext"
	"git.blackforestbytes.com/BlackForestBytes/goext/exerr"
)

func fileExists(p string) bool {
	_, err := os.Stat(p)
	if os.IsNotExist(err) {
		return false
	}
	return true
}

type State struct {
	ETag     string `json:"etag"`
	Size     int64  `json:"size"`
	Checksum string `json:"checksum"`
}

func (app *Application) readState() *State {
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

func (app *Application) calcLocalChecksum() (string, error) {
	bin, err := os.ReadFile(app.dbFile)
	if err != nil {
		return "", exerr.Wrap(err, "").Build()
	}

	return cryptext.BytesSha256(bin), nil
}
