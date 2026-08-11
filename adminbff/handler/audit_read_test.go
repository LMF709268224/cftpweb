package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	gauditpb "github.com/afnandelfin620-star/cftptest/cftp/gaudit"
	"github.com/go-chi/chi/v5"
	"google.golang.org/grpc"
)

type auditReadClientStub struct {
	gauditpb.AuditServiceClient
	listRequest   *gauditpb.ListAuditLogsRequest
	detailRequest *gauditpb.GetAuditLogDetailRequest
}

func (s *auditReadClientStub) ListAuditLogs(_ context.Context, req *gauditpb.ListAuditLogsRequest, _ ...grpc.CallOption) (*gauditpb.ListAuditLogsResponse, error) {
	s.listRequest = req
	return &gauditpb.ListAuditLogsResponse{
		Items: []*gauditpb.AuditLogSummary{{
			AuditUlid:           "audit-1",
			CreatedAt:           "2026-08-11T00:00:00Z",
			SourceService:       "gcreds",
			Action:              "READ",
			Status:              "SUCCESS",
			SummaryText:         "Viewed credential application",
			OperatorId:          "admin-1",
			OperatorName:        "Regression Admin",
			ResourceType:        "credential_application",
			ResourceId:          "application-1",
			ResourceDisplayName: "Regression Application",
		}},
		HasMore:    true,
		NextCursor: "next-page",
	}, nil
}

func (s *auditReadClientStub) GetAuditLogDetail(_ context.Context, req *gauditpb.GetAuditLogDetailRequest, _ ...grpc.CallOption) (*gauditpb.AuditLogItem, error) {
	s.detailRequest = req
	return &gauditpb.AuditLogItem{
		Summary: &gauditpb.AuditLogSummary{
			AuditUlid:           "audit-1",
			SourceService:       "gcreds",
			Action:              "READ",
			Status:              "SUCCESS",
			ResourceId:          "application-1",
			ResourceDisplayName: "Regression Application",
		},
		Details: `{"field":"status"}`,
	}, nil
}

func TestListAuditLogsPassesReadOnlyFilters(t *testing.T) {
	client := &auditReadClientStub{}
	h := &Handler{Audit: client}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/audit/logs?operator_id=admin-1&source_service=gcreds&action=READ&resource_type=credential_application&resource_id=application-1&status=SUCCESS&keyword=credential&cursor=current-page&page_size=10", nil)

	h.ListAuditLogs(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	filters := client.listRequest.GetFilters()
	if filters.GetOperatorId() != "admin-1" || filters.GetSourceService() != "gcreds" || filters.GetAction() != "READ" || filters.GetResourceType() != "credential_application" || filters.GetResourceId() != "application-1" || filters.GetStatus() != "SUCCESS" || filters.GetKeyword() != "credential" {
		t.Fatalf("audit filters = %+v", filters)
	}
	if client.listRequest.GetCursor() != "current-page" || client.listRequest.GetPageSize() != 10 {
		t.Fatalf("list request = %+v", client.listRequest)
	}
	var payload struct {
		Data struct {
			Items []struct {
				AuditULID   string `json:"audit_ulid"`
				SummaryText string `json:"summary_text"`
			} `json:"items"`
			HasMore bool `json:"has_more"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(payload.Data.Items) != 1 || payload.Data.Items[0].AuditULID != "audit-1" || payload.Data.Items[0].SummaryText != "Viewed credential application" || !payload.Data.HasMore {
		t.Fatalf("audit page = %+v", payload.Data)
	}
}

func TestGetAuditLogDetailReturnsReadOnlyDetail(t *testing.T) {
	client := &auditReadClientStub{}
	h := &Handler{Audit: client}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/audit/logs/audit-1", nil)
	routeContext := chi.NewRouteContext()
	routeContext.URLParams.Add("audit_ulid", "audit-1")
	request = request.WithContext(context.WithValue(request.Context(), chi.RouteCtxKey, routeContext))

	h.GetAuditLogDetail(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if client.detailRequest.GetAuditUlid() != "audit-1" {
		t.Fatalf("detail request = %+v", client.detailRequest)
	}
	var payload struct {
		Data struct {
			Summary struct {
				AuditULID string `json:"audit_ulid"`
				Action    string `json:"action"`
			} `json:"summary"`
			Details string `json:"details"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Data.Summary.AuditULID != "audit-1" || payload.Data.Summary.Action != "READ" || payload.Data.Details != `{"field":"status"}` {
		t.Fatalf("audit detail = %+v", payload.Data)
	}
}
