package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sort"
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

type routeAccess string

const (
	routePublic    routeAccess = "public"
	routeOptional  routeAccess = "optional"
	routeProtected routeAccess = "protected"
)

type routeExpectation struct {
	method string
	path   string
	access routeAccess
}

var candidateRouteContract = []routeExpectation{
	{method: http.MethodGet, path: "/health", access: routePublic},
	{method: http.MethodPost, path: "/api/public/webhooks/exams/callback/{urlType}/{examId}", access: routePublic},
	{method: http.MethodPost, path: "/api/public/telemetry/", access: routePublic},
	{method: http.MethodGet, path: "/api/public/config", access: routePublic},
	{method: http.MethodGet, path: "/api/public/config/organization", access: routePublic},
	{method: http.MethodGet, path: "/api/auth/login-url", access: routePublic},
	{method: http.MethodPost, path: "/api/auth/login", access: routePublic},
	{method: http.MethodPost, path: "/api/auth/logout", access: routePublic},
	{method: http.MethodPost, path: "/api/auth/refresh", access: routePublic},
	{method: http.MethodGet, path: "/api/pipeline/resource-preview", access: routeProtected},
	{method: http.MethodHead, path: "/api/pipeline/resource-preview", access: routeProtected},
	{method: http.MethodGet, path: "/api/pipeline/lessons/{lessonId}/preview", access: routeProtected},
	{method: http.MethodHead, path: "/api/pipeline/lessons/{lessonId}/preview", access: routeProtected},
	{method: http.MethodGet, path: "/api/membership/plans", access: routeOptional},
	{method: http.MethodGet, path: "/api/mall/pipelines/", access: routeOptional},
	{method: http.MethodGet, path: "/api/mall/pipelines/{pipelineId}", access: routeOptional},
	{method: http.MethodGet, path: "/api/mall/pipelines/{pipelineId}/thumbnail-url", access: routeOptional},
	{method: http.MethodGet, path: "/api/mall/pipelines/{pipelineId}/runtime", access: routeOptional},
	{method: http.MethodGet, path: "/api/mall/pipelines/{pipelineId}/timeline", access: routeOptional},
	{method: http.MethodGet, path: "/api/mall/bundles/", access: routeOptional},
	{method: http.MethodGet, path: "/api/mall/bundles/{bundleId}", access: routeOptional},
	{method: http.MethodGet, path: "/api/mall/bundles/{bundleId}/thumbnail-url", access: routeOptional},
	{method: http.MethodGet, path: "/api/mall/courses/{courseId}", access: routeOptional},
	{method: http.MethodGet, path: "/api/mall/courses/{courseId}/thumbnail-url", access: routeOptional},
	{method: http.MethodGet, path: "/api/user/me", access: routeProtected},
	{method: http.MethodPut, path: "/api/user/profile", access: routeProtected},
	{method: http.MethodPost, path: "/api/user/profile/email/send-code", access: routeProtected},
	{method: http.MethodPut, path: "/api/user/profile/email", access: routeProtected},
	{method: http.MethodPut, path: "/api/user/password", access: routeProtected},
	{method: http.MethodGet, path: "/api/membership/active", access: routeProtected},
	{method: http.MethodGet, path: "/api/membership/history", access: routeProtected},
	{method: http.MethodGet, path: "/api/membership/billings", access: routeProtected},
	{method: http.MethodPost, path: "/api/membership/cancel", access: routeProtected},
	{method: http.MethodGet, path: "/api/mall/bundles/{bundleId}/pricing-detail", access: routeProtected},
	{method: http.MethodPost, path: "/api/mall/bundles/{bundleId}/purchase", access: routeProtected},
	{method: http.MethodPost, path: "/api/mall/bundles/{bundleId}/unlock", access: routeProtected},
	{method: http.MethodPost, path: "/api/mall/pipelines/{pipelineId}/stages/{stageId}/purchase", access: routeProtected},
	{method: http.MethodPost, path: "/api/mall/stage-orders/{stageOrderId}/exemptions", access: routeProtected},
	{method: http.MethodPost, path: "/api/mall/payments/preview", access: routeProtected},
	{method: http.MethodPost, path: "/api/mall/payments/initiate", access: routeProtected},
	{method: http.MethodGet, path: "/api/pipeline/", access: routeProtected},
	{method: http.MethodGet, path: "/api/pipeline/materials", access: routeProtected},
	{method: http.MethodGet, path: "/api/pipeline/materials/{materialId}/url", access: routeProtected},
	{method: http.MethodGet, path: "/api/pipeline/resource-preview-url", access: routeProtected},
	{method: http.MethodGet, path: "/api/pipeline/courses/{courseId}/complete", access: routeProtected},
	{method: http.MethodGet, path: "/api/pipeline/lessons/{lessonId}", access: routeProtected},
	{method: http.MethodGet, path: "/api/pipeline/lessons/{lessonId}/preview-url", access: routeProtected},
	{method: http.MethodPost, path: "/api/pipeline/lessons/{lessonId}/complete", access: routeProtected},
	{method: http.MethodGet, path: "/api/pipeline/{pipelineUlid}/certificate-url", access: routeProtected},
	{method: http.MethodPost, path: "/api/progress/courses/{courseId}/sync", access: routeProtected},
	{method: http.MethodPost, path: "/api/progress/", access: routeProtected},
	{method: http.MethodGet, path: "/api/progress/", access: routeProtected},
	{method: http.MethodGet, path: "/api/enrollments/", access: routeProtected},
	{method: http.MethodGet, path: "/api/enrollments/{enrollmentId}", access: routeProtected},
	{method: http.MethodGet, path: "/api/resource-packs/", access: routeProtected},
	{method: http.MethodGet, path: "/api/resource-packs/{pack_id}/files", access: routeProtected},
	{method: http.MethodGet, path: "/api/resource-pack-files/{file_id}/view-url", access: routeProtected},
	{method: http.MethodGet, path: "/api/resource-pack-files/{file_id}/thumbnail-url", access: routeProtected},
	{method: http.MethodGet, path: "/api/resource-pack-files/{file_id}/preview-url", access: routeProtected},
	{method: http.MethodPost, path: "/api/quizzes/{quizId}/take", access: routeProtected},
	{method: http.MethodPost, path: "/api/quizzes/{quizId}/complete", access: routeProtected},
	{method: http.MethodGet, path: "/api/quizzes/attempts/{attemptId}/paper", access: routeProtected},
	{method: http.MethodGet, path: "/api/quizzes/attempts/{attemptId}/detail", access: routeProtected},
	{method: http.MethodPost, path: "/api/quizzes/attempts/{attemptId}/draft", access: routeProtected},
	{method: http.MethodPost, path: "/api/quizzes/attempts/{attemptId}/submit", access: routeProtected},
	{method: http.MethodGet, path: "/api/exams/", access: routeProtected},
	{method: http.MethodGet, path: "/api/exams/history", access: routeProtected},
	{method: http.MethodPost, path: "/api/exams/units/{courseUnitUlid}/signup", access: routeProtected},
	{method: http.MethodPost, path: "/api/exams/units/{courseUnitUlid}/retake", access: routeProtected},
	{method: http.MethodPost, path: "/api/exams/units/{courseUnitUlid}/retake-payment", access: routeProtected},
	{method: http.MethodPost, path: "/api/exams/units/{courseUnitUlid}/exemption", access: routeProtected},
	{method: http.MethodGet, path: "/api/exams/{examId}/schedule-url", access: routeProtected},
	{method: http.MethodGet, path: "/api/exams/{examId}/result", access: routeProtected},
	{method: http.MethodGet, path: "/api/exams/{examId}/schedule-callback/{urlType}", access: routeProtected},
	{method: http.MethodPost, path: "/api/exams/{examId}/schedule-callback", access: routeProtected},
	{method: http.MethodGet, path: "/api/credentials/definitions", access: routeProtected},
	{method: http.MethodGet, path: "/api/credentials/applications", access: routeProtected},
	{method: http.MethodGet, path: "/api/credentials/actionable-count", access: routeProtected},
	{method: http.MethodGet, path: "/api/credentials/qualifications", access: routeProtected},
	{method: http.MethodGet, path: "/api/credentials/upload-permission", access: routeProtected},
	{method: http.MethodPost, path: "/api/credentials/application-orders", access: routeProtected},
	{method: http.MethodPost, path: "/api/credentials/upload-url", access: routeProtected},
	{method: http.MethodPost, path: "/api/credentials/submit", access: routeProtected},
	{method: http.MethodPut, path: "/api/credentials/update", access: routeProtected},
	{method: http.MethodGet, path: "/api/certificates/", access: routeProtected},
	{method: http.MethodGet, path: "/api/orders/", access: routeProtected},
	{method: http.MethodGet, path: "/api/orders/{orderId}", access: routeProtected},
	{method: http.MethodPost, path: "/api/orders/cancel", access: routeProtected},
	{method: http.MethodGet, path: "/api/invoices/{orderId}", access: routeProtected},
	{method: http.MethodGet, path: "/api/invoices/{orderId}/pdf", access: routeProtected},
	{method: http.MethodGet, path: "/api/messages/", access: routeProtected},
	{method: http.MethodGet, path: "/api/messages/unread-count", access: routeProtected},
	{method: http.MethodPut, path: "/api/messages/read", access: routeProtected},
	{method: http.MethodPost, path: "/api/messages/delete", access: routeProtected},
	{method: http.MethodGet, path: "/api/messages/{messageId}", access: routeProtected},
	{method: http.MethodGet, path: "/api/dashboard/", access: routeProtected},
	{method: http.MethodGet, path: "/api/dashboard/stats", access: routeProtected},
	{method: http.MethodPost, path: "/api/telemetry", access: routeProtected},
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

func TestRouterMatchesCandidateRouteContract(t *testing.T) {
	router, ok := newTestRouter().(chi.Routes)
	if !ok {
		t.Fatal("router does not expose chi route metadata")
	}

	registered := make(map[string]struct{}, len(candidateRouteContract))
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

	expected := make(map[string]struct{}, len(candidateRouteContract))
	for _, route := range candidateRouteContract {
		key := route.method + " " + route.path
		if _, exists := expected[key]; exists {
			t.Fatalf("duplicate route in candidate contract: %q", key)
		}
		expected[key] = struct{}{}
	}

	var missing []string
	for route := range expected {
		if _, exists := registered[route]; !exists {
			missing = append(missing, route)
		}
	}
	var unexpected []string
	for route := range registered {
		if _, exists := expected[route]; !exists {
			unexpected = append(unexpected, route)
		}
	}
	sort.Strings(missing)
	sort.Strings(unexpected)
	if len(missing) > 0 || len(unexpected) > 0 {
		t.Fatalf("candidate route contract mismatch\nmissing: %v\nunexpected: %v", missing, unexpected)
	}
}

func TestProtectedRoutesRequireAuthentication(t *testing.T) {
	router := newTestRouter()
	for _, route := range candidateRouteContract {
		if route.access != routeProtected {
			continue
		}
		route := route
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(route.method, materializeRoutePath(route.path), nil)

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
