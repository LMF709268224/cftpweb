package handler

import (
	"encoding/json"
	"fmt"
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

type importCourseJSON struct {
	Title    string              `json:"title"`
	Chapters []importChapterJSON `json:"chapters"`
}

type importChapterJSON struct {
	Title   string             `json:"title"`
	Lessons []importLessonJSON `json:"lessons"`
}

type importLessonJSON struct {
	Title          string `json:"title"`
	LessonType     any    `json:"lesson_type"`
	Body           string `json:"body"`
	MediaObjectKey string `json:"media_object_key"`
	ExternalURL    string `json:"external_url"`
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
	if !validateImportCourseJSON(w, courseJSON) {
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

func validateImportCourseJSON(w http.ResponseWriter, value string) bool {
	var course importCourseJSON
	if err := json.Unmarshal([]byte(value), &course); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "course_json is invalid")
		return false
	}
	if strings.TrimSpace(course.Title) == "" {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "course_json.title is required")
		return false
	}
	for chapterIndex, chapter := range course.Chapters {
		chapterPath := fmt.Sprintf("course_json.chapters[%d]", chapterIndex)
		if strings.TrimSpace(chapter.Title) == "" {
			WriteError(w, http.StatusBadRequest, ErrInvalidRequest, chapterPath+".title is required")
			return false
		}
		for lessonIndex, lesson := range chapter.Lessons {
			lessonPath := fmt.Sprintf("%s.lessons[%d]", chapterPath, lessonIndex)
			if strings.TrimSpace(lesson.Title) == "" {
				WriteError(w, http.StatusBadRequest, ErrInvalidRequest, lessonPath+".title is required")
				return false
			}
			if !validateImportLessonPayload(w, lesson, lessonPath) {
				return false
			}
		}
	}
	return true
}

func validateImportLessonPayload(w http.ResponseWriter, lesson importLessonJSON, lessonPath string) bool {
	switch normalizeImportLessonType(lesson.LessonType) {
	case "text":
		if strings.TrimSpace(lesson.Body) == "" {
			WriteError(w, http.StatusBadRequest, ErrInvalidRequest, lessonPath+".body is required")
			return false
		}
	case "link":
		if strings.TrimSpace(lesson.ExternalURL) == "" {
			WriteError(w, http.StatusBadRequest, ErrInvalidRequest, lessonPath+".external_url is required")
			return false
		}
	case "video", "pdf", "image", "audio", "file":
		if strings.TrimSpace(lesson.MediaObjectKey) == "" {
			WriteError(w, http.StatusBadRequest, ErrInvalidRequest, lessonPath+".media_object_key is required")
			return false
		}
	case "":
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, lessonPath+".lesson_type is required")
		return false
	default:
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, lessonPath+".lesson_type is invalid")
		return false
	}
	return true
}

func normalizeImportLessonType(value any) string {
	switch lessonType := value.(type) {
	case string:
		return strings.ToLower(strings.TrimSpace(lessonType))
	case float64:
		switch int(lessonType) {
		case int(lmspb.LessonType_LESSON_TYPE_TEXT):
			return "text"
		case int(lmspb.LessonType_LESSON_TYPE_VIDEO):
			return "video"
		case int(lmspb.LessonType_LESSON_TYPE_PDF):
			return "pdf"
		case int(lmspb.LessonType_LESSON_TYPE_IMAGE):
			return "image"
		case int(lmspb.LessonType_LESSON_TYPE_AUDIO):
			return "audio"
		case int(lmspb.LessonType_LESSON_TYPE_LINK):
			return "link"
		case int(lmspb.LessonType_LESSON_TYPE_FILE):
			return "file"
		}
	}
	return ""
}
