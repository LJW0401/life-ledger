// Package web serves the embedded React build and implements SPA fallback
// behavior for non-API routes.
package web

import (
	"fmt"
	"io/fs"
	"net/http"
	"net/url"
	"path"
	"strings"

	webdist "life-ledger/web"
)

// NewHandler returns a handler backed by the embedded frontend dist directory.
func NewHandler() (http.Handler, error) {
	dist, err := fs.Sub(webdist.Assets, "dist")
	if err != nil {
		return nil, fmt.Errorf("frontend dist not embedded: %w", err)
	}
	return newHandler(dist)
}

func newHandler(dist fs.FS) (http.Handler, error) {
	if _, err := fs.Stat(dist, "index.html"); err != nil {
		return nil, fmt.Errorf("frontend dist missing index.html; run npm run build: %w", err)
	}
	files := http.FileServer(http.FS(dist))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.Redirect(w, r, "/important-dates", http.StatusFound)
			return
		}
		name := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if name != "." {
			if info, err := fs.Stat(dist, name); err == nil && !info.IsDir() {
				files.ServeHTTP(w, r)
				return
			}
		}
		r2 := new(http.Request)
		*r2 = *r
		r2.URL = cloneURL(r.URL)
		r2.URL.Path = "/"
		http.ServeFileFS(w, r2, dist, "index.html")
	}), nil
}

func cloneURL(u *url.URL) *url.URL {
	copy := *u
	return &copy
}
