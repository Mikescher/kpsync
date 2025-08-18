package log

//TODO

func Fatal(msg string) {
	panic(msg)
}

func FatalErr(msg string, err error) {
	if err != nil {
		panic("ERROR: " + msg + "\n" + err.Error())
	} else {
		panic("ERROR: " + msg)
	}
}

func LogError(msg string, err error) {
	if err != nil {
		println("ERROR: " + msg + "\n" + err.Error())
	} else {
		println("ERROR: " + msg)
	}
}

func LogInfo(msg string) {
	println("INFO: " + msg)
}
