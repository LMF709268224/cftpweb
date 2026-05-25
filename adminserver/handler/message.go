package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/oklog/ulid/v2"

	gmsgpb "github.com/afnandelfin620-star/cftptest/cftp/gmsg"
)

type SendMessageInput struct {
	UserIds    []string `json:"user_ids"`
	TemplateId string   `json:"template_id"`
	Payload    string   `json:"payload"`
	MsgType    int32    `json:"msg_type"` // 0 或不传默认为 1 (系统通知)
}

func (h *Handler) SendMessage(w http.ResponseWriter, r *http.Request) {
	var input SendMessageInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}

	messageID := ulid.Make().String()
	senderID := CandidateID(r)

	msgType := gmsgpb.MsgType_SYSTEM_NOTICE
	if input.MsgType > 0 {
		msgType = gmsgpb.MsgType(input.MsgType)
	}

	resp, err := h.Gmsg.SendMessage(r.Context(), &gmsgpb.SendMessageRequest{
		UserIds:    input.UserIds,
		MessageId:  messageID,
		TemplateId: input.TemplateId,
		Payload:    input.Payload,
		MsgType:    msgType,
		MsgSource:  gmsgpb.MsgSource_MANUAL_ADMIN,
		SenderId:   senderID,
	})

	if err != nil {
		slog.Error("SendMessage failed", "error", err)
		WriteError(w, http.StatusInternalServerError, ErrInternal, "failed to send message")
		return
	}

	WriteJSON(w, http.StatusOK, map[string]interface{}{
		"count":  resp.Count,
		"status": resp.Status,
	})
}

// ListSentMessages GET /api/messages/sent
func (h *Handler) ListSentMessages(w http.ResponseWriter, r *http.Request) {
	// TODO: gmsg 尚未提供查询所有已发送消息的 gRPC 接口
	// 目前先返回空数组占位
	WriteJSON(w, http.StatusOK, map[string]interface{}{
		"messages": []interface{}{},
	})
}

// CreateTemplate POST /api/messages/templates
func (h *Handler) CreateTemplate(w http.ResponseWriter, r *http.Request) {
	var req gmsgpb.CreateTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}

	req.TemplateId = ulid.Make().String()

	// Fix Go template syntax by injecting dot for variables like {{name}} -> {{.name}}
	req.TitleTpl = tplVarRegex.ReplaceAllString(req.TitleTpl, "{{.$1}}")
	req.ContentTpl = tplVarRegex.ReplaceAllString(req.ContentTpl, "{{.$1}}")

	resp, err := h.Gmsg.CreateTemplate(r.Context(), &req)
	if err != nil {
		slog.Error("CreateTemplate failed", "error", err)
		WriteError(w, http.StatusInternalServerError, ErrInternal, "failed to create template")
		return
	}

	WriteJSON(w, http.StatusOK, resp)
}

// ListTemplates GET /api/messages/templates
func (h *Handler) ListTemplates(w http.ResponseWriter, r *http.Request) {
	keyword := r.URL.Query().Get("keyword")

	resp, err := h.Gmsg.ListTemplates(r.Context(), &gmsgpb.ListTemplatesRequest{
		Keyword:  keyword,
		Page:     1,
		PageSize: 100,
	})

	if err != nil {
		slog.Error("ListTemplates failed", "error", err)
		WriteError(w, http.StatusInternalServerError, ErrInternal, "failed to list templates")
		return
	}

	WriteJSON(w, http.StatusOK, resp)
}

// UpdateTemplate PUT /api/messages/templates
func (h *Handler) UpdateTemplate(w http.ResponseWriter, r *http.Request) {
	var req gmsgpb.UpdateTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}

	// Fix Go template syntax by injecting dot for variables like {{name}} -> {{.name}}
	req.TitleTpl = tplVarRegex.ReplaceAllString(req.TitleTpl, "{{.$1}}")
	req.ContentTpl = tplVarRegex.ReplaceAllString(req.ContentTpl, "{{.$1}}")

	resp, err := h.Gmsg.UpdateTemplate(r.Context(), &req)
	if err != nil {
		slog.Error("UpdateTemplate failed", "error", err)
		WriteError(w, http.StatusInternalServerError, ErrInternal, "failed to update template")
		return
	}

	WriteJSON(w, http.StatusOK, resp)
}

// DeleteMessageTemplate DELETE /api/messages/templates
func (h *Handler) DeleteMessageTemplate(w http.ResponseWriter, r *http.Request) {
	WriteError(w, http.StatusNotImplemented, ErrNotImplemented, "not implemented in gmsg microservice")
}
