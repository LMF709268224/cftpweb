package handler

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	gccpb "github.com/afnandelfin620-star/cftptest/cftp/gcc"
	lmspb "github.com/afnandelfin620-star/cftptest/cftp/glms"
	gprog "github.com/afnandelfin620-star/cftptest/cftp/gprog"
	"github.com/go-chi/chi/v5"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const pdfPreviewStreamTimeout = 30 * time.Minute

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
		PipelineUlid: pipelineUlid,
		Limit:        100,
		Offset:       0,
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
		Total: resp.GetTotal(),
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
			Query: &gccpb.GetPipelineRequest_PipelineId{PipelineId: summary.PipelineCcUlid},
		}); configErr == nil {
			summary.PipelineName = strings.TrimSpace(config.GetName())
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
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, map[string]string{"view_url": resp.GetViewUrl()})
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
			CandidateId: CandidateID(r),
			CourseId:    courseID,
		})
		if err == nil {
			if t := strings.TrimSpace(summaryResp.GetCourse().GetTitle()); t != "" {
				title = t
			}
		}

		resp, err := h.Lms.ListCourseMaterials(r.Context(), &lmspb.ListCourseMaterialsCandidateRequest{
			CandidateId: CandidateID(r),
			CourseId:    courseID,
		})
		if err != nil {
			slog.Warn("failed to list candidate course materials", "error", err, "course_id", courseID)
			continue
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
		CandidateId: CandidateID(r),
		MaterialId:  materialID,
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
	if !requireRequestFields(w, material.GetCourseId(), "course_id", material.GetFileObjectKey(), "file_object_key") {
		return
	}

	courseIDs, err := h.candidateCourseIDs(r, candidateID)
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	if !slices.Contains(courseIDs, material.GetCourseId()) {
		WriteError(w, http.StatusForbidden, ErrForbidden, "material is not available for current candidate")
		return
	}

	viewResp, err := h.Lms.CreateViewURL(r.Context(), &lmspb.CreateViewURLCandidateRequest{
		CandidateId: CandidateID(r),
		ObjectKey:   material.GetFileObjectKey(),
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
		CandidateId: CandidateID(r),
		CourseId:    courseID,
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	completeCourse := resp.GetCompleteCourse()

	if len(completeCourse.GetMaterials()) == 0 {
		matResp, err := h.Lms.ListCourseMaterials(r.Context(), &lmspb.ListCourseMaterialsCandidateRequest{
			CandidateId: candidateID,
			CourseId:    courseID,
		})
		if err == nil && matResp != nil {
			var materials []*lmspb.CourseMaterial
			for _, summary := range matResp.GetMaterials() {
				materials = append(materials, &lmspb.CourseMaterial{
					MaterialId:    summary.GetMaterialId(),
					CourseId:      summary.GetCourseId(),
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

	if completeCourse.GetSupplementaryMaterial() == nil {
		suppResp, err := h.Lms.GetCourseSupplementaryMaterialAdmin(r.Context(), &lmspb.GetCourseSupplementaryMaterialRequest{
			CourseId: courseID,
		})
		if err == nil && suppResp != nil && suppResp.GetMaterial() != nil {
			completeCourse.SupplementaryMaterial = suppResp.GetMaterial()
		} else if err != nil {
			slog.Warn("failed to load candidate course supplementary material", "error", err, "course_id", courseID)
		}
	}

	quizProgress := h.quizProgressByCourse(r, candidateID, completeCourse)
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
		CandidateId: candidateID,
		LessonId:    lessonID,
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
		CandidateId: candidateID,
		Status:      status,
		PageSize:    pageSize,
		PageToken:   pageToken,
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
		CandidateId: CandidateID(r),
		LessonId:    lessonID,
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) GetLessonURL(w http.ResponseWriter, r *http.Request) {
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

	params := url.Values{}
	params.Set("lessonId", lessonID)
	if title := strings.TrimSpace(lesson.GetTitle()); title != "" {
		params.Set("title", title)
	}

	WriteJSON(w, http.StatusOK, GetAccessURLRsp{
		URL:       "/pdf-preview?" + params.Encode(),
		ExpiresAt: viewResp.GetExpiresAt(),
	})
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

	expiresAt := time.Now().Add(time.Hour * 24).Unix()
	filename := sanitizeFilename(lesson.GetTitle()) + ".pdf"
	token := h.signPDFPreviewToken(candidateID, "lesson", lessonID, viewURL, filename, expiresAt)
	params := url.Values{}
	params.Set("token", token)
	previewURL := "/api/public/pdf-preview/lessons/" + url.PathEscape(lessonID) + "?" + params.Encode()

	WriteJSON(w, http.StatusOK, GetAccessURLRsp{
		URL:       previewURL,
		ExpiresAt: firstNonEmpty(viewResp.GetExpiresAt(), time.Unix(expiresAt, 0).UTC().Format(time.RFC3339)),
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

	expiresAt := time.Now().Add(time.Hour).Unix()
	token := h.signPDFPreviewToken(candidateID, "resource", resourceURL, resourceURL, "resource.pdf", expiresAt)
	params := url.Values{}
	params.Set("src", resourceURL)
	params.Set("token", token)

	WriteJSON(w, http.StatusOK, GetAccessURLRsp{
		URL:       "/api/public/pdf-preview/resource?" + params.Encode(),
		ExpiresAt: time.Unix(expiresAt, 0).UTC().Format(time.RFC3339),
	})
}

func (h *Handler) PreviewLessonPDF(w http.ResponseWriter, r *http.Request) {
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
	if strings.TrimSpace(viewResp.GetViewUrl()) == "" {
		WriteError(w, http.StatusBadGateway, ErrServiceUnavailable, "empty view url")
		return
	}

	method := http.MethodGet
	if r.Method == http.MethodHead {
		method = http.MethodHead
	}
	proxyReq, err := http.NewRequestWithContext(r.Context(), method, viewResp.GetViewUrl(), nil)
	if err != nil {
		WriteError(w, http.StatusBadGateway, ErrServiceUnavailable, "invalid view url")
		return
	}
	if rangeHeader := strings.TrimSpace(r.Header.Get("Range")); rangeHeader != "" {
		proxyReq.Header.Set("Range", rangeHeader)
	}
	if ifRange := strings.TrimSpace(r.Header.Get("If-Range")); ifRange != "" {
		proxyReq.Header.Set("If-Range", ifRange)
	}

	proxyResp, err := http.DefaultClient.Do(proxyReq)
	if err != nil {
		WriteError(w, http.StatusBadGateway, ErrServiceUnavailable, "failed to open lesson preview")
		return
	}
	defer proxyResp.Body.Close()

	copyPreviewHeaders(w.Header(), proxyResp.Header)
	if w.Header().Get("Accept-Ranges") == "" {
		w.Header().Set("Accept-Ranges", "bytes")
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`inline; filename="%s"`, sanitizeFilename(lesson.GetTitle())+".pdf"))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(proxyResp.StatusCode)

	if r.Method == http.MethodHead {
		return
	}
	extendPDFPreviewWriteDeadline(w)
	if _, err := io.Copy(w, proxyResp.Body); err != nil {
		slog.Warn("failed to stream lesson preview", "error", err, "lesson_id", lessonID)
	}
}

func (h *Handler) PreviewLessonPDFPublic(w http.ResponseWriter, r *http.Request) {
	lessonID := strings.TrimSpace(chi.URLParam(r, "lessonId"))
	token := strings.TrimSpace(r.URL.Query().Get("token"))
	preview, ok := h.verifyPDFPreviewToken(token, "lesson", lessonID)
	if !ok || !requireRequestFields(w, preview.CandidateID, "candidate_id", lessonID, "lesson_id") {
		WriteError(w, http.StatusUnauthorized, ErrUnauthorized, "invalid or expired preview token")
		return
	}

	if !isValidPreviewResourceURL(preview.SourceURL) {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid preview url")
		return
	}

	h.streamPDFPreview(w, r, preview.SourceURL, preview.Filename, "lesson", lessonID)
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

	method := http.MethodGet
	if r.Method == http.MethodHead {
		method = http.MethodHead
	}
	proxyReq, err := http.NewRequestWithContext(r.Context(), method, resourceURL, nil)
	if err != nil {
		WriteError(w, http.StatusBadGateway, ErrServiceUnavailable, "invalid resource url")
		return
	}
	if rangeHeader := strings.TrimSpace(r.Header.Get("Range")); rangeHeader != "" {
		proxyReq.Header.Set("Range", rangeHeader)
	}
	if ifRange := strings.TrimSpace(r.Header.Get("If-Range")); ifRange != "" {
		proxyReq.Header.Set("If-Range", ifRange)
	}

	proxyResp, err := http.DefaultClient.Do(proxyReq)
	if err != nil {
		WriteError(w, http.StatusBadGateway, ErrServiceUnavailable, "failed to open resource preview")
		return
	}
	defer proxyResp.Body.Close()

	copyPreviewHeaders(w.Header(), proxyResp.Header)
	if w.Header().Get("Accept-Ranges") == "" {
		w.Header().Set("Accept-Ranges", "bytes")
	}
	contentType := strings.TrimSpace(proxyResp.Header.Get("Content-Type"))
	if contentType == "" || contentType == "application/octet-stream" {
		contentType = "application/pdf"
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", `inline; filename="resource.pdf"`)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(proxyResp.StatusCode)

	if r.Method == http.MethodHead {
		return
	}
	extendPDFPreviewWriteDeadline(w)
	if _, err := io.Copy(w, proxyResp.Body); err != nil {
		slog.Warn("failed to stream resource preview", "error", err, "resource_url", resourceURL)
	}
}

func (h *Handler) PreviewResourceURLPublic(w http.ResponseWriter, r *http.Request) {
	resourceURL := strings.TrimSpace(r.URL.Query().Get("src"))
	token := strings.TrimSpace(r.URL.Query().Get("token"))
	preview, ok := h.verifyPDFPreviewToken(token, "resource", resourceURL)
	if !ok || !requireRequestFields(w, preview.CandidateID, "candidate_id", resourceURL, "src") {
		WriteError(w, http.StatusUnauthorized, ErrUnauthorized, "invalid or expired preview token")
		return
	}
	if !isValidPreviewResourceURL(preview.SourceURL) {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid resource url")
		return
	}

	h.streamPDFPreview(w, r, preview.SourceURL, preview.Filename, "resource", resourceURL)
}

func (h *Handler) PreviewResourcePackFilePDFPublic(w http.ResponseWriter, r *http.Request) {
	fileID := strings.TrimSpace(chi.URLParam(r, "fileId"))
	token := strings.TrimSpace(r.URL.Query().Get("token"))
	preview, ok := h.verifyPDFPreviewToken(token, "resource-pack-file", fileID)
	if !ok || !requireRequestFields(w, preview.CandidateID, "candidate_id", fileID, "file_id") {
		WriteError(w, http.StatusUnauthorized, ErrUnauthorized, "invalid or expired preview token")
		return
	}
	if !isValidPreviewResourceURL(preview.SourceURL) {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid preview url")
		return
	}

	h.streamPDFPreview(w, r, preview.SourceURL, preview.Filename, "resource-pack-file", fileID)
}

func (h *Handler) streamPDFPreview(w http.ResponseWriter, r *http.Request, sourceURL string, filename string, logKind string, logID string) {
	method := http.MethodGet
	if r.Method == http.MethodHead {
		method = http.MethodHead
	}
	proxyReq, err := http.NewRequestWithContext(r.Context(), method, sourceURL, nil)
	if err != nil {
		WriteError(w, http.StatusBadGateway, ErrServiceUnavailable, "invalid preview url")
		return
	}
	if rangeHeader := strings.TrimSpace(r.Header.Get("Range")); rangeHeader != "" {
		proxyReq.Header.Set("Range", rangeHeader)
	}
	if ifRange := strings.TrimSpace(r.Header.Get("If-Range")); ifRange != "" {
		proxyReq.Header.Set("If-Range", ifRange)
	}

	proxyResp, err := http.DefaultClient.Do(proxyReq)
	if err != nil {
		WriteError(w, http.StatusBadGateway, ErrServiceUnavailable, "failed to open preview")
		return
	}
	defer proxyResp.Body.Close()

	if proxyResp.StatusCode < 200 || proxyResp.StatusCode >= 300 {
		slog.Warn("preview upstream returned non-2xx", "kind", logKind, "id", logID, "status", proxyResp.StatusCode)
		WriteError(w, http.StatusBadGateway, ErrServiceUnavailable, "failed to open preview")
		return
	}

	copyPreviewHeaders(w.Header(), proxyResp.Header)
	if w.Header().Get("Accept-Ranges") == "" {
		w.Header().Set("Accept-Ranges", "bytes")
	}
	contentType := strings.TrimSpace(proxyResp.Header.Get("Content-Type"))
	if contentType == "" || contentType == "application/octet-stream" {
		contentType = "application/pdf"
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`inline; filename="%s"`, sanitizeFilename(filename)))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(proxyResp.StatusCode)

	if r.Method == http.MethodHead {
		return
	}
	extendPDFPreviewWriteDeadline(w)
	if _, err := io.Copy(w, proxyResp.Body); err != nil {
		slog.Warn("failed to stream preview", "error", err, "kind", logKind, "id", logID)
	}
}

func extendPDFPreviewWriteDeadline(w http.ResponseWriter) {
	if err := http.NewResponseController(w).SetWriteDeadline(time.Now().Add(pdfPreviewStreamTimeout)); err != nil {
		slog.Warn("failed to extend pdf preview write deadline", "error", err)
	}
}

type pdfPreviewClaims struct {
	CandidateID string
	SourceURL   string
	Filename    string
}

func (h *Handler) signPDFPreviewToken(candidateID string, resourceKind string, resourceID string, sourceURL string, filename string, expiresAt int64) string {
	payload := strings.Join([]string{candidateID, resourceKind, resourceID, sourceURL, filename, strconv.FormatInt(expiresAt, 10)}, "\n")
	mac := hmac.New(sha256.New, h.pdfPreviewSigningKey())
	_, _ = mac.Write([]byte(payload))
	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return base64.RawURLEncoding.EncodeToString([]byte(payload + "\n" + signature))
}

func (h *Handler) verifyPDFPreviewToken(token string, resourceKind string, resourceID string) (pdfPreviewClaims, bool) {
	decoded, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return pdfPreviewClaims{}, false
	}
	parts := strings.Split(string(decoded), "\n")
	if len(parts) != 7 {
		return pdfPreviewClaims{}, false
	}
	candidateID := parts[0]
	if parts[1] != resourceKind || parts[2] != resourceID {
		return pdfPreviewClaims{}, false
	}
	expiresAt, err := strconv.ParseInt(parts[5], 10, 64)
	if err != nil || time.Now().Unix() > expiresAt {
		return pdfPreviewClaims{}, false
	}

	payload := strings.Join(parts[:6], "\n")
	mac := hmac.New(sha256.New, h.pdfPreviewSigningKey())
	_, _ = mac.Write([]byte(payload))
	expected := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(parts[6])) {
		return pdfPreviewClaims{}, false
	}
	return pdfPreviewClaims{
		CandidateID: candidateID,
		SourceURL:   parts[3],
		Filename:    parts[4],
	}, candidateID != "" && parts[3] != ""
}

func (h *Handler) pdfPreviewSigningKey() []byte {
	key := strings.TrimSpace(h.CasdoorClientSecret)
	if key == "" {
		key = strings.TrimSpace(h.CasdoorClientId)
	}
	if key == "" {
		key = "candidate-pdf-preview"
	}
	return []byte(key)
}

func isValidPreviewResourceURL(resourceURL string) bool {
	parsed, err := url.Parse(resourceURL)
	return err == nil && parsed != nil && (parsed.Scheme == "http" || parsed.Scheme == "https") && parsed.Host != ""
}

func (h *Handler) lessonViewURL(ctx context.Context, candidateID, lessonID string) (*lmspb.CreateViewURLResponse, *lmspb.Lesson, error) {
	lessonResp, err := h.Lms.GetLessonDetail(ctx, &lmspb.GetLessonDetailCandidateRequest{
		CandidateId: candidateID,
		LessonId:    lessonID,
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
		CandidateId: candidateID,
		ObjectKey:   lesson.GetMediaObjectKey(),
	})
	if err != nil {
		return nil, nil, err
	}
	return viewResp, lesson, nil
}

func copyPreviewHeaders(dst, src http.Header) {
	for _, key := range []string{"Accept-Ranges", "Content-Length", "Content-Range", "Last-Modified", "ETag"} {
		if value := src.Get(key); value != "" {
			dst.Set(key, value)
		}
	}
}

func (h *Handler) GetCandidateEnrollmentDetail(w http.ResponseWriter, r *http.Request) {
	candidateID := CandidateID(r)
	enrollmentID := strings.TrimSpace(chi.URLParam(r, "enrollmentId"))
	if !requireRequestFields(w, candidateID, "candidate_id", enrollmentID, "enrollment_id") {
		return
	}

	resp, err := h.Lms.GetCandidateEnrollmentDetail(r.Context(), &lmspb.GetCandidateEnrollmentDetailRequest{
		CandidateId:  candidateID,
		EnrollmentId: enrollmentID,
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) findEnrollmentIdByCourse(ctx context.Context, candidateID, courseID string) (string, error) {
	resp, err := h.Lms.ListCandidateEnrollments(ctx, &lmspb.ListCandidateEnrollmentsRequest{
		CandidateId: candidateID,
		PageSize:    1000,
	})
	if err != nil {
		return "", err
	}
	for _, e := range resp.GetEnrollments() {
		if e.GetCourseId() == courseID {
			return e.GetEnrollmentId(), nil
		}
	}
	return "", fmt.Errorf("enrollment not found")
}

func (h *Handler) SyncCourseProgress(w http.ResponseWriter, r *http.Request) {
	candidateID := CandidateID(r)
	courseID := strings.TrimSpace(chi.URLParam(r, "courseId"))
	if !requireRequestFields(w, candidateID, "candidate_id", courseID, "course_id") {
		return
	}

	enrollmentID, err := h.findEnrollmentIdByCourse(r.Context(), candidateID, courseID)
	if err != nil {
		WriteJSON(w, http.StatusOK, SyncCourseProgressRsp{
			Success:            true,
			CourseStatus:       "learning",
			ProgressPercentage: 0,
		})
		return
	}

	resp, err := h.Lms.GetCandidateEnrollmentDetail(r.Context(), &lmspb.GetCandidateEnrollmentDetailRequest{
		CandidateId:  candidateID,
		EnrollmentId: enrollmentID,
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
		resp, err := h.Lms.CompleteLessonLearning(r.Context(), &lmspb.CompleteLessonLearningRequest{
			CandidateId: candidateID,
			LessonId:    materialID,
		})
		if err != nil {
			rejected++
			continue
		}
		_ = resp
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

	resp, err := h.Lms.ListCandidateEnrollments(r.Context(), &lmspb.ListCandidateEnrollmentsRequest{
		CandidateId: candidateID,
		PageSize:    1000,
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	var records []ProgressRecord
	targetLessonID := strings.TrimSpace(r.URL.Query().Get("lessonId"))

	for _, e := range resp.GetEnrollments() {
		detail, err := h.Lms.GetCandidateEnrollmentDetail(r.Context(), &lmspb.GetCandidateEnrollmentDetailRequest{
			CandidateId:  candidateID,
			EnrollmentId: e.GetEnrollmentId(),
		})
		if err != nil {
			continue
		}

		for _, lessonId := range detail.GetCompletedLessonIds() {
			if targetLessonID != "" && lessonId != targetLessonID {
				continue
			}
			records = append(records, ProgressRecord{
				CandidateId:     candidateID,
				MaterialId:      lessonId,
				CoursePackageId: e.GetCourseId(),
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
		Status:           p.Status,
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
			courseID := strings.TrimSpace(unit.GetGlmsCourseId())
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
	resp, err := h.Lms.ListCandidateEnrollments(r.Context(), &lmspb.ListCandidateEnrollmentsRequest{
		CandidateId: candidateID,
		PageSize:    200,
	})
	if err != nil {
		return nil, err
	}

	out := make(map[string]uint32, len(resp.GetEnrollments()))
	for _, enrollment := range resp.GetEnrollments() {
		if enrollment == nil {
			continue
		}
		courseID := strings.TrimSpace(enrollment.GetCourseId())
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
			Query: &gccpb.GetPipelineRequest_PipelineId{PipelineId: pipelineID},
		})
		if err != nil {
			slog.Warn("failed to get candidate pipeline config", "error", err, "pipeline_id", pipelineID)
			continue
		}
		for _, stage := range config.GetStages() {
			for _, unit := range stage.GetUnits() {
				courseID := strings.TrimSpace(unit.GetGlmsCourseId())
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
		ID:          material.GetMaterialId(),
		CourseID:    material.GetCourseId(),
		CourseTitle: courseTitle,
		Title:       material.GetTitle(),
		Type:        int32(material.GetMaterialType()),
		FileKey:     material.GetFileObjectKey(),
		FileSize:    int64(material.GetFileSize()),
		FileHash:    material.GetFileHash(),
	}
}

func (h *Handler) quizProgressByCourse(r *http.Request, candidateID string, course *lmspb.CompleteCourse) map[string]QuizProgressItem {
	quizIDs := collectCourseQuizIDs(course)
	out := make(map[string]QuizProgressItem, len(quizIDs))
	for _, quizID := range quizIDs {
		item := QuizProgressItem{QuizID: quizID}
		resp, err := h.Lms.ListQuizAttemptsAdmin(r.Context(), &lmspb.ListQuizAttemptsRequest{
			QuizId:   quizID,
			UserId:   candidateID,
			PageSize: 20,
		})
		if err != nil {
			slog.Warn("failed to list candidate quiz attempts", "error", err, "candidate_id", candidateID, "quiz_id", quizID)
			out[quizID] = item
			continue
		}
		for _, attempt := range resp.GetAttempts() {
			if attempt == nil {
				continue
			}
			if item.AttemptID == "" {
				item.AttemptID = attempt.GetAttemptId()
				item.Status = attempt.GetStatus()
			}
			if attempt.GetIsPassed() {
				item.AttemptID = attempt.GetAttemptId()
				item.Status = attempt.GetStatus()
				item.IsPassed = true
				break
			}
		}
		out[quizID] = item
	}
	return out
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
		quizID := strings.TrimSpace(detail.GetQuiz().GetQuizId())
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
