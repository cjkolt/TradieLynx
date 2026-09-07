package tradielynx

import (
	"embed"
	"io/fs"
	"net/http"
)

/*
These funcs will be changed later for when more than just /static/ needs to be embedded -
perhaps through a loop and arrayed embed.FS vars w/ funcs that carry string vars for folder
names.
*/

// This directive embeds files. Do not move or delete this file!
// Do not mess with the go comments either.
//
//go:embed static/*
var staticFiles embed.FS

// Called in main
func UseEmbeddedContent() {
	http.Handle("/static/", http.StripPrefix("/static/",
		http.FileServer(StaticFileSystem())))
}

// Helper called by UseEmbeddedContent()
func StaticFileSystem() http.FileSystem {
	staticFileSystem, _ := fs.Sub(staticFiles, "static")
	return http.FS(staticFileSystem)
}
