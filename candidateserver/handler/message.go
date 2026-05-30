package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	gmsgpb "github.com/afnandelfin620-star/cftptest/cftp/gmsg"
)

// ListMessages  GET /api/messages  消息列表
func (h *Handler) ListMessages(w http.ResponseWriter, r *http.Request) {
	candidateID := CandidateID(r)

	limitStr := r.URL.Query().Get("limit")
	lastIdStr := r.URL.Query().Get("lastId")

	limit := 10 // default
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}

	var lastId uint64
	if lastIdStr != "" {
		if lid, err := strconv.ParseUint(lastIdStr, 10, 64); err == nil {
			lastId = lid
		}
	}

	rsp, err := h.Gmsg.ListMessages(r.Context(), &gmsgpb.ListMessagesRequest{
		UserId: candidateID,
		Limit:  uint32(limit),
		LastId: lastId,
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	out := MessageListRsp{
		Messages: make([]MessageItem, 0, len(rsp.GetMessages())),
	}

	// 缓存模板以避免重复请求
	templates := make(map[string]*gmsgpb.Template)

	for _, msg := range rsp.GetMessages() {
		tplId := msg.GetTemplateId()
		var tpl *gmsgpb.Template

		if t, ok := templates[tplId]; ok {
			tpl = t
		} else if tplId != "" {
			tResp, err := h.Gmsg.GetTemplate(r.Context(), &gmsgpb.GetTemplateRequest{TemplateId: tplId})
			if err == nil && tResp != nil {
				tpl = tResp
				templates[tplId] = tResp
			}
		}

		// MessageItem no longer carries per-message payload in the latest gmsg proto.
		payloadStr := "{}"
		if tpl != nil {
			finalPayload := map[string]string{
				"title":   tpl.TitleTpl,
				"content": tpl.ContentTpl,
			}
			if b, err := json.Marshal(finalPayload); err == nil {
				payloadStr = string(b)
			}
		}

		out.Messages = append(out.Messages, MessageItem{
			Id:         msg.GetId(),
			MessageId:  msg.GetMessageId(),
			UserId:     msg.GetUserId(),
			TemplateId: tplId,
			Payload:    payloadStr,
			MsgType:    msg.GetMsgType(),
			MsgSource:  msg.GetMsgSource(),
			SenderId:   msg.GetSenderId(),
			Status:     msg.GetStatus(),
			CreatedAt:  msg.GetCreatedAt(),
		})
	}

	WriteJSON(w, http.StatusOK, out)
}

// MarkMessagesRead  PUT /api/messages/read  标记消息已读
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
