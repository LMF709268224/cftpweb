package handler

import (
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
		UserId: candidateID,
		Limit:  uint32(limit),
		LastId: lastID,
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	out := MessageListRsp{
		Messages: make([]MessageItem, 0, len(rsp.GetMessages())),
	}
	for _, msg := range rsp.GetMessages() {
		out.Messages = append(out.Messages, MessageItem{
			Id:         msg.GetId(),
			MessageId:  msg.GetMessageId(),
			UserId:     msg.GetUserId(),
			TemplateId: msg.GetTemplatePath(),
			Payload:    "{}",
			MsgType:    msg.GetMsgType(),
			MsgSource:  msg.GetMsgSource(),
			SenderId:   msg.GetSenderId(),
			Status:     msg.GetStatus(),
			CreatedAt:  msg.GetCreatedAt(),
		})
	}

	WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) MarkMessagesRead(w http.ResponseWriter, r *http.Request) {
	candidateID := CandidateID(r)

	var input MessageOperationInput
	if err := ReadJSON(r, &input); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid request body: "+err.Error())
		return
	}

	_, err := h.Gmsg.MarkAsRead(r.Context(), &gmsgpb.MarkAsReadRequest{
		UserId:     candidateID,
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
		UserId:     candidateID,
		MessageIds: input.MessageIDs,
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, nil)
}

func (h *Handler) GetMessageDetail(w http.ResponseWriter, r *http.Request) {
	candidateID := CandidateID(r)
	messageID := chi.URLParam(r, "messageId")

	if messageID == "" {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "messageId is required")
		return
	}

	msg, err := h.Gmsg.GetMessageDetail(r.Context(), &gmsgpb.GetMessageDetailRequest{
		UserId:    candidateID,
		MessageId: messageID,
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	title := ""
	content := ""

	templatePath := msg.GetTemplatePath()
	if templatePath != "" {
		tResp, err := h.Gmsg.GetTemplate(r.Context(), &gmsgpb.GetTemplateRequest{Path: templatePath})
		if err == nil && tResp != nil {
			title = tResp.GetTitleTpl()
			content = tResp.GetContentTpl()

			if msg.GetPayload() != "" {
				var vars map[string]interface{}
				if err := json.Unmarshal([]byte(msg.GetPayload()), &vars); err == nil {
					for k, v := range vars {
						vStr := fmt.Sprintf("%v", v)
						title = strings.ReplaceAll(title, "{{"+k+"}}", vStr)
						content = strings.ReplaceAll(content, "{{"+k+"}}", vStr)
					}
				}
			}
		}
	} else if msg.GetPayload() != "" {
		var vars map[string]interface{}
		if err := json.Unmarshal([]byte(msg.GetPayload()), &vars); err == nil {
			if t, ok := vars["title"].(string); ok {
				title = t
			}
			if c, ok := vars["content"].(string); ok {
				content = c
			}
		}
	}

	out := map[string]interface{}{
		"id":         msg.GetId(),
		"message_id": msg.GetMessageId(),
		"msg_type":   msg.GetMsgType().String(),
		"status":     msg.GetStatus().String(),
		"created_at": msg.GetCreatedAt(),
		"title":      title,
		"content":    content,
		"payload":    msg.GetPayload(),
	}
	WriteJSON(w, http.StatusOK, out)
}
