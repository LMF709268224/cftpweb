package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	lmspb "github.com/afnandelfin620-star/cftptest/cftp/glms"
)

type importLmsContentRequest struct {
	Scope         string `json:"scope,omitempty"`
	Type          string `json:"type,omitempty"`
	CategoryTips  string `json:"category_tips,omitempty"`
	CourseJSON    string `json:"course_json,omitempty"`
	QuizJSON      string `json:"quiz_json,omitempty"`
	QuizzableType int32  `json:"quizzable_type,omitempty"`
	QuizzableID   string `json:"quizzable_id,omitempty"`
}

// ImportLmsContent POST /api/lms/import
func (h *Handler) ImportLmsContent(w http.ResponseWriter, r *http.Request) {
	var req importLmsContentRequest
	if err := ReadJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}

	scope := normalizeLmsImportScope(req.Scope)
	if scope == "" {
		scope = normalizeLmsImportScope(req.Type)
	}
	switch scope {
	case "course":
		h.importLmsCourse(w, r, req)
	case "quiz":
		h.importLmsQuiz(w, r, req)
	default:
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "scope is required")
	}
}

func normalizeLmsImportScope(scope string) string {
	switch strings.ToLower(strings.TrimSpace(scope)) {
	case "course", "full_course":
		return "course"
	case "quiz", "quizzes":
		return "quiz"
	default:
		return ""
	}
}

func (h *Handler) importLmsCourse(w http.ResponseWriter, r *http.Request, req importLmsContentRequest) {
	courseJSON := strings.TrimSpace(req.CourseJSON)
	if !requireRequestField(w, courseJSON, "course_json") {
		return
	}
	if !requireValidJSONString(w, courseJSON, "course_json") {
		return
	}

	resp, err := h.Lms.ImportCourse(r.Context(), &lmspb.ImportCourseRequest{
		CategoryTips: strings.TrimSpace(req.CategoryTips),
		CourseJson:   courseJSON,
	})
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) importLmsQuiz(w http.ResponseWriter, r *http.Request, req importLmsContentRequest) {
	quizJSON := strings.TrimSpace(req.QuizJSON)
	if !requireRequestField(w, quizJSON, "quiz_json") || !requireRequestField(w, req.QuizzableID, "quizzable_id") {
		return
	}
	if req.QuizzableType == int32(lmspb.QuizzableType_QUIZZABLE_TYPE_UNSPECIFIED) {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "quizzable_type is required")
		return
	}
	if !requireValidJSONString(w, quizJSON, "quiz_json") {
		return
	}

	resp, err := h.Lms.ImportQuiz(r.Context(), &lmspb.ImportQuizRequest{
		QuizzableType: lmspb.QuizzableType(req.QuizzableType),
		QuizzableId:   strings.TrimSpace(req.QuizzableID),
		QuizJson:      quizJSON,
	})
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func requireValidJSONString(w http.ResponseWriter, value string, name string) bool {
	var parsed map[string]any
	if err := json.Unmarshal([]byte(value), &parsed); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, name+" is invalid")
		return false
	}
	if len(parsed) == 0 {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, name+" must be a non-empty JSON object")
		return false
	}
	return true
}
