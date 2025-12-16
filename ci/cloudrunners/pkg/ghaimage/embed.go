package ghaimage

import (
	_ "embed"
)

//go:embed build-image.sh
var buildImage []byte
