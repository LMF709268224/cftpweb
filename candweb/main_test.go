package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
)

func TestStaticHandlerServesPrecompressedAssets(t *testing.T) {
	distFS := fstest.MapFS{
		"index.html":       {Data: []byte("<html>app</html>")},
		"assets/app.js":    {Data: []byte("raw-javascript")},
		"assets/app.js.br": {Data: []byte("brotli-javascript")},
		"assets/app.js.gz": {Data: []byte("gzip-javascript")},
	}

	tests := []struct {
		name           string
		acceptEncoding string
		wantEncoding   string
		wantBody       string
	}{
		{name: "brotli", acceptEncoding: "gzip, deflate, br", wantEncoding: "br", wantBody: "brotli-javascript"},
		{name: "gzip", acceptEncoding: "gzip", wantEncoding: "gzip", wantBody: "gzip-javascript"},
		{name: "raw", wantBody: "raw-javascript"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/assets/app.js", nil)
			request.Header.Set("Accept-Encoding", tt.acceptEncoding)
			response := httptest.NewRecorder()

			newStaticHandler(distFS).ServeHTTP(response, request)

			result := response.Result()
			defer result.Body.Close()
			body, err := io.ReadAll(result.Body)
			if err != nil {
				t.Fatalf("read response: %v", err)
			}
			if got := result.Header.Get("Content-Encoding"); got != tt.wantEncoding {
				t.Fatalf("Content-Encoding = %q, want %q", got, tt.wantEncoding)
			}
			if got := string(body); got != tt.wantBody {
				t.Fatalf("body = %q, want %q", got, tt.wantBody)
			}
			if got := result.Header.Get("Cache-Control"); got != "public, max-age=31536000, immutable" {
				t.Fatalf("Cache-Control = %q", got)
			}
			if got := result.Header.Get("Vary"); got != "Accept-Encoding" {
				t.Fatalf("Vary = %q", got)
			}
		})
	}
}

func TestStaticHandlerFallsBackToIndex(t *testing.T) {
	distFS := fstest.MapFS{
		"index.html": {Data: []byte("<html>app</html>")},
	}
	request := httptest.NewRequest(http.MethodGet, "/orders", nil)
	response := httptest.NewRecorder()

	newStaticHandler(distFS).ServeHTTP(response, request)

	if got := response.Body.String(); got != "<html>app</html>" {
		t.Fatalf("body = %q", got)
	}
	if got := response.Header().Get("Cache-Control"); got != "no-cache" {
		t.Fatalf("Cache-Control = %q", got)
	}
}
