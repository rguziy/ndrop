package webui

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed all:web/out
var assets embed.FS

// Handler serves the embedded ndrop web UI.
func Handler() http.Handler {
	sub, err := fs.Sub(assets, "web/out")
	if err != nil {
		panic(err)
	}
	return http.FileServer(http.FS(sub))
}
