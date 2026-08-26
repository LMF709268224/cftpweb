package server

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sort"
	"strings"
	"testing"

	"adminbff/config"
	"adminbff/handler"

	"github.com/go-chi/chi/v5"
)

const (
	adminRouteCount       = 285
	adminRouteFingerprint = "8e28347b4074838e6ddbe97b6da3041dbc65994279ce01bdd7db68935775d995"
)

var adminPublicRoutes = map[string]struct{}{
	http.MethodGet + " /health":             {},
	http.MethodGet + " /api/auth/login-url": {},
	http.MethodPost + " /api/auth/login":    {},
	http.MethodPost + " /api/auth/logout":   {},
	http.MethodPost + " /api/auth/refresh":  {},
}

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

func registeredAdminRoutes(t *testing.T) []string {
	t.Helper()
	router, ok := newTestRouter().(chi.Routes)
	if !ok {
		t.Fatal("router does not expose chi route metadata")
	}

	var routes []string
	if err := chi.Walk(router, func(
		method string,
		route string,
		_ http.Handler,
		_ ...func(http.Handler) http.Handler,
	) error {
		routes = append(routes, method+" "+route)
		return nil
	}); err != nil {
		t.Fatalf("walk routes: %v", err)
	}
	sort.Strings(routes)
	return routes
}

func TestRouterHealth(t *testing.T) {
	recorder := httptest.NewRecorder()
	newTestRouter().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/health", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%q", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	response := decodeTestResponse(t, recorder)
	var data struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(response.Data, &data); err != nil {
		t.Fatalf("decode health data: %v", err)
	}
	if data.Status != "ok" {
		t.Fatalf("health status = %q, want ok", data.Status)
	}
}

func TestRouterUnknownPathReturnsJSON404(t *testing.T) {
	recorder := httptest.NewRecorder()
	newTestRouter().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/not-a-real-route", nil))

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d; body=%q", recorder.Code, http.StatusNotFound, recorder.Body.String())
	}
	if contentType := recorder.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "application/json") {
		t.Fatalf("Content-Type = %q, want JSON", contentType)
	}
}

func TestRouterMatchesAdminRouteFingerprint(t *testing.T) {
	routes := registeredAdminRoutes(t)
	hash := sha256.Sum256([]byte(strings.Join(routes, "\n")))
	fingerprint := hex.EncodeToString(hash[:])
	if len(routes) != adminRouteCount || fingerprint != adminRouteFingerprint {
		t.Fatalf(
			"admin route contract changed\ncount: %d\nfingerprint: %s\nroutes:\n%s",
			len(routes),
			fingerprint,
			strings.Join(routes, "\n"),
		)
	}
}

func TestProtectedRoutesRequireAuthentication(t *testing.T) {
	router := newTestRouter()
	for _, route := range registeredAdminRoutes(t) {
		if _, public := adminPublicRoutes[route]; public {
			continue
		}
		parts := strings.SplitN(route, " ", 2)
		if len(parts) != 2 || !strings.HasPrefix(parts[1], "/api/") {
			t.Fatalf("route %q is neither public nor a protected API route", route)
		}

		route := route
		t.Run(route, func(t *testing.T) {
			parts := strings.SplitN(route, " ", 2)
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(parts[0], materializeRoutePath(parts[1]), nil)
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

var routeParameterPattern = regexp.MustCompile(`\{[^}/]+\}`)

func materializeRoutePath(pattern string) string {
	return routeParameterPattern.ReplaceAllString(pattern, "test-id")
}

func TestProtectedRouteRejectsMalformedAuthCookie(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/mall/orders", nil)
	request.AddCookie(&http.Cookie{Name: "access_token", Value: "z1.2.incomplete"})

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
	t.Setenv(config.EnvCORSOrigins, "https://admin.example.test")
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodOptions, "/api/mall/orders", nil)
	request.Header.Set("Origin", "https://admin.example.test")

	newTestRouter().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNoContent)
	}
	if origin := recorder.Header().Get("Access-Control-Allow-Origin"); origin != "https://admin.example.test" {
		t.Fatalf("Access-Control-Allow-Origin = %q", origin)
	}
	if credentials := recorder.Header().Get("Access-Control-Allow-Credentials"); credentials != "true" {
		t.Fatalf("Access-Control-Allow-Credentials = %q, want true", credentials)
	}
}

func TestRouterCORSDoesNotAllowUnknownOrigin(t *testing.T) {
	t.Setenv(config.EnvCORSOrigins, "https://admin.example.test")
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodOptions, "/api/mall/orders", nil)
	request.Header.Set("Origin", "https://evil.example")

	newTestRouter().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNoContent)
	}
	if origin := recorder.Header().Get("Access-Control-Allow-Origin"); origin != "" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want empty", origin)
	}
}
