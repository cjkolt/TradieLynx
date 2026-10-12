package tradielynx

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed static/*
var staticFiles embed.FS

func UseEmbeddedContent() {
	http.Handle("/static/", http.StripPrefix("/static/",
		http.FileServer(StaticFileSystem())))
}

func StaticFileSystem() http.FileSystem {
	staticFileSystem, err := fs.Sub(staticFiles, "static")
	if err != nil {
		panic(err)
	}
	return http.FS(staticFileSystem)
}
