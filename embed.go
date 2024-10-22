package main

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed web/dist
var Static embed.FS

func getFileSystem() http.FileSystem {
	fsys, err := fs.Sub(Static, "web/dist")
	if err != nil {
		return nil
	}
	return http.FS(fsys)
}
