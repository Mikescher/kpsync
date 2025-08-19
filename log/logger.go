package log

import (
	"git.blackforestbytes.com/BlackForestBytes/goext/exerr"
)

//TODO

func Fatal(msg string) {
	panic(msg)
}

func FatalErr(msg string, err error) {
	if err != nil {
		println("FATAL: " + msg)
		println("       " + err.Error())
		println(exerr.FromError(err).FormatLog(exerr.LogPrintOverview))
		panic(0)
	} else {
		panic("FATAL: " + msg)
	}
}

func LogError(msg string, err error) {
	if err != nil {
		println("ERROR: " + msg)
		println("       " + err.Error())
		println(exerr.FromError(err).FormatLog(exerr.LogPrintOverview))
	} else {
		println("ERROR: " + msg)
	}
}

func LogInfo(msg string) {
	println("INFO: " + msg)
}
