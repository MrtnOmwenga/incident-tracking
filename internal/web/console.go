package web

import (
	"embed"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

// The console's build output (console/, built with Vite) lands in console/ here and is embedded.
// Without a build, a placeholder page explains how to make one.
//
//go:embed all:console
var consoleFS embed.FS

// console serves the single-page app: real files as they are (hashed assets cached for a year),
// and index.html for every other path under /console/, so client-side routes work on reload.
func (s *Server) console() http.Handler {
	files, _ := fs.Sub(consoleFS, "console")
	index := "index.html"
	if _, err := fs.Stat(files, index); err != nil {
		index = "placeholder.html"
	}
	serveIndex := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache")
		http.ServeFileFS(w, r, files, index)
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(path.Clean(r.URL.Path), "/console")
		name = strings.TrimPrefix(name, "/")
		if name == "" || name == "index.html" {
			serveIndex(w, r)
			return
		}
		if f, err := fs.Stat(files, name); err == nil && !f.IsDir() {
			if strings.HasPrefix(name, "assets/") {
				w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			}
			http.ServeFileFS(w, r, files, name)
			return
		}
		if strings.HasPrefix(name, "assets/") {
			http.NotFound(w, r) // a missing script must not come back as HTML
			return
		}
		serveIndex(w, r)
	})
}
