package handler

import (
	"net/http"
	"strings"

	lmspb "github.com/afnandelfin620-star/cftptest/cftp/glms"
	"github.com/go-chi/chi/v5"
)

func (h *Handler) TakeQuiz(w http.ResponseWriter, r *http.Request) {
	candidateID := CandidateID(r)
	quizID := strings.TrimSpace(chi.URLParam(r, "quizId"))
	if !requireRequestFields(w, candidateID, "candidate_id", quizID, "quiz_id") {
		return
	}

	resp, err := h.Lms.TakeQuiz(r.Context(), &lmspb.TakeQuizRequest{
		CandidateId: candidateID,
		QuizId:      quizID,
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) GetQuizPaper(w http.ResponseWriter, r *http.Request) {
	candidateID := CandidateID(r)
	attemptID := strings.TrimSpace(chi.URLParam(r, "attemptId"))
	if !requireRequestFields(w, candidateID, "candidate_id", attemptID, "attempt_id") {
		return
	}

	resp, err := h.Lms.GetCandidateQuizPaper(r.Context(), &lmspb.GetCandidateQuizPaperRequest{
		CandidateId: candidateID,
		AttemptId:   attemptID,
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) SubmitQuiz(w http.ResponseWriter, r *http.Request) {
	candidateID := CandidateID(r)
	attemptID := strings.TrimSpace(chi.URLParam(r, "attemptId"))
	if !requireRequestFields(w, candidateID, "candidate_id", attemptID, "attempt_id") {
		return
	}

	var input SubmitQuizInput
	if err := ReadJSON(r, &input); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid request body: "+err.Error())
		return
	}

	submissions := make([]*lmspb.QuizAnswerSubmission, 0, len(input.Submissions))
	for _, sub := range input.Submissions {
		submissions = append(submissions, &lmspb.QuizAnswerSubmission{
			QuestionId:        sub.QuestionId,
			SelectedOptionIds: sub.SelectedOptionIds,
		})
	}

	resp, err := h.Lms.SubmitQuiz(r.Context(), &lmspb.SubmitQuizRequest{
		CandidateId: candidateID,
		AttemptId:   attemptID,
		Submissions: submissions,
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, resp)
}
