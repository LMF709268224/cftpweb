package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	gcredspb "github.com/afnandelfin620-star/cftptest/cftp/gcreds"
	"github.com/go-chi/chi/v5"
	"google.golang.org/grpc"
)

type credentialListClient struct {
	gcredspb.CredentialServiceClient
	listRequest  *gcredspb.ListCredentialsRequest
	countRequest *gcredspb.GetCredentialCountRequest
}

func (s *credentialListClient) ListCredentials(_ context.Context, request *gcredspb.ListCredentialsRequest, _ ...grpc.CallOption) (*gcredspb.ListCredentialsResponse, error) {
	s.listRequest = request
	return &gcredspb.ListCredentialsResponse{
		Credentials: []*gcredspb.CredentialSummary{{
			CredUlid:      "credential-current",
			CandidateUlid: "candidate-current",
			CredDefUlid:   "definition-current",
			Status:        gcredspb.CredentialStatus_CREDENTIAL_STATUS_REVOKED,
			Source:        gcredspb.CredentialSource_CREDENTIAL_SOURCE_PDF_CERT,
			AuditRemark:   "duplicate record",
			IsCurrent:     true,
			AuditTime:     "2026-09-20T08:00:00Z",
		}},
		HasMore:    true,
		NextCursor: "next-page",
		PrevCursor: "previous-page",
	}, nil
}

func (s *credentialListClient) GetCredentialCount(_ context.Context, request *gcredspb.GetCredentialCountRequest, _ ...grpc.CallOption) (*gcredspb.GetCredentialCountResponse, error) {
	s.countRequest = request
	return &gcredspb.GetCredentialCountResponse{Count: 1}, nil
}

func (s *credentialListClient) ListCredentialDefinitions(_ context.Context, _ *gcredspb.ListCredentialDefinitionsRequest, _ ...grpc.CallOption) (*gcredspb.ListCredentialDefinitionsResponse, error) {
	return &gcredspb.ListCredentialDefinitionsResponse{Definitions: []*gcredspb.CredentialDefinitionSummary{{
		CredDefUlid: "definition-current",
		Name:        "CFtP Certification",
	}}}, nil
}

func (s *credentialListClient) GetCredentialDetail(_ context.Context, request *gcredspb.GetCredentialDetailRequest, _ ...grpc.CallOption) (*gcredspb.Credential, error) {
	return &gcredspb.Credential{
		CredUlid:      request.GetCredUlid(),
		CandidateUlid: "candidate-current",
		CredDefUlid:   "definition-current",
		Status:        gcredspb.CredentialStatus_CREDENTIAL_STATUS_REVOKED,
		AuditRemark:   "duplicate record",
		Source:        gcredspb.CredentialSource_CREDENTIAL_SOURCE_APPLICATION,
		IsCurrent:     true,
		AuditTime:     "2026-09-20T08:00:00Z",
	}, nil
}

func (s *credentialListClient) GetCredentialDefinitionDetail(_ context.Context, request *gcredspb.GetCredentialDefinitionDetailRequest, _ ...grpc.CallOption) (*gcredspb.CredentialDefinition, error) {
	return &gcredspb.CredentialDefinition{CredDefUlid: request.GetCredDefUlid(), Name: "CFtP Certification"}, nil
}

func TestListCredentialsForwardsCurrentFilterAndReturnsLifecycleFields(t *testing.T) {
	client := &credentialListClient{}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/credentials?is_current=true&page_size=50&source=2", nil)

	(&Handler{Creds: client}).ListCredentials(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if client.listRequest == nil || client.listRequest.GetFilters() == nil || client.listRequest.GetFilters().IsCurrent == nil || !client.listRequest.GetFilters().GetIsCurrent() {
		t.Fatalf("list request did not forward is_current=true: %+v", client.listRequest)
	}
	if client.countRequest == nil || client.countRequest.GetFilters() == nil || client.countRequest.GetFilters().IsCurrent == nil || !client.countRequest.GetFilters().GetIsCurrent() {
		t.Fatalf("count request did not forward is_current=true: %+v", client.countRequest)
	}
	if client.listRequest.GetFilters().GetSource() != gcredspb.CredentialSource_CREDENTIAL_SOURCE_PDF_CERT || client.countRequest.GetFilters().GetSource() != gcredspb.CredentialSource_CREDENTIAL_SOURCE_PDF_CERT {
		t.Fatalf("source filter differs between list and count: list=%v count=%v", client.listRequest.GetFilters().GetSource(), client.countRequest.GetFilters().GetSource())
	}

	var payload struct {
		Data struct {
			Credentials []struct {
				CredentialName string `json:"credential_name"`
				AuditRemark    string `json:"audit_remark"`
				AuditTime      string `json:"audit_time"`
				IsCurrent      bool   `json:"is_current"`
				Source         int    `json:"source"`
			} `json:"credentials"`
			NextCursor string `json:"next_cursor"`
			PrevCursor string `json:"prev_cursor"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(payload.Data.Credentials) != 1 {
		t.Fatalf("credentials = %d, want 1", len(payload.Data.Credentials))
	}
	credential := payload.Data.Credentials[0]
	if credential.CredentialName != "CFtP Certification" || credential.AuditRemark != "duplicate record" || credential.AuditTime != "2026-09-20T08:00:00Z" || !credential.IsCurrent || credential.Source != 2 {
		t.Fatalf("credential payload = %+v", credential)
	}
	if payload.Data.NextCursor != "next-page" || payload.Data.PrevCursor != "previous-page" {
		t.Fatalf("pagination cursors = next %q, prev %q", payload.Data.NextCursor, payload.Data.PrevCursor)
	}
}

func TestListCredentialsForwardsApplicationSource(t *testing.T) {
	client := &credentialListClient{}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/credentials?source=1", nil)

	(&Handler{Creds: client}).ListCredentials(recorder, request)

	if recorder.Code != http.StatusOK || client.listRequest.GetFilters().GetSource() != gcredspb.CredentialSource_CREDENTIAL_SOURCE_APPLICATION || client.countRequest.GetFilters().GetSource() != gcredspb.CredentialSource_CREDENTIAL_SOURCE_APPLICATION {
		t.Fatalf("application source filter not forwarded to list and count: status=%d list=%+v count=%+v", recorder.Code, client.listRequest, client.countRequest)
	}
}

func TestListCredentialsRejectsInvalidSource(t *testing.T) {
	for _, source := range []string{"application", "pdf_cert", "3", "", "1&source=2"} {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/api/credentials?source="+source, nil)
		(&Handler{}).ListCredentials(recorder, request)
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("source %q: status=%d, want 400", source, recorder.Code)
		}
	}
}

func TestListCredentialsRejectsInvalidCurrentFilter(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/credentials?is_current=latest", nil)

	(&Handler{}).ListCredentials(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusBadRequest, recorder.Body.String())
	}
}

func TestGetCredentialDetailReturnsLifecycleFields(t *testing.T) {
	client := &credentialListClient{}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/credentials/credential-current", nil)
	routeContext := chi.NewRouteContext()
	routeContext.URLParams.Add("cred_ulid", "credential-current")
	request = request.WithContext(context.WithValue(request.Context(), chi.RouteCtxKey, routeContext))

	(&Handler{Creds: client}).GetCredentialDetail(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	var payload struct {
		Data struct {
			CredentialName string `json:"credential_name"`
			AuditRemark    string `json:"audit_remark"`
			AuditTime      string `json:"audit_time"`
			IsCurrent      bool   `json:"is_current"`
			Source         int    `json:"source"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Data.CredentialName != "CFtP Certification" || payload.Data.AuditRemark != "duplicate record" || payload.Data.AuditTime != "2026-09-20T08:00:00Z" || !payload.Data.IsCurrent || payload.Data.Source != 1 {
		t.Fatalf("credential detail payload = %+v", payload.Data)
	}
}
