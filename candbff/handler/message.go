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
	messageScanBatchSize    = 100
	maxMessageScanRows      = 2000
)

func (h *Handler) ListMessages(w http.ResponseWriter, r *http.Request) {
	candidateID := CandidateID(r)

	limit := parsePositiveIntQuery(r, "limit", defaultMessageListLimit)
	if limit > maxMessageListLimit {
		limit = maxMessageListLimit
	}
	lastID := uint64(parseNonNegativeIntQuery(r, "lastId", 0))
	msgTypeParam := firstNonEmpty(r.URL.Query().Get("msg_type"), r.URL.Query().Get("type"))
	msgType, hasMsgType, ok := parseCandidateMessageType(msgTypeParam)
	if !ok {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "unsupported msg_type")
		return
	}

	var messages []MessageItem
	var hasMore bool
	var err error
	if hasMsgType {
		messages, hasMore, err = h.listMessagesByType(r.Context(), candidateID, msgType, limit, lastID)
	} else {
		messages, hasMore, err = h.listMessagesPage(r.Context(), candidateID, limit, lastID)
	}
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, MessageListRsp{
		Messages: messages,
		HasMore:  hasMore,
	})
}

func (h *Handler) listMessagesPage(ctx context.Context, candidateID string, limit int, lastID uint64) ([]MessageItem, bool, error) {
	rsp, err := h.Gmsg.ListMessages(ctx, &gmsgpb.ListMessagesRequest{
		UserUlid: candidateID,
		Limit:    uint32(limit),
		LastId:   lastID,
	})
	if err != nil {
		return nil, false, err
	}

	messages := make([]MessageItem, 0, len(rsp.GetMessages()))
	for _, msg := range rsp.GetMessages() {
		messages = append(messages, h.messageItem(ctx, msg))
	}
	return messages, rsp.GetHasMore(), nil
}

func (h *Handler) listMessagesByType(ctx context.Context, candidateID string, msgType gmsgpb.MsgType, limit int, lastID uint64) ([]MessageItem, bool, error) {
	messages := make([]MessageItem, 0, limit)
	cursor := lastID
	scanned := 0
	for scanned < maxMessageScanRows {
		rsp, err := h.Gmsg.ListMessages(ctx, &gmsgpb.ListMessagesRequest{
			UserUlid: candidateID,
			Limit:    messageScanBatchSize,
			LastId:   cursor,
		})
		if err != nil {
			return nil, false, err
		}
		items := rsp.GetMessages()
		if len(items) == 0 {
			return messages, false, nil
		}
		scanned += len(items)
		for _, msg := range items {
			cursor = msg.GetId()
			if msg.GetMsgType() != msgType {
				continue
			}
			messages = append(messages, h.messageItem(ctx, msg))
			if len(messages) >= limit {
				return messages, rsp.GetHasMore() || h.hasMoreMessagesOfType(ctx, candidateID, msgType, cursor), nil
			}
		}
		if !rsp.GetHasMore() {
			return messages, false, nil
		}
	}
	return messages, true, nil
}

func (h *Handler) hasMoreMessagesOfType(ctx context.Context, candidateID string, msgType gmsgpb.MsgType, lastID uint64) bool {
	cursor := lastID
	scanned := 0
	for scanned < maxMessageScanRows {
		rsp, err := h.Gmsg.ListMessages(ctx, &gmsgpb.ListMessagesRequest{
			UserUlid: candidateID,
			Limit:    messageScanBatchSize,
			LastId:   cursor,
		})
		if err != nil {
			return false
		}
		items := rsp.GetMessages()
		if len(items) == 0 {
			return false
		}
		scanned += len(items)
		for _, msg := range items {
			cursor = msg.GetId()
			if msg.GetMsgType() == msgType {
				return true
			}
		}
		if !rsp.GetHasMore() {
			return false
		}
	}
	return true
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

func (h *Handler) GetMessageTypeCounts(w http.ResponseWriter, r *http.Request) {
	candidateID := CandidateID(r)
	counts, err := h.messageTypeCounts(r.Context(), candidateID)
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	total := uint32(0)
	for _, count := range counts {
		total += count
	}
	WriteJSON(w, http.StatusOK, MessageTypeCountsRsp{
		Total:  total,
		Counts: counts,
	})
}

func (h *Handler) messageTypeCounts(ctx context.Context, candidateID string) (map[string]uint32, error) {
	counts := map[string]uint32{
		"system":       0,
		"announcement": 0,
		"score":        0,
		"payment":      0,
		"other":        0,
	}
	cursor := uint64(0)
	scanned := 0
	for scanned < maxMessageScanRows {
		rsp, err := h.Gmsg.ListMessages(ctx, &gmsgpb.ListMessagesRequest{
			UserUlid: candidateID,
			Limit:    messageScanBatchSize,
			LastId:   cursor,
		})
		if err != nil {
			return counts, err
		}
		items := rsp.GetMessages()
		if len(items) == 0 {
			return counts, nil
		}
		scanned += len(items)
		for _, msg := range items {
			cursor = msg.GetId()
			counts[candidateMessageTypeKey(msg.GetMsgType())]++
		}
		if !rsp.GetHasMore() {
			return counts, nil
		}
	}
	return counts, nil
}

func parseCandidateMessageType(raw string) (gmsgpb.MsgType, bool, bool) {
	value := strings.TrimSpace(raw)
	if value == "" || strings.EqualFold(value, "all") {
		return gmsgpb.MsgType_UNKNOWN_TYPE, false, true
	}
	normalized := strings.ToUpper(strings.ReplaceAll(strings.ReplaceAll(value, "-", "_"), " ", "_"))
	switch normalized {
	case "SYSTEM", "SYSTEM_NOTICE", "1":
		return gmsgpb.MsgType_SYSTEM_NOTICE, true, true
	case "ANNOUNCEMENT", "NOTICE", "EXAM", "EXAM_REMIND", "2":
		return gmsgpb.MsgType_EXAM_REMIND, true, true
	case "SCORE", "TRANSCRIPT", "SCORE_REPORT", "PROMOTION", "3":
		return gmsgpb.MsgType_SCORE_REPORT, true, true
	case "PAYMENT", "PAYMENT_NOTICE", "4":
		return gmsgpb.MsgType_PAYMENT_NOTICE, true, true
	case "OTHER", "5":
		return gmsgpb.MsgType_OTHER, true, true
	default:
		if n, err := strconv.Atoi(normalized); err == nil {
			msgType := gmsgpb.MsgType(n)
			if msgType >= gmsgpb.MsgType_SYSTEM_NOTICE && msgType <= gmsgpb.MsgType_OTHER {
				return msgType, true, true
			}
		}
		return gmsgpb.MsgType_UNKNOWN_TYPE, false, false
	}
}

func candidateMessageTypeKey(msgType gmsgpb.MsgType) string {
	switch msgType {
	case gmsgpb.MsgType_SYSTEM_NOTICE:
		return "system"
	case gmsgpb.MsgType_EXAM_REMIND:
		return "announcement"
	case gmsgpb.MsgType_SCORE_REPORT:
		return "score"
	case gmsgpb.MsgType_PAYMENT_NOTICE:
		return "payment"
	case gmsgpb.MsgType_OTHER:
		return "other"
	default:
		return "other"
	}
}

func (h *Handler) GetUnreadMessageCount(w http.ResponseWriter, r *http.Request) {
	candidateID := CandidateID(r)

	rsp, err := h.Gmsg.GetMessageCount(r.Context(), &gmsgpb.GetMessageCountRequest{
		UserUlid: candidateID,
		Status:   gmsgpb.MessageStatus_UNREAD.Enum(),
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
