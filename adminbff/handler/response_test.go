package handler

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestReadJSONRejectsOversizedBody(t *testing.T) {
	body := `{"value":"` + strings.Repeat("x", 1<<20) + `"}`
	req := httptest.NewRequest("POST", "/", strings.NewReader(body))

	var input struct {
		Value string `json:"value"`
	}
	if err := ReadJSON(req, &input); err == nil {
		t.Fatal("ReadJSON() error = nil, want request size error")
	}
}
