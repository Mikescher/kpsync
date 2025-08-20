package app

import (
	"fmt"
	"strings"

	"git.blackforestbytes.com/BlackForestBytes/goext/cmdext"
	"git.blackforestbytes.com/BlackForestBytes/goext/exerr"
)

func (app *Application) showErrorNotification(msg string, body string) {
	app.LogDebug("{notify-send} " + msg)

	res, err := cmdext.
		Runner("notify-send").
		Arg("--urgency=critical").
		Arg("--app-name=kpsync").
		Arg("--print-id").
		Arg(msg).
		Arg(body).
		Run()
	if err != nil {
		app.LogError("Failed to show notification", err)
		return
	}

	if res.ExitCode != 0 {
		app.LogError("Failed to show notification", nil)
		app.LogDebug(fmt.Sprintf("ExitCode: %d", res.ExitCode))
		app.LogDebug(fmt.Sprintf("Stdout: %s", res.StdOut))
		app.LogDebug(fmt.Sprintf("Stderr: %s", res.StdErr))
		return
	}

	app.LogDebug(fmt.Sprintf("Displayed notification with id %s", res.StdOut))
}

func (app *Application) showSuccessNotification(msg string, body string) {
	app.LogDebug("{notify-send} " + msg)

	res, err := cmdext.
		Runner("notify-send").
		Arg("--urgency=critical").
		Arg("--app-name=kpsync").
		Arg("--print-id").
		Arg(msg).
		Arg(body).
		Run()
	if err != nil {
		app.LogError("Failed to show notification", err)
		return
	}

	if res.ExitCode != 0 {
		app.LogError("Failed to show notification", nil)
		app.LogDebug(fmt.Sprintf("ExitCode: %d", res.ExitCode))
		app.LogDebug(fmt.Sprintf("Stdout: %s", res.StdOut))
		app.LogDebug(fmt.Sprintf("Stderr: %s", res.StdErr))
		return
	}

	app.LogDebug(fmt.Sprintf("Displayed notification with id %s", res.StdOut))
}

func (app *Application) showChoiceNotification(msg string, body string, options map[string]string) (string, error) {
	app.LogDebug(fmt.Sprintf("{notify-send} %s {%d choices}", msg, len(options)))

	bldr := cmdext.
		Runner("notify-send").
		Arg("--urgency=critical").
		Arg("--expire-time=0").
		Arg("--wait").
		Arg("--app-name=kpsync")

	for kOpt, vOpt := range options {
		bldr = bldr.Arg("--action=" + kOpt + "=" + vOpt)
	}

	bldr = bldr.Arg(msg).Arg(body)

	res, err := bldr.Run()
	if err != nil {
		app.LogError("Failed to show choice-notification", err)
		return "", exerr.Wrap(err, "").Build()
	}

	if res.ExitCode != 0 {
		return "", nil
	}

	return strings.TrimSpace(res.StdOut), nil
}
