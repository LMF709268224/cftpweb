package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	gcredspb "github.com/afnandelfin620-star/cftptest/cftp/gcreds"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type applicationCredentialClientStub struct {
	gcredspb.CredentialServiceClient
	auditRequest *gcredspb.AuditApplicationRequest
	auditError   error
}

func (s *applicationCredentialClientStub) AuditApplication(
	_ context.Context,
	req *gcredspb.AuditApplicationRequest,
	_ ...grpc.CallOption,
) (*gcredspb.ApplicationSummary, error) {
	s.auditRequest = req
	if s.auditError != nil {
		return nil, s.auditError
	}
	return &gcredspb.ApplicationSummary{AppUlid: req.GetAppUlid()}, nil
}

func TestAuditApplicationForwardsAdminDecision(t *testing.T) {
	tests := []struct {
		name          string
		body          string
		applicationID string
		approved      bool
		remark        string
		resubmit      bool
	}{
		{
			name:          "approve canonical application id",
			body:          `{"application_id":" app-approve ","approved":true}`,
			applicationID: "app-approve",
			approved:      true,
		},
		{
			name:          "reject legacy app id",
			body:          `{"app_id":"app-reject","reject_reason":"invalid evidence"}`,
			applicationID: "app-reject",
			remark:        "invalid evidence",
		},
		{
			name:          "request resubmission by app ulid",
			body:          `{"app_ulid":"app-resubmit","reject_reason":"upload a clearer file","require_resubmit":true}`,
			applicationID: "app-resubmit",
			remark:        "upload a clearer file",
			resubmit:      true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := &applicationCredentialClientStub{}
			h := &Handler{Creds: client}
			request := httptest.NewRequest(http.MethodPost, "/api/applications/audit", strings.NewReader(test.body))
			request = request.WithContext(WithCandidate(request.Context(), "admin-1", "admin@example.test", "Admin", "token"))
			recorder := httptest.NewRecorder()
			before := time.Now().AddDate(2, 0, 0).Add(-time.Minute)

			h.AuditApplication(recorder, request)

			if recorder.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
			}
			if client.auditRequest == nil {
				t.Fatal("AuditApplication() did not call the credential service")
			}
			got := client.auditRequest
			if got.GetAppUlid() != test.applicationID || got.GetApproved() != test.approved || got.GetAuditRemark() != test.remark || got.GetAllowReupload() != test.resubmit {
				t.Fatalf("audit request = %+v", got)
			}
			if got.GetAuditorUlid() != "admin-1" {
				t.Fatalf("auditor_ulid = %q, want admin-1", got.GetAuditorUlid())
			}
			validUntil, err := time.Parse(time.RFC3339, got.GetValidUntil())
			if err != nil {
				t.Fatalf("valid_until = %q: %v", got.GetValidUntil(), err)
			}
			if validUntil.Before(before) || validUntil.After(time.Now().AddDate(2, 0, 0).Add(time.Minute)) {
				t.Fatalf("valid_until = %s, want approximately two years from now", validUntil)
			}
		})
	}
}

func TestAuditApplicationRejectsInvalidRequestsBeforeDownstream(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "malformed JSON", body: `{"application_id":`},
		{name: "missing application id", body: `{"approved":true}`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := &applicationCredentialClientStub{}
			h := &Handler{Creds: client}
			recorder := httptest.NewRecorder()

			h.AuditApplication(recorder, httptest.NewRequest(http.MethodPost, "/api/applications/audit", strings.NewReader(test.body)))

			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusBadRequest, recorder.Body.String())
			}
			if client.auditRequest != nil {
				t.Fatal("credential service was called for an invalid audit request")
			}
		})
	}
}

func TestAuditApplicationMapsDownstreamUnavailable(t *testing.T) {
	client := &applicationCredentialClientStub{auditError: status.Error(codes.Unavailable, "credentials unavailable")}
	h := &Handler{Creds: client}
	recorder := httptest.NewRecorder()

	h.AuditApplication(recorder, httptest.NewRequest(
		http.MethodPost,
		"/api/applications/audit",
		strings.NewReader(`{"application_id":"app-1","approved":true}`),
	))

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusServiceUnavailable, recorder.Body.String())
	}
}
