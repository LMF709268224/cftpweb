package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCatalogHandlersReportUnavailableContract(t *testing.T) {
	tests := []struct {
		name   string
		method string
		target string
		handle func(*Handler, http.ResponseWriter, *http.Request)
	}{
		{name: "list", method: http.MethodGet, target: "/api/catalogs", handle: (*Handler).ListCatalogs},
		{name: "create", method: http.MethodPost, target: "/api/catalogs", handle: (*Handler).CreateCatalog},
		{name: "update", method: http.MethodPut, target: "/api/catalogs/catalog-1", handle: (*Handler).UpdateCatalog},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			test.handle(&Handler{}, recorder, httptest.NewRequest(test.method, test.target, nil))

			if recorder.Code != http.StatusNotImplemented {
				t.Fatalf("status = %d, want %d; body = %s", recorder.Code, http.StatusNotImplemented, recorder.Body.String())
			}
			var response struct {
				ErrorCode ErrorCode `json:"error_code"`
			}
			if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if response.ErrorCode != ErrNotImplemented {
				t.Fatalf("error code = %q, want %q", response.ErrorCode, ErrNotImplemented)
			}
		})
	}
}
