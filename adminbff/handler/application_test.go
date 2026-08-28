package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	gcredspb "github.com/afnandelfin620-star/cftptest/cftp/gcreds"
	"github.com/go-chi/chi/v5"
	"google.golang.org/grpc"
)

type applicationReadClientStub struct {
	gcredspb.CredentialServiceClient
	listRequest   *gcredspb.ListApplicationsRequest
	countRequests []*gcredspb.GetApplicationCountRequest
	detailRequest *gcredspb.GetApplicationDetailRequest
	countMu       sync.Mutex
}

func (s *applicationReadClientStub) ListApplications(
	_ context.Context,
	req *gcredspb.ListApplicationsRequest,
	_ ...grpc.CallOption,
) (*gcredspb.ListApplicationsResponse, error) {
	s.listRequest = req
	return &gcredspb.ListApplicationsResponse{
		Applications: []*gcredspb.ApplicationSummary{{
			AppUlid:       "app-1",
			CandidateUlid: "candidate-1",
			CredDefUlid:   "credential-1",
			Status:        "PENDING",
		}},
		HasMore:    true,
		NextCursor: "next-page",
	}, nil
}

func (s *applicationReadClientStub) GetApplicationCount(
	_ context.Context,
	req *gcredspb.GetApplicationCountRequest,
	_ ...grpc.CallOption,
) (*gcredspb.GetApplicationCountResponse, error) {
	s.countMu.Lock()
	s.countRequests = append(s.countRequests, req)
	s.countMu.Unlock()
	statuses := req.GetFilters().GetStatuses()
	if len(statuses) == 0 {
		return &gcredspb.GetApplicationCountResponse{Count: 10}, nil
	}
	switch strings.ToUpper(statuses[0]) {
	case "PENDING":
		return &gcredspb.GetApplicationCountResponse{Count: 5}, nil
	case "APPROVED":
		return &gcredspb.GetApplicationCountResponse{Count: 3}, nil
	case "REJECTED", "REUPLOAD":
		return &gcredspb.GetApplicationCountResponse{Count: 1}, nil
	default:
		return &gcredspb.GetApplicationCountResponse{}, nil
	}
}

func (s *applicationReadClientStub) ListCredentialDefinitions(
	_ context.Context,
	_ *gcredspb.ListCredentialDefinitionsRequest,
	_ ...grpc.CallOption,
) (*gcredspb.ListCredentialDefinitionsResponse, error) {
	return &gcredspb.ListCredentialDefinitionsResponse{
		Definitions: []*gcredspb.CredentialDefinitionSummary{{
			CredDefUlid: "credential-1",
			Name:        "Regression Credential",
		}},
	}, nil
}

func (s *applicationReadClientStub) GetApplicationDetail(
	_ context.Context,
	req *gcredspb.GetApplicationDetailRequest,
	_ ...grpc.CallOption,
) (*gcredspb.Application, error) {
	s.detailRequest = req
	return &gcredspb.Application{
		AppUlid:       req.GetAppUlid(),
		CandidateUlid: "candidate-1",
		CredDefUlid:   "credential-1",
		Status:        "PENDING",
		CreatedAt:     "2026-08-11T00:00:00Z",
	}, nil
}

func (s *applicationReadClientStub) GetCredentialDefinitionDetail(
	_ context.Context,
	_ *gcredspb.GetCredentialDefinitionDetailRequest,
	_ ...grpc.CallOption,
) (*gcredspb.CredentialDefinition, error) {
	return &gcredspb.CredentialDefinition{Name: "Regression Credential"}, nil
}

func TestListApplicationsReturnsReadOnlyApplicationPage(t *testing.T) {
	client := &applicationReadClientStub{}
	h := &Handler{Creds: client}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/applications?status=PENDING&page_size=10", nil)

	h.ListApplications(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if client.listRequest == nil {
		t.Fatal("ListApplications() did not call the credential service")
	}
	if got := client.listRequest.GetFilters().GetStatuses(); len(got) != 1 || got[0] != "PENDING" {
		t.Fatalf("statuses = %v, want [PENDING]", got)
	}
	if client.listRequest.GetPageSize() != 10 {
		t.Fatalf("page_size = %d, want 10", client.listRequest.GetPageSize())
	}
	countRequestsByStatus := make(map[string]int)
	for _, countRequest := range client.countRequests {
		statuses := countRequest.GetFilters().GetStatuses()
		key := "ALL"
		if len(statuses) > 0 {
			key = strings.ToUpper(statuses[0])
		}
		countRequestsByStatus[key]++
	}
	for _, status := range []string{"ALL", "PENDING", "APPROVED", "REJECTED", "REUPLOAD"} {
		if countRequestsByStatus[status] == 0 {
			t.Fatalf("application status subtotal did not count %s: %+v", status, countRequestsByStatus)
		}
	}

	var payload struct {
		Data struct {
			Applications    []map[string]interface{}    `json:"applications"`
			Total           uint32                      `json:"total"`
			StatusSubtotals []applicationStatusSubtotal `json:"status_subtotals"`
			NextCursor      string                      `json:"next_cursor"`
			HasMore         bool                        `json:"has_more"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(payload.Data.Applications) != 1 || payload.Data.Total != 5 || payload.Data.NextCursor != "next-page" || !payload.Data.HasMore {
		t.Fatalf("application page = %+v", payload.Data)
	}
	subtotals := make(map[string]uint32, len(payload.Data.StatusSubtotals))
	for _, subtotal := range payload.Data.StatusSubtotals {
		subtotals[subtotal.Status] = subtotal.Count
	}
	if payload.Data.Applications[0]["cred_def_name"] != "Regression Credential" || len(subtotals) != 5 || subtotals[""] != 10 || subtotals["Pending"] != 5 || subtotals["Approved"] != 3 || subtotals["Rejected"] != 1 || subtotals["Reupload"] != 1 {
		t.Fatalf("application status subtotals = %+v / %+v", payload.Data.Applications[0], payload.Data.StatusSubtotals)
	}
}

func TestGetApplicationReturnsReadOnlyDetail(t *testing.T) {
	client := &applicationReadClientStub{}
	h := &Handler{Creds: client}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/applications/app-1", nil)
	routeContext := chi.NewRouteContext()
	routeContext.URLParams.Add("app_id", "app-1")
	request = request.WithContext(context.WithValue(request.Context(), chi.RouteCtxKey, routeContext))

	h.GetApplication(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if client.detailRequest == nil || client.detailRequest.GetAppUlid() != "app-1" {
		t.Fatalf("detail request = %+v", client.detailRequest)
	}

	var payload struct {
		Data map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Data["app_ulid"] != "app-1" || payload.Data["cred_def_name"] != "Regression Credential" || payload.Data["status"] != "PENDING" {
		t.Fatalf("application detail = %+v", payload.Data)
	}
}
