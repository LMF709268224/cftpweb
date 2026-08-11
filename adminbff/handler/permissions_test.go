package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	gcredspb "github.com/afnandelfin620-star/cftptest/cftp/gcreds"
	"google.golang.org/grpc"
)

type permissionCredentialClientStub struct {
	gcredspb.CredentialServiceClient
	grantRequest  *gcredspb.GrantUploadPermissionRequest
	revokeRequest *gcredspb.RevokeUploadPermissionRequest
}

func (s *permissionCredentialClientStub) GrantUploadPermission(
	_ context.Context,
	req *gcredspb.GrantUploadPermissionRequest,
	_ ...grpc.CallOption,
) (*gcredspb.UploadPermissionResponse, error) {
	s.grantRequest = req
	return &gcredspb.UploadPermissionResponse{}, nil
}

func (s *permissionCredentialClientStub) RevokeUploadPermission(
	_ context.Context,
	req *gcredspb.RevokeUploadPermissionRequest,
	_ ...grpc.CallOption,
) (*gcredspb.UploadPermissionResponse, error) {
	s.revokeRequest = req
	return &gcredspb.UploadPermissionResponse{}, nil
}

func TestUploadPermissionActionsNormalizeIDsAndUseSessionAdmin(t *testing.T) {
	tests := []struct {
		name   string
		handle func(*Handler, http.ResponseWriter, *http.Request)
		assert func(*testing.T, *permissionCredentialClientStub)
	}{
		{
			name:   "grant",
			handle: (*Handler).GrantUploadPermission,
			assert: func(t *testing.T, client *permissionCredentialClientStub) {
				t.Helper()
				if client.grantRequest == nil {
					t.Fatal("GrantUploadPermission() did not call the credential service")
				}
				got := client.grantRequest
				if got.GetCandidateUlid() != "candidate-1" || got.GetCredDefUlid() != "credential-1" || got.GetOperatorUlid() != "admin-1" || got.GetReason() != "manual review" || got.GetSourceSystem() != "admin_ui" {
					t.Fatalf("grant request = %+v", got)
				}
			},
		},
		{
			name:   "revoke",
			handle: (*Handler).RevokeUploadPermission,
			assert: func(t *testing.T, client *permissionCredentialClientStub) {
				t.Helper()
				if client.revokeRequest == nil {
					t.Fatal("RevokeUploadPermission() did not call the credential service")
				}
				got := client.revokeRequest
				if got.GetCandidateUlid() != "candidate-1" || got.GetCredDefUlid() != "credential-1" || got.GetOperatorUlid() != "admin-1" || got.GetReason() != "manual review" || got.GetSourceSystem() != "admin_ui" {
					t.Fatalf("revoke request = %+v", got)
				}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := &permissionCredentialClientStub{}
			h := &Handler{Creds: client}
			request := httptest.NewRequest(http.MethodPost, "/api/permissions/"+test.name, strings.NewReader(
				`{"candidate_id":" candidate-1 ","cred_def_id":" credential-1 ","reason":"manual review"}`,
			))
			request = request.WithContext(WithCandidate(request.Context(), "admin-1", "admin@example.test", "Admin", "token"))
			recorder := httptest.NewRecorder()

			test.handle(h, recorder, request)

			if recorder.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
			}
			test.assert(t, client)
		})
	}
}

func TestUploadPermissionActionsRequireBothBusinessIDs(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "missing candidate", body: `{"cred_def_ulid":"credential-1"}`},
		{name: "missing credential", body: `{"candidate_ulid":"candidate-1"}`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := &permissionCredentialClientStub{}
			h := &Handler{Creds: client}
			recorder := httptest.NewRecorder()

			h.GrantUploadPermission(recorder, httptest.NewRequest(http.MethodPost, "/api/permissions/grant", strings.NewReader(test.body)))

			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusBadRequest, recorder.Body.String())
			}
			if client.grantRequest != nil {
				t.Fatal("credential service was called without both business IDs")
			}
		})
	}
}
