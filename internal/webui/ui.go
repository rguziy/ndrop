package webui

import (
    "bytes"
	"embed"
	"io/fs"
	"net/http"
    "path/filepath"
    "strings"

    "github.com/rguziy/ndrop/internal/version"
)

//go:embed all:web/out
var assets embed.FS

// Handler serves the embedded ndrop web UI.
func Handler() http.Handler {
	sub, err := fs.Sub(assets, "web/out")
	if err != nil {
		panic(err)
	}

    fileServer := http.FileServer(http.FS(sub))

    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := filepath.Clean(r.URL.Path)

		if path == "/" || path == "." || strings.HasSuffix(path, "index.html") {
			content, err := fs.ReadFile(sub, "index.html")
			if err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			processedContent := bytes.ReplaceAll(content, []byte("{{.Version}}"), []byte(version.Version))

			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			w.Write(processedContent)
			return
		}

		fileServer.ServeHTTP(w, r)
	})
}
