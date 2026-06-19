package main

import (
	"embed"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

//go:embed web/build
var embedFS embed.FS

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	buildFS, err := fs.Sub(embedFS, "web/build")
	if err != nil {
		log.Fatalf("failed to locate embedded build folder: %v", err)
	}

	fileServer := http.FileServer(http.FS(buildFS))

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Clean the path
		path := filepath.Clean(r.URL.Path)
		if path == "." || path == "/" {
			path = "index.html"
		} else {
			path = strings.TrimPrefix(path, "/")
		}

		// 1. Try to open the file directly (for static assets like _next/static, favicon.ico, images, etc.)
		f, err := buildFS.Open(path)
		if err == nil {
			stat, err := f.Stat()
			if err == nil && !stat.IsDir() {
				f.Close()
				// Cache control headers
				if strings.HasPrefix(r.URL.Path, "/_next/static/") || strings.HasPrefix(r.URL.Path, "/assets/") {
					w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
				} else {
					w.Header().Set("Cache-Control", "public, max-age=86400")
				}
				fileServer.ServeHTTP(w, r)
				return
			}
			f.Close()
		}

		// 2. Next.js Static Export routing helper
		// E.g., request "/callback", try "/callback.html"
		htmlPath := path + ".html"
		f, err = buildFS.Open(htmlPath)
		if err == nil {
			stat, err := f.Stat()
			if err == nil && !stat.IsDir() {
				f.Close()
				w.Header().Set("Cache-Control", "no-cache")
				r.URL.Path = "/" + htmlPath
				fileServer.ServeHTTP(w, r)
				return
			}
			f.Close()
		}

		// E.g. request "/courses", check if "/courses/index.html" exists
		indexPath := filepath.Join(path, "index.html")
		// Convert backslashes to forward slashes for fs.FS compatibility
		indexPath = filepath.ToSlash(indexPath)
		f, err = buildFS.Open(indexPath)
		if err == nil {
			stat, err := f.Stat()
			if err == nil && !stat.IsDir() {
				f.Close()
				w.Header().Set("Cache-Control", "no-cache")
				r.URL.Path = "/" + indexPath
				fileServer.ServeHTTP(w, r)
				return
			}
			f.Close()
		}

		// 3. Fallback: If not found, fallback to index.html for SPA client-side routing
		fallbackFile, err := buildFS.Open("index.html")
		if err != nil {
			http.Error(w, "index.html not found", http.StatusNotFound)
			return
		}
		defer fallbackFile.Close()

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		_, _ = io.Copy(w, fallbackFile)
	})

	log.Printf("adminweb listening on port %s", port)
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
