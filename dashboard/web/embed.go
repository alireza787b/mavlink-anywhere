package web

import (
	"embed"
	"io/fs"
)

//go:embed static/*
var staticFiles embed.FS

// StaticFS returns a filesystem rooted at the static directory.
func StaticFS() fs.FS {
	sub, err := fs.Sub(staticFiles, "static")
	if err != nil {
		panic("failed to get static subdirectory: " + err.Error())
	}
	return sub
}
