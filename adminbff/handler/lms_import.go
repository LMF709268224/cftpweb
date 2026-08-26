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
	Title                string                  `json:"title"`
	CourseGPath          string                  `json:"course_gpath"`
	Description          string                  `json:"description,omitempty"`
	CategoryTips         string                  `json:"category_tips,omitempty"`
	DurationMin          uint32                  `json:"duration_min,omitempty"`
	CertificationEnabled bool                    `json:"certification_enabled,omitempty"`
	CertificationDefULID string                  `json:"certification_def_ulid,omitempty"`
	ThumbnailObjectKey   string                  `json:"thumbnail_object_key,omitempty"`
	ThumbnailFileHash    string                  `json:"thumbnail_file_hash,omitempty"`
	Chapters             []importChapterJSON     `json:"chapters"`
	Quizzes              []importChapterQuizJSON `json:"quizzes,omitempty"`
}

type importChapterJSON struct {
	Title     string             `json:"title"`
	SortOrder uint32             `json:"sort_order,omitempty"`
	Lessons   []importLessonJSON `json:"lessons"`
}

type importLessonJSON struct {
	Title                  string `json:"title"`
	SortOrder              uint32 `json:"sort_order,omitempty"`
	LessonType             any    `json:"lesson_type"`
	Body                   string `json:"body"`
	MediaObjectKey         string `json:"media_object_key"`
	MediaFileHash          string `json:"media_file_hash"`
	ExternalURL            string `json:"external_url"`
	ExternalCoursewareULID string `json:"external_courseware_ulid"`
	VideoProvider          string `json:"video_provider"`
	VideoStreamUID         string `json:"video_stream_uid"`
	VideoEmbedCode         string `json:"video_embed_code"`
	MetaJSON               string `json:"meta_json"`
}

// importChapterQuizJSON is the optional package extension used by adminweb.
// GLMS imports the course tree and quizzes through separate RPCs, so the BFF
// resolves chapter_title to the newly-created chapter ULID after the course
// import succeeds.
type importChapterQuizJSON struct {
	ChapterTitle       string                   `json:"chapter_title"`
	Title              string                   `json:"title"`
	Description        string                   `json:"description"`
	PassingScore       uint32                   `json:"passing_score"`
	TimeLimit          uint32                   `json:"time_limit"`
	RandomizeQuestions bool                     `json:"randomize_questions"`
	QuizType           any                      `json:"quiz_type"`
	Questions          []importQuizQuestionJSON `json:"questions"`
}

type importQuizQuestionJSON struct {
	QuestionText   string                 `json:"question_text"`
	QuestionType   any                    `json:"question_type"`
	Points         uint32                 `json:"points"`
	SortOrder      uint32                 `json:"sort_order"`
	IsRequired     bool                   `json:"is_required"`
	Explanation    string                 `json:"explanation"`
	MediaItemsJSON string                 `json:"media_items_json"`
	Options        []importQuizOptionJSON `json:"options"`
}

type importQuizOptionJSON struct {
	OptionText string `json:"option_text"`
	IsCorrect  bool   `json:"is_correct"`
	SortOrder  uint32 `json:"sort_order"`
}

type importedChapterQuizResult struct {
	ChapterTitle string                    `json:"chapter_title"`
	Quiz         *lmspb.ImportQuizResponse `json:"quiz"`
}

type importedCoursePackageResponse struct {
	Course  *lmspb.ImportCourseResponse `json:"course"`
	Quizzes []importedChapterQuizResult `json:"quizzes"`
}

type importedChapterResponse struct {
	ChapterULID string `json:"chapter_ulid"`
	LessonCount int    `json:"lesson_count"`
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

// ImportLmsChapter POST /api/lms/courses/{course_id}/chapters/import
func (h *Handler) ImportLmsChapter(w http.ResponseWriter, r *http.Request) {
	courseID, ok := requiredURLParam(w, r, "course_id")
	if !ok {
		return
	}

	var chapter importChapterJSON
	if err := ReadJSON(r, &chapter); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}
	if chapter.SortOrder == 0 {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "chapter.sort_order is required")
		return
	}
	if !validateImportChapter(w, chapter, "chapter") {
		return
	}

	chapterID := newLmsID()
	chapterResp, err := h.Lms.CreateChapterAdmin(r.Context(), &lmspb.CreateChapterRequest{
		ChapterUlid: chapterID,
		CourseUlid:  courseID,
		Title:       strings.TrimSpace(chapter.Title),
		SortOrder:   chapter.SortOrder,
	})
	if err != nil {
		writeLmsError(w, err)
		return
	}
	chapterID = chapterResp.GetChapterUlid()

	for lessonIndex, lesson := range chapter.Lessons {
		lessonReq := buildImportLessonRequest(chapterID, lesson, uint32(lessonIndex+1))
		if _, err := h.Lms.CreateLessonAdmin(r.Context(), lessonReq); err != nil {
			writeLmsError(w, err)
			return
		}
	}

	WriteJSON(w, http.StatusOK, importedChapterResponse{
		ChapterULID: chapterID,
		LessonCount: len(chapter.Lessons),
	})
}

func (h *Handler) importLmsCourse(w http.ResponseWriter, r *http.Request, req importLmsContentRequest) {
	courseJSON := strings.TrimSpace(req.CourseJSON)
	if !requireRequestField(w, courseJSON, "course_json") {
		return
	}
	if !requireValidJSONString(w, courseJSON, "course_json") {
		return
	}
	var packageJSON importCourseJSON
	if err := json.Unmarshal([]byte(courseJSON), &packageJSON); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "course_json is invalid")
		return
	}
	if !validateImportCoursePackage(w, packageJSON) {
		return
	}

	// Forward the complete GLMS course document. Only the package wrapper and
	// BFF-managed quizzes are removed; filtering to a hand-written field subset
	// would silently discard valid microservice fields such as course_gpath.
	coursePayload, err := buildImportCoursePayload(courseJSON)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, ErrInternal, "failed to prepare course import")
		return
	}

	resp, err := h.Lms.ImportCourseAdmin(r.Context(), &lmspb.ImportCourseRequest{
		CategoryTips: firstNonEmpty(strings.TrimSpace(req.CategoryTips), strings.TrimSpace(packageJSON.CategoryTips)),
		CourseJson:   string(coursePayload),
	})
	if err != nil {
		writeLmsError(w, err)
		return
	}

	if len(packageJSON.Quizzes) == 0 {
		WriteJSON(w, http.StatusOK, resp)
		return
	}

	chapters, err := h.Lms.ListChaptersAdmin(r.Context(), &lmspb.ListChaptersRequest{CourseUlid: resp.GetCourseUlid()})
	if err != nil {
		writeLmsError(w, err)
		return
	}
	chapterIDs := make(map[string]string, len(chapters.GetChapters()))
	for _, chapter := range chapters.GetChapters() {
		if chapter != nil {
			chapterIDs[strings.TrimSpace(chapter.GetTitle())] = chapter.GetChapterUlid()
		}
	}
	results := make([]importedChapterQuizResult, 0, len(packageJSON.Quizzes))
	for _, quiz := range packageJSON.Quizzes {
		chapterID := chapterIDs[strings.TrimSpace(quiz.ChapterTitle)]
		if chapterID == "" {
			WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "quiz chapter_title does not match an imported chapter: "+quiz.ChapterTitle)
			return
		}
		quizPayload, err := json.Marshal(struct {
			Title              string                   `json:"title"`
			Description        string                   `json:"description,omitempty"`
			PassingScore       uint32                   `json:"passing_score"`
			TimeLimit          uint32                   `json:"time_limit"`
			RandomizeQuestions bool                     `json:"randomize_questions"`
			QuizType           any                      `json:"quiz_type"`
			Questions          []importQuizQuestionJSON `json:"questions"`
		}{quiz.Title, quiz.Description, quiz.PassingScore, quiz.TimeLimit, quiz.RandomizeQuestions, quiz.QuizType, quiz.Questions})
		if err != nil {
			WriteError(w, http.StatusInternalServerError, ErrInternal, "failed to prepare quiz import")
			return
		}
		quizResp, err := h.Lms.ImportQuizAdmin(r.Context(), &lmspb.ImportQuizRequest{
			QuizzableType: lmspb.QuizzableType_QUIZZABLE_TYPE_CHAPTER,
			QuizzableUlid: chapterID,
			QuizJson:      string(quizPayload),
		})
		if err != nil {
			writeLmsError(w, err)
			return
		}
		results = append(results, importedChapterQuizResult{ChapterTitle: quiz.ChapterTitle, Quiz: quizResp})
	}
	WriteJSON(w, http.StatusOK, importedCoursePackageResponse{Course: resp, Quizzes: results})
}

func buildImportCoursePayload(courseJSON string) ([]byte, error) {
	var payload map[string]json.RawMessage
	if err := json.Unmarshal([]byte(courseJSON), &payload); err != nil {
		return nil, err
	}
	delete(payload, "package_type")
	delete(payload, "package_version")
	delete(payload, "category_tips")
	delete(payload, "quizzes")
	return json.Marshal(payload)
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

	resp, err := h.Lms.ImportQuizAdmin(r.Context(), &lmspb.ImportQuizRequest{
		QuizzableType: lmspb.QuizzableType(req.QuizzableType),
		QuizzableUlid: strings.TrimSpace(req.QuizzableID),
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
	return validateImportCoursePackage(w, course)
}

func validateImportCoursePackage(w http.ResponseWriter, course importCourseJSON) bool {
	if strings.TrimSpace(course.Title) == "" {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "course_json.title is required")
		return false
	}
	if strings.TrimSpace(course.CourseGPath) == "" {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "course_json.course_gpath is required")
		return false
	}
	for chapterIndex, chapter := range course.Chapters {
		chapterPath := fmt.Sprintf("course_json.chapters[%d]", chapterIndex)
		if !validateImportChapter(w, chapter, chapterPath) {
			return false
		}
	}
	for quizIndex, quiz := range course.Quizzes {
		quizPath := fmt.Sprintf("course_json.quizzes[%d]", quizIndex)
		if strings.TrimSpace(quiz.ChapterTitle) == "" || strings.TrimSpace(quiz.Title) == "" {
			WriteError(w, http.StatusBadRequest, ErrInvalidRequest, quizPath+".chapter_title and title are required")
			return false
		}
		if quiz.PassingScore > 100 {
			WriteError(w, http.StatusBadRequest, ErrInvalidRequest, quizPath+".passing_score must be between 0 and 100")
			return false
		}
		for questionIndex, question := range quiz.Questions {
			questionPath := fmt.Sprintf("%s.questions[%d]", quizPath, questionIndex)
			if strings.TrimSpace(question.QuestionText) == "" || len(question.Options) == 0 {
				WriteError(w, http.StatusBadRequest, ErrInvalidRequest, questionPath+".question_text and options are required")
				return false
			}
			for optionIndex, option := range question.Options {
				if strings.TrimSpace(option.OptionText) == "" {
					WriteError(w, http.StatusBadRequest, ErrInvalidRequest, fmt.Sprintf("%s.options[%d].option_text is required", questionPath, optionIndex))
					return false
				}
			}
		}
	}
	return true
}

func validateImportChapter(w http.ResponseWriter, chapter importChapterJSON, chapterPath string) bool {
	if strings.TrimSpace(chapter.Title) == "" {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, chapterPath+".title is required")
		return false
	}
	if len(chapter.Lessons) == 0 {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, chapterPath+".lessons must not be empty")
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
	case "token_link":
		if strings.TrimSpace(lesson.ExternalCoursewareULID) == "" {
			WriteError(w, http.StatusBadRequest, ErrInvalidRequest, lessonPath+".external_courseware_ulid is required")
			return false
		}
	case "video", "pdf", "image", "audio", "file":
		if strings.TrimSpace(lesson.MediaObjectKey) == "" && strings.TrimSpace(lesson.VideoStreamUID) == "" {
			WriteError(w, http.StatusBadRequest, ErrInvalidRequest, lessonPath+".media_object_key or video_stream_uid is required")
			return false
		}
		if strings.TrimSpace(lesson.VideoStreamUID) != "" && strings.TrimSpace(lesson.VideoProvider) == "" {
			WriteError(w, http.StatusBadRequest, ErrInvalidRequest, lessonPath+".video_provider is required when video_stream_uid is provided")
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
		case int(lmspb.LessonType_LESSON_TYPE_TOKEN_LINK):
			return "token_link"
		}
	}
	return ""
}

func buildImportLessonRequest(chapterID string, lesson importLessonJSON, fallbackSortOrder uint32) *lmspb.CreateLessonRequest {
	lessonType := lmspb.LessonType_LESSON_TYPE_UNSPECIFIED
	switch normalizeImportLessonType(lesson.LessonType) {
	case "video":
		lessonType = lmspb.LessonType_LESSON_TYPE_VIDEO
	case "text":
		lessonType = lmspb.LessonType_LESSON_TYPE_TEXT
	case "pdf":
		lessonType = lmspb.LessonType_LESSON_TYPE_PDF
	case "image":
		lessonType = lmspb.LessonType_LESSON_TYPE_IMAGE
	case "audio":
		lessonType = lmspb.LessonType_LESSON_TYPE_AUDIO
	case "file":
		lessonType = lmspb.LessonType_LESSON_TYPE_FILE
	case "link":
		lessonType = lmspb.LessonType_LESSON_TYPE_LINK
	case "token_link":
		lessonType = lmspb.LessonType_LESSON_TYPE_TOKEN_LINK
	}

	sortOrder := lesson.SortOrder
	if sortOrder == 0 {
		sortOrder = fallbackSortOrder
	}
	metaJSON := strings.TrimSpace(lesson.MetaJSON)
	if metaJSON == "" {
		metaJSON = "{}"
	}

	return &lmspb.CreateLessonRequest{
		LessonUlid:             newLmsID(),
		ChapterUlid:            chapterID,
		Title:                  strings.TrimSpace(lesson.Title),
		SortOrder:              sortOrder,
		LessonType:             lessonType,
		Body:                   lesson.Body,
		MediaObjectKey:         strings.TrimSpace(lesson.MediaObjectKey),
		MediaFileHash:          strings.TrimSpace(lesson.MediaFileHash),
		ExternalUrl:            strings.TrimSpace(lesson.ExternalURL),
		ExternalCoursewareUlid: strings.TrimSpace(lesson.ExternalCoursewareULID),
		VideoProvider:          strings.TrimSpace(lesson.VideoProvider),
		VideoStreamUid:         strings.TrimSpace(lesson.VideoStreamUID),
		VideoEmbedCode:         strings.TrimSpace(lesson.VideoEmbedCode),
		MetaJson:               metaJSON,
	}
}
