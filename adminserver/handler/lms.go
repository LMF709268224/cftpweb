package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	lmspb "github.com/afnandelfin620-star/cftptest/cftp/glms"
	"github.com/go-chi/chi/v5"
	"github.com/oklog/ulid/v2"
)

const defaultUploadContentType = "application/octet-stream"

type versionOnlyReq struct {
	Version uint32 `json:"version"`
}

func newLmsID() string {
	return ulid.Make().String()
}

func readVersionParam(r *http.Request) (uint32, error) {
	if raw := strings.TrimSpace(r.URL.Query().Get("version")); raw != "" {
		version, err := strconv.ParseUint(raw, 10, 32)
		if err != nil {
			return 0, err
		}
		return uint32(version), nil
	}

	var body versionOnlyReq
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		if errors.Is(err, io.EOF) {
			return 0, nil
		}
		return 0, err
	}
	return body.Version, nil
}

func parseBoolQuery(r *http.Request, key string) bool {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	return raw == "1" || strings.EqualFold(raw, "true") || strings.EqualFold(raw, "yes")
}

func parseUint32Query(r *http.Request, key string) uint32 {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return 0
	}
	value, err := strconv.ParseUint(raw, 10, 32)
	if err != nil {
		return 0
	}
	return uint32(value)
}

func parseEnumQuery(r *http.Request, key string) int32 {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return 0
	}
	value, err := strconv.ParseInt(raw, 10, 32)
	if err != nil {
		return 0
	}
	return int32(value)
}

func writeLmsError(w http.ResponseWriter, err error) {
	HandleGrpcError(w, err)
}

// ListLmsCourses GET /api/lms/courses
func (h *Handler) ListLmsCourses(w http.ResponseWriter, r *http.Request) {
	resp, err := h.Lms.ListCourses(r.Context(), &lmspb.ListCoursesRequest{
		CategoryId:    r.URL.Query().Get("category_id"),
		PublishedOnly: parseBoolQuery(r, "published_only"),
	})
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// CreateLmsCourse POST /api/lms/courses
func (h *Handler) CreateLmsCourse(w http.ResponseWriter, r *http.Request) {
	var req lmspb.CreateCourseRequest
	if err := ReadJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}
	req.CourseId = newLmsID()

	resp, err := h.Lms.CreateCourse(r.Context(), &req)
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// GetLmsCourse GET /api/lms/courses/{course_id}
func (h *Handler) GetLmsCourse(w http.ResponseWriter, r *http.Request) {
	resp, err := h.Lms.GetCourse(r.Context(), &lmspb.GetCourseRequest{
		CourseId: chi.URLParam(r, "course_id"),
	})
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// GetCompleteLmsCourse GET /api/lms/courses/{course_id}/complete
func (h *Handler) GetCompleteLmsCourse(w http.ResponseWriter, r *http.Request) {
	resp, err := h.Lms.GetCompleteCourse(r.Context(), &lmspb.GetCompleteCourseRequest{
		CourseId: chi.URLParam(r, "course_id"),
	})
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// UpdateLmsCourse PUT /api/lms/courses/{course_id}
func (h *Handler) UpdateLmsCourse(w http.ResponseWriter, r *http.Request) {
	var req lmspb.UpdateCourseRequest
	if err := ReadJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}
	req.CourseId = chi.URLParam(r, "course_id")

	resp, err := h.Lms.UpdateCourse(r.Context(), &req)
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// DeleteLmsCourse DELETE /api/lms/courses/{course_id}
func (h *Handler) DeleteLmsCourse(w http.ResponseWriter, r *http.Request) {
	version, err := readVersionParam(r)
	if err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid version")
		return
	}

	resp, err := h.Lms.DeleteCourse(r.Context(), &lmspb.DeleteCourseRequest{
		CourseId: chi.URLParam(r, "course_id"),
		Version:  version,
	})
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// PublishLmsCourse POST /api/lms/courses/{course_id}/publish
func (h *Handler) PublishLmsCourse(w http.ResponseWriter, r *http.Request) {
	var body versionOnlyReq
	if err := ReadJSON(r, &body); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}

	resp, err := h.Lms.PublishCourse(r.Context(), &lmspb.PublishCourseRequest{
		CourseId: chi.URLParam(r, "course_id"),
		Version:  body.Version,
	})
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// UnpublishLmsCourse POST /api/lms/courses/{course_id}/unpublish
func (h *Handler) UnpublishLmsCourse(w http.ResponseWriter, r *http.Request) {
	var body versionOnlyReq
	if err := ReadJSON(r, &body); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}

	resp, err := h.Lms.UnpublishCourse(r.Context(), &lmspb.UnpublishCourseRequest{
		CourseId: chi.URLParam(r, "course_id"),
		Version:  body.Version,
	})
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// ListLmsCourseMaterials GET /api/lms/courses/{course_id}/materials
func (h *Handler) ListLmsCourseMaterials(w http.ResponseWriter, r *http.Request) {
	resp, err := h.Lms.ListCourseMaterials(r.Context(), &lmspb.ListCourseMaterialsRequest{
		CourseId: chi.URLParam(r, "course_id"),
	})
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// CreateLmsCourseMaterial POST /api/lms/courses/{course_id}/materials
func (h *Handler) CreateLmsCourseMaterial(w http.ResponseWriter, r *http.Request) {
	var req lmspb.CreateCourseMaterialRequest
	if err := ReadJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}
	req.MaterialId = newLmsID()
	req.CourseId = chi.URLParam(r, "course_id")

	resp, err := h.Lms.CreateCourseMaterial(r.Context(), &req)
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// GetLmsCourseMaterial GET /api/lms/materials/{material_id}
func (h *Handler) GetLmsCourseMaterial(w http.ResponseWriter, r *http.Request) {
	resp, err := h.Lms.GetCourseMaterial(r.Context(), &lmspb.GetCourseMaterialRequest{
		MaterialId: chi.URLParam(r, "material_id"),
	})
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// UpdateLmsCourseMaterial PUT /api/lms/materials/{material_id}
func (h *Handler) UpdateLmsCourseMaterial(w http.ResponseWriter, r *http.Request) {
	var req lmspb.UpdateCourseMaterialRequest
	if err := ReadJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}
	req.MaterialId = chi.URLParam(r, "material_id")

	resp, err := h.Lms.UpdateCourseMaterial(r.Context(), &req)
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// DeleteLmsCourseMaterial DELETE /api/lms/materials/{material_id}
func (h *Handler) DeleteLmsCourseMaterial(w http.ResponseWriter, r *http.Request) {
	version, err := readVersionParam(r)
	if err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid version")
		return
	}

	resp, err := h.Lms.DeleteCourseMaterial(r.Context(), &lmspb.DeleteCourseMaterialRequest{
		MaterialId: chi.URLParam(r, "material_id"),
		Version:    version,
	})
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// ReorderLmsCourseMaterials POST /api/lms/courses/{course_id}/materials/reorder
func (h *Handler) ReorderLmsCourseMaterials(w http.ResponseWriter, r *http.Request) {
	var req lmspb.ReorderCourseMaterialsRequest
	if err := ReadJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}
	req.CourseId = chi.URLParam(r, "course_id")

	resp, err := h.Lms.ReorderCourseMaterials(r.Context(), &req)
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// ListLmsChapters GET /api/lms/courses/{course_id}/chapters
func (h *Handler) ListLmsChapters(w http.ResponseWriter, r *http.Request) {
	resp, err := h.Lms.ListChapters(r.Context(), &lmspb.ListChaptersRequest{
		CourseId: chi.URLParam(r, "course_id"),
	})
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// CreateLmsChapter POST /api/lms/courses/{course_id}/chapters
func (h *Handler) CreateLmsChapter(w http.ResponseWriter, r *http.Request) {
	var req lmspb.CreateChapterRequest
	if err := ReadJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}
	req.ChapterId = newLmsID()
	req.CourseId = chi.URLParam(r, "course_id")

	resp, err := h.Lms.CreateChapter(r.Context(), &req)
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// GetLmsChapter GET /api/lms/chapters/{chapter_id}
func (h *Handler) GetLmsChapter(w http.ResponseWriter, r *http.Request) {
	resp, err := h.Lms.GetChapter(r.Context(), &lmspb.GetChapterRequest{
		ChapterId: chi.URLParam(r, "chapter_id"),
	})
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// UpdateLmsChapter PUT /api/lms/chapters/{chapter_id}
func (h *Handler) UpdateLmsChapter(w http.ResponseWriter, r *http.Request) {
	var req lmspb.UpdateChapterRequest
	if err := ReadJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}
	req.ChapterId = chi.URLParam(r, "chapter_id")

	resp, err := h.Lms.UpdateChapter(r.Context(), &req)
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// DeleteLmsChapter DELETE /api/lms/chapters/{chapter_id}
func (h *Handler) DeleteLmsChapter(w http.ResponseWriter, r *http.Request) {
	version, err := readVersionParam(r)
	if err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid version")
		return
	}

	resp, err := h.Lms.DeleteChapter(r.Context(), &lmspb.DeleteChapterRequest{
		ChapterId: chi.URLParam(r, "chapter_id"),
		Version:   version,
	})
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// ReorderLmsChapters POST /api/lms/courses/{course_id}/chapters/reorder
func (h *Handler) ReorderLmsChapters(w http.ResponseWriter, r *http.Request) {
	var req lmspb.ReorderChaptersRequest
	if err := ReadJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}
	req.CourseId = chi.URLParam(r, "course_id")

	resp, err := h.Lms.ReorderChapters(r.Context(), &req)
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// ListLmsLessons GET /api/lms/chapters/{chapter_id}/lessons
func (h *Handler) ListLmsLessons(w http.ResponseWriter, r *http.Request) {
	resp, err := h.Lms.ListLessons(r.Context(), &lmspb.ListLessonsRequest{
		ChapterId: chi.URLParam(r, "chapter_id"),
	})
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// CreateLmsLesson POST /api/lms/chapters/{chapter_id}/lessons
func (h *Handler) CreateLmsLesson(w http.ResponseWriter, r *http.Request) {
	var req lmspb.CreateLessonRequest
	if err := ReadJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}
	req.LessonId = newLmsID()
	req.ChapterId = chi.URLParam(r, "chapter_id")
	if strings.TrimSpace(req.MetaJson) == "" {
		req.MetaJson = "{}"
	}

	resp, err := h.Lms.CreateLesson(r.Context(), &req)
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// GetLmsLesson GET /api/lms/lessons/{lesson_id}
func (h *Handler) GetLmsLesson(w http.ResponseWriter, r *http.Request) {
	resp, err := h.Lms.GetLesson(r.Context(), &lmspb.GetLessonRequest{
		LessonId: chi.URLParam(r, "lesson_id"),
	})
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// UpdateLmsLesson PUT /api/lms/lessons/{lesson_id}
func (h *Handler) UpdateLmsLesson(w http.ResponseWriter, r *http.Request) {
	var req lmspb.UpdateLessonRequest
	if err := ReadJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}
	req.LessonId = chi.URLParam(r, "lesson_id")
	if strings.TrimSpace(req.MetaJson) == "" {
		req.MetaJson = "{}"
	}

	resp, err := h.Lms.UpdateLesson(r.Context(), &req)
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// DeleteLmsLesson DELETE /api/lms/lessons/{lesson_id}
func (h *Handler) DeleteLmsLesson(w http.ResponseWriter, r *http.Request) {
	version, err := readVersionParam(r)
	if err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid version")
		return
	}

	resp, err := h.Lms.DeleteLesson(r.Context(), &lmspb.DeleteLessonRequest{
		LessonId: chi.URLParam(r, "lesson_id"),
		Version:  version,
	})
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// ReorderLmsLessons POST /api/lms/chapters/{chapter_id}/lessons/reorder
func (h *Handler) ReorderLmsLessons(w http.ResponseWriter, r *http.Request) {
	var req lmspb.ReorderLessonsRequest
	if err := ReadJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}
	req.ChapterId = chi.URLParam(r, "chapter_id")

	resp, err := h.Lms.ReorderLessons(r.Context(), &req)
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// ListLmsPrerequisites GET /api/lms/prerequisites
func (h *Handler) ListLmsPrerequisites(w http.ResponseWriter, r *http.Request) {
	resp, err := h.Lms.ListPrerequisites(r.Context(), &lmspb.ListPrerequisitesRequest{
		TargetEntityType: lmspb.EntityType(parseEnumQuery(r, "target_entity_type")),
		TargetEntityId:   r.URL.Query().Get("target_entity_id"),
	})
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// CreateLmsPrerequisite POST /api/lms/prerequisites
func (h *Handler) CreateLmsPrerequisite(w http.ResponseWriter, r *http.Request) {
	var req lmspb.CreatePrerequisiteRequest
	if err := ReadJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}
	req.PrerequisiteId = newLmsID()

	resp, err := h.Lms.CreatePrerequisite(r.Context(), &req)
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// GetLmsPrerequisite GET /api/lms/prerequisites/{prerequisite_id}
func (h *Handler) GetLmsPrerequisite(w http.ResponseWriter, r *http.Request) {
	resp, err := h.Lms.GetPrerequisite(r.Context(), &lmspb.GetPrerequisiteRequest{
		PrerequisiteId: chi.URLParam(r, "prerequisite_id"),
	})
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// UpdateLmsPrerequisite PUT /api/lms/prerequisites/{prerequisite_id}
func (h *Handler) UpdateLmsPrerequisite(w http.ResponseWriter, r *http.Request) {
	var req lmspb.UpdatePrerequisiteRequest
	if err := ReadJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}
	req.PrerequisiteId = chi.URLParam(r, "prerequisite_id")

	resp, err := h.Lms.UpdatePrerequisite(r.Context(), &req)
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// DeleteLmsPrerequisite DELETE /api/lms/prerequisites/{prerequisite_id}
func (h *Handler) DeleteLmsPrerequisite(w http.ResponseWriter, r *http.Request) {
	version, err := readVersionParam(r)
	if err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid version")
		return
	}

	resp, err := h.Lms.DeletePrerequisite(r.Context(), &lmspb.DeletePrerequisiteRequest{
		PrerequisiteId: chi.URLParam(r, "prerequisite_id"),
		Version:        version,
	})
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// ListLmsQuizzes GET /api/lms/quizzes
func (h *Handler) ListLmsQuizzes(w http.ResponseWriter, r *http.Request) {
	resp, err := h.Lms.ListQuizzes(r.Context(), &lmspb.ListQuizzesRequest{
		QuizzableType: lmspb.QuizzableType(parseEnumQuery(r, "quizzable_type")),
		QuizzableId:   r.URL.Query().Get("quizzable_id"),
	})
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// CreateLmsQuiz POST /api/lms/quizzes
func (h *Handler) CreateLmsQuiz(w http.ResponseWriter, r *http.Request) {
	var req lmspb.CreateQuizRequest
	if err := ReadJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}
	req.QuizId = newLmsID()

	resp, err := h.Lms.CreateQuiz(r.Context(), &req)
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// GetLmsQuiz GET /api/lms/quizzes/{quiz_id}
func (h *Handler) GetLmsQuiz(w http.ResponseWriter, r *http.Request) {
	resp, err := h.Lms.GetQuiz(r.Context(), &lmspb.GetQuizRequest{
		QuizId: chi.URLParam(r, "quiz_id"),
	})
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// UpdateLmsQuiz PUT /api/lms/quizzes/{quiz_id}
func (h *Handler) UpdateLmsQuiz(w http.ResponseWriter, r *http.Request) {
	var req lmspb.UpdateQuizRequest
	if err := ReadJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}
	req.QuizId = chi.URLParam(r, "quiz_id")

	resp, err := h.Lms.UpdateQuiz(r.Context(), &req)
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// DeleteLmsQuiz DELETE /api/lms/quizzes/{quiz_id}
func (h *Handler) DeleteLmsQuiz(w http.ResponseWriter, r *http.Request) {
	version, err := readVersionParam(r)
	if err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid version")
		return
	}

	resp, err := h.Lms.DeleteQuiz(r.Context(), &lmspb.DeleteQuizRequest{
		QuizId:  chi.URLParam(r, "quiz_id"),
		Version: version,
	})
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// ListLmsQuizQuestions GET /api/lms/quizzes/{quiz_id}/questions
func (h *Handler) ListLmsQuizQuestions(w http.ResponseWriter, r *http.Request) {
	resp, err := h.Lms.ListQuizQuestions(r.Context(), &lmspb.ListQuizQuestionsRequest{
		QuizId: chi.URLParam(r, "quiz_id"),
	})
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// CreateLmsQuizQuestion POST /api/lms/quizzes/{quiz_id}/questions
func (h *Handler) CreateLmsQuizQuestion(w http.ResponseWriter, r *http.Request) {
	var req lmspb.CreateQuizQuestionRequest
	if err := ReadJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}
	req.QuestionId = newLmsID()
	req.QuizId = chi.URLParam(r, "quiz_id")
	if strings.TrimSpace(req.MediaItemsJson) == "" {
		req.MediaItemsJson = "[]"
	}

	resp, err := h.Lms.CreateQuizQuestion(r.Context(), &req)
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// GetLmsQuizQuestion GET /api/lms/questions/{question_id}
func (h *Handler) GetLmsQuizQuestion(w http.ResponseWriter, r *http.Request) {
	resp, err := h.Lms.GetQuizQuestion(r.Context(), &lmspb.GetQuizQuestionRequest{
		QuestionId: chi.URLParam(r, "question_id"),
	})
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// UpdateLmsQuizQuestion PUT /api/lms/questions/{question_id}
func (h *Handler) UpdateLmsQuizQuestion(w http.ResponseWriter, r *http.Request) {
	var req lmspb.UpdateQuizQuestionRequest
	if err := ReadJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}
	req.QuestionId = chi.URLParam(r, "question_id")
	if strings.TrimSpace(req.MediaItemsJson) == "" {
		req.MediaItemsJson = "[]"
	}

	resp, err := h.Lms.UpdateQuizQuestion(r.Context(), &req)
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// DeleteLmsQuizQuestion DELETE /api/lms/questions/{question_id}
func (h *Handler) DeleteLmsQuizQuestion(w http.ResponseWriter, r *http.Request) {
	version, err := readVersionParam(r)
	if err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid version")
		return
	}

	resp, err := h.Lms.DeleteQuizQuestion(r.Context(), &lmspb.DeleteQuizQuestionRequest{
		QuestionId: chi.URLParam(r, "question_id"),
		Version:    version,
	})
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// ReorderLmsQuizQuestions POST /api/lms/quizzes/{quiz_id}/questions/reorder
func (h *Handler) ReorderLmsQuizQuestions(w http.ResponseWriter, r *http.Request) {
	var req lmspb.ReorderQuizQuestionsRequest
	if err := ReadJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}
	req.QuizId = chi.URLParam(r, "quiz_id")

	resp, err := h.Lms.ReorderQuizQuestions(r.Context(), &req)
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// ListLmsQuizOptions GET /api/lms/questions/{question_id}/options
func (h *Handler) ListLmsQuizOptions(w http.ResponseWriter, r *http.Request) {
	resp, err := h.Lms.ListQuizOptions(r.Context(), &lmspb.ListQuizOptionsRequest{
		QuestionId: chi.URLParam(r, "question_id"),
	})
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// CreateLmsQuizOption POST /api/lms/questions/{question_id}/options
func (h *Handler) CreateLmsQuizOption(w http.ResponseWriter, r *http.Request) {
	var req lmspb.CreateQuizOptionRequest
	if err := ReadJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}
	req.OptionId = newLmsID()
	req.QuestionId = chi.URLParam(r, "question_id")

	resp, err := h.Lms.CreateQuizOption(r.Context(), &req)
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// GetLmsQuizOption GET /api/lms/options/{option_id}
func (h *Handler) GetLmsQuizOption(w http.ResponseWriter, r *http.Request) {
	resp, err := h.Lms.GetQuizOption(r.Context(), &lmspb.GetQuizOptionRequest{
		OptionId: chi.URLParam(r, "option_id"),
	})
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// UpdateLmsQuizOption PUT /api/lms/options/{option_id}
func (h *Handler) UpdateLmsQuizOption(w http.ResponseWriter, r *http.Request) {
	var req lmspb.UpdateQuizOptionRequest
	if err := ReadJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}
	req.OptionId = chi.URLParam(r, "option_id")

	resp, err := h.Lms.UpdateQuizOption(r.Context(), &req)
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// DeleteLmsQuizOption DELETE /api/lms/options/{option_id}
func (h *Handler) DeleteLmsQuizOption(w http.ResponseWriter, r *http.Request) {
	version, err := readVersionParam(r)
	if err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid version")
		return
	}

	resp, err := h.Lms.DeleteQuizOption(r.Context(), &lmspb.DeleteQuizOptionRequest{
		OptionId: chi.URLParam(r, "option_id"),
		Version:  version,
	})
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// ReorderLmsQuizOptions POST /api/lms/questions/{question_id}/options/reorder
func (h *Handler) ReorderLmsQuizOptions(w http.ResponseWriter, r *http.Request) {
	var req lmspb.ReorderQuizOptionsRequest
	if err := ReadJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}
	req.QuestionId = chi.URLParam(r, "question_id")

	resp, err := h.Lms.ReorderQuizOptions(r.Context(), &req)
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// ListLmsObjects GET /api/lms/objects
func (h *Handler) ListLmsObjects(w http.ResponseWriter, r *http.Request) {
	resp, err := h.Lms.ListObjects(r.Context(), &lmspb.ListObjectsRequest{
		Prefix:    r.URL.Query().Get("prefix"),
		PageSize:  parseUint32Query(r, "page_size"),
		PageToken: r.URL.Query().Get("page_token"),
	})
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// CreateLmsUploadURL POST /api/lms/upload-url
func (h *Handler) CreateLmsUploadURL(w http.ResponseWriter, r *http.Request) {
	var req lmspb.CreateUploadURLRequest
	if err := ReadJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}
	if strings.TrimSpace(req.ContentType) == "" {
		req.ContentType = defaultUploadContentType
	}

	resp, err := h.Lms.CreateUploadURL(r.Context(), &req)
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// CreateLmsViewURL POST /api/lms/view-url
func (h *Handler) CreateLmsViewURL(w http.ResponseWriter, r *http.Request) {
	var req lmspb.CreateViewURLRequest
	if err := ReadJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}

	resp, err := h.Lms.CreateViewURL(r.Context(), &req)
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}
