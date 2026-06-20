package main

import (
	"net/http"
	"os"
	"path/filepath"
)

func spaHandler(dir string) http.Handler {
	fs := http.FileServer(http.Dir(dir))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := filepath.Join(dir, filepath.Clean(r.URL.Path))
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			fs.ServeHTTP(w, r)
			return
		}
		// SPA fallback is only for browser navigation. Unknown non-GET API-like
		// requests should not be swallowed as index.html with HTTP 200.
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.ServeFile(w, r, filepath.Join(dir, "index.html"))
	})
}
