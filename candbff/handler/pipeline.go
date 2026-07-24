package handler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"

	gccpb "github.com/afnandelfin620-star/cftptest/cftp/gcc"
	lmspb "github.com/afnandelfin620-star/cftptest/cftp/glms"
	gprog "github.com/afnandelfin620-star/cftptest/cftp/gprog"
	"github.com/go-chi/chi/v5"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var errCandidateEnrollmentNotFound = errors.New("candidate enrollment not found")

// GetPipelineTimeline GET /api/mall/pipelines/{pipelineId}/timeline
func (h *Handler) GetPipelineTimeline(w http.ResponseWriter, r *http.Request) {
	candidateID := CandidateID(r)
	pipelineID := strings.TrimSpace(chi.URLParam(r, "pipelineId"))
	if !requireRequestFields(w, candidateID, "candidate_id", pipelineID, "pipeline_id") {
		return
	}

	candidatePipelines, err := h.Gprog.ListCandidatePipelines(r.Context(), &gprog.ListCandidatePipelinesReq{
		CandidateUlid: candidateID,
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	pipelineUlid := ""
	for _, pipeline := range candidatePipelines.GetPipelines() {
		if pipeline != nil && pipeline.GetPipelineCcUlid() == pipelineID {
			pipelineUlid = pipeline.GetPipelineUlid()
			break
		}
	}
	if strings.TrimSpace(pipelineUlid) == "" {
		WriteError(w, http.StatusNotFound, ErrNotFound, "pipeline instance not found")
		return
	}

	resp, err := h.Gprog.ListStatusTransitionLogs(r.Context(), &gprog.ListStatusTransitionLogsReq{
		Filters: &gprog.StatusTransitionLogFilters{
			PipelineUlid: pipelineUlid,
		},
		PageSize: 100,
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	logs := make([]StatusTransitionLogSummary, 0, len(resp.GetLogs()))
	for _, logItem := range resp.GetLogs() {
		if logItem == nil {
			continue
		}
		logs = append(logs, StatusTransitionLogSummary{
			TransitionUlid: logItem.GetTransitionUlid(),
			EntityType:     logItem.GetEntityType(),
			EntityUlid:     logItem.GetEntityUlid(),
			FromStatus:     normalizeProgTimelineStatus(logItem.GetEntityType(), logItem.GetFromStatus()),
			ToStatus:       normalizeProgTimelineStatus(logItem.GetEntityType(), logItem.GetToStatus()),
			ReasonCode:     logItem.GetReasonCode(),
			ReasonMessage:  logItem.GetReasonMessage(),
			TriggerSource:  logItem.GetTriggerSource(),
			EventType:      logItem.GetEventType(),
			CreatedAt:      logItem.GetCreatedAt(),
		})
	}

	WriteJSON(w, http.StatusOK, PipelineTimelineRsp{
		Logs:  logs,
		Total: int32(len(logs)),
	})
}

func normalizeProgTimelineStatus(entityType, statusText string) string {
	statusText = strings.TrimSpace(statusText)
	if statusText == "" {
		return ""
	}

	var (
		value int32
		ok    bool
	)
	switch strings.ToUpper(strings.TrimSpace(entityType)) {
	case "PIPELINE":
		value, ok = gprog.PipelineStatus_value[statusText]
		if !ok {
			value, ok = gprog.PipelineStatus_value["PIPELINE_STATUS_"+statusText]
		}
	case "STAGE":
		value, ok = gprog.StageStatus_value[statusText]
		if !ok {
			value, ok = gprog.StageStatus_value["STAGE_STATUS_"+statusText]
		}
	case "COURSE_UNIT":
		value, ok = gprog.CourseUnitStatus_value[statusText]
		if !ok {
			value, ok = gprog.CourseUnitStatus_value["COURSE_UNIT_STATUS_"+statusText]
		}
	}
	if ok {
		return strconv.Itoa(int(value))
	}
	return statusText
}

func (h *Handler) ListMyPipelines(w http.ResponseWriter, r *http.Request) {
	candidateID := CandidateID(r)

	resp, err := h.Gprog.ListCandidatePipelines(r.Context(), &gprog.ListCandidatePipelinesReq{
		CandidateUlid: candidateID,
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	out := ListMyPipelinesRsp{
		List: make([]PipelineSummary, 0, len(resp.GetPipelines())),
	}

	enrollmentProgress, err := h.candidateEnrollmentProgressByCourse(r, candidateID)
	if err != nil {
		slog.Warn("failed to load candidate enrollment progress", "error", err, "candidate_id", candidateID)
	}

	for _, p := range resp.GetPipelines() {
		summary := toPipelineSummary(p)
		if config, configErr := h.Gcc.GetPipeline(r.Context(), &gccpb.GetPipelineRequest{
			Query: &gccpb.GetPipelineRequest_PipelineUlid{PipelineUlid: summary.PipelineCcUlid},
		}); configErr == nil {
			summary.PipelineName = strings.TrimSpace(config.GetName())
			summary.Description = strings.TrimSpace(config.GetDescription())
			if progress, ok := pipelineProgressFromCourseEnrollments(config, enrollmentProgress); ok {
				summary.ProgressAvailable = true
				summary.Progress = progress
				summary.LmsProgress = uint32(progress + 0.5)
			}
			if strings.TrimSpace(summary.CurrentStageUlid) != "" {
				if runtimeResp, runtimeErr := h.Gprog.GetPipelineDetail(r.Context(), &gprog.GetPipelineDetailReq{
					PipelineUlid: summary.PipelineUlid,
				}); runtimeErr == nil {
					summary.CurrentStageName = currentStageNameFromRuntime(config, runtimeResp, summary.CurrentStageUlid)
				}
			}
		} else {
			slog.Warn("failed to load candidate pipeline config for display", "error", configErr, "pipeline_cc_ulid", summary.PipelineCcUlid)
		}
		out.List = append(out.List, summary)
	}

	WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) GetPipelineCertificateViewURL(w http.ResponseWriter, r *http.Request) {
	candidateID := CandidateID(r)
	pipelineULID := strings.TrimSpace(chi.URLParam(r, "pipelineUlid"))
	if !requireRequestFields(w, candidateID, "candidate_id", pipelineULID, "pipeline_ulid") {
		return
	}

	resp, err := h.Gprog.GetPipelineCertificateViewURL(r.Context(), &gprog.GetPipelineCertificateViewURLReq{
		CandidateUlid: candidateID,
		PipelineUlid:  pipelineULID,
	})
	if err == nil {
		WriteJSON(w, http.StatusOK, map[string]string{"view_url": strings.TrimSpace(resp.GetViewUrl())})
		return
	}

	HandleGrpcError(w, err)
}

func (h *Handler) ListMaterials(w http.ResponseWriter, r *http.Request) {
	candidateID := CandidateID(r)
	if !requireRequestField(w, candidateID, "candidate_id") {
		return
	}

	courseIDs, err := h.candidateCourseIDs(r, candidateID)
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	out := MaterialListRsp{
		Materials: make([]MaterialListItem, 0),
	}
	for _, courseID := range courseIDs {
		title := courseID
		summaryResp, err := h.Lms.GetCourseSummary(r.Context(), &lmspb.GetCourseSummaryCandidateRequest{
			CandidateUlid: CandidateID(r),
			CourseUlid:    courseID,
		})
		if err == nil {
			if t := strings.TrimSpace(summaryResp.GetCourse().GetTitle()); t != "" {
				title = t
			}
		}

		resp, err := h.Lms.ListCourseMaterials(r.Context(), &lmspb.ListCourseMaterialsCandidateRequest{
			CandidateUlid: CandidateID(r),
			CourseUlid:    courseID,
		})
		if err != nil {
			HandleGrpcError(w, err)
			return
		}
		for _, material := range resp.GetMaterials() {
			out.Materials = append(out.Materials, materialSummaryToListItem(material, title))
		}
	}

	WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) GetAccessURL(w http.ResponseWriter, r *http.Request) {
	candidateID := CandidateID(r)
	materialID := strings.TrimSpace(chi.URLParam(r, "materialId"))
	if !requireRequestFields(w, candidateID, "candidate_id", materialID, "material_id") {
		return
	}

	materialResp, err := h.Lms.GetCourseMaterial(r.Context(), &lmspb.GetCourseMaterialCandidateRequest{
		CandidateUlid: CandidateID(r),
		MaterialUlid:  materialID,
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	material := materialResp.GetMaterial()
	if material == nil {
		WriteError(w, http.StatusNotFound, ErrNotFound, "material not found")
		return
	}
	if !requireRequestFields(w, material.GetCourseUlid(), "course_id", material.GetFileObjectKey(), "file_object_key") {
		return
	}

	courseIDs, err := h.candidateCourseIDs(r, candidateID)
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	if !slices.Contains(courseIDs, material.GetCourseUlid()) {
		WriteError(w, http.StatusForbidden, ErrForbidden, "material is not available for current candidate")
		return
	}

	viewResp, err := h.Lms.CreateViewURL(r.Context(), &lmspb.CreateViewURLCandidateRequest{
		CandidateUlid: CandidateID(r),
		ObjectKey:     material.GetFileObjectKey(),
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, GetAccessURLRsp{
		URL:       viewResp.GetViewUrl(),
		ExpiresAt: viewResp.GetExpiresAt(),
	})
}

func (h *Handler) GetPipelineCourse(w http.ResponseWriter, r *http.Request) {
	candidateID := CandidateID(r)
	courseID := strings.TrimSpace(chi.URLParam(r, "courseId"))
	if !requireRequestFields(w, candidateID, "candidate_id", courseID, "course_id") {
		return
	}

	courseIDs, err := h.candidateCourseIDs(r, candidateID)
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	if !slices.Contains(courseIDs, courseID) {
		WriteError(w, http.StatusForbidden, ErrForbidden, "course is not available for current candidate")
		return
	}

	resp, err := h.Lms.GetCompleteCourse(r.Context(), &lmspb.GetCompleteCourseCandidateRequest{
		CandidateUlid: CandidateID(r),
		CourseUlid:    courseID,
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	completeCourse := resp.GetCompleteCourse()

	if len(completeCourse.GetMaterials()) == 0 {
		matResp, err := h.Lms.ListCourseMaterials(r.Context(), &lmspb.ListCourseMaterialsCandidateRequest{
			CandidateUlid: candidateID,
			CourseUlid:    courseID,
		})
		if err != nil {
			HandleGrpcError(w, err)
			return
		}
		if matResp != nil {
			var materials []*lmspb.CourseMaterial
			for _, summary := range matResp.GetMaterials() {
				materials = append(materials, &lmspb.CourseMaterial{
					MaterialUlid:  summary.GetMaterialUlid(),
					CourseUlid:    summary.GetCourseUlid(),
					Title:         summary.GetTitle(),
					MaterialType:  summary.GetMaterialType(),
					FileObjectKey: summary.GetFileObjectKey(),
					FileSize:      summary.GetFileSize(),
					SortOrder:     summary.GetSortOrder(),
					Version:       summary.GetVersion(),
					CreatedAt:     summary.GetCreatedAt(),
					UpdatedAt:     summary.GetUpdatedAt(),
					FileHash:      summary.GetFileHash(),
				})
			}
			completeCourse.Materials = materials
		}
	}

	quizProgress, err := h.quizProgressByCourse(r.Context(), candidateID, completeCourse)
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, PipelineCourseRsp{
		CompleteCourse: completeCourse,
		QuizProgress:   quizProgress,
	})
}

func (h *Handler) CompletePipelineLesson(w http.ResponseWriter, r *http.Request) {
	candidateID := CandidateID(r)
	lessonID := strings.TrimSpace(chi.URLParam(r, "lessonId"))
	if !requireRequestFields(w, candidateID, "candidate_id", lessonID, "lesson_id") {
		return
	}

	resp, err := h.Lms.CompleteLessonLearning(r.Context(), &lmspb.CompleteLessonLearningRequest{
		CandidateUlid: candidateID,
		LessonUlid:    lessonID,
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) ListCandidateEnrollments(w http.ResponseWriter, r *http.Request) {
	candidateID := CandidateID(r)
	if !requireRequestField(w, candidateID, "candidate_id") {
		return
	}

	status := r.URL.Query().Get("status")
	pageSizeStr := r.URL.Query().Get("pageSize")
	pageToken := r.URL.Query().Get("pageToken")

	pageSize := uint32(20)
	if pageSizeStr != "" {
		if ps, err := strconv.ParseUint(pageSizeStr, 10, 32); err == nil && ps > 0 {
			pageSize = uint32(ps)
		}
	}

	resp, err := h.Lms.ListCandidateEnrollments(r.Context(), &lmspb.ListCandidateEnrollmentsRequest{
		Filters: &lmspb.CandidateEnrollmentFilters{
			CandidateUlid: candidateID,
			Status:        status,
		},
		PageSize: pageSize,
		Cursor:   pageToken,
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) GetPipelineLessonDetail(w http.ResponseWriter, r *http.Request) {
	lessonID := strings.TrimSpace(chi.URLParam(r, "lessonId"))
	if !requireRequestField(w, lessonID, "lesson_id") {
		return
	}

	resp, err := h.Lms.GetLessonDetail(r.Context(), &lmspb.GetLessonDetailCandidateRequest{
		CandidateUlid: CandidateID(r),
		LessonUlid:    lessonID,
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) GetLessonPreviewURL(w http.ResponseWriter, r *http.Request) {
	candidateID := CandidateID(r)
	lessonID := strings.TrimSpace(chi.URLParam(r, "lessonId"))
	if !requireRequestFields(w, candidateID, "candidate_id", lessonID, "lesson_id") {
		return
	}

	viewResp, lesson, err := h.lessonViewURL(r.Context(), candidateID, lessonID)
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	viewURL := strings.TrimSpace(viewResp.GetViewUrl())
	if viewURL == "" {
		WriteError(w, http.StatusBadGateway, ErrServiceUnavailable, "empty view url")
		return
	}

	WriteJSON(w, http.StatusOK, GetAccessURLRsp{
		URL:       viewURL,
		ExpiresAt: viewResp.GetExpiresAt(),
		Title:     lesson.GetTitle(),
	})
}

func (h *Handler) GetResourcePreviewURL(w http.ResponseWriter, r *http.Request) {
	candidateID := CandidateID(r)
	resourceURL := strings.TrimSpace(r.URL.Query().Get("src"))
	if !requireRequestFields(w, candidateID, "candidate_id", resourceURL, "src") {
		return
	}
	if !isValidPreviewResourceURL(resourceURL) {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid resource url")
		return
	}

	WriteJSON(w, http.StatusOK, GetAccessURLRsp{
		URL: resourceURL,
	})
}

func (h *Handler) PreviewLessonPDF(w http.ResponseWriter, r *http.Request) {
	candidateID := CandidateID(r)
	lessonID := strings.TrimSpace(chi.URLParam(r, "lessonId"))
	if !requireRequestFields(w, candidateID, "candidate_id", lessonID, "lesson_id") {
		return
	}

	viewResp, _, err := h.lessonViewURL(r.Context(), candidateID, lessonID)
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	if strings.TrimSpace(viewResp.GetViewUrl()) == "" {
		WriteError(w, http.StatusBadGateway, ErrServiceUnavailable, "empty view url")
		return
	}
	redirectPreview(w, r, viewResp.GetViewUrl())
}

func (h *Handler) PreviewResourceURL(w http.ResponseWriter, r *http.Request) {
	candidateID := CandidateID(r)
	resourceURL := strings.TrimSpace(r.URL.Query().Get("src"))
	if !requireRequestFields(w, candidateID, "candidate_id", resourceURL, "src") {
		return
	}

	if !isValidPreviewResourceURL(resourceURL) {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid resource url")
		return
	}

	redirectPreview(w, r, resourceURL)
}

func redirectPreview(w http.ResponseWriter, r *http.Request, sourceURL string) {
	sourceURL = strings.TrimSpace(sourceURL)
	if sourceURL == "" {
		WriteError(w, http.StatusBadGateway, ErrServiceUnavailable, "empty preview url")
		return
	}
	if r.Method == http.MethodHead {
		w.Header().Set("Location", sourceURL)
		w.WriteHeader(http.StatusTemporaryRedirect)
		return
	}
	http.Redirect(w, r, sourceURL, http.StatusTemporaryRedirect)
}

func isValidPreviewResourceURL(resourceURL string) bool {
	parsed, err := url.Parse(resourceURL)
	if err != nil || parsed == nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return false
	}
	hostname := parsed.Hostname()
	if hostname == "localhost" || hostname == "127.0.0.1" || hostname == "::1" {
		return false
	}
	ips, err := net.LookupIP(hostname)
	if err != nil {
		return false
	}
	for _, ip := range ips {
		if ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
			return false
		}
	}
	return true
}

func (h *Handler) lessonViewURL(ctx context.Context, candidateID, lessonID string) (*lmspb.CreateViewURLResponse, *lmspb.Lesson, error) {
	lessonResp, err := h.Lms.GetLessonDetail(ctx, &lmspb.GetLessonDetailCandidateRequest{
		CandidateUlid: candidateID,
		LessonUlid:    lessonID,
	})
	if err != nil {
		return nil, nil, err
	}
	lesson := lessonResp.GetLesson()
	if lesson == nil {
		return nil, nil, status.Error(codes.NotFound, "lesson not found")
	}
	if lesson.GetMediaObjectKey() == "" {
		return nil, nil, status.Error(codes.InvalidArgument, "lesson has no media object key")
	}

	viewResp, err := h.Lms.CreateViewURL(ctx, &lmspb.CreateViewURLCandidateRequest{
		CandidateUlid: candidateID,
		ObjectKey:     lesson.GetMediaObjectKey(),
	})
	if err != nil {
		return nil, nil, err
	}
	return viewResp, lesson, nil
}

func (h *Handler) GetCandidateEnrollmentDetail(w http.ResponseWriter, r *http.Request) {
	candidateID := CandidateID(r)
	enrollmentID := strings.TrimSpace(chi.URLParam(r, "enrollmentId"))
	if !requireRequestFields(w, candidateID, "candidate_id", enrollmentID, "enrollment_id") {
		return
	}

	resp, err := h.Lms.GetCandidateEnrollmentDetail(r.Context(), &lmspb.GetCandidateEnrollmentDetailRequest{
		CandidateUlid: candidateID,
		EnrollmentId:  enrollmentID,
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) findEnrollmentIdByCourse(ctx context.Context, candidateID, courseID string) (string, error) {
	enrollments, err := h.listCandidateEnrollments(ctx, candidateID)
	if err != nil {
		return "", err
	}
	for _, e := range enrollments {
		if e.GetCourseUlid() == courseID && strings.TrimSpace(e.GetEnrollmentId()) != "" {
			return strings.TrimSpace(e.GetEnrollmentId()), nil
		}
	}
	return "", errCandidateEnrollmentNotFound
}

func (h *Handler) SyncCourseProgress(w http.ResponseWriter, r *http.Request) {
	candidateID := CandidateID(r)
	courseID := strings.TrimSpace(chi.URLParam(r, "courseId"))
	if !requireRequestFields(w, candidateID, "candidate_id", courseID, "course_id") {
		return
	}

	enrollmentID, err := h.findEnrollmentIdByCourse(r.Context(), candidateID, courseID)
	if err != nil {
		if !errors.Is(err, errCandidateEnrollmentNotFound) {
			HandleGrpcError(w, err)
			return
		}
		WriteJSON(w, http.StatusOK, SyncCourseProgressRsp{
			Success:            true,
			CourseStatus:       "learning",
			ProgressPercentage: 0,
		})
		return
	}

	resp, err := h.Lms.GetCandidateEnrollmentDetail(r.Context(), &lmspb.GetCandidateEnrollmentDetailRequest{
		CandidateUlid: candidateID,
		EnrollmentId:  enrollmentID,
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, SyncCourseProgressRsp{
		Success:               true,
		CourseStatus:          resp.GetStatus(),
		ProgressPercentage:    resp.GetProgressPercentage(),
		CompletedLessonsCount: resp.GetCompletedLessons(),
		PassedQuizzesCount:    resp.GetPassedQuizzes(),
	})
}

func (h *Handler) ReportProgress(w http.ResponseWriter, r *http.Request) {
	candidateID := CandidateID(r)
	if !requireRequestField(w, candidateID, "candidate_id") {
		return
	}

	var input ReportProgressInput
	if err := ReadJSON(r, &input); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid request body: "+err.Error())
		return
	}
	if len(input.Records) == 0 {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "records is required")
		return
	}

	accepted := int32(0)
	rejected := int32(0)
	for _, record := range input.Records {
		materialID := strings.TrimSpace(record.MaterialID)
		if materialID == "" {
			rejected++
			continue
		}
		_, err := h.Lms.CompleteLessonLearning(r.Context(), &lmspb.CompleteLessonLearningRequest{
			CandidateUlid: candidateID,
			LessonUlid:    materialID,
		})
		if err != nil {
			rejected++
			continue
		}
		accepted++
	}

	WriteJSON(w, http.StatusOK, ReportProgressRsp{
		AcceptedCount: accepted,
		RejectedCount: rejected,
	})
}

func (h *Handler) GetProgress(w http.ResponseWriter, r *http.Request) {
	candidateID := CandidateID(r)
	if !requireRequestField(w, candidateID, "candidate_id") {
		return
	}

	enrollments, err := h.listCandidateEnrollments(r.Context(), candidateID)
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	var records []ProgressRecord
	targetLessonID := strings.TrimSpace(r.URL.Query().Get("lessonId"))

	for _, e := range enrollments {
		detail, err := h.Lms.GetCandidateEnrollmentDetail(r.Context(), &lmspb.GetCandidateEnrollmentDetailRequest{
			CandidateUlid: candidateID,
			EnrollmentId:  e.GetEnrollmentId(),
		})
		if err != nil {
			HandleGrpcError(w, err)
			return
		}

		for _, lessonId := range detail.GetCompletedLessonIds() {
			if targetLessonID != "" && lessonId != targetLessonID {
				continue
			}
			records = append(records, ProgressRecord{
				CandidateUlid:   candidateID,
				MaterialUlid:    lessonId,
				CoursePackageId: e.GetCourseUlid(),
				ProgressType:    "completed",
				ProgressValue:   100,
			})
		}
	}

	if records == nil {
		records = []ProgressRecord{}
	}

	WriteJSON(w, http.StatusOK, GetProgressRsp{Records: records})
}

func toPipelineSummary(p *gprog.PipelineSummary) PipelineSummary {
	if p == nil {
		return PipelineSummary{}
	}
	return PipelineSummary{
		PipelineUlid:     p.PipelineUlid,
		CandidateUlid:    p.CandidateUlid,
		PipelineCcUlid:   p.PipelineCcUlid,
		Status:           p.Status.String(),
		CurrentStageUlid: p.CurrentStageUlid,
		LmsProgress:      0,
		StartedAt:        p.StartedAt,
		CompletedAt:      p.CompletedAt,
		CreatedAt:        p.CreatedAt,
	}
}

func pipelineProgressFromCourseEnrollments(config *gccpb.PipelineConfig, enrollmentProgress map[string]uint32) (float64, bool) {
	if config == nil || len(enrollmentProgress) == 0 {
		return 0, false
	}

	var total float64
	var count int
	for _, stage := range config.GetStages() {
		if stage == nil {
			continue
		}
		for _, unit := range stage.GetUnits() {
			if unit == nil {
				continue
			}
			courseID := strings.TrimSpace(unit.GetGlmsCourseUlid())
			if courseID == "" {
				continue
			}
			progress, ok := enrollmentProgress[courseID]
			if !ok {
				continue
			}
			total += float64(progress)
			count++
		}
	}
	if count == 0 {
		return 0, false
	}

	return total / float64(count), true
}

func currentStageNameFromRuntime(config *gccpb.PipelineConfig, runtime *gprog.GetPipelineDetailRsp, currentStageUlid string) string {
	if config == nil || runtime == nil {
		return ""
	}
	currentStageUlid = strings.TrimSpace(currentStageUlid)
	if currentStageUlid == "" {
		return ""
	}
	for _, stage := range runtime.GetStages() {
		if stage == nil || stage.GetStage() == nil {
			continue
		}
		if strings.TrimSpace(stage.GetStage().GetStageUlid()) != currentStageUlid {
			continue
		}
		return stageConfigNameByID(config, stage.GetStage().GetStageCcUlid())
	}
	return ""
}

func (h *Handler) candidateEnrollmentProgressByCourse(r *http.Request, candidateID string) (map[string]uint32, error) {
	enrollments, err := h.listCandidateEnrollments(r.Context(), candidateID)
	if err != nil {
		return nil, err
	}

	out := make(map[string]uint32, len(enrollments))
	for _, enrollment := range enrollments {
		if enrollment == nil {
			continue
		}
		courseID := strings.TrimSpace(enrollment.GetCourseUlid())
		if courseID == "" {
			continue
		}
		progress := enrollment.GetProgressPercentage()
		if current, ok := out[courseID]; !ok || progress > current {
			out[courseID] = progress
		}
	}

	return out, nil
}

func (h *Handler) listCandidateEnrollments(ctx context.Context, candidateID string) ([]*lmspb.CandidateEnrollmentSummary, error) {
	const pageSize uint32 = 100
	enrollments := make([]*lmspb.CandidateEnrollmentSummary, 0)
	cursor := ""
	seen := make(map[string]struct{})

	for page := 0; page < 1000; page++ {
		resp, err := h.Lms.ListCandidateEnrollments(ctx, &lmspb.ListCandidateEnrollmentsRequest{
			Filters: &lmspb.CandidateEnrollmentFilters{
				CandidateUlid: candidateID,
			},
			Cursor:   cursor,
			PageSize: pageSize,
		})
		if err != nil {
			return nil, err
		}
		enrollments = append(enrollments, resp.GetEnrollments()...)
		if !resp.GetHasMore() {
			return enrollments, nil
		}
		nextCursor := strings.TrimSpace(resp.GetNextCursor())
		if nextCursor == "" || nextCursor == cursor {
			return nil, fmt.Errorf("candidate enrollment cursor did not advance")
		}
		if _, ok := seen[nextCursor]; ok {
			return nil, fmt.Errorf("candidate enrollment cursor loop detected")
		}
		seen[nextCursor] = struct{}{}
		cursor = nextCursor
	}

	return nil, fmt.Errorf("candidate enrollment pagination exceeded max pages")
}

func (h *Handler) candidateCourseIDs(r *http.Request, candidateID string) ([]string, error) {
	resp, err := h.Gprog.ListCandidatePipelines(r.Context(), &gprog.ListCandidatePipelinesReq{
		CandidateUlid: candidateID,
	})
	if err != nil {
		return nil, err
	}

	courseIDs := make([]string, 0)
	seen := make(map[string]struct{})
	for _, pipeline := range resp.GetPipelines() {
		pipelineID := strings.TrimSpace(pipeline.GetPipelineCcUlid())
		if pipelineID == "" {
			continue
		}
		config, err := h.Gcc.GetPipeline(r.Context(), &gccpb.GetPipelineRequest{
			Query: &gccpb.GetPipelineRequest_PipelineUlid{PipelineUlid: pipelineID},
		})
		if err != nil {
			return nil, fmt.Errorf("get candidate pipeline config %q: %w", pipelineID, err)
		}
		for _, stage := range config.GetStages() {
			for _, unit := range stage.GetUnits() {
				courseID := strings.TrimSpace(unit.GetGlmsCourseUlid())
				if courseID == "" {
					continue
				}
				if _, ok := seen[courseID]; ok {
					continue
				}
				seen[courseID] = struct{}{}
				courseIDs = append(courseIDs, courseID)
			}
		}
	}

	return courseIDs, nil
}

func materialSummaryToListItem(material *lmspb.CourseMaterialSummary, courseTitle string) MaterialListItem {
	if material == nil {
		return MaterialListItem{}
	}
	return MaterialListItem{
		ID:          material.GetMaterialUlid(),
		CourseID:    material.GetCourseUlid(),
		CourseTitle: courseTitle,
		Title:       material.GetTitle(),
		Type:        int32(material.GetMaterialType()),
		FileKey:     material.GetFileObjectKey(),
		FileSize:    int64(material.GetFileSize()),
		FileHash:    material.GetFileHash(),
	}
}

func (h *Handler) quizProgressByCourse(ctx context.Context, candidateID string, course *lmspb.CompleteCourse) (map[string]QuizProgressItem, error) {
	quizIDs := collectCourseQuizIDs(course)
	out := make(map[string]QuizProgressItem, len(quizIDs))
	for _, quizID := range quizIDs {
		item := QuizProgressItem{QuizID: quizID}
		resp, err := h.Lms.ListQuizAttemptsCandidate(ctx, &lmspb.ListQuizAttemptsCandidateRequest{
			Filters: &lmspb.QuizAttemptsCandidateFilters{
				QuizUlid:      quizID,
				CandidateUlid: candidateID,
			},
			PageSize: 20,
		})
		if err != nil {
			return nil, fmt.Errorf("list candidate quiz attempts for quiz %q: %w", quizID, err)
		}
		for _, attempt := range resp.GetAttempts() {
			if attempt == nil {
				continue
			}
			if item.AttemptID == "" {
				item.AttemptID = attempt.GetAttemptId()
				item.Status = attempt.GetStatus()
			}
			if attempt.GetPassStatus() == lmspb.QuizPassStatus_QUIZ_PASS_STATUS_PASSED {
				item.AttemptID = attempt.GetAttemptId()
				item.Status = attempt.GetStatus()
				item.IsPassed = true
				break
			}
		}
		out[quizID] = item
	}
	return out, nil
}

func collectCourseQuizIDs(course *lmspb.CompleteCourse) []string {
	if course == nil {
		return nil
	}
	seen := make(map[string]struct{})
	ids := make([]string, 0)
	addQuiz := func(detail *lmspb.QuizDetail) {
		if detail == nil || detail.GetQuiz() == nil {
			return
		}
		quizID := strings.TrimSpace(detail.GetQuiz().GetQuizUlid())
		if quizID == "" {
			return
		}
		if _, ok := seen[quizID]; ok {
			return
		}
		seen[quizID] = struct{}{}
		ids = append(ids, quizID)
	}
	for _, quiz := range course.GetQuizzes() {
		addQuiz(quiz)
	}
	for _, chapter := range course.GetChapters() {
		if chapter == nil {
			continue
		}
		for _, quiz := range chapter.GetQuizzes() {
			addQuiz(quiz)
		}
		for _, lesson := range chapter.GetLessons() {
			if lesson == nil {
				continue
			}
			for _, quiz := range lesson.GetQuizzes() {
				addQuiz(quiz)
			}
		}
	}
	return ids
}
