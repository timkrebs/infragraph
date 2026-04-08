package ui

import (
	"embed"
	"io"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

//go:embed dist/*
var distFS embed.FS

// Handler returns an http.Handler that serves the embedded SPA.
// All paths that don't match a static file are rewritten to index.html
// so that client-side routing works.
func Handler() http.Handler {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		panic("ui: embed.FS missing dist: " + err.Error())
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Strip the /ui/ prefix.
		p := strings.TrimPrefix(r.URL.Path, "/ui")
		p = strings.TrimPrefix(p, "/")
		if p == "" {
			p = "index.html"
		}

		// Try to open the requested file.
		f, err := sub.Open(p)
		if err != nil {
			// SPA fallback: serve index.html for any unknown path.
			p = "index.html"
			f, err = sub.Open(p)
			if err != nil {
				http.NotFound(w, r)
				return
			}
		}
		defer f.Close()

		// If it's a directory, serve its index.html.
		stat, err := f.Stat()
		if err != nil {
			http.NotFound(w, r)
			return
		}
		if stat.IsDir() {
			f.Close()
			p = path.Join(p, "index.html")
			f, err = sub.Open(p)
			if err != nil {
				http.NotFound(w, r)
				return
			}
			defer f.Close()
			stat, _ = f.Stat()
		}

		// Set content type based on extension.
		ext := path.Ext(p)
		switch ext {
		case ".html":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
		case ".js":
			w.Header().Set("Content-Type", "application/javascript")
		case ".css":
			w.Header().Set("Content-Type", "text/css")
		case ".json":
			w.Header().Set("Content-Type", "application/json")
		case ".svg":
			w.Header().Set("Content-Type", "image/svg+xml")
		case ".png":
			w.Header().Set("Content-Type", "image/png")
		case ".ico":
			w.Header().Set("Content-Type", "image/x-icon")
		}

		// Cache static assets (hashed filenames), but not index.html.
		if ext != ".html" && strings.Contains(p, "assets/") {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		} else {
			w.Header().Set("Cache-Control", "no-cache")
		}

		w.WriteHeader(http.StatusOK)
		io.Copy(w, f.(io.Reader))
	})
}
