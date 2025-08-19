package app

import (
	"io"
	"net/http"
	"os"
	"time"

	"git.blackforestbytes.com/BlackForestBytes/goext/cryptext"
	"git.blackforestbytes.com/BlackForestBytes/goext/exerr"
	"git.blackforestbytes.com/BlackForestBytes/goext/timeext"
)

func (app *Application) downloadDatabase() (string, time.Time, string, int64, error) {

	client := http.Client{Timeout: 90 * time.Second}

	req, err := http.NewRequest("GET", app.config.WebDAVURL, nil)
	if err != nil {
		return "", time.Time{}, "", 0, exerr.Wrap(err, "").Build()
	}

	req.SetBasicAuth(app.config.WebDAVUser, app.config.WebDAVPass)

	resp, err := client.Do(req)
	if err != nil {
		return "", time.Time{}, "", 0, exerr.Wrap(err, "Failed to download remote database").Build()
	}

	if resp.StatusCode != http.StatusOK {
		return "", time.Time{}, "", 0, exerr.New(exerr.TypeInternal, "Failed to download remote database").Int("sc", resp.StatusCode).Build()
	}

	bin, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", time.Time{}, "", 0, exerr.Wrap(err, "Failed to read response body").Build()
	}

	etag := resp.Header.Get("ETag")
	if etag == "" {
		return "", time.Time{}, "", 0, exerr.New(exerr.TypeInternal, "ETag header is missing").Build()
	}

	lmStr := resp.Header.Get("Last-Modified")

	lm, err := time.Parse("Mon, 02 Jan 2006 15:04:05 MST", lmStr)
	if err != nil {
		return "", time.Time{}, "", 0, exerr.Wrap(err, "Failed to parse Last-Modified header").Build()
	}

	lm = lm.In(timeext.TimezoneBerlin)

	sha := cryptext.BytesSha256(bin)

	sz := int64(len(bin))

	err = os.WriteFile(app.dbFile, bin, 0644)
	if err != nil {
		return "", time.Time{}, "", 0, exerr.Wrap(err, "Failed to write database file").Build()
	}

	return etag, lm, sha, sz, nil
}

func (app *Application) getRemoteETag() (string, time.Time, error) {
	client := http.Client{Timeout: 90 * time.Second}

	req, err := http.NewRequest("HEAD", app.config.WebDAVURL, nil)
	if err != nil {
		return "", time.Time{}, exerr.Wrap(err, "").Build()
	}

	req.SetBasicAuth(app.config.WebDAVUser, app.config.WebDAVPass)

	resp, err := client.Do(req)
	if err != nil {
		return "", time.Time{}, exerr.Wrap(err, "Failed to download remote database").Build()
	}

	if resp.StatusCode != http.StatusOK {
		return "", time.Time{}, exerr.New(exerr.TypeInternal, "Failed to download remote database").Int("sc", resp.StatusCode).Build()
	}

	etag := resp.Header.Get("ETag")
	if etag == "" {
		return "", time.Time{}, exerr.New(exerr.TypeInternal, "ETag header is missing").Build()
	}

	lmStr := resp.Header.Get("Last-Modified")

	lm, err := time.Parse("Mon, 02 Jan 2006 15:04:05 MST", lmStr)
	if err != nil {
		return "", time.Time{}, exerr.Wrap(err, "Failed to parse Last-Modified header").Build()
	}

	lm = lm.In(timeext.TimezoneBerlin)

	return etag, lm, nil
}
