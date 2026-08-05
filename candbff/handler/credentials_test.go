package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	gcredspb "github.com/afnandelfin620-star/cftptest/cftp/gcreds"
	"google.golang.org/grpc"
)

type credentialClientStub struct {
	gcredspb.CredentialServiceClient
	listCandidateApplicationsResponse *gcredspb.ListApplicationsResponse
	listCandidateApplicationsRequest  *gcredspb.ListApplicationsRequest
}

func (s *credentialClientStub) ListCandidateApplications(
	_ context.Context,
	req *gcredspb.ListApplicationsRequest,
	_ ...grpc.CallOption,
) (*gcredspb.ListApplicationsResponse, error) {
	s.listCandidateApplicationsRequest = req
	return s.listCandidateApplicationsResponse, nil
}

func TestLatestCredentialApplicationUsesCandidateScopedLatestQuery(t *testing.T) {
	const (
		candidateID = "01J00000000000000000000000"
		credDefID   = "01J00000000000000000000001"
		appID       = "01J00000000000000000000002"
	)
	client := &credentialClientStub{
		listCandidateApplicationsResponse: &gcredspb.ListApplicationsResponse{
			Applications: []*gcredspb.ApplicationSummary{
				{
					AppUlid:       appID,
					CandidateUlid: candidateID,
					CredDefUlid:   credDefID,
					Status:        "PENDING",
				},
			},
		},
	}
	h := &Handler{Creds: client}

	got, err := h.latestCredentialApplication(context.Background(), candidateID, credDefID)
	if err != nil {
		t.Fatalf("latestCredentialApplication returned error: %v", err)
	}
	if got["app_ulid"] != appID {
		t.Fatalf("app_ulid = %v, want %q", got["app_ulid"], appID)
	}

	req := client.listCandidateApplicationsRequest
	if req == nil {
		t.Fatal("ListCandidateApplications was not called")
	}
	if req.GetFilters().GetCandidateUlid() != candidateID {
		t.Fatalf("candidate_ulid = %q, want %q", req.GetFilters().GetCandidateUlid(), candidateID)
	}
	if req.GetFilters().GetCredDefUlid() != credDefID {
		t.Fatalf("cred_def_ulid = %q, want %q", req.GetFilters().GetCredDefUlid(), credDefID)
	}
	if req.GetPageSize() != 1 {
		t.Fatalf("page_size = %d, want 1", req.GetPageSize())
	}
	if req.GetSortOrder() != gcredspb.SortOrder_SORT_ORDER_DESC {
		t.Fatalf("sort_order = %v, want SORT_ORDER_DESC", req.GetSortOrder())
	}
}

func TestLatestCredentialApplicationReturnsNilWhenNoApplicationExists(t *testing.T) {
	client := &credentialClientStub{
		listCandidateApplicationsResponse: &gcredspb.ListApplicationsResponse{},
	}
	h := &Handler{Creds: client}

	got, err := h.latestCredentialApplication(
		context.Background(),
		"01J00000000000000000000000",
		"01J00000000000000000000001",
	)
	if err != nil {
		t.Fatalf("latestCredentialApplication returned error: %v", err)
	}
	if got != nil {
		t.Fatalf("latest application = %#v, want nil", got)
	}
}

func TestCredentialHandlersRejectInvalidRequestsBeforeCallingServices(t *testing.T) {
	tests := []struct {
		name   string
		method string
		target string
		body   string
		handle func(*Handler, http.ResponseWriter, *http.Request)
	}{
		{
			name:   "qualification IDs are required",
			method: http.MethodGet,
			target: "/api/credentials/qualifications",
			handle: func(h *Handler, w http.ResponseWriter, r *http.Request) {
				h.CheckCandidateQualifications(w, r)
			},
		},
		{
			name:   "application order fields are required",
			method: http.MethodPost,
			target: "/api/credentials/application-orders",
			body:   `{}`,
			handle: func(h *Handler, w http.ResponseWriter, r *http.Request) {
				h.CreateCredentialApplicationOrder(w, r)
			},
		},
		{
			name:   "upload fields are required",
			method: http.MethodPost,
			target: "/api/credentials/upload-url",
			body:   `{}`,
			handle: func(h *Handler, w http.ResponseWriter, r *http.Request) {
				h.RequestUploadUrl(w, r)
			},
		},
		{
			name:   "credential definition is required for submission",
			method: http.MethodPost,
			target: "/api/credentials/submit",
			body:   `{}`,
			handle: func(h *Handler, w http.ResponseWriter, r *http.Request) {
				h.SubmitApplication(w, r)
			},
		},
		{
			name:   "all submitted file fields are required",
			method: http.MethodPost,
			target: "/api/credentials/submit",
			body:   `{"cred_def_ulid":"credential-1","files":[{"file_hash":"hash","file_name":"proof.pdf","file_ext":"pdf"}]}`,
			handle: func(h *Handler, w http.ResponseWriter, r *http.Request) {
				h.SubmitApplication(w, r)
			},
		},
		{
			name:   "application ID is required for update",
			method: http.MethodPut,
			target: "/api/credentials/update",
			body:   `{}`,
			handle: func(h *Handler, w http.ResponseWriter, r *http.Request) {
				h.UpdateApplication(w, r)
			},
		},
		{
			name:   "all updated file fields are required",
			method: http.MethodPut,
			target: "/api/credentials/update",
			body:   `{"app_ulid":"application-1","files":[{"file_hash":"hash","file_name":"proof.pdf","file_ext":"pdf"}]}`,
			handle: func(h *Handler, w http.ResponseWriter, r *http.Request) {
				h.UpdateApplication(w, r)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := newCandidateHandlerRequest(test.method, test.target, test.body, "candidate-1", nil)
			recorder := httptest.NewRecorder()

			test.handle(&Handler{}, recorder, request)

			assertHandlerAPIError(t, recorder, http.StatusBadRequest, ErrInvalidRequest)
		})
	}
}
