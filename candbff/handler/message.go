package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	gmsgpb "github.com/afnandelfin620-star/cftptest/cftp/gmsg"
)

func (h *Handler) ListMessages(w http.ResponseWriter, r *http.Request) {
	candidateID := CandidateID(r)

	limit := 10
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	var lastID uint64
	if lastIDStr := r.URL.Query().Get("lastId"); lastIDStr != "" {
		if parsed, err := strconv.ParseUint(lastIDStr, 10, 64); err == nil {
			lastID = parsed
		}
	}

	rsp, err := h.Gmsg.ListMessages(r.Context(), &gmsgpb.ListMessagesRequest{
		UserUlid: candidateID,
		Limit:    uint32(limit),
		LastId:   lastID,
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	out := MessageListRsp{
		Messages: make([]MessageItem, 0, len(rsp.GetMessages())),
		HasMore:  rsp.GetHasMore(),
	}
	for _, msg := range rsp.GetMessages() {
		title, content := h.renderMessageSummary(r.Context(), msg)
		out.Messages = append(out.Messages, MessageItem{
			Id:              msg.GetId(),
			MessageId:       msg.GetMessageUlid(),
			UserUlid:        msg.GetUserUlid(),
			TemplateId:      msg.GetTemplatePath(),
			Payload:         msg.GetPayload(),
			TemplatePayload: msg.GetPayload(),
			Title:           title,
			Content:         content,
			MsgType:         msg.GetMsgType(),
			MsgSource:       msg.GetMsgSource(),
			SenderId:        msg.GetSenderUlid(),
			Status:          msg.GetStatus(),
			CreatedAt:       msg.GetCreatedAt(),
		})
	}

	WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) GetUnreadMessageCount(w http.ResponseWriter, r *http.Request) {
	candidateID := CandidateID(r)

	rsp, err := h.Gmsg.ListMessages(r.Context(), &gmsgpb.ListMessagesRequest{
		UserUlid: candidateID,
		Status:   gmsgpb.MessageStatus_UNREAD.Enum(),
		Limit:    99,
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, MessageUnreadCountRsp{
		UnreadCount: uint32(len(rsp.GetMessages())),
	})
}

func (h *Handler) MarkMessagesRead(w http.ResponseWriter, r *http.Request) {
	candidateID := CandidateID(r)

	var input MessageOperationInput
	if err := ReadJSON(r, &input); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid request body: "+err.Error())
		return
	}

	_, err := h.Gmsg.MarkAsRead(r.Context(), &gmsgpb.MarkAsReadRequest{
		UserUlid:   candidateID,
		MessageIds: input.MessageIDs,
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, nil)
}

func (h *Handler) DeleteMessage(w http.ResponseWriter, r *http.Request) {
	candidateID := CandidateID(r)

	var input MessageOperationInput
	if err := ReadJSON(r, &input); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid request body: "+err.Error())
		return
	}

	_, err := h.Gmsg.DeleteMessages(r.Context(), &gmsgpb.DeleteMessagesRequest{
		UserUlid:   candidateID,
		MessageIds: input.MessageIDs,
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, nil)
}

func (h *Handler) GetMessage(w http.ResponseWriter, r *http.Request) {
	candidateID := CandidateID(r)
	messageID := chi.URLParam(r, "messageId")

	if messageID == "" {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "messageId is required")
		return
	}

	msg, err := h.Gmsg.GetMessage(r.Context(), &gmsgpb.GetMessageRequest{
		UserUlid:    candidateID,
		MessageUlid: messageID,
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	title, content := h.renderMessageSummary(r.Context(), msg)

	out := map[string]interface{}{
		"id":               msg.GetId(),
		"message_id":       msg.GetMessageUlid(),
		"msg_type":         msg.GetMsgType().String(),
		"status":           msg.GetStatus().String(),
		"created_at":       msg.GetCreatedAt(),
		"title":            title,
		"content":          content,
		"payload":          msg.GetPayload(),
		"template_payload": msg.GetPayload(),
	}
	WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) renderMessageSummary(ctx context.Context, msg *gmsgpb.MessageItem) (string, string) {
	if msg == nil {
		return "", ""
	}

	var vars map[string]interface{}
	if msg.GetPayload() != "" {
		_ = json.Unmarshal([]byte(msg.GetPayload()), &vars)
	}

	title := ""
	content := ""
	if templatePath := msg.GetTemplatePath(); templatePath != "" {
		if tResp, err := h.Gmsg.GetTemplate(ctx, &gmsgpb.GetTemplateRequest{Path: templatePath}); err == nil && tResp != nil {
			title = renderTemplateString(tResp.GetTitleTpl(), vars)
			content = renderTemplateString(tResp.GetContentTpl(), vars)
		}
	}

	if title == "" {
		title = stringFromPayload(vars, "title", "name", "subject")
	}
	if content == "" {
		content = stringFromPayload(vars, "content", "message", "description", "remark")
	}
	if content == "" {
		content = compactPayloadSummary(vars)
	}
	if content == "" {
		content = msg.GetPayload()
	}

	return title, content
}

func renderTemplateString(tpl string, vars map[string]interface{}) string {
	if tpl == "" || len(vars) == 0 {
		return tpl
	}
	out := tpl
	for k, v := range vars {
		vStr := fmt.Sprintf("%v", v)
		out = strings.ReplaceAll(out, "{{"+k+"}}", vStr)
		out = strings.ReplaceAll(out, "{{ "+k+" }}", vStr)
	}
	return out
}

func stringFromPayload(vars map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if value, ok := vars[key]; ok {
			text := strings.TrimSpace(fmt.Sprintf("%v", value))
			if text != "" && text != "<nil>" {
				return text
			}
		}
	}
	return ""
}

func compactPayloadSummary(vars map[string]interface{}) string {
	if len(vars) == 0 {
		return ""
	}
	parts := make([]string, 0, len(vars))
	for key, value := range vars {
		text := strings.TrimSpace(fmt.Sprintf("%v", value))
		if text == "" || text == "<nil>" {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s: %s", key, text))
		if len(parts) >= 4 {
			break
		}
	}
	return strings.Join(parts, " · ")
}
