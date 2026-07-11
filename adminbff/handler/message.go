package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/oklog/ulid/v2"

	gmsgpb "github.com/afnandelfin620-star/cftptest/cftp/gmsg"
)

type SendMessageInput struct {
	UserUlids    []string `json:"user_ids"`
	TemplateId   string   `json:"template_id"`
	TemplatePath string   `json:"template_path"`
	Payload      string   `json:"payload"`
	MsgType      int32    `json:"msg_type"`
}

func (h *Handler) SendMessage(w http.ResponseWriter, r *http.Request) {
	var input SendMessageInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}

	templatePath := firstNonEmpty(input.TemplatePath, input.TemplateId)
	if len(input.UserUlids) == 0 {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "user_ids is required")
		return
	}
	if !requireRequestField(w, templatePath, "template_path") {
		return
	}

	msgType := gmsgpb.MsgType_SYSTEM_NOTICE
	if input.MsgType > 0 {
		msgType = gmsgpb.MsgType(input.MsgType)
	}

	resp, err := h.Gmsg.SendMessage(r.Context(), &gmsgpb.SendMessageRequest{
		UserIds:      input.UserUlids,
		MessageUlid:  ulid.Make().String(),
		TemplatePath: templatePath,
		Payload:      input.Payload,
		MsgType:      msgType,
		MsgSource:    gmsgpb.MsgSource_MANUAL_ADMIN,
		SenderUlid:   AdminID(r),
	})
	if err != nil {
		slog.Error("SendMessage failed", "error", err)
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, map[string]interface{}{
		"count":  resp.GetCount(),
		"status": resp.GetStatus(),
	})
}

func (h *Handler) ListSentMessages(w http.ResponseWriter, r *http.Request) {
	page := parseCursorPage(r, 50)

	var statusPtr *gmsgpb.MessageStatus
	if statusStr := r.URL.Query().Get("status"); statusStr != "" {
		if v, err := strconv.Atoi(statusStr); err == nil {
			statusEnum := gmsgpb.MessageStatus(v)
			statusPtr = &statusEnum
		}
	}

	filters := &gmsgpb.MessageAdminFilters{
		Status: statusPtr,
	}

	resp, err := h.Gmsg.ListMessagesAdmin(r.Context(), &gmsgpb.ListMessagesAdminRequest{
		Filters:   filters,
		Cursor:    page.Cursor,
		PageSize:  page.PageSize,
		SortOrder: gmsgpb.SortOrder(page.Sort),
	})
	if err != nil {
		slog.Error("ListMessagesAdmin failed", "error", err)
		HandleGrpcError(w, err)
		return
	}

	countResp, err := h.Gmsg.GetMessageCountAdmin(r.Context(), &gmsgpb.GetMessageCountAdminRequest{
		Filters: filters,
	})
	if err != nil {
		slog.Error("GetMessageCountAdmin failed", "error", err)
		HandleGrpcError(w, err)
		return
	}

	payload := jsonPayloadObject(resp)
	payload["total"] = countResp.GetCount()
	WriteJSON(w, http.StatusOK, payload)
}

func (h *Handler) CreateTemplate(w http.ResponseWriter, r *http.Request) {
	var req gmsgpb.CreateTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}

	if !requireRequestFields(w, req.Path, "path", req.TitleTpl, "title_tpl", req.ContentTpl, "content_tpl") {
		return
	}
	if req.Description == "" {
		req.Description = "-"
	}
	parameterSchema, err := normalizeParameterSchema(req.ParameterSchema)
	if err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "parameter_schema must be valid JSON")
		return
	}
	req.ParameterSchema = parameterSchema

	resp, err := h.Gmsg.CreateTemplate(r.Context(), &req)
	if err != nil {
		slog.Error("CreateTemplate failed", "error", err)
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) ListTemplates(w http.ResponseWriter, r *http.Request) {
	keyword := r.URL.Query().Get("keyword")
	page := parseCursorPage(r, 10)
	filters := &gmsgpb.TemplateFilters{
		Keyword: keyword,
	}
	resp, err := h.Gmsg.ListTemplates(r.Context(), &gmsgpb.ListTemplatesRequest{
		Filters:   filters,
		Cursor:    page.Cursor,
		PageSize:  page.PageSize,
		SortOrder: gmsgpb.SortOrder(page.Sort),
	})
	if err != nil {
		slog.Error("ListTemplates failed", "error", err)
		HandleGrpcError(w, err)
		return
	}

	countResp, err := h.Gmsg.GetTemplateCount(r.Context(), &gmsgpb.GetTemplateCountRequest{
		Filters: filters,
	})
	if err != nil {
		slog.Error("GetTemplateCount failed", "error", err)
		HandleGrpcError(w, err)
		return
	}

	payload := jsonPayloadObject(resp)
	payload["page_size"] = page.PageSize
	payload["total"] = countResp.GetCount()
	WriteJSON(w, http.StatusOK, payload)
}

func (h *Handler) GetMessageTemplate(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if !requireRequestField(w, path, "path") {
		return
	}
	resp, err := h.Gmsg.GetTemplateDetail(r.Context(), &gmsgpb.GetTemplateDetailRequest{Path: path})
	if err != nil {
		slog.Error("GetMessageTemplate failed", "error", err)
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) UpdateTemplate(w http.ResponseWriter, r *http.Request) {
	var req gmsgpb.UpdateTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}
	if !requireRequestFields(w, req.Path, "path", req.TitleTpl, "title_tpl", req.ContentTpl, "content_tpl") {
		return
	}
	parameterSchema, err := normalizeParameterSchema(req.ParameterSchema)
	if err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "parameter_schema must be valid JSON")
		return
	}
	req.ParameterSchema = parameterSchema

	resp, err := h.Gmsg.UpdateTemplate(r.Context(), &req)
	if err != nil {
		slog.Error("UpdateTemplate failed", "error", err)
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) DeleteMessageTemplate(w http.ResponseWriter, r *http.Request) {
	// TODO(microservice-missing-api): gmsg does not provide DeleteTemplate yet.
	WriteError(w, http.StatusNotImplemented, ErrNotImplemented, "not implemented in gmsg microservice")
}

func (h *Handler) RevokeMessage(w http.ResponseWriter, r *http.Request) {
	var req gmsgpb.RevokeMessageRequest
	if err := ReadJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}
	if strings.TrimSpace(req.AdminUlid) == "" {
		req.AdminUlid = adminActorID(r)
	}
	if !requireRequestFields(w, req.UserUlid, "user_ulid", req.MessageUlid, "message_ulid", req.AdminUlid, "admin_ulid") {
		return
	}

	resp, err := h.Gmsg.RevokeMessage(r.Context(), &req)
	if err != nil {
		slog.Error("RevokeMessage failed", "error", err)
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) GetMessageStats(w http.ResponseWriter, r *http.Request) {
	resp, err := h.Gmsg.GetMessageStats(r.Context(), &gmsgpb.GetMessageStatsRequest{})
	if err != nil {
		slog.Error("GetMessageStats failed", "error", err)
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) GetMessageBuiltInPath(w http.ResponseWriter, r *http.Request) {
	req := &gmsgpb.GetBuiltInPathRequest{}
	if path := strings.TrimSpace(r.URL.Query().Get("path")); path != "" {
		req.Query = &gmsgpb.GetBuiltInPathRequest_Path{Path: path}
	} else if rawType := strings.TrimSpace(r.URL.Query().Get("path_type")); rawType != "" {
		parsed, err := strconv.ParseInt(rawType, 10, 32)
		if err != nil || parsed <= 0 {
			WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "path_type must be a positive integer")
			return
		}
		req.Query = &gmsgpb.GetBuiltInPathRequest_PathType{PathType: gmsgpb.BuiltInMsgPathType(parsed)}
	} else {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "path or path_type is required")
		return
	}

	resp, err := h.Gmsg.GetBuiltInPath(r.Context(), req)
	if err != nil {
		slog.Error("GetMessageBuiltInPath failed", "error", err)
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) GetAllMessageBuiltInPaths(w http.ResponseWriter, r *http.Request) {
	resp, err := h.Gmsg.GetAllBuiltInPaths(r.Context(), &gmsgpb.GetAllBuiltInPathsRequest{})
	if err != nil {
		slog.Error("GetAllMessageBuiltInPaths failed", "error", err)
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}
