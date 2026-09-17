package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"adminbff/handler"

	"github.com/afnandelfin620-star/cftptest/cftp/util"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type recordingAuditPublisher struct {
	events []*util.NatsAuditEvent
}

func (p *recordingAuditPublisher) Publish(_ context.Context, event *util.NatsAuditEvent) error {
	p.events = append(p.events, event)
	return nil
}

func (p *recordingAuditPublisher) Close() error { return nil }

func auditTestRouter(publisher auditEventPublisher, method, path string, endpoint http.HandlerFunc) http.Handler {
	s := &Server{auditPublisher: publisher}
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := handler.WithCandidate(r.Context(), "01ADMINULID00000000000000", "admin@example.test", "Admin User", "")
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})
	r.Use(s.auditMiddleware)
	r.MethodFunc(method, path, endpoint)
	return r
}

func TestAuditMiddlewarePublishesSearchableMutation(t *testing.T) {
	publisher := &recordingAuditPublisher{}
	router := auditTestRouter(publisher, http.MethodPut, "/api/catalogs/{catalog_id}", func(w http.ResponseWriter, _ *http.Request) {
		handler.WriteJSON(w, http.StatusOK, map[string]any{"updated": true})
	})
	request := httptest.NewRequest(http.MethodPut, "/api/catalogs/catalog-123", strings.NewReader(`{"name":"New name"}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("User-Agent", "audit-test")
	request.Header.Set("X-Trace-ID", "trace-123")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if len(publisher.events) != 1 {
		t.Fatalf("published events = %d, want 1", len(publisher.events))
	}
	event := publisher.events[0]
	if event.Action != "update_catalog" || event.Status != "SUCCESS" {
		t.Fatalf("event action/status = %q/%q", event.Action, event.Status)
	}
	if event.Resource.Type != "catalog" || event.Resource.ID != "catalog-123" {
		t.Fatalf("event resource = %+v", event.Resource)
	}
	if event.Operator.ID != "01ADMINULID00000000000000" || event.Operator.Name != "Admin User" || event.Operator.Email != "admin@example.test" || event.Operator.Role != "admin" {
		t.Fatalf("event operator = %+v", event.Operator)
	}
	if event.Context.TraceID != "trace-123" || event.Context.UserAgent != "audit-test" || event.Context.RequestURI != "/api/catalogs/catalog-123" {
		t.Fatalf("event context = %+v", event.Context)
	}
	var details map[string]any
	if err := json.Unmarshal([]byte(event.Details), &details); err != nil {
		t.Fatalf("decode details: %v", err)
	}
	if details["route_pattern"] != "/api/catalogs/{catalog_id}" || details["http_status"] != float64(http.StatusOK) {
		t.Fatalf("details = %+v", details)
	}
	if strings.Contains(event.Details, "New name") {
		t.Fatalf("details leaked request body: %s", event.Details)
	}
}

func TestAuditMiddlewareExtractsCreatedResourceIDFromResponse(t *testing.T) {
	publisher := &recordingAuditPublisher{}
	var matchedPattern string
	router := auditTestRouter(publisher, http.MethodPost, "/api/catalogs/", func(w http.ResponseWriter, r *http.Request) {
		matchedPattern = chi.RouteContext(r.Context()).RoutePattern()
		handler.WriteJSON(w, http.StatusCreated, map[string]any{"catalog_id": "catalog-created"})
	})
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/catalogs/", strings.NewReader(`{"name":"Catalog"}`)))

	if len(publisher.events) != 1 || publisher.events[0].Resource.ID != "catalog-created" {
		t.Fatalf("route pattern = %q; events = %+v", matchedPattern, publisher.events)
	}
}

func TestAuditMiddlewareRecordsFailedMutationWithoutSensitiveBody(t *testing.T) {
	publisher := &recordingAuditPublisher{}
	router := auditTestRouter(publisher, http.MethodPut, "/api/user/password", func(w http.ResponseWriter, _ *http.Request) {
		handler.WriteError(w, http.StatusConflict, handler.ErrPrecondition, "password rejected")
	})
	request := httptest.NewRequest(http.MethodPut, "/api/user/password", strings.NewReader(`{"old_password":"secret-old","new_password":"secret-new"}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if len(publisher.events) != 1 {
		t.Fatalf("published events = %d, want 1", len(publisher.events))
	}
	event := publisher.events[0]
	if event.Status != "FAILED" {
		t.Fatalf("status = %q, want FAILED", event.Status)
	}
	if strings.Contains(event.Details, "secret-old") || strings.Contains(event.Details, "secret-new") {
		t.Fatalf("details leaked password: %s", event.Details)
	}
	if !strings.Contains(event.Details, string(handler.ErrPrecondition)) {
		t.Fatalf("details did not include safe error code: %s", event.Details)
	}
}

func TestApplicationReviewUsesDecisionSpecificAction(t *testing.T) {
	spec := adminAuditOperations[route(http.MethodPost, "/api/applications/audit")]
	if action := resolveAuditAction(spec, []byte(`{"approved":true}`)); action != "approve_application" {
		t.Fatalf("approved action = %q", action)
	}
	if action := resolveAuditAction(spec, []byte(`{"approved":false}`)); action != "reject_application" {
		t.Fatalf("rejected action = %q", action)
	}
}

func TestAuditMiddlewareRecordsPanickedMutationAsFailed(t *testing.T) {
	publisher := &recordingAuditPublisher{}
	router := auditTestRouter(publisher, http.MethodPut, "/api/catalogs/{catalog_id}", func(http.ResponseWriter, *http.Request) {
		panic("test panic")
	})
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPut, "/api/catalogs/catalog-123", nil))

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", recorder.Code)
	}
	if len(publisher.events) != 1 || publisher.events[0].Status != "FAILED" {
		t.Fatalf("events = %+v", publisher.events)
	}
}

func TestAdminAuditTaxonomyUsesStableSearchableNames(t *testing.T) {
	namePattern := regexp.MustCompile(`^[a-z][a-z0-9]*(?:_[a-z0-9]+)*$`)
	for key, spec := range adminAuditOperations {
		if !namePattern.MatchString(spec.Action) || len(spec.Action) > 128 {
			t.Errorf("%s %s has invalid action %q", key.Method, key.Path, spec.Action)
		}
		if !namePattern.MatchString(spec.ResourceType) || len(spec.ResourceType) > 64 {
			t.Errorf("%s %s has invalid resource type %q", key.Method, key.Path, spec.ResourceType)
		}
		if spec.BoolAction != nil {
			if !namePattern.MatchString(spec.BoolAction.TrueAction) || !namePattern.MatchString(spec.BoolAction.FalseAction) {
				t.Errorf("%s %s has invalid conditional actions %+v", key.Method, key.Path, spec.BoolAction)
			}
		}
	}
}
