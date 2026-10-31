package frontend

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var rawDistFS embed.FS

// DistFS returns the embedded frontend/dist filesystem for Wails AssetFileServerFS.
func DistFS() fs.FS {
	sub, err := fs.Sub(rawDistFS, "dist")
	if err != nil {
		return rawDistFS
	}
	return sub
}
