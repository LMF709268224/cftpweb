package handler

import (
	"net/http"
	"strings"

	gauditpb "github.com/afnandelfin620-star/cftptest/cftp/gaudit"
)

// ListAuditLogs GET /api/audit/logs
func (h *Handler) ListAuditLogs(w http.ResponseWriter, r *http.Request) {
	page := parseCursorPage(r, 20)

	resp, err := h.Audit.ListAuditLogs(r.Context(), &gauditpb.ListAuditLogsRequest{
		Filters: &gauditpb.AuditLogFilters{
			OperatorId:    strings.TrimSpace(r.URL.Query().Get("operator_id")),
			SourceService: strings.TrimSpace(r.URL.Query().Get("source_service")),
			Action:        strings.TrimSpace(r.URL.Query().Get("action")),
			ResourceType:  strings.TrimSpace(r.URL.Query().Get("resource_type")),
			ResourceId:    strings.TrimSpace(r.URL.Query().Get("resource_id")),
			Status:        strings.TrimSpace(r.URL.Query().Get("status")),
			StartTime:     strings.TrimSpace(r.URL.Query().Get("start_time")),
			EndTime:       strings.TrimSpace(r.URL.Query().Get("end_time")),
			Keyword:       strings.TrimSpace(r.URL.Query().Get("keyword")),
		},
		Cursor:   page.Cursor,
		PageSize: page.PageSize,
		SortOrder: gauditpb.SortOrder(page.Sort),
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// GetAuditLogDetail GET /api/audit/logs/{audit_ulid}
func (h *Handler) GetAuditLogDetail(w http.ResponseWriter, r *http.Request) {
	auditULID, ok := requiredURLParam(w, r, "audit_ulid")
	if !ok {
		return
	}

	resp, err := h.Audit.GetAuditLogDetail(r.Context(), &gauditpb.GetAuditLogDetailRequest{
		AuditUlid: auditULID,
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}
