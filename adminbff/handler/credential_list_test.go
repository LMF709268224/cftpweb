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
	return &gcredspb.ListCredentialsResponse{Credentials: []*gcredspb.CredentialSummary{{
		CredUlid:      "credential-current",
		CandidateUlid: "candidate-current",
		CredDefUlid:   "definition-current",
		Status:        gcredspb.CredentialStatus_CREDENTIAL_STATUS_REVOKED,
		AuditRemark:   "duplicate record",
		IsCurrent:     true,
		AuditTime:     "2026-09-20T08:00:00Z",
	}}}, nil
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
	request := httptest.NewRequest(http.MethodGet, "/api/credentials?is_current=true&page_size=50", nil)

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

	var payload struct {
		Data struct {
			Credentials []struct {
				CredentialName string `json:"credential_name"`
				AuditRemark    string `json:"audit_remark"`
				AuditTime      string `json:"audit_time"`
				IsCurrent      bool   `json:"is_current"`
			} `json:"credentials"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(payload.Data.Credentials) != 1 {
		t.Fatalf("credentials = %d, want 1", len(payload.Data.Credentials))
	}
	credential := payload.Data.Credentials[0]
	if credential.CredentialName != "CFtP Certification" || credential.AuditRemark != "duplicate record" || credential.AuditTime != "2026-09-20T08:00:00Z" || !credential.IsCurrent {
		t.Fatalf("credential payload = %+v", credential)
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
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Data.CredentialName != "CFtP Certification" || payload.Data.AuditRemark != "duplicate record" || payload.Data.AuditTime != "2026-09-20T08:00:00Z" || !payload.Data.IsCurrent {
		t.Fatalf("credential detail payload = %+v", payload.Data)
	}
}
