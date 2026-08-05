package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"candbff/config"
	"candbff/handler"

	"github.com/go-chi/chi/v5"
)

type testAPIResponse struct {
	Code      int               `json:"code"`
	ErrorCode handler.ErrorCode `json:"error_code"`
	Data      json.RawMessage   `json:"data"`
}

func newTestRouter() http.Handler {
	return (&Server{}).buildRouter(&handler.Handler{})
}

func decodeTestResponse(t *testing.T, recorder *httptest.ResponseRecorder) testAPIResponse {
	t.Helper()

	var response testAPIResponse
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v; body=%q", err, recorder.Body.String())
	}
	return response
}

func TestRouterHealth(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/health", nil)

	newTestRouter().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%q", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	response := decodeTestResponse(t, recorder)
	if response.Code != http.StatusOK {
		t.Fatalf("response code = %d, want %d", response.Code, http.StatusOK)
	}

	var data struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(response.Data, &data); err != nil {
		t.Fatalf("decode health data: %v", err)
	}
	if data.Status != "ok" {
		t.Fatalf("health status = %q, want %q", data.Status, "ok")
	}
}

func TestRouterUnknownPathReturnsJSON404(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/not-a-real-route", nil)

	newTestRouter().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d; body=%q", recorder.Code, http.StatusNotFound, recorder.Body.String())
	}
	if contentType := recorder.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "application/json") {
		t.Fatalf("Content-Type = %q, want JSON", contentType)
	}
}

func TestRouterRegistersCriticalCandidateRoutes(t *testing.T) {
	router, ok := newTestRouter().(chi.Routes)
	if !ok {
		t.Fatal("router does not expose chi route metadata")
	}

	registered := make(map[string]struct{})
	if err := chi.Walk(router, func(
		method string,
		route string,
		_ http.Handler,
		_ ...func(http.Handler) http.Handler,
	) error {
		registered[method+" "+route] = struct{}{}
		return nil
	}); err != nil {
		t.Fatalf("walk routes: %v", err)
	}

	expected := []string{
		http.MethodGet + " /health",
		http.MethodPost + " /api/public/webhooks/exams/callback/{urlType}/{examId}",
		http.MethodGet + " /api/auth/login-url",
		http.MethodGet + " /api/membership/plans",
		http.MethodGet + " /api/mall/pipelines/",
		http.MethodGet + " /api/user/me",
		http.MethodPost + " /api/mall/payments/initiate",
		http.MethodPost + " /api/pipeline/lessons/{lessonId}/complete",
		http.MethodPost + " /api/exams/units/{courseUnitUlid}/signup",
		http.MethodPost + " /api/credentials/submit",
		http.MethodGet + " /api/orders/",
		http.MethodGet + " /api/invoices/{orderId}/pdf",
	}

	for _, route := range expected {
		if _, exists := registered[route]; !exists {
			t.Errorf("critical route %q is not registered", route)
		}
	}
}

func TestProtectedRoutesRequireAuthentication(t *testing.T) {
	tests := []struct {
		name   string
		method string
		path   string
	}{
		{name: "current user", method: http.MethodGet, path: "/api/user/me"},
		{name: "orders", method: http.MethodGet, path: "/api/orders"},
		{name: "initiate payment", method: http.MethodPost, path: "/api/mall/payments/initiate"},
		{name: "exam signup", method: http.MethodPost, path: "/api/exams/units/unit-1/signup"},
		{name: "credential submission", method: http.MethodPost, path: "/api/credentials/submit"},
		{name: "resource preview", method: http.MethodGet, path: "/api/pipeline/resource-preview"},
	}

	router := newTestRouter()
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(test.method, test.path, nil)

			router.ServeHTTP(recorder, request)

			if recorder.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want %d; body=%q", recorder.Code, http.StatusUnauthorized, recorder.Body.String())
			}
			response := decodeTestResponse(t, recorder)
			if response.ErrorCode != handler.ErrUnauthorized {
				t.Fatalf("error_code = %q, want %q", response.ErrorCode, handler.ErrUnauthorized)
			}
		})
	}
}

func TestProtectedRouteRejectsMalformedAuthCookie(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/orders", nil)
	request.AddCookie(&http.Cookie{
		Name:  "access_token",
		Value: "z1.2.incomplete",
	})

	newTestRouter().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d; body=%q", recorder.Code, http.StatusUnauthorized, recorder.Body.String())
	}
	response := decodeTestResponse(t, recorder)
	if response.ErrorCode != handler.ErrInvalidToken {
		t.Fatalf("error_code = %q, want %q", response.ErrorCode, handler.ErrInvalidToken)
	}
}

func TestRouterCORSPreflightForAllowedOrigin(t *testing.T) {
	t.Setenv(config.EnvCORSOrigins, "https://cftpcand.llwan.top")

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodOptions, "/api/orders", nil)
	request.Header.Set("Origin", "https://cftpcand.llwan.top")

	newTestRouter().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNoContent)
	}
	if origin := recorder.Header().Get("Access-Control-Allow-Origin"); origin != "https://cftpcand.llwan.top" {
		t.Fatalf("Access-Control-Allow-Origin = %q", origin)
	}
	if credentials := recorder.Header().Get("Access-Control-Allow-Credentials"); credentials != "true" {
		t.Fatalf("Access-Control-Allow-Credentials = %q, want true", credentials)
	}
}

func TestRouterCORSDoesNotAllowUnknownOrigin(t *testing.T) {
	t.Setenv(config.EnvCORSOrigins, "https://cftpcand.llwan.top")

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodOptions, "/api/orders", nil)
	request.Header.Set("Origin", "https://evil.example")

	newTestRouter().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNoContent)
	}
	if origin := recorder.Header().Get("Access-Control-Allow-Origin"); origin != "" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want empty", origin)
	}
}
