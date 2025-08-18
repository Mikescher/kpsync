package main

import (
	"git.blackforestbytes.com/BlackForestBytes/goext/exerr"
	"git.blackforestbytes.com/BlackForestBytes/goext/langext"
	"mikescher.com/kpsync/app"
)

func main() {
	exerr.Init(exerr.ErrorPackageConfigInit{
		ZeroLogErrTraces: langext.PFalse,
		ZeroLogAllTraces: langext.PFalse,
	})

	kpApp := app.NewApplication()
	kpApp.Run()
}
