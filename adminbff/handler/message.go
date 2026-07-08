package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

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
	page := 1
	pageSize := 50
	if p := r.URL.Query().Get("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}
	if ps := r.URL.Query().Get("page_size"); ps != "" {
		if v, err := strconv.Atoi(ps); err == nil && v > 0 {
			pageSize = v
		}
	}

	var statusPtr *gmsgpb.MessageStatus
	if statusStr := r.URL.Query().Get("status"); statusStr != "" {
		if v, err := strconv.Atoi(statusStr); err == nil {
			statusEnum := gmsgpb.MessageStatus(v)
			statusPtr = &statusEnum
		}
	}

	resp, err := h.Gmsg.ListMessagesAdmin(r.Context(), &gmsgpb.ListMessagesAdminRequest{
		Page:     uint32(page),
		PageSize: uint32(pageSize),
		Status:   statusPtr,
	})
	if err != nil {
		slog.Error("ListMessagesAdmin failed", "error", err)
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
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
	page := parsePositiveIntQuery(r, "page", 1)
	pageSize := parsePositiveIntQuery(r, "page_size", 10)
	resp, err := h.Gmsg.ListTemplates(r.Context(), &gmsgpb.ListTemplatesRequest{
		Keyword:  keyword,
		Page:     uint32(page),
		PageSize: uint32(pageSize),
	})
	if err != nil {
		slog.Error("ListTemplates failed", "error", err)
		HandleGrpcError(w, err)
		return
	}
	payload := jsonPayloadObject(resp)
	payload["page"] = page
	payload["page_size"] = pageSize
	WriteJSON(w, http.StatusOK, payload)
}

func (h *Handler) GetMessageTemplate(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if !requireRequestField(w, path, "path") {
		return
	}
	resp, err := h.Gmsg.GetTemplate(r.Context(), &gmsgpb.GetTemplateRequest{Path: path})
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
