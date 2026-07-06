package handler

import (
	"net/http"
	"strings"

	gauditpb "github.com/afnandelfin620-star/cftptest/cftp/gaudit"
)

const defaultAuditLogPageSize uint32 = 20
const maxAuditLogPageSize uint32 = 100

// ListAuditLogs GET /api/audit/logs
func (h *Handler) ListAuditLogs(w http.ResponseWriter, r *http.Request) {
	page := parseUint32Query(r, "page")
	if page == 0 {
		page = 1
	}
	pageSize := parseUint32Query(r, "page_size")
	if pageSize == 0 {
		pageSize = defaultAuditLogPageSize
	}
	if pageSize > maxAuditLogPageSize {
		pageSize = maxAuditLogPageSize
	}

	resp, err := h.Audit.ListAuditLogs(r.Context(), &gauditpb.ListAuditLogsRequest{
		OperatorId:    strings.TrimSpace(r.URL.Query().Get("operator_id")),
		SourceService: strings.TrimSpace(r.URL.Query().Get("source_service")),
		Action:        strings.TrimSpace(r.URL.Query().Get("action")),
		ResourceType:  strings.TrimSpace(r.URL.Query().Get("resource_type")),
		ResourceId:    strings.TrimSpace(r.URL.Query().Get("resource_id")),
		Status:        strings.TrimSpace(r.URL.Query().Get("status")),
		StartTime:     strings.TrimSpace(r.URL.Query().Get("start_time")),
		EndTime:       strings.TrimSpace(r.URL.Query().Get("end_time")),
		Keyword:       strings.TrimSpace(r.URL.Query().Get("keyword")),
		Page:          page,
		PageSize:      pageSize,
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
