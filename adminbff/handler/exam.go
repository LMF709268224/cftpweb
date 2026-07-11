package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	gexampb "github.com/afnandelfin620-star/cftptest/cftp/gexam"
	"github.com/go-chi/chi/v5"
)

func (h *Handler) ListAdminExams(w http.ResponseWriter, r *http.Request) {
	page := parseCursorPage(r, 10)

	filters := &gexampb.ExamFilters{
		Status:             optionalString(r.URL.Query().Get("status")),
		ResultStatus:       optionalString(r.URL.Query().Get("result_status")),
		CandidateUlid:      optionalString(r.URL.Query().Get("candidate_ulid")),
		ConfirmationNumber: optionalString(r.URL.Query().Get("confirmation_number")),
		CourseUnitUlid:     optionalString(r.URL.Query().Get("course_unit_ulid")),
	}

	total, err := countCursorAll(r.Context(), func(ctx context.Context, cursor string, limit uint32) (uint32, string, error) {
		resp, err := h.Gexam.GetExamCount(ctx, &gexampb.GetExamCountRequest{
			Filters:   filters,
			Limit:     limit,
			Cursor:    cursor,
			SortOrder: gexampb.SortOrder(page.Sort),
		})
		if err != nil {
			return 0, "", err
		}
		return resp.GetCount(), resp.GetNextCursor(), nil
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	req := &gexampb.ListExamsRequest{
		Filters:   filters,
		Cursor:    page.Cursor,
		PageSize:  page.PageSize,
		SortOrder: gexampb.SortOrder(page.Sort),
	}

	resp, err := h.Gexam.ListExams(r.Context(), req)
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, map[string]interface{}{
		"total":       total.Total,
		"exact":       total.Exact,
		"has_more":    resp.GetHasMore(),
		"next_cursor": resp.GetNextCursor(),
		"prev_cursor": resp.GetPrevCursor(),
		"exams":       resp.GetExams(),
	})
}

func (h *Handler) GetAdminExamDetail(w http.ResponseWriter, r *http.Request) {
	examULID := chi.URLParam(r, "exam_ulid")
	if !requireRequestField(w, examULID, "exam_ulid") {
		return
	}

	resp, err := h.Gexam.GetExamDetail(r.Context(), &gexampb.GetExamRequest{ExamUlid: examULID})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	payload := jsonPayloadObject(resp)
	h.attachCandidateName(payload, resp.GetCandidateUlid())
	WriteJSON(w, http.StatusOK, payload)
}

func (h *Handler) GetAdminExamResult(w http.ResponseWriter, r *http.Request) {
	examULID := chi.URLParam(r, "exam_ulid")
	if !requireRequestField(w, examULID, "exam_ulid") {
		return
	}

	resp, err := h.Gexam.GetExamResultDetail(r.Context(), &gexampb.GetExamRequest{ExamUlid: examULID})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) GetAdminExamTransitions(w http.ResponseWriter, r *http.Request) {
	examULID := chi.URLParam(r, "exam_ulid")
	if !requireRequestField(w, examULID, "exam_ulid") {
		return
	}

	resp, err := h.Gexam.GetExamStatusTransitions(r.Context(), &gexampb.GetExamRequest{ExamUlid: examULID})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) SyncAdminExamResult(w http.ResponseWriter, r *http.Request) {
	examULID := chi.URLParam(r, "exam_ulid")
	if !requireRequestField(w, examULID, "exam_ulid") {
		return
	}

	resp, err := h.Gexam.SyncExamResult(r.Context(), &gexampb.GetExamRequest{ExamUlid: examULID})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) ListWebhookMessages(w http.ResponseWriter, r *http.Request) {
	page := parseCursorPage(r, 50)

	var statusPtr *string
	if status := r.URL.Query().Get("status"); status != "" {
		statusPtr = &status
	}

	resp, err := h.Gexam.ListWebhookMessages(r.Context(), &gexampb.ListWebhookMessagesRequest{
		Filters: &gexampb.WebhookFilters{
			ProcessedStatus: statusPtr,
		},
		Cursor:   page.Cursor,
		PageSize: page.PageSize,
		SortOrder: gexampb.SortOrder(page.Sort),
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
