package assets

import (
	_ "embed"
)

//go:embed iconInit.png
var IconInit []byte

//go:embed iconDefault.png
var IconDefault []byte

//go:embed iconDownload.png
var IconDownload []byte

//go:embed iconUpload.png
var IconUpload []byte

//go:embed iconUploadConflict.png
var IconUploadConflict []byte
