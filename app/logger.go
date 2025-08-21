package app

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"
	"syscall"
	"time"

	"git.blackforestbytes.com/BlackForestBytes/goext/exerr"
	"git.blackforestbytes.com/BlackForestBytes/goext/langext"
	"git.blackforestbytes.com/BlackForestBytes/goext/termext"
)

type LogMessage struct {
	Line      int
	Prefix    string
	Message   string
	ColorFunc func(string) string
}

func colDefault(v string) string {
	return v
}

func (app *Application) LogFatal(msg string) {
	app.logInternal("[F]", msg, termext.Red)
	panic(0)
}

func (app *Application) LogFatalErr(msg string, err error) {
	if err != nil {
		app.logInternal("[F]", msg+"\n"+err.Error()+"\n"+exerr.FromError(err).FormatLog(exerr.LogPrintOverview), termext.Red)
	} else {
		app.logInternal("[F]", msg, termext.Red)
	}

	panic(0)
}

func (app *Application) LogError(msg string, err error) {
	if err != nil {
		app.logInternal("[E]", msg+"\n"+err.Error()+"\n"+exerr.FromError(err).FormatLog(exerr.LogPrintOverview), termext.Red)
	} else {
		app.logInternal("[E]", msg, termext.Red)
	}
}

func (app *Application) LogWarn(msg string) {
	app.logInternal("[W]", msg, termext.Red)
}

func (app *Application) LogInfo(msg string) {
	app.logInternal("[I]", msg, colDefault)
}

func (app *Application) LogDebug(msg string) {
	app.logInternal("[D]", msg, termext.Gray)
}

func (app *Application) logInternal(pf string, msg string, cf func(_ string) string) {
	app.logLock.Lock()
	defer app.logLock.Unlock()

	c := cf

	if !termext.SupportsColors() && !app.config.ForceColors {
		c = func(s string) string { return s }
	}

	for i, s := range strings.Split(msg, "\n") {
		if i == 0 {
			println(c(pf + " " + s))
			app.logList = append(app.logList, LogMessage{i, pf, s, cf})
			if app.logFile != nil {
				_, err := app.logFile.WriteString(pf + " " + s + "\n")
				if err != nil {
					app.fallbackLog("[!] Failed to write logfile: " + err.Error())
				}
			}
			app.logBroadcaster.Publish("", LogMessage{i, pf, s, cf})
		} else {
			println(c(langext.StrRepeat(" ", len(pf)+1) + s))
			app.logList = append(app.logList, LogMessage{i, pf, s, cf})
			if app.logFile != nil {
				_, err := app.logFile.WriteString(langext.StrRepeat(" ", len(pf)+1) + s + "\n")
				if err != nil {
					app.fallbackLog("[!] Failed to write logfile: " + err.Error())
				}
			}
			app.logBroadcaster.Publish("", LogMessage{i, pf, s, cf})
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

	app.logList = append(app.logList, LogMessage{0, "", "", func(v string) string { return v }})

	app.logBroadcaster.Publish("", LogMessage{0, "", "", func(v string) string { return v }})
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
		_, err := app.logFile.WriteString(v.Prefix + " " + v.Message + "\n")
		if err != nil {
			app.fallbackLog("[!] Failed to write logfile: " + err.Error())
		}
	}

	err := app.logFile.Sync()
	if err != nil {
		app.fallbackLog("[!] Failed to flush logfile: " + err.Error())
	}
}

func (app *Application) openLogFile() {
	err := exec.Command("xdg-open", path.Join(app.config.WorkDir, "kpsync.log")).Start()
	if err != nil {
		app.LogError("Failed to open log file with xdg-open", err)
		return
	}
}

func (app *Application) openLogFifo() {
	filePath := path.Join(app.config.WorkDir, fmt.Sprintf("kpsync.%s.fifo", langext.RandBase62(8)))

	app.LogDebug(fmt.Sprintf("Creating fifo file at '%s'", filePath))
	err := syscall.Mkfifo(filePath, 0640)
	if err != nil {
		app.LogError("Failed to create fifo file", err)
		return
	}

	defer func() { _ = syscall.Unlink(filePath) }()

	listernerStopSig := make(chan bool, 8)

	go func() {

		app.LogDebug(fmt.Sprintf("Opening fifo file '%s'", filePath))

		f, err := os.OpenFile(filePath, os.O_WRONLY, 0600)
		if err != nil {
			app.LogError("Failed to open fifo file", err)
			return
		}
		defer func() { _ = f.Close() }()

		app.LogDebug(fmt.Sprintf("Initializing fifo with %d past entries", len(app.logList)))
		app.logLock.Lock()
		for _, item := range app.logList {
			if item.Line == 0 {
				_, _ = f.WriteString(item.ColorFunc(item.Prefix+" "+item.Message) + "\n")
			} else {
				_, _ = f.WriteString(item.ColorFunc(langext.StrRepeat(" ", len(item.Prefix)+1)+item.Message) + "\n")
			}
		}
		app.logLock.Unlock()

		sub := app.logBroadcaster.SubscribeByCallback("", func(msg LogMessage) {
			if msg.Line == 0 {
				_, _ = f.WriteString(msg.ColorFunc(msg.Prefix+" "+msg.Message) + "\n")
			} else {
				_, _ = f.WriteString(msg.ColorFunc(langext.StrRepeat(" ", len(msg.Prefix)+1)+msg.Message) + "\n")
			}
		})
		defer sub.Unsubscribe()

		app.LogDebug(fmt.Sprintf("Starting fifo log listener"))

		<-listernerStopSig

		app.LogDebug(fmt.Sprintf("Finished fifo log listener"))
		app.LogLine()

	}()

	time.Sleep(100 * time.Millisecond)

	te := strings.Split(app.config.TerminalEmulator, " ")[0]
	tc := strings.Split(app.config.TerminalEmulator, " ")[1:]
	tc = append(tc, fmt.Sprintf("cat \"%s\"", filePath))
	//tc = append(tc, "bash")

	proc := exec.Command(te, tc...)

	app.LogDebug(fmt.Sprintf("Starting terminal-emulator '%s' [%v]", te, tc))
	err = proc.Start()
	if err != nil {
		app.LogError("Failed to start terminal emulator", err)
		return
	}

	app.LogDebug("Terminal-emulator started - waiting for exit")
	app.LogLine()

	_ = proc.Wait()

	app.LogDebug("Terminal-emulator exited - stopping fifo pipe")

	listernerStopSig <- true
}
