package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	gexampb "github.com/LMF709268224/cftpproto/gexam"
)

func (h *Handler) ListWebhookMessages(w http.ResponseWriter, r *http.Request) {
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

	var statusPtr *string
	if status := r.URL.Query().Get("status"); status != "" {
		statusPtr = &status
	}

	resp, err := h.Gexam.ListWebhookMessages(r.Context(), &gexampb.ListWebhookMessagesRequest{
		Page:            uint32(page),
		PageSize:        uint32(pageSize),
		ProcessedStatus: statusPtr,
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) GetWebhookMessageDetail(w http.ResponseWriter, r *http.Request) {
	msgFp := strings.TrimSpace(r.URL.Query().Get("msg_fp"))
	if !requireRequestField(w, msgFp, "msg_fp") {
		return
	}

	resp, err := h.Gexam.GetWebhookMessageDetail(r.Context(), &gexampb.GetWebhookMessageDetailRequest{
		MsgFp: msgFp,
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) ReprocessWebhookMessage(w http.ResponseWriter, r *http.Request) {
	var input struct {
		WebhookMsgId int64 `json:"webhook_msg_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}

	req := &gexampb.ReprocessWebhookMessageRequest{
		WebhookMsgId: input.WebhookMsgId,
	}

	resp, err := h.Gexam.ReprocessWebhookMessage(r.Context(), req)
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}
