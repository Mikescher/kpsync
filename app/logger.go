package app

import (
	"strings"

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
		panic(0)
	} else {
		app.logInternal("[F] ", msg, termext.Red)
		panic(0)
	}
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
	if !termext.SupportsColors() && !app.config.ForceColors {
		c = func(s string) string { return s }
	}

	for i, s := range strings.Split(msg, "\n") {
		if i == 0 {
			println(c(pf + s))
		} else {
			println(c(langext.StrRepeat(" ", len(pf)) + s))
		}
	}
}

func (app *Application) LogLine() {
	println()
}
