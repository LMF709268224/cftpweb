package main

import (
	"embed"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	pathpkg "path"
	"strings"
)

//go:embed all:vue-web/dist
var embedFS embed.FS

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	distFS, err := fs.Sub(embedFS, "vue-web/dist")
	if err != nil {
		log.Fatalf("failed to locate embedded dist folder: %v", err)
	}

	fileServer := http.FileServer(http.FS(distFS))

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := pathpkg.Clean(r.URL.Path)
		if path == "." || path == "/" {
			path = "index.html"
		} else {
			path = strings.TrimPrefix(path, "/")
		}

		f, err := distFS.Open(path)
		if err == nil {
			stat, err := f.Stat()
			f.Close()
			if err == nil && !stat.IsDir() {
				if strings.HasPrefix(r.URL.Path, "/assets/") {
					w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
					if strings.HasSuffix(path, ".js") {
						w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
					} else if strings.HasSuffix(path, ".css") {
						w.Header().Set("Content-Type", "text/css; charset=utf-8")
					}
				} else if path == "index.html" || strings.HasSuffix(path, ".html") {
					w.Header().Set("Cache-Control", "no-cache")
				} else {
					w.Header().Set("Cache-Control", "public, max-age=86400")
				}
				fileServer.ServeHTTP(w, r)
				return
			}
		}

		if strings.HasPrefix(r.URL.Path, "/assets/") {
			http.NotFound(w, r)
			return
		}

		indexFile, err := distFS.Open("index.html")
		if err != nil {
			http.Error(w, "index.html not found", http.StatusNotFound)
			return
		}
		defer indexFile.Close()

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		_, _ = io.Copy(w, indexFile)
	})

	log.Printf("adminweb listening on port %s", port)
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
