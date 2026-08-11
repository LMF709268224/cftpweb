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

type pdfRequestReadClientStub struct {
	gcredspb.CredentialServiceClient
	listRequest   *gcredspb.ListPdfRequestsRequest
	detailRequest *gcredspb.GetPdfRequestRequest
}

func (s *pdfRequestReadClientStub) ListPdfRequests(_ context.Context, req *gcredspb.ListPdfRequestsRequest, _ ...grpc.CallOption) (*gcredspb.ListPdfRequestsResponse, error) {
	s.listRequest = req
	return &gcredspb.ListPdfRequestsResponse{
		Requests: []*gcredspb.PdfRequestSummary{{
			RequestUlid:   "pdf-request-1",
			BusinessUnit:  "gprog",
			CandidateUlid: "candidate-1",
			DegreeNo:      "CERT-2026-001",
			Status:        gcredspb.PdfRequestStatus(3),
			CreatedAt:     "2026-08-11T00:00:00Z",
		}},
		HasMore:    true,
		NextCursor: "next-page",
	}, nil
}

func (s *pdfRequestReadClientStub) GetPdfRequestDetail(_ context.Context, req *gcredspb.GetPdfRequestRequest, _ ...grpc.CallOption) (*gcredspb.PdfRequest, error) {
	s.detailRequest = req
	return &gcredspb.PdfRequest{
		RequestUlid:    "pdf-request-1",
		BusinessUnit:   "gprog",
		CandidateUlid:  "candidate-1",
		CredDefUlid:    "credential-definition-1",
		DegreeNo:       "CERT-2026-001",
		TemplateUlid:   "template-1",
		TemplateParams: `{"name":"Regression Candidate"}`,
		Status:         gcredspb.PdfRequestStatus(3),
		PdfFileHash:    "sha256-regression",
		CreatedAt:      "2026-08-11T00:00:00Z",
	}, nil
}

func TestListPdfRequestsReturnsReadOnlyPage(t *testing.T) {
	client := &pdfRequestReadClientStub{}
	h := &Handler{Creds: client}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/pdf-requests?cursor=current-page&page_size=10&sort=1", nil)

	h.ListPdfRequests(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if client.listRequest.GetCursor() != "current-page" || client.listRequest.GetPageSize() != 10 || int32(client.listRequest.GetSortOrder()) != 1 {
		t.Fatalf("list request = %+v", client.listRequest)
	}
	var payload struct {
		Data struct {
			Requests []struct {
				RequestULID string `json:"request_ulid"`
				DegreeNo    string `json:"degree_no"`
				Status      int32  `json:"status"`
			} `json:"requests"`
			HasMore    bool   `json:"has_more"`
			NextCursor string `json:"next_cursor"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(payload.Data.Requests) != 1 || payload.Data.Requests[0].RequestULID != "pdf-request-1" || payload.Data.Requests[0].DegreeNo != "CERT-2026-001" || payload.Data.Requests[0].Status != 3 || !payload.Data.HasMore || payload.Data.NextCursor != "next-page" {
		t.Fatalf("PDF request page = %+v", payload.Data)
	}
}

func TestGetPdfRequestDetailReturnsReadOnlyDetail(t *testing.T) {
	client := &pdfRequestReadClientStub{}
	h := &Handler{Creds: client}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/pdf-requests/pdf-request-1/detail", nil)
	routeContext := chi.NewRouteContext()
	routeContext.URLParams.Add("request_ulid", "pdf-request-1")
	request = request.WithContext(context.WithValue(request.Context(), chi.RouteCtxKey, routeContext))

	h.GetPdfRequestDetail(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if client.detailRequest.GetRequestUlid() != "pdf-request-1" {
		t.Fatalf("detail request = %+v", client.detailRequest)
	}
	var payload struct {
		Data struct {
			RequestULID    string `json:"request_ulid"`
			TemplateParams string `json:"template_params"`
			PdfFileHash    string `json:"pdf_file_hash"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Data.RequestULID != "pdf-request-1" || payload.Data.TemplateParams == "" || payload.Data.PdfFileHash != "sha256-regression" {
		t.Fatalf("PDF request detail = %+v", payload.Data)
	}
}
