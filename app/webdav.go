package app

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"git.blackforestbytes.com/BlackForestBytes/goext/cryptext"
	"git.blackforestbytes.com/BlackForestBytes/goext/exerr"
	"git.blackforestbytes.com/BlackForestBytes/goext/timeext"
)

var ETagConflictError = errors.New("ETag conflict")

func (app *Application) downloadDatabase() (string, time.Time, string, int64, error) {

	prevTT := app.currSysTrayTooltop
	defer app.setTrayTooltip(prevTT)

	client := http.Client{Timeout: 90 * time.Second}

	req, err := http.NewRequest("GET", app.config.WebDAVURL, nil)
	if err != nil {
		return "", time.Time{}, "", 0, exerr.Wrap(err, "").Build()
	}

	req.SetBasicAuth(app.config.WebDAVUser, app.config.WebDAVPass)

	t0 := time.Now()
	app.LogDebug(fmt.Sprintf("{HTTP} Starting WebDAV download..."))

	resp, err := client.Do(req)
	if err != nil {
		return "", time.Time{}, "", 0, exerr.Wrap(err, "Failed to download remote database").Build()
	}

	if resp.StatusCode != http.StatusOK {
		return "", time.Time{}, "", 0, exerr.New(exerr.TypeInternal, "Failed to download remote database").Int("sc", resp.StatusCode).Build()
	}

	currTT := ""
	progressCallback := func(current int64, total int64) {
		newTT := fmt.Sprintf("Downloading (%.0f%%)", float64(current)/float64(total)*100)
		if currTT != newTT {
			app.setTrayTooltip(newTT)
			currTT = newTT
		}
	}

	bin, err := ReadAllWithProgress(resp.Body, resp.ContentLength, progressCallback)
	if err != nil {
		return "", time.Time{}, "", 0, exerr.Wrap(err, "Failed to read response body").Build()
	}

	app.LogDebug(fmt.Sprintf("{HTTP} Finished WebDAV download in %s", time.Since(t0)))

	etag, lm, err := app.parseHeader(resp)
	if err != nil {
		return "", time.Time{}, "", 0, exerr.Wrap(err, "").Build()
	}

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

	t0 := time.Now()
	app.LogDebug(fmt.Sprintf("{HTTP} Starting WebDAV HEAD-request..."))

	resp, err := client.Do(req)
	if err != nil {
		return "", time.Time{}, exerr.Wrap(err, "Failed to download remote database").Build()
	}

	app.LogDebug(fmt.Sprintf("{HTTP} Finished WebDAV request in %s", time.Since(t0)))

	if resp.StatusCode != http.StatusOK {
		return "", time.Time{}, exerr.New(exerr.TypeInternal, "Failed to download remote database").Int("sc", resp.StatusCode).Build()
	}

	etag, lm, err := app.parseHeader(resp)
	if err != nil {
		return "", time.Time{}, exerr.Wrap(err, "").Build()
	}

	return etag, lm, nil
}

func (app *Application) uploadDatabase(etagIfMatch *string) (string, time.Time, string, int64, error) {

	prevTT := app.currSysTrayTooltop
	defer app.setTrayTooltip(prevTT)

	client := http.Client{Timeout: 90 * time.Second}

	req, err := http.NewRequest("PUT", app.config.WebDAVURL, nil)
	if err != nil {
		return "", time.Time{}, "", 0, exerr.Wrap(err, "").Build()
	}

	req.SetBasicAuth(app.config.WebDAVUser, app.config.WebDAVPass)

	if etagIfMatch != nil {
		req.Header.Set("If-Match", "\""+*etagIfMatch+"\"")
	}

	bin, err := os.ReadFile(app.dbFile)
	if err != nil {
		return "", time.Time{}, "", 0, exerr.Wrap(err, "Failed to read database file").Build()
	}

	sha := cryptext.BytesSha256(bin)

	sz := int64(len(bin))

	currTT := ""
	progressCallback := func(current int64, total int64) {
		newTT := fmt.Sprintf("Uploading (%.0f%%)", float64(current)/float64(total)*100)
		if currTT != newTT {
			app.setTrayTooltip(newTT)
			currTT = newTT
		}
	}

	req.ContentLength = sz
	req.Body = NewProgressReader(bytes.NewReader(bin), int64(len(bin)), progressCallback)

	t0 := time.Now()
	app.LogDebug(fmt.Sprintf("{HTTP} Starting WebDAV upload..."))

	resp, err := client.Do(req)
	if err != nil {
		return "", time.Time{}, "", 0, exerr.Wrap(err, "Failed to upload remote database").Build()
	}
	defer func() { _ = resp.Body.Close() }()

	app.LogDebug(fmt.Sprintf("{HTTP} Finished WebDAV upload in %s", time.Since(t0)))

	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusNoContent {

		etag, lm, err := app.parseHeader(resp)
		if err != nil {
			return "", time.Time{}, "", 0, exerr.Wrap(err, "").Build()
		}

		return etag, lm, sha, sz, nil
	}

	if resp.StatusCode == http.StatusPreconditionFailed {
		return "", time.Time{}, "", 0, ETagConflictError
	}

	return "", time.Time{}, "", 0, exerr.New(exerr.TypeInternal, fmt.Sprintf("Failed to upload remote database (statuscode: %d)", resp.StatusCode)).Int("sc", resp.StatusCode).Build()
}

func (app *Application) parseHeader(resp *http.Response) (string, time.Time, error) {
	var err error
	
	etag := resp.Header.Get("ETag")
	if etag == "" {
		return "", time.Time{}, exerr.New(exerr.TypeInternal, "ETag header is missing").Build()
	}
	etag = strings.Trim(etag, "\"\r\n ")

	var lm time.Time

	lmStr := resp.Header.Get("Last-Modified")
	if lmStr == "" {
		lm = time.Now().In(timeext.TimezoneBerlin)
		app.LogDebug("Last-Modified header is missing, using current time as fallback")
	} else {
		lm, err = time.Parse("Mon, 02 Jan 2006 15:04:05 MST", lmStr)
		if err != nil {
			return "", time.Time{}, exerr.Wrap(err, "Failed to parse Last-Modified header").Build()
		}
		lm = lm.In(timeext.TimezoneBerlin)
	}

	return etag, lm, nil
}
