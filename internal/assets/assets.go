// Package assets provides embedded assets for the application.
package assets

import (
	_ "embed"
)

// Favicon is the embedded favicon image.
//
//go:embed favicon.ico
var Favicon []byte
