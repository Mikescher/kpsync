package app

import (
	"strings"

	"git.blackforestbytes.com/BlackForestBytes/goext/dataext"
	"git.blackforestbytes.com/BlackForestBytes/goext/exerr"
	"git.blackforestbytes.com/BlackForestBytes/goext/langext"
	"git.blackforestbytes.com/BlackForestBytes/goext/termext"
)

func colDefault(v string) string {
	return v
}

func (app *Application) LogFatal(msg string) {
	app.logInternal("[F] ", msg, termext.Red)
	panic(0)
}

func (app *Application) LogFatalErr(msg string, err error) {
	if err != nil {
		app.logInternal("[F] ", msg+"\n"+err.Error()+"\n"+exerr.FromError(err).FormatLog(exerr.LogPrintOverview), termext.Red)
	} else {
		app.logInternal("[F] ", msg, termext.Red)
	}

	panic(0)
}

func (app *Application) LogError(msg string, err error) {
	if err != nil {
		app.logInternal("[E] ", msg+"\n"+err.Error()+"\n"+exerr.FromError(err).FormatLog(exerr.LogPrintOverview), termext.Red)
	} else {
		app.logInternal("[E] ", msg, termext.Red)
	}
}

func (app *Application) LogWarn(msg string) {
	app.logInternal("[W] ", msg, termext.Red)
}

func (app *Application) LogInfo(msg string) {
	app.logInternal("[I] ", msg, colDefault)
}

func (app *Application) LogDebug(msg string) {
	app.logInternal("[D] ", msg, termext.Gray)
}

func (app *Application) logInternal(pf string, msg string, c func(_ string) string) {
	app.logLock.Lock()
	defer app.logLock.Unlock()

	if !termext.SupportsColors() && !app.config.ForceColors {
		c = func(s string) string { return s }
	}

	for i, s := range strings.Split(msg, "\n") {
		if i == 0 {
			println(c(pf + s))
			app.logList = append(app.logList, dataext.NewTriple(strings.TrimSpace(pf), s, c))
			if app.logFile != nil {
				_, err := app.logFile.WriteString(pf + s + "\n")
				if err != nil {
					app.fallbackLog("[!] Failed to write logfile: " + err.Error())
				}
			}
		} else {
			println(c(langext.StrRepeat(" ", len(pf)) + s))
			app.logList = append(app.logList, dataext.NewTriple(strings.TrimSpace(pf), s, c))
			if app.logFile != nil {
				_, err := app.logFile.WriteString(langext.StrRepeat(" ", len(pf)) + s + "\n")
				if err != nil {
					app.fallbackLog("[!] Failed to write logfile: " + err.Error())
				}
			}
		}
	}

	if app.logFile != nil {
		err := app.logFile.Sync()
		if err != nil {
			app.fallbackLog("[!] Failed to flush logfile: " + err.Error())
		}
	}
}

func (app *Application) LogLine() {
	app.logLock.Lock()
	defer app.logLock.Unlock()

	println()

	if app.logFile != nil {
		_, err := app.logFile.WriteString("\n")
		if err != nil {
			app.fallbackLog("[!] Failed to write logfile: " + err.Error())
		}
		err = app.logFile.Sync()
		if err != nil {
			app.fallbackLog("[!] Failed to flush logfile: " + err.Error())
		}
	}

	app.logList = append(app.logList, dataext.NewTriple("", "", func(v string) string { return v }))
}

func (app *Application) fallbackLog(s string) {
	if termext.SupportsColors() || app.config.ForceColors {
		s = termext.Red(s)
	}

	println(s)
}

func (app *Application) writeOutStartupLogs() {
	app.logLock.Lock()
	defer app.logLock.Unlock()

	for _, v := range app.logList {
		_, err := app.logFile.WriteString(v.V1 + " " + v.V2 + "\n")
		if err != nil {
			app.fallbackLog("[!] Failed to write logfile: " + err.Error())
		}
	}

	err := app.logFile.Sync()
	if err != nil {
		app.fallbackLog("[!] Failed to flush logfile: " + err.Error())
	}
}
