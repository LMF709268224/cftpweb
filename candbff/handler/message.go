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

const (
	defaultMessageListLimit = 10
	maxMessageListLimit     = 50
)

func (h *Handler) ListMessages(w http.ResponseWriter, r *http.Request) {
	candidateID := CandidateID(r)

	page := parseCursorPage(r, defaultMessageListLimit)
	if page.PageSize > maxMessageListLimit {
		page.PageSize = maxMessageListLimit
	}
	status, ok := parseCandidateMessageStatus(r.URL.Query().Get("status"))
	if !ok {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "unsupported status")
		return
	}

	messages, nextCursor, hasMore, err := h.listMessagesPage(r.Context(), candidateID, status, page)
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, MessageListRsp{
		Messages:   messages,
		HasMore:    hasMore,
		NextCursor: nextCursor,
	})
}

func (h *Handler) listMessagesPage(ctx context.Context, candidateID string, status *gmsgpb.MessageStatus, page cursorPage) ([]MessageItem, string, bool, error) {
	rsp, err := h.Gmsg.ListMessages(ctx, &gmsgpb.ListMessagesRequest{
		Filters: &gmsgpb.MessageFilters{
			UserUlid: candidateID,
			Status:   status,
		},
		Cursor:   page.Cursor,
		PageSize: page.PageSize,
		SortOrder: gmsgpb.SortOrder(page.Sort),
	})
	if err != nil {
		return nil, "", false, err
	}

	messages := make([]MessageItem, 0, len(rsp.GetMessages()))
	for _, msg := range rsp.GetMessages() {
		messages = append(messages, h.messageItem(ctx, msg))
	}
	return messages, rsp.GetNextCursor(), rsp.GetHasMore(), nil
}

func (h *Handler) messageItem(ctx context.Context, msg *gmsgpb.MessageItem) MessageItem {
	title, content := h.renderMessageSummary(ctx, msg)
	return MessageItem{
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
	}
}

func parseCandidateMessageStatus(raw string) (*gmsgpb.MessageStatus, bool) {
	value := strings.TrimSpace(raw)
	if value == "" || strings.EqualFold(value, "all") {
		return nil, true
	}
	normalized := strings.ToUpper(strings.ReplaceAll(strings.ReplaceAll(value, "-", "_"), " ", "_"))
	switch normalized {
	case "UNREAD", "MESSAGE_STATUS_UNREAD", "0":
		status := gmsgpb.MessageStatus_UNREAD
		return &status, true
	case "READ", "MESSAGE_STATUS_READ", "1":
		status := gmsgpb.MessageStatus_READ
		return &status, true
	case "DELETED", "MESSAGE_STATUS_DELETED", "2":
		status := gmsgpb.MessageStatus_DELETED
		return &status, true
	case "REVOKED", "MESSAGE_STATUS_REVOKED", "3":
		status := gmsgpb.MessageStatus_REVOKED
		return &status, true
	default:
		if n, err := strconv.Atoi(normalized); err == nil {
			status := gmsgpb.MessageStatus(n)
			if status >= gmsgpb.MessageStatus_UNREAD && status <= gmsgpb.MessageStatus_REVOKED {
				return &status, true
			}
		}
		return nil, false
	}
}

func (h *Handler) GetUnreadMessageCount(w http.ResponseWriter, r *http.Request) {
	candidateID := CandidateID(r)

	rsp, err := h.Gmsg.GetMessageCount(r.Context(), &gmsgpb.GetMessageCountRequest{
		Filters: &gmsgpb.MessageFilters{
			UserUlid: candidateID,
			Status:   gmsgpb.MessageStatus_UNREAD.Enum(),
		},
		Limit: 99,
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, MessageUnreadCountRsp{
		UnreadCount: rsp.GetCount(),
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
