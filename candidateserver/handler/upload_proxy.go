package handler

import (
	"fmt"
	"io"
	"net/http"
)

// UploadProxy POST /api/credentials/upload-proxy
// Temporary proxy to bypass S3 CORS issues. To be deleted later.
func (h *Handler) UploadProxy(w http.ResponseWriter, r *http.Request) {
	uploadUrl := r.URL.Query().Get("url")
	if uploadUrl == "" {
		http.Error(w, "missing target url", http.StatusBadRequest)
		return
	}

	req, err := http.NewRequest(http.MethodPut, uploadUrl, r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to create proxy request: %v", err), http.StatusInternalServerError)
		return
	}

	// Forward Content-Type and Content-Length
	req.Header.Set("Content-Type", r.Header.Get("Content-Type"))
	req.ContentLength = r.ContentLength

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, fmt.Sprintf("proxy request failed: %v", err), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		http.Error(w, fmt.Sprintf("S3 returned error: %s - %s", resp.Status, string(bodyBytes)), resp.StatusCode)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"success": true}`))
}
