package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	lmspb "github.com/afnandelfin620-star/cftptest/cftp/glms"
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

type lmsLessonPayload interface {
	GetLessonType() lmspb.LessonType
	GetBody() string
	GetMediaObjectKey() string
	GetExternalUrl() string
}

func validateLmsLessonPayload(w http.ResponseWriter, req lmsLessonPayload) bool {
	switch req.GetLessonType() {
	case lmspb.LessonType_LESSON_TYPE_TEXT:
		return requireRequestField(w, req.GetBody(), "body")
	case lmspb.LessonType_LESSON_TYPE_LINK:
		return requireRequestField(w, req.GetExternalUrl(), "external_url")
	case lmspb.LessonType_LESSON_TYPE_VIDEO,
		lmspb.LessonType_LESSON_TYPE_PDF,
		lmspb.LessonType_LESSON_TYPE_IMAGE,
		lmspb.LessonType_LESSON_TYPE_AUDIO,
		lmspb.LessonType_LESSON_TYPE_FILE:
		return requireRequestField(w, req.GetMediaObjectKey(), "media_object_key")
	default:
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "lesson_type is invalid")
		return false
	}
}

func validateLmsPrerequisitePayload(
	w http.ResponseWriter,
	requiredType lmspb.EntityType,
	requiredID string,
	requiredResult lmspb.PrerequisiteResult,
	targetType lmspb.EntityType,
	targetID string,
) bool {
	if requiredType == lmspb.EntityType_ENTITY_TYPE_UNSPECIFIED {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "required_entity_type is required")
		return false
	}
	if !requireRequestField(w, requiredID, "required_entity_id") {
		return false
	}
	if requiredResult == lmspb.PrerequisiteResult_PREREQUISITE_RESULT_UNSPECIFIED {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "required_result is required")
		return false
	}
	if targetType == lmspb.EntityType_ENTITY_TYPE_UNSPECIFIED {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "target_entity_type is required")
		return false
	}
	return requireRequestField(w, targetID, "target_entity_id")
}

func validateLmsUploadURLPayload(w http.ResponseWriter, req *lmspb.CreateUploadURLRequest) bool {
	if !requireRequestField(w, req.FileName, "file_name") {
		return false
	}
	if !requireRequestField(w, req.FileHash, "file_hash") {
		return false
	}
	switch req.UploadType {
	case lmspb.UploadType_UPLOAD_TYPE_COURSE_THUMBNAIL:
		return requireRequestField(w, req.CourseUlid, "course_id")
	case lmspb.UploadType_UPLOAD_TYPE_COURSE_MATERIAL:
		return requireRequestFields(w, req.CourseUlid, "course_id", req.MaterialUlid, "material_id")
	case lmspb.UploadType_UPLOAD_TYPE_LESSON_ASSET:
		return requireRequestFields(w, req.CourseUlid, "course_id", req.ChapterUlid, "chapter_id", req.LessonUlid, "lesson_id")
	case lmspb.UploadType_UPLOAD_TYPE_QUIZ_ASSET:
		return requireRequestFields(w, req.CourseUlid, "course_id", req.QuizUlid, "quiz_id")
	case lmspb.UploadType_UPLOAD_TYPE_RESOURCE_PACK_THUMBNAIL:
		return requireRequestField(w, req.PackId, "pack_id")
	case lmspb.UploadType_UPLOAD_TYPE_RESOURCE_PACK_FILE:
		return requireRequestFields(w, req.PackId, "pack_id", req.ResourcePackFileId, "resource_pack_file_id")
	default:
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "upload_type is required")
		return false
	}
}

// ListLmsCourses GET /api/lms/courses
func (h *Handler) ListLmsCourses(w http.ResponseWriter, r *http.Request) {
	page := parseCursorPage(r, 20)
	resp, err := h.Lms.ListCoursesAdmin(r.Context(), &lmspb.ListCoursesRequest{
		Filters: &lmspb.CourseFilters{
			CategoryTips:  r.URL.Query().Get("category_tips"),
			PublishedOnly: parseBoolQuery(r, "published_only"),
		},
		PageSize: page.PageSize,
		SortOrder: lmspb.SortOrder(page.Sort),
		Cursor:   firstNonEmpty(r.URL.Query().Get("cursor"), r.URL.Query().Get("page_token")),
	})
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// CreateLmsCourse POST /api/lms/courses
func (h *Handler) CreateLmsCourse(w http.ResponseWriter, r *http.Request) {
	var input struct {
		lmspb.CreateCourseDraftRequest
		FromCourseUlid string `json:"from_course_id"`
	}
	if err := ReadJSON(r, &input); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}
	if strings.TrimSpace(input.FromCourseUlid) != "" {
		req := &lmspb.DuplicateCourseDraftRequest{
			FromCourseUlid: strings.TrimSpace(input.FromCourseUlid),
			CourseUlid:     newLmsID(),
			Title:          input.Title,
		}
		if !requireRequestFields(w, req.FromCourseUlid, "from_course_id", req.Title, "title") {
			return
		}
		resp, err := h.Lms.DuplicateCourseDraftAdmin(r.Context(), req)
		if err != nil {
			writeLmsError(w, err)
			return
		}
		WriteJSON(w, http.StatusOK, resp)
		return
	}

	req := input.CreateCourseDraftRequest
	req.CourseUlid = newLmsID()
	if !requireRequestField(w, req.Title, "title") {
		return
	}

	resp, err := h.Lms.CreateCourseDraftAdmin(r.Context(), &req)
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// GetLmsCourse GET /api/lms/courses/{course_id}
func (h *Handler) GetLmsCourse(w http.ResponseWriter, r *http.Request) {
	courseID, ok := requiredURLParam(w, r, "course_id")
	if !ok {
		return
	}
	resp, err := h.Lms.GetCourseSummaryAdmin(r.Context(), &lmspb.GetCourseRequest{
		CourseUlid: courseID,
	})
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// GetLmsCourseDetail GET /api/lms/courses/{course_id}/detail
func (h *Handler) GetLmsCourseDetail(w http.ResponseWriter, r *http.Request) {
	courseID, ok := requiredURLParam(w, r, "course_id")
	if !ok {
		return
	}
	resp, err := h.Lms.GetCourseDetailAdmin(r.Context(), &lmspb.GetCourseDetailRequest{
		CourseUlid: courseID,
	})
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// GetCompleteLmsCourse GET /api/lms/courses/{course_id}/complete
func (h *Handler) GetCompleteLmsCourse(w http.ResponseWriter, r *http.Request) {
	courseID, ok := requiredURLParam(w, r, "course_id")
	if !ok {
		return
	}
	resp, err := h.Lms.GetCompleteCourseAdmin(r.Context(), &lmspb.GetCompleteCourseRequest{
		CourseUlid: courseID,
	})
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// UpdateLmsCourse PUT /api/lms/courses/{course_id}
func (h *Handler) UpdateLmsCourse(w http.ResponseWriter, r *http.Request) {
	courseID, ok := requiredURLParam(w, r, "course_id")
	if !ok {
		return
	}
	var req lmspb.UpdateCourseRequest
	if err := ReadJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}
	req.CourseUlid = courseID
	if !requireRequestField(w, req.Title, "title") || !requirePositiveVersion(w, req.Version) {
		return
	}

	resp, err := h.Lms.UpdateCourseAdmin(r.Context(), &req)
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// DeleteLmsCourse DELETE /api/lms/courses/{course_id}
func (h *Handler) DeleteLmsCourse(w http.ResponseWriter, r *http.Request) {
	courseID, ok := requiredURLParam(w, r, "course_id")
	if !ok {
		return
	}
	version, err := readVersionParam(r)
	if err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid version")
		return
	}
	if !requirePositiveVersion(w, version) {
		return
	}

	resp, err := h.Lms.DeleteCourseAdmin(r.Context(), &lmspb.DeleteCourseRequest{
		CourseUlid: courseID,
		Version:    version,
	})
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// PublishLmsCourse POST /api/lms/courses/{course_id}/publish
func (h *Handler) PublishLmsCourse(w http.ResponseWriter, r *http.Request) {
	courseID, ok := requiredURLParam(w, r, "course_id")
	if !ok {
		return
	}
	var body versionOnlyReq
	if err := ReadJSON(r, &body); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}
	if !requirePositiveVersion(w, body.Version) {
		return
	}

	resp, err := h.Lms.PublishCourseAdmin(r.Context(), &lmspb.PublishCourseRequest{
		CourseUlid: courseID,
		Version:    body.Version,
	})
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// ListLmsCourseEnrollmentsForAdmin GET /api/lms/courses/{course_id}/enrollments
func (h *Handler) ListLmsCourseEnrollmentsForAdmin(w http.ResponseWriter, r *http.Request) {
	courseID, ok := requiredURLParam(w, r, "course_id")
	if !ok {
		return
	}
	page := parseCursorPage(r, 20)
	resp, err := h.Lms.ListCourseEnrollmentsForAdmin(r.Context(), &lmspb.ListCourseEnrollmentsForAdminRequest{
		Filters: &lmspb.CourseEnrollmentForAdminFilters{
			CourseUlid: courseID,
			Status:     r.URL.Query().Get("status"),
		},
		PageSize: page.PageSize,
		SortOrder: lmspb.SortOrder(page.Sort),
		Cursor:   firstNonEmpty(r.URL.Query().Get("cursor"), r.URL.Query().Get("page_token")),
	})
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// GetLmsCandidateProgressForAdmin GET /api/lms/courses/{course_id}/candidates/{candidate_id}/progress
func (h *Handler) GetLmsCandidateProgressForAdmin(w http.ResponseWriter, r *http.Request) {
	courseID, ok := requiredURLParam(w, r, "course_id")
	if !ok {
		return
	}
	candidateID, ok := requiredURLParam(w, r, "candidate_id")
	if !ok {
		return
	}
	resp, err := h.Lms.GetCandidateProgressForAdmin(r.Context(), &lmspb.GetCandidateProgressForAdminRequest{
		CandidateUlid: candidateID,
		CourseUlid:    courseID,
	})
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// ListLmsCourseEnrollments GET /api/lms/enrollments
func (h *Handler) ListLmsCourseEnrollments(w http.ResponseWriter, r *http.Request) {
	page := parseCursorPage(r, 20)
	resp, err := h.Lms.ListCourseEnrollmentsAdmin(r.Context(), &lmspb.ListCourseEnrollmentsRequest{
		Filters: &lmspb.CourseEnrollmentFilters{
			CandidateUlid: r.URL.Query().Get("candidate_id"),
			CourseUlid:    r.URL.Query().Get("course_id"),
			BizUnit:       r.URL.Query().Get("biz_unit"),
			Status:        r.URL.Query().Get("status"),
		},
		PageSize: page.PageSize,
		SortOrder: lmspb.SortOrder(page.Sort),
		Cursor:   firstNonEmpty(r.URL.Query().Get("cursor"), r.URL.Query().Get("page_token")),
	})
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// GetLmsCourseEnrollmentDetail GET /api/lms/enrollments/{enrollment_id}
func (h *Handler) GetLmsCourseEnrollmentDetail(w http.ResponseWriter, r *http.Request) {
	enrollmentID, ok := requiredURLParam(w, r, "enrollment_id")
	if !ok {
		return
	}
	resp, err := h.Lms.GetCourseEnrollmentDetailAdmin(r.Context(), &lmspb.GetCourseEnrollmentDetailRequest{
		EnrollmentId: enrollmentID,
	})
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// ListLmsLessonProgress GET /api/lms/lesson-progress
func (h *Handler) ListLmsLessonProgress(w http.ResponseWriter, r *http.Request) {
	candidateID := r.URL.Query().Get("candidate_id")
	if !requireRequestField(w, candidateID, "candidate_id") {
		return
	}
	page := parseCursorPage(r, 20)
	resp, err := h.Lms.ListLessonProgressAdmin(r.Context(), &lmspb.ListLessonProgressRequest{
		Filters: &lmspb.LessonProgressFilters{
			CandidateUlid: candidateID,
			LessonUlid:    r.URL.Query().Get("lesson_id"),
			Status:        r.URL.Query().Get("status"),
		},
		PageSize: page.PageSize,
		SortOrder: lmspb.SortOrder(page.Sort),
		Cursor:   firstNonEmpty(r.URL.Query().Get("cursor"), r.URL.Query().Get("page_token")),
	})
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// GetLmsLessonProgressDetail GET /api/lms/lessons/{lesson_id}/progress
func (h *Handler) GetLmsLessonProgressDetail(w http.ResponseWriter, r *http.Request) {
	lessonID, ok := requiredURLParam(w, r, "lesson_id")
	if !ok {
		return
	}
	candidateID := r.URL.Query().Get("candidate_id")
	if !requireRequestField(w, candidateID, "candidate_id") {
		return
	}
	resp, err := h.Lms.GetLessonProgressDetailAdmin(r.Context(), &lmspb.GetLessonProgressDetailRequest{
		UserUlid:   candidateID,
		LessonUlid: lessonID,
	})
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// ListLmsChapterProgress GET /api/lms/chapter-progress
func (h *Handler) ListLmsChapterProgress(w http.ResponseWriter, r *http.Request) {
	candidateID := r.URL.Query().Get("candidate_id")
	courseID := r.URL.Query().Get("course_id")
	if !requireRequestFields(w, candidateID, "candidate_id", courseID, "course_id") {
		return
	}
	page := parseCursorPage(r, 20)
	resp, err := h.Lms.ListChapterProgressAdmin(r.Context(), &lmspb.ListChapterProgressRequest{
		Filters: &lmspb.ChapterProgressFilters{
			CandidateUlid: candidateID,
			CourseUlid:    courseID,
		},
		PageSize: page.PageSize,
		SortOrder: lmspb.SortOrder(page.Sort),
		Cursor:   firstNonEmpty(r.URL.Query().Get("cursor"), r.URL.Query().Get("page_token")),
	})
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// GetLmsChapterProgressDetail GET /api/lms/chapters/{chapter_id}/progress
func (h *Handler) GetLmsChapterProgressDetail(w http.ResponseWriter, r *http.Request) {
	chapterID, ok := requiredURLParam(w, r, "chapter_id")
	if !ok {
		return
	}
	candidateID := r.URL.Query().Get("candidate_id")
	if !requireRequestField(w, candidateID, "candidate_id") {
		return
	}
	resp, err := h.Lms.GetChapterProgressDetailAdmin(r.Context(), &lmspb.GetChapterProgressDetailRequest{
		CandidateUlid: candidateID,
		ChapterUlid:   chapterID,
	})
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// ListLmsQuizAttempts GET /api/lms/quizzes/{quiz_id}/attempts
func (h *Handler) ListLmsQuizAttempts(w http.ResponseWriter, r *http.Request) {
	quizID, ok := requiredURLParam(w, r, "quiz_id")
	if !ok {
		return
	}
	page := parseCursorPage(r, 20)
	resp, err := h.Lms.ListQuizAttemptsAdmin(r.Context(), &lmspb.ListQuizAttemptsRequest{
		Filters: &lmspb.QuizAttemptsFilters{
			QuizUlid: quizID,
			UserUlid: r.URL.Query().Get("candidate_id"),
			Status:   r.URL.Query().Get("status"),
		},
		PageSize: page.PageSize,
		SortOrder: lmspb.SortOrder(page.Sort),
		Cursor:   firstNonEmpty(r.URL.Query().Get("cursor"), r.URL.Query().Get("page_token")),
	})
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// GetLmsQuizAttemptDetail GET /api/lms/quiz-attempts/{attempt_id}
func (h *Handler) GetLmsQuizAttemptDetail(w http.ResponseWriter, r *http.Request) {
	attemptID, ok := requiredURLParam(w, r, "attempt_id")
	if !ok {
		return
	}
	resp, err := h.Lms.GetQuizAttemptDetailAdmin(r.Context(), &lmspb.GetQuizAttemptDetailRequest{
		AttemptId: attemptID,
	})
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// ListLmsCourseMaterials GET /api/lms/courses/{course_id}/materials
func (h *Handler) ListLmsCourseMaterials(w http.ResponseWriter, r *http.Request) {
	courseID, ok := requiredURLParam(w, r, "course_id")
	if !ok {
		return
	}
	resp, err := h.Lms.ListCourseMaterialsAdmin(r.Context(), &lmspb.ListCourseMaterialsRequest{
		CourseUlid: courseID,
	})
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// BatchEnrollLmsCandidateCourses POST /api/lms/enrollments/batch
func (h *Handler) BatchEnrollLmsCandidateCourses(w http.ResponseWriter, r *http.Request) {
	var req lmspb.BatchEnrollCandidateCoursesRequest
	if err := ReadJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}
	if len(req.CourseUlids) == 0 {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "course_ids is required")
		return
	}
	if req.BizUnit == "" {
		req.BizUnit = "adminserver"
	}
	if !requireRequestFields(w, req.CandidateUlid, "candidate_id", req.BizUnit, "biz_unit") {
		return
	}
	resp, err := h.Lms.BatchEnrollCandidateCoursesAdmin(r.Context(), &req)
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// SyncLmsCourseProgress POST /api/lms/courses/{course_id}/progress/sync
func (h *Handler) SyncLmsCourseProgress(w http.ResponseWriter, r *http.Request) {
	courseID, ok := requiredURLParam(w, r, "course_id")
	if !ok {
		return
	}
	var req lmspb.SyncCourseProgressRequest
	if err := ReadJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}
	req.CourseUlid = courseID
	if !requireRequestFields(w, req.CandidateUlid, "candidate_id", req.CourseUlid, "course_id") {
		return
	}
	resp, err := h.Lms.SyncCourseProgressAdmin(r.Context(), &req)
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// CreateLmsCourseMaterial POST /api/lms/courses/{course_id}/materials
func (h *Handler) CreateLmsCourseMaterial(w http.ResponseWriter, r *http.Request) {
	courseID, ok := requiredURLParam(w, r, "course_id")
	if !ok {
		return
	}
	var req lmspb.CreateCourseMaterialRequest
	if err := ReadJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}
	if strings.TrimSpace(req.MaterialUlid) == "" {
		req.MaterialUlid = newLmsID()
	}
	req.CourseUlid = courseID
	if !requireRequestFields(w, req.Title, "title", req.FileObjectKey, "file_object_key") {
		return
	}
	if req.MaterialType == lmspb.MaterialType_MATERIAL_TYPE_UNSPECIFIED {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "material_type is required")
		return
	}

	resp, err := h.Lms.CreateCourseMaterialAdmin(r.Context(), &req)
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// GetLmsCourseMaterial GET /api/lms/materials/{material_id}
func (h *Handler) GetLmsCourseMaterial(w http.ResponseWriter, r *http.Request) {
	materialID, ok := requiredURLParam(w, r, "material_id")
	if !ok {
		return
	}
	resp, err := h.Lms.GetCourseMaterialDetailAdmin(r.Context(), &lmspb.GetCourseMaterialRequest{
		MaterialUlid: materialID,
	})
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// UpdateLmsCourseMaterial PUT /api/lms/materials/{material_id}
func (h *Handler) UpdateLmsCourseMaterial(w http.ResponseWriter, r *http.Request) {
	materialID, ok := requiredURLParam(w, r, "material_id")
	if !ok {
		return
	}
	var req lmspb.UpdateCourseMaterialRequest
	if err := ReadJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}
	req.MaterialUlid = materialID
	if !requireRequestFields(w, req.Title, "title", req.FileObjectKey, "file_object_key") || !requirePositiveVersion(w, req.Version) {
		return
	}
	if req.MaterialType == lmspb.MaterialType_MATERIAL_TYPE_UNSPECIFIED {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "material_type is required")
		return
	}

	resp, err := h.Lms.UpdateCourseMaterialAdmin(r.Context(), &req)
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// DeleteLmsCourseMaterial DELETE /api/lms/materials/{material_id}
func (h *Handler) DeleteLmsCourseMaterial(w http.ResponseWriter, r *http.Request) {
	materialID, ok := requiredURLParam(w, r, "material_id")
	if !ok {
		return
	}
	version, err := readVersionParam(r)
	if err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid version")
		return
	}
	if !requirePositiveVersion(w, version) {
		return
	}

	resp, err := h.Lms.DeleteCourseMaterialAdmin(r.Context(), &lmspb.DeleteCourseMaterialRequest{
		MaterialUlid: materialID,
		Version:      version,
	})
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// ReorderLmsCourseMaterials POST /api/lms/courses/{course_id}/materials/reorder
func (h *Handler) ReorderLmsCourseMaterials(w http.ResponseWriter, r *http.Request) {
	courseID, ok := requiredURLParam(w, r, "course_id")
	if !ok {
		return
	}
	var req lmspb.ReorderCourseMaterialsRequest
	if err := ReadJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}
	req.CourseUlid = courseID
	if !requireReorderItems(w, req.Items) {
		return
	}

	resp, err := h.Lms.ReorderCourseMaterialsAdmin(r.Context(), &req)
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// ListLmsChapters GET /api/lms/courses/{course_id}/chapters
func (h *Handler) ListLmsChapters(w http.ResponseWriter, r *http.Request) {
	courseID, ok := requiredURLParam(w, r, "course_id")
	if !ok {
		return
	}
	resp, err := h.Lms.ListChaptersAdmin(r.Context(), &lmspb.ListChaptersRequest{
		CourseUlid: courseID,
	})
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// CreateLmsChapter POST /api/lms/courses/{course_id}/chapters
func (h *Handler) CreateLmsChapter(w http.ResponseWriter, r *http.Request) {
	courseID, ok := requiredURLParam(w, r, "course_id")
	if !ok {
		return
	}
	var req lmspb.CreateChapterRequest
	if err := ReadJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}
	req.ChapterUlid = newLmsID()
	req.CourseUlid = courseID
	if !requireRequestField(w, req.Title, "title") {
		return
	}

	resp, err := h.Lms.CreateChapterAdmin(r.Context(), &req)
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// GetLmsChapter GET /api/lms/chapters/{chapter_id}
func (h *Handler) GetLmsChapter(w http.ResponseWriter, r *http.Request) {
	chapterID, ok := requiredURLParam(w, r, "chapter_id")
	if !ok {
		return
	}
	resp, err := h.Lms.GetChapterDetailAdmin(r.Context(), &lmspb.GetChapterRequest{
		ChapterUlid: chapterID,
	})
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// UpdateLmsChapter PUT /api/lms/chapters/{chapter_id}
func (h *Handler) UpdateLmsChapter(w http.ResponseWriter, r *http.Request) {
	chapterID, ok := requiredURLParam(w, r, "chapter_id")
	if !ok {
		return
	}
	var req lmspb.UpdateChapterRequest
	if err := ReadJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}
	req.ChapterUlid = chapterID
	if !requireRequestField(w, req.Title, "title") || !requirePositiveVersion(w, req.Version) {
		return
	}

	resp, err := h.Lms.UpdateChapterAdmin(r.Context(), &req)
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// DeleteLmsChapter DELETE /api/lms/chapters/{chapter_id}
func (h *Handler) DeleteLmsChapter(w http.ResponseWriter, r *http.Request) {
	chapterID, ok := requiredURLParam(w, r, "chapter_id")
	if !ok {
		return
	}
	version, err := readVersionParam(r)
	if err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid version")
		return
	}
	if !requirePositiveVersion(w, version) {
		return
	}

	resp, err := h.Lms.DeleteChapterAdmin(r.Context(), &lmspb.DeleteChapterRequest{
		ChapterUlid: chapterID,
		Version:     version,
	})
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// ReorderLmsChapters POST /api/lms/courses/{course_id}/chapters/reorder
func (h *Handler) ReorderLmsChapters(w http.ResponseWriter, r *http.Request) {
	courseID, ok := requiredURLParam(w, r, "course_id")
	if !ok {
		return
	}
	var req lmspb.ReorderChaptersRequest
	if err := ReadJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}
	req.CourseUlid = courseID
	if !requireReorderItems(w, req.Items) {
		return
	}

	resp, err := h.Lms.ReorderChaptersAdmin(r.Context(), &req)
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// ListLmsLessons GET /api/lms/chapters/{chapter_id}/lessons
func (h *Handler) ListLmsLessons(w http.ResponseWriter, r *http.Request) {
	chapterID, ok := requiredURLParam(w, r, "chapter_id")
	if !ok {
		return
	}
	resp, err := h.Lms.ListLessonsAdmin(r.Context(), &lmspb.ListLessonsRequest{
		ChapterUlid: chapterID,
	})
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// ListLmsLessonsByCourse GET /api/lms/courses/{course_id}/lessons
func (h *Handler) ListLmsLessonsByCourse(w http.ResponseWriter, r *http.Request) {
	courseID, ok := requiredURLParam(w, r, "course_id")
	if !ok {
		return
	}
	page := parseCursorPage(r, 20)
	resp, err := h.Lms.ListLessonsByCourseAdmin(r.Context(), &lmspb.ListLessonsByCourseRequest{
		Filters: &lmspb.LessonFilters{
			CourseUlid: courseID,
		},
		PageSize: page.PageSize,
		SortOrder: lmspb.SortOrder(page.Sort),
		Cursor:   firstNonEmpty(r.URL.Query().Get("cursor"), r.URL.Query().Get("page_token")),
	})
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// CreateLmsLesson POST /api/lms/chapters/{chapter_id}/lessons
func (h *Handler) CreateLmsLesson(w http.ResponseWriter, r *http.Request) {
	chapterID, ok := requiredURLParam(w, r, "chapter_id")
	if !ok {
		return
	}
	var req lmspb.CreateLessonRequest
	if err := ReadJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}
	req.LessonUlid = newLmsID()
	req.ChapterUlid = chapterID
	if strings.TrimSpace(req.MetaJson) == "" {
		req.MetaJson = "{}"
	}
	if !requireRequestField(w, req.Title, "title") {
		return
	}
	if req.LessonType == lmspb.LessonType_LESSON_TYPE_UNSPECIFIED {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "lesson_type is required")
		return
	}
	if !validateLmsLessonPayload(w, &req) {
		return
	}

	resp, err := h.Lms.CreateLessonAdmin(r.Context(), &req)
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// GetLmsLesson GET /api/lms/lessons/{lesson_id}
func (h *Handler) GetLmsLesson(w http.ResponseWriter, r *http.Request) {
	lessonID, ok := requiredURLParam(w, r, "lesson_id")
	if !ok {
		return
	}
	resp, err := h.Lms.GetLessonDetailAdmin(r.Context(), &lmspb.GetLessonRequest{
		LessonUlid: lessonID,
	})
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// UpdateLmsLesson PUT /api/lms/lessons/{lesson_id}
func (h *Handler) UpdateLmsLesson(w http.ResponseWriter, r *http.Request) {
	lessonID, ok := requiredURLParam(w, r, "lesson_id")
	if !ok {
		return
	}
	var req lmspb.UpdateLessonRequest
	if err := ReadJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}
	req.LessonUlid = lessonID
	if strings.TrimSpace(req.MetaJson) == "" {
		req.MetaJson = "{}"
	}
	if !requireRequestField(w, req.Title, "title") || !requirePositiveVersion(w, req.Version) {
		return
	}
	if req.LessonType == lmspb.LessonType_LESSON_TYPE_UNSPECIFIED {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "lesson_type is required")
		return
	}
	if !validateLmsLessonPayload(w, &req) {
		return
	}

	resp, err := h.Lms.UpdateLessonAdmin(r.Context(), &req)
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// DeleteLmsLesson DELETE /api/lms/lessons/{lesson_id}
func (h *Handler) DeleteLmsLesson(w http.ResponseWriter, r *http.Request) {
	lessonID, ok := requiredURLParam(w, r, "lesson_id")
	if !ok {
		return
	}
	version, err := readVersionParam(r)
	if err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid version")
		return
	}
	if !requirePositiveVersion(w, version) {
		return
	}

	resp, err := h.Lms.DeleteLessonAdmin(r.Context(), &lmspb.DeleteLessonRequest{
		LessonUlid: lessonID,
		Version:    version,
	})
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// ReorderLmsLessons POST /api/lms/chapters/{chapter_id}/lessons/reorder
func (h *Handler) ReorderLmsLessons(w http.ResponseWriter, r *http.Request) {
	chapterID, ok := requiredURLParam(w, r, "chapter_id")
	if !ok {
		return
	}
	var req lmspb.ReorderLessonsRequest
	if err := ReadJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}
	req.ChapterUlid = chapterID
	if !requireReorderItems(w, req.Items) {
		return
	}

	resp, err := h.Lms.ReorderLessonsAdmin(r.Context(), &req)
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// ListLmsPrerequisites GET /api/lms/prerequisites
func (h *Handler) ListLmsPrerequisites(w http.ResponseWriter, r *http.Request) {
	resp, err := h.Lms.ListPrerequisitesAdmin(r.Context(), &lmspb.ListPrerequisitesRequest{
		TargetEntityType: lmspb.EntityType(parseEnumQuery(r, "target_entity_type")),
		TargetEntityUlid: r.URL.Query().Get("target_entity_id"),
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
	req.PrerequisiteUlid = newLmsID()
	if !validateLmsPrerequisitePayload(w, req.RequiredEntityType, req.RequiredEntityUlid, req.RequiredResult, req.TargetEntityType, req.TargetEntityUlid) {
		return
	}

	resp, err := h.Lms.CreatePrerequisiteAdmin(r.Context(), &req)
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// GetLmsPrerequisite GET /api/lms/prerequisites/{prerequisite_id}
func (h *Handler) GetLmsPrerequisite(w http.ResponseWriter, r *http.Request) {
	prerequisiteID, ok := requiredURLParam(w, r, "prerequisite_id")
	if !ok {
		return
	}
	resp, err := h.Lms.GetPrerequisiteDetailAdmin(r.Context(), &lmspb.GetPrerequisiteRequest{
		PrerequisiteUlid: prerequisiteID,
	})
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// UpdateLmsPrerequisite PUT /api/lms/prerequisites/{prerequisite_id}
func (h *Handler) UpdateLmsPrerequisite(w http.ResponseWriter, r *http.Request) {
	prerequisiteID, ok := requiredURLParam(w, r, "prerequisite_id")
	if !ok {
		return
	}
	var req lmspb.UpdatePrerequisiteRequest
	if err := ReadJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}
	req.PrerequisiteUlid = prerequisiteID
	if !validateLmsPrerequisitePayload(w, req.RequiredEntityType, req.RequiredEntityUlid, req.RequiredResult, req.TargetEntityType, req.TargetEntityUlid) || !requirePositiveVersion(w, req.Version) {
		return
	}

	resp, err := h.Lms.UpdatePrerequisiteAdmin(r.Context(), &req)
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// DeleteLmsPrerequisite DELETE /api/lms/prerequisites/{prerequisite_id}
func (h *Handler) DeleteLmsPrerequisite(w http.ResponseWriter, r *http.Request) {
	prerequisiteID, ok := requiredURLParam(w, r, "prerequisite_id")
	if !ok {
		return
	}
	version, err := readVersionParam(r)
	if err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid version")
		return
	}
	if !requirePositiveVersion(w, version) {
		return
	}

	resp, err := h.Lms.DeletePrerequisiteAdmin(r.Context(), &lmspb.DeletePrerequisiteRequest{
		PrerequisiteUlid: prerequisiteID,
		Version:          version,
	})
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// ListLmsQuizzes GET /api/lms/quizzes
func (h *Handler) ListLmsQuizzes(w http.ResponseWriter, r *http.Request) {
	resp, err := h.Lms.ListQuizzesAdmin(r.Context(), &lmspb.ListQuizzesRequest{
		QuizzableType: lmspb.QuizzableType(parseEnumQuery(r, "quizzable_type")),
		QuizzableUlid: r.URL.Query().Get("quizzable_id"),
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
	req.QuizUlid = newLmsID()
	if !requireRequestFields(w, req.Title, "title", req.QuizzableUlid, "quizzable_id") {
		return
	}
	if req.QuizzableType == lmspb.QuizzableType_QUIZZABLE_TYPE_UNSPECIFIED {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "quizzable_type is required")
		return
	}

	resp, err := h.Lms.CreateQuizAdmin(r.Context(), &req)
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// GetLmsQuiz GET /api/lms/quizzes/{quiz_id}
func (h *Handler) GetLmsQuiz(w http.ResponseWriter, r *http.Request) {
	quizID, ok := requiredURLParam(w, r, "quiz_id")
	if !ok {
		return
	}
	resp, err := h.Lms.GetQuizDetailAdmin(r.Context(), &lmspb.GetQuizDetailRequest{
		QuizUlid: quizID,
	})
	if err != nil {
		writeLmsError(w, err)
		return
	}
	payload := jsonPayloadObject(resp)
	if detail := resp.GetQuizDetail(); detail != nil {
		payload["quiz"] = detail.GetQuiz()
	}
	WriteJSON(w, http.StatusOK, payload)
}

// UpdateLmsQuiz PUT /api/lms/quizzes/{quiz_id}
func (h *Handler) UpdateLmsQuiz(w http.ResponseWriter, r *http.Request) {
	quizID, ok := requiredURLParam(w, r, "quiz_id")
	if !ok {
		return
	}
	var req lmspb.UpdateQuizRequest
	if err := ReadJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}
	req.QuizUlid = quizID
	if !requireRequestField(w, req.Title, "title") || !requirePositiveVersion(w, req.Version) {
		return
	}

	resp, err := h.Lms.UpdateQuizAdmin(r.Context(), &req)
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// DeleteLmsQuiz DELETE /api/lms/quizzes/{quiz_id}
func (h *Handler) DeleteLmsQuiz(w http.ResponseWriter, r *http.Request) {
	quizID, ok := requiredURLParam(w, r, "quiz_id")
	if !ok {
		return
	}
	version, err := readVersionParam(r)
	if err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid version")
		return
	}
	if !requirePositiveVersion(w, version) {
		return
	}

	resp, err := h.Lms.DeleteQuizAdmin(r.Context(), &lmspb.DeleteQuizRequest{
		QuizUlid: quizID,
		Version:  version,
	})
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// ListLmsQuizQuestions GET /api/lms/quizzes/{quiz_id}/questions
func (h *Handler) ListLmsQuizQuestions(w http.ResponseWriter, r *http.Request) {
	quizID, ok := requiredURLParam(w, r, "quiz_id")
	if !ok {
		return
	}
	resp, err := h.Lms.ListQuizQuestionsAdmin(r.Context(), &lmspb.ListQuizQuestionsRequest{
		QuizUlid: quizID,
	})
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// CreateLmsQuizQuestion POST /api/lms/quizzes/{quiz_id}/questions
func (h *Handler) CreateLmsQuizQuestion(w http.ResponseWriter, r *http.Request) {
	quizID, ok := requiredURLParam(w, r, "quiz_id")
	if !ok {
		return
	}
	var req lmspb.CreateQuizQuestionRequest
	if err := ReadJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}
	req.QuestionUlid = newLmsID()
	req.QuizUlid = quizID
	if strings.TrimSpace(req.MediaItemsJson) == "" {
		req.MediaItemsJson = "[]"
	}
	if !requireRequestField(w, req.QuestionText, "question_text") {
		return
	}
	if req.QuestionType == lmspb.QuizQuestionType_QUIZ_QUESTION_TYPE_UNSPECIFIED {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "question_type is required")
		return
	}

	resp, err := h.Lms.CreateQuizQuestionAdmin(r.Context(), &req)
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// GetLmsQuizQuestion GET /api/lms/questions/{question_id}
func (h *Handler) GetLmsQuizQuestion(w http.ResponseWriter, r *http.Request) {
	questionID, ok := requiredURLParam(w, r, "question_id")
	if !ok {
		return
	}
	resp, err := h.Lms.GetQuizQuestionDetailAdmin(r.Context(), &lmspb.GetQuizQuestionRequest{
		QuestionUlid: questionID,
	})
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// UpdateLmsQuizQuestion PUT /api/lms/questions/{question_id}
func (h *Handler) UpdateLmsQuizQuestion(w http.ResponseWriter, r *http.Request) {
	questionID, ok := requiredURLParam(w, r, "question_id")
	if !ok {
		return
	}
	var req lmspb.UpdateQuizQuestionRequest
	if err := ReadJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}
	req.QuestionUlid = questionID
	if strings.TrimSpace(req.MediaItemsJson) == "" {
		req.MediaItemsJson = "[]"
	}
	if !requireRequestField(w, req.QuestionText, "question_text") || !requirePositiveVersion(w, req.Version) {
		return
	}
	if req.QuestionType == lmspb.QuizQuestionType_QUIZ_QUESTION_TYPE_UNSPECIFIED {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "question_type is required")
		return
	}

	resp, err := h.Lms.UpdateQuizQuestionAdmin(r.Context(), &req)
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// DeleteLmsQuizQuestion DELETE /api/lms/questions/{question_id}
func (h *Handler) DeleteLmsQuizQuestion(w http.ResponseWriter, r *http.Request) {
	questionID, ok := requiredURLParam(w, r, "question_id")
	if !ok {
		return
	}
	version, err := readVersionParam(r)
	if err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid version")
		return
	}
	if !requirePositiveVersion(w, version) {
		return
	}

	resp, err := h.Lms.DeleteQuizQuestionAdmin(r.Context(), &lmspb.DeleteQuizQuestionRequest{
		QuestionUlid: questionID,
		Version:      version,
	})
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// ReorderLmsQuizQuestions POST /api/lms/quizzes/{quiz_id}/questions/reorder
func (h *Handler) ReorderLmsQuizQuestions(w http.ResponseWriter, r *http.Request) {
	quizID, ok := requiredURLParam(w, r, "quiz_id")
	if !ok {
		return
	}
	var req lmspb.ReorderQuizQuestionsRequest
	if err := ReadJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}
	req.QuizUlid = quizID
	if !requireReorderItems(w, req.Items) {
		return
	}

	resp, err := h.Lms.ReorderQuizQuestionsAdmin(r.Context(), &req)
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// ListLmsQuizOptions GET /api/lms/questions/{question_id}/options
func (h *Handler) ListLmsQuizOptions(w http.ResponseWriter, r *http.Request) {
	questionID, ok := requiredURLParam(w, r, "question_id")
	if !ok {
		return
	}
	resp, err := h.Lms.ListQuizOptionsAdmin(r.Context(), &lmspb.ListQuizOptionsRequest{
		QuestionUlid: questionID,
	})
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// CreateLmsQuizOption POST /api/lms/questions/{question_id}/options
func (h *Handler) CreateLmsQuizOption(w http.ResponseWriter, r *http.Request) {
	questionID, ok := requiredURLParam(w, r, "question_id")
	if !ok {
		return
	}
	var req lmspb.CreateQuizOptionRequest
	if err := ReadJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}
	req.OptionUlid = newLmsID()
	req.QuestionUlid = questionID
	if !requireRequestField(w, req.OptionText, "option_text") {
		return
	}

	resp, err := h.Lms.CreateQuizOptionAdmin(r.Context(), &req)
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// GetLmsQuizOption GET /api/lms/options/{option_id}
func (h *Handler) GetLmsQuizOption(w http.ResponseWriter, r *http.Request) {
	optionID, ok := requiredURLParam(w, r, "option_id")
	if !ok {
		return
	}
	resp, err := h.Lms.GetQuizOptionDetailAdmin(r.Context(), &lmspb.GetQuizOptionRequest{
		OptionUlid: optionID,
	})
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// UpdateLmsQuizOption PUT /api/lms/options/{option_id}
func (h *Handler) UpdateLmsQuizOption(w http.ResponseWriter, r *http.Request) {
	optionID, ok := requiredURLParam(w, r, "option_id")
	if !ok {
		return
	}
	var req lmspb.UpdateQuizOptionRequest
	if err := ReadJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}
	req.OptionUlid = optionID
	if !requireRequestField(w, req.OptionText, "option_text") || !requirePositiveVersion(w, req.Version) {
		return
	}

	resp, err := h.Lms.UpdateQuizOptionAdmin(r.Context(), &req)
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// DeleteLmsQuizOption DELETE /api/lms/options/{option_id}
func (h *Handler) DeleteLmsQuizOption(w http.ResponseWriter, r *http.Request) {
	optionID, ok := requiredURLParam(w, r, "option_id")
	if !ok {
		return
	}
	version, err := readVersionParam(r)
	if err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid version")
		return
	}
	if !requirePositiveVersion(w, version) {
		return
	}

	resp, err := h.Lms.DeleteQuizOptionAdmin(r.Context(), &lmspb.DeleteQuizOptionRequest{
		OptionUlid: optionID,
		Version:    version,
	})
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// ReorderLmsQuizOptions POST /api/lms/questions/{question_id}/options/reorder
func (h *Handler) ReorderLmsQuizOptions(w http.ResponseWriter, r *http.Request) {
	questionID, ok := requiredURLParam(w, r, "question_id")
	if !ok {
		return
	}
	var req lmspb.ReorderQuizOptionsRequest
	if err := ReadJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}
	req.QuestionUlid = questionID
	if !requireReorderItems(w, req.Items) {
		return
	}

	resp, err := h.Lms.ReorderQuizOptionsAdmin(r.Context(), &req)
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// ListLmsObjects GET /api/lms/objects
func (h *Handler) ListLmsObjects(w http.ResponseWriter, r *http.Request) {
	page := parseCursorPage(r, 20)
	resp, err := h.Lms.ListObjectsAdmin(r.Context(), &lmspb.ListObjectsRequest{
		Filters: &lmspb.ObjectFilters{
			Prefix: r.URL.Query().Get("prefix"),
		},
		PageSize: page.PageSize,
		SortOrder: lmspb.SortOrder(page.Sort),
		Cursor:   firstNonEmpty(r.URL.Query().Get("cursor"), r.URL.Query().Get("page_token")),
	})
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// ListLmsCourseAssets GET /api/lms/assets
func (h *Handler) ListLmsCourseAssets(w http.ResponseWriter, r *http.Request) {
	page := parseCursorPage(r, 20)
	resp, err := h.Lms.ListCourseAssetsAdmin(r.Context(), &lmspb.ListCourseAssetsRequest{
		Filters: &lmspb.CourseAssetFilters{
			Status:       r.URL.Query().Get("status"),
			AssetType:    r.URL.Query().Get("asset_type"),
			AssociatedId: r.URL.Query().Get("associated_id"),
		},
		PageSize: page.PageSize,
		SortOrder: lmspb.SortOrder(page.Sort),
		Cursor:   firstNonEmpty(r.URL.Query().Get("cursor"), r.URL.Query().Get("page_token")),
	})
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// GetLmsCourseAssetDetail GET /api/lms/assets/detail?object_key=...
func (h *Handler) GetLmsCourseAssetDetail(w http.ResponseWriter, r *http.Request) {
	objectKey := strings.TrimSpace(r.URL.Query().Get("object_key"))
	associatedID := strings.TrimSpace(r.URL.Query().Get("associated_id"))
	if !requireRequestFields(w, objectKey, "object_key", associatedID, "associated_id") {
		return
	}
	resp, err := h.Lms.GetCourseAssetDetailAdmin(r.Context(), &lmspb.GetCourseAssetDetailRequest{
		ObjectKey:    objectKey,
		AssociatedId: associatedID,
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
	if !validateLmsUploadURLPayload(w, &req) {
		return
	}

	resp, err := h.Lms.CreateUploadURLAdmin(r.Context(), &req)
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
	if !requireRequestField(w, req.ObjectKey, "object_key") {
		return
	}

	resp, err := h.Lms.CreateViewURLAdmin(r.Context(), &req)
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// ListLmsBrokenAssets GET /api/lms/broken-assets
func (h *Handler) ListLmsBrokenAssets(w http.ResponseWriter, r *http.Request) {
	page := parseCursorPage(r, 20)
	resp, err := h.Lms.ListBrokenAssetsAdmin(r.Context(), &lmspb.ListBrokenAssetsRequest{
		Filters: &lmspb.BrokenAssetFilters{
			AssetType: r.URL.Query().Get("asset_type"),
		},
		PageSize: page.PageSize,
		SortOrder: lmspb.SortOrder(page.Sort),
		Cursor:   firstNonEmpty(r.URL.Query().Get("cursor"), r.URL.Query().Get("page_token")),
	})
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// CleanUpDeprecatedCourseAssets POST /api/lms/courses/{course_id}/cleanup-assets
func (h *Handler) CleanUpDeprecatedCourseAssets(w http.ResponseWriter, r *http.Request) {
	courseID, ok := requiredURLParam(w, r, "course_id")
	if !ok {
		return
	}

	resp, err := h.Lms.CleanUpDeprecatedCourseAssetsAdmin(r.Context(), &lmspb.CleanUpDeprecatedCourseAssetsRequest{
		CourseId: courseID,
	})
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// CreateLmsCourseSupplementaryMaterial POST /api/lms/courses/{course_id}/supplementary-material
func (h *Handler) CreateLmsCourseSupplementaryMaterial(w http.ResponseWriter, r *http.Request) {
	courseID, ok := requiredURLParam(w, r, "course_id")
	if !ok {
		return
	}
	var req lmspb.CreateCourseSupplementaryMaterialRequest
	if err := ReadJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}
	if req.MaterialUlid == "" {
		req.MaterialUlid = newLmsID()
	}
	req.CourseUlid = courseID
	if !requireRequestFields(w, req.Kind, "kind", req.DataJson, "data_json") {
		return
	}

	resp, err := h.Lms.CreateCourseSupplementaryMaterialAdmin(r.Context(), &req)
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// GetLmsCourseSupplementaryMaterial GET /api/lms/courses/{course_id}/supplementary-material
func (h *Handler) GetLmsCourseSupplementaryMaterial(w http.ResponseWriter, r *http.Request) {
	courseID, ok := requiredURLParam(w, r, "course_id")
	if !ok {
		return
	}
	resp, err := h.Lms.GetCourseSupplementaryMaterialAdmin(r.Context(), &lmspb.GetCourseSupplementaryMaterialRequest{
		CourseUlid: courseID,
	})
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// UpdateLmsCourseSupplementaryMaterial PUT /api/lms/courses/{course_id}/supplementary-material/{material_id}
func (h *Handler) UpdateLmsCourseSupplementaryMaterial(w http.ResponseWriter, r *http.Request) {
	materialID, ok := requiredURLParam(w, r, "material_id")
	if !ok {
		return
	}
	var req lmspb.UpdateCourseSupplementaryMaterialRequest
	if err := ReadJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}
	req.MaterialUlid = materialID
	if !requireRequestFields(w, req.Kind, "kind", req.DataJson, "data_json") || !requirePositiveVersion(w, req.Version) {
		return
	}

	resp, err := h.Lms.UpdateCourseSupplementaryMaterialAdmin(r.Context(), &req)
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// DeleteLmsCourseSupplementaryMaterial DELETE /api/lms/courses/{course_id}/supplementary-material/{material_id}
func (h *Handler) DeleteLmsCourseSupplementaryMaterial(w http.ResponseWriter, r *http.Request) {
	materialID, ok := requiredURLParam(w, r, "material_id")
	if !ok {
		return
	}
	version, err := readVersionParam(r)
	if err != nil || !requirePositiveVersion(w, version) {
		if err != nil {
			WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid version")
		}
		return
	}

	resp, err := h.Lms.DeleteCourseSupplementaryMaterialAdmin(r.Context(), &lmspb.DeleteCourseSupplementaryMaterialRequest{
		MaterialUlid: materialID,
		Version:      version,
	})
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// GrantLmsCourseAccessPermission POST /api/lms/courses/{course_id}/permissions/grant
func (h *Handler) GrantLmsCourseAccessPermission(w http.ResponseWriter, r *http.Request) {
	courseID, ok := requiredURLParam(w, r, "course_id")
	if !ok {
		return
	}
	var req lmspb.GrantCourseAccessPermissionAdminRequest
	if err := ReadJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}
	req.CourseUlid = courseID
	req.OperatorUlid = AdminID(r)
	if !requireRequestFields(w, req.CandidateUlid, "candidate_id", req.BizUnit, "biz_unit") {
		return
	}
	if !requireRequestField(w, req.OperatorUlid, "operator_id") {
		return
	}

	resp, err := h.Lms.GrantCourseAccessPermissionAdmin(r.Context(), &req)
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// RevokeLmsCourseAccessPermission POST /api/lms/courses/{course_id}/permissions/revoke
func (h *Handler) RevokeLmsCourseAccessPermission(w http.ResponseWriter, r *http.Request) {
	courseID, ok := requiredURLParam(w, r, "course_id")
	if !ok {
		return
	}
	var req lmspb.RevokeCourseAccessPermissionAdminRequest
	if err := ReadJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}
	req.CourseUlid = courseID
	req.OperatorUlid = AdminID(r)
	if !requireRequestFields(w, req.CandidateUlid, "candidate_id", req.BizUnit, "biz_unit") {
		return
	}
	if !requireRequestField(w, req.OperatorUlid, "operator_id") {
		return
	}

	resp, err := h.Lms.RevokeCourseAccessPermissionAdmin(r.Context(), &req)
	if err != nil {
		writeLmsError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}
