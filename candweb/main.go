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

func newStaticHandler(distFS fs.FS) http.Handler {
	fileServer := http.FileServer(http.FS(distFS))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := pathpkg.Clean(r.URL.Path)
		if path == "." || path == "/" {
			path = "index.html"
		} else {
			path = strings.TrimPrefix(path, "/")
		}

		// Try to open file
		f, err := distFS.Open(path)
		if err == nil {
			stat, err := f.Stat()
			f.Close()
			if err == nil && !stat.IsDir() {
				if strings.HasPrefix(r.URL.Path, "/assets/") {
					w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
					w.Header().Set("Vary", "Accept-Encoding")
					if strings.HasSuffix(path, ".js") {
						w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
					} else if strings.HasSuffix(path, ".css") {
						w.Header().Set("Content-Type", "text/css; charset=utf-8")
					}
					if servePrecompressedAsset(w, r, distFS, path) {
						return
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

		// Fallback to index.html for SPA
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
}

func servePrecompressedAsset(w http.ResponseWriter, r *http.Request, distFS fs.FS, path string) bool {
	variants := []struct {
		encoding string
		suffix   string
	}{
		{encoding: "br", suffix: ".br"},
		{encoding: "gzip", suffix: ".gz"},
	}

	for _, variant := range variants {
		if !acceptsEncoding(r.Header.Get("Accept-Encoding"), variant.encoding) {
			continue
		}

		filename := path + variant.suffix
		file, err := distFS.Open(filename)
		if err != nil {
			continue
		}

		seeker, ok := file.(io.ReadSeeker)
		if !ok {
			file.Close()
			continue
		}
		stat, err := file.Stat()
		if err != nil {
			file.Close()
			continue
		}

		w.Header().Set("Content-Encoding", variant.encoding)
		http.ServeContent(w, r, path, stat.ModTime(), seeker)
		file.Close()
		return true
	}

	return false
}

func acceptsEncoding(header string, encoding string) bool {
	for _, item := range strings.Split(strings.ToLower(header), ",") {
		parts := strings.Split(item, ";")
		name := strings.TrimSpace(parts[0])
		if name != encoding && name != "*" {
			continue
		}
		for _, parameter := range parts[1:] {
			if strings.TrimSpace(parameter) == "q=0" {
				return false
			}
		}
		return true
	}
	return false
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	distFS, err := fs.Sub(embedFS, "vue-web/dist")
	if err != nil {
		log.Fatalf("failed to locate embedded dist folder: %v", err)
	}

	log.Printf("candweb listening on port %s", port)
	if err := http.ListenAndServe(":"+port, newStaticHandler(distFS)); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
