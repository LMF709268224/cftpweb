package handler

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	gcredspb "github.com/afnandelfin620-star/cftptest/cftp/gcreds"
	gmsgpb "github.com/afnandelfin620-star/cftptest/cftp/gmsg"
	"google.golang.org/grpc"
)

type adminWriteCredentialClient struct {
	gcredspb.CredentialServiceClient
	request *gcredspb.AuditApplicationRequest
}

func (s *adminWriteCredentialClient) AuditApplication(_ context.Context, request *gcredspb.AuditApplicationRequest, _ ...grpc.CallOption) (*gcredspb.ApplicationSummary, error) {
	s.request = request
	return &gcredspb.ApplicationSummary{AppUlid: request.GetAppUlid()}, nil
}

type adminWriteMessageClient struct {
	gmsgpb.MessageServiceClient
	request *gmsgpb.RevokeMessageRequest
}

func (s *adminWriteMessageClient) RevokeMessage(_ context.Context, request *gmsgpb.RevokeMessageRequest, _ ...grpc.CallOption) (*gmsgpb.CommonResponse, error) {
	s.request = request
	return &gmsgpb.CommonResponse{Success: true}, nil
}

func TestAuditApplicationUsesSelectedExpiryAndAuthenticatedAdmin(t *testing.T) {
	client := &adminWriteCredentialClient{}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/applications/audit", bytes.NewBufferString(`{
		"application_id":"application-1",
		"approved":true,
		"reject_reason":"approved",
		"valid_until":"2099-08-12T15:30:00+08:00"
	}`))
	request = request.WithContext(WithCandidate(request.Context(), "admin-from-token", "admin@example.test", "Admin", "token"))

	(&Handler{Creds: client}).AuditApplication(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if client.request == nil {
		t.Fatal("AuditApplication was not called")
	}
	if client.request.GetAuditorUlid() != "admin-from-token" || client.request.GetValidUntil() != "2099-08-12T07:30:00Z" {
		t.Fatalf("audit request = %+v", client.request)
	}
}

func TestAuditApplicationRejectsInvalidExpiry(t *testing.T) {
	client := &adminWriteCredentialClient{}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/applications/audit", bytes.NewBufferString(`{
		"application_id":"application-1",
		"approved":true,
		"valid_until":"not-a-timestamp"
	}`))
	request = request.WithContext(WithCandidate(request.Context(), "admin-from-token", "admin@example.test", "Admin", "token"))

	(&Handler{Creds: client}).AuditApplication(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusBadRequest, recorder.Body.String())
	}
	if client.request != nil {
		t.Fatalf("unexpected downstream request = %+v", client.request)
	}
}

func TestAuditApplicationRejectsMissingAuthenticatedAdmin(t *testing.T) {
	client := &adminWriteCredentialClient{}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/applications/audit", bytes.NewBufferString(`{
		"application_id":"application-1",
		"approved":true,
		"valid_until":"2099-08-12T07:30:00Z"
	}`))

	(&Handler{Creds: client}).AuditApplication(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusBadRequest, recorder.Body.String())
	}
	if client.request != nil {
		t.Fatalf("unexpected downstream request = %+v", client.request)
	}
}

func TestRevokeMessageIgnoresRequestAdminIdentity(t *testing.T) {
	client := &adminWriteMessageClient{}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/messages/revoke", bytes.NewBufferString(`{
		"user_ulid":"candidate-1",
		"message_ulid":"message-1",
		"admin_ulid":"spoofed-admin"
	}`))
	request = request.WithContext(WithCandidate(request.Context(), "admin-from-token", "admin@example.test", "Admin", "token"))

	(&Handler{Gmsg: client}).RevokeMessage(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if client.request == nil || client.request.GetAdminUlid() != "admin-from-token" {
		t.Fatalf("revoke request = %+v", client.request)
	}
}
