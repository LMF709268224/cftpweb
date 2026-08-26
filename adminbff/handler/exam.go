package handler

import (
	"context"
	"fmt"
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

func (h *Handler) ListPendingGradingExams(w http.ResponseWriter, r *http.Request) {
	page := parseCursorPage(r, 10)
	filters := &gexampb.PendingGradingExamFilters{
		ProgramCode: strings.TrimSpace(r.URL.Query().Get("program_code")),
		ExamCode:    strings.TrimSpace(r.URL.Query().Get("exam_code")),
		Keyword:     strings.TrimSpace(r.URL.Query().Get("keyword")),
	}

	total, err := countCursorAll(r.Context(), func(ctx context.Context, cursor string, limit uint32) (uint32, string, error) {
		resp, err := h.Gexam.GetPendingGradingExamCount(ctx, &gexampb.GetPendingGradingExamCountRequest{
			Filters:   filters,
			Cursor:    cursor,
			Limit:     int32(limit),
			SortOrder: gexampb.SortOrder(page.Sort),
		})
		if err != nil {
			return 0, "", err
		}
		return uint32(resp.GetCount()), resp.GetNextCursor(), nil
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	resp, err := h.Gexam.ListPendingGradingExams(r.Context(), &gexampb.ListPendingGradingExamsRequest{
		Filters:   filters,
		PageSize:  int32(page.PageSize),
		Cursor:    page.Cursor,
		SortOrder: gexampb.SortOrder(page.Sort),
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, map[string]interface{}{
		"items":       resp.GetItems(),
		"total":       total.Total,
		"total_exact": total.Exact,
		"next_cursor": resp.GetNextCursor(),
		"prev_cursor": resp.GetPrevCursor(),
		"has_more":    resp.GetHasMore(),
	})
}

func (h *Handler) GetExamEssayDetails(w http.ResponseWriter, r *http.Request) {
	examULID := strings.TrimSpace(chi.URLParam(r, "exam_ulid"))
	if !requireRequestField(w, examULID, "exam_ulid") {
		return
	}
	resp, err := h.Gexam.GetExamEssayDetails(r.Context(), &gexampb.GetExamEssayDetailsRequest{ExamUlid: examULID})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) SubmitExamEssayGrade(w http.ResponseWriter, r *http.Request) {
	examULID := strings.TrimSpace(chi.URLParam(r, "exam_ulid"))
	graderID := strings.TrimSpace(AdminID(r))
	graderName := strings.TrimSpace(AdminName(r))
	if !requireRequestFields(w, examULID, "exam_ulid", graderID, "grader_id", graderName, "grader_name") {
		return
	}
	if len([]rune(graderName)) > 100 {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "grader_name must be at most 100 characters")
		return
	}

	var body struct {
		IsPassed       bool   `json:"is_passed"`
		OverallComment string `json:"overall_comment"`
		Items          []struct {
			QuestionSeq int32    `json:"question_seq"`
			Score       float64  `json:"score"`
			MaxScore    *float64 `json:"max_score"`
			Comment     string   `json:"comment"`
		} `json:"items"`
	}
	if err := ReadJSON(r, &body); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid request body")
		return
	}
	body.OverallComment = strings.TrimSpace(body.OverallComment)
	if len([]rune(body.OverallComment)) > 1000 {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "overall_comment must be at most 1000 characters")
		return
	}
	if len(body.Items) == 0 {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "items is required")
		return
	}

	items := make([]*gexampb.ExamEssayGradeItem, 0, len(body.Items))
	seenQuestions := make(map[int32]struct{}, len(body.Items))
	for index, item := range body.Items {
		if item.QuestionSeq <= 0 || item.Score < 0 {
			WriteError(w, http.StatusBadRequest, ErrInvalidRequest, fmt.Sprintf("items[%d] has an invalid question_seq or score", index))
			return
		}
		if _, exists := seenQuestions[item.QuestionSeq]; exists {
			WriteError(w, http.StatusBadRequest, ErrInvalidRequest, fmt.Sprintf("question_seq %d is duplicated", item.QuestionSeq))
			return
		}
		seenQuestions[item.QuestionSeq] = struct{}{}
		if item.MaxScore != nil && (*item.MaxScore <= 0 || item.Score > *item.MaxScore) {
			WriteError(w, http.StatusBadRequest, ErrInvalidRequest, fmt.Sprintf("items[%d].score must be between 0 and max_score", index))
			return
		}
		items = append(items, &gexampb.ExamEssayGradeItem{
			QuestionSeq: item.QuestionSeq,
			Score:       item.Score,
			MaxScore:    item.MaxScore,
			Comment:     strings.TrimSpace(item.Comment),
		})
	}

	resp, err := h.Gexam.SubmitExamEssayGrade(r.Context(), &gexampb.SubmitExamEssayGradeRequest{
		ExamUlid:       examULID,
		GraderId:       graderID,
		GraderName:     graderName,
		IsPassed:       body.IsPassed,
		Items:          items,
		OverallComment: body.OverallComment,
	})
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
		Cursor:    page.Cursor,
		PageSize:  page.PageSize,
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
	if err := ReadJSON(r, &input); err != nil {
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
