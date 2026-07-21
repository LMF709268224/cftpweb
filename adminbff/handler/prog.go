package handler

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	gprogpb "github.com/afnandelfin620-star/cftptest/cftp/gprog"
	"github.com/go-chi/chi/v5"
)

func (h *Handler) ListProgPipelines(w http.ResponseWriter, r *http.Request) {
	page := parseCursorPage(r, 20)
	filters := &gprogpb.PipelineFilters{
		CandidateUlid:  strings.TrimSpace(r.URL.Query().Get("candidate_ulid")),
		PipelineCcUlid: strings.TrimSpace(r.URL.Query().Get("pipeline_cc_ulid")),
		Status:         gprogpb.PipelineStatus(parseEnumQuery(r, "status")),
	}

	total, err := countCursorAll(r.Context(), func(ctx context.Context, cursor string, limit uint32) (uint32, string, error) {
		resp, err := h.Gprog.GetPipelineCount(ctx, &gprogpb.GetPipelineCountRequest{
			Filters:   filters,
			Limit:     int32(limit),
			Cursor:    cursor,
			SortOrder: gprogpb.SortOrder(page.Sort),
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

	req := &gprogpb.ListPipelinesReq{
		Filters:   filters,
		Cursor:    page.Cursor,
		PageSize:  int32(page.PageSize),
		SortOrder: gprogpb.SortOrder(page.Sort),
	}

	resp, err := h.Gprog.ListPipelines(r.Context(), req)
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	items := make([]map[string]interface{}, 0, len(resp.GetPipelines()))
	for _, pipeline := range resp.GetPipelines() {
		if pipeline == nil {
			continue
		}
		item := jsonPayloadObject(pipeline)
		h.attachCandidateName(item, pipeline.GetCandidateUlid())
		items = append(items, item)
	}

	WriteJSON(w, http.StatusOK, map[string]interface{}{
		"pipelines":   items,
		"total":       total.Total,
		"total_label": total.Label(),
		"total_exact": total.Exact,
		"next_cursor": resp.GetNextCursor(),
		"has_more":    resp.GetHasMore(),
	})
}

func (h *Handler) GetProgPipelineDetail(w http.ResponseWriter, r *http.Request) {
	pipelineULID := strings.TrimSpace(chi.URLParam(r, "pipeline_ulid"))
	if !requireRequestField(w, pipelineULID, "pipeline_ulid") {
		return
	}

	resp, err := h.Gprog.GetPipelineDetail(r.Context(), &gprogpb.GetPipelineDetailReq{
		PipelineUlid: pipelineULID,
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	payload := jsonPayloadObject(resp)
	if pipeline := resp.GetPipeline(); pipeline != nil {
		pipelinePayload := jsonPayloadObject(pipeline)
		h.attachCandidateName(pipelinePayload, pipeline.GetCandidateUlid())
		payload["pipeline"] = pipelinePayload
	}
	WriteJSON(w, http.StatusOK, payload)
}

func (h *Handler) AdminTriggerProgNextStage(w http.ResponseWriter, r *http.Request) {
	pipelineULID := strings.TrimSpace(chi.URLParam(r, "pipeline_ulid"))
	if !requireRequestField(w, pipelineULID, "pipeline_ulid") {
		return
	}

	var input struct {
		ReasonMessage string `json:"reason_message"`
	}
	if err := ReadJSON(r, &input); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}
	adminID := AdminID(r)
	if adminID == "" {
		adminID = "adminserver"
	}

	resp, err := h.Gprog.AdminTriggerNextStage(r.Context(), &gprogpb.AdminTriggerNextStageReq{
		PipelineUlid:  pipelineULID,
		AdminUlid:     adminID,
		ReasonMessage: strings.TrimSpace(input.ReasonMessage),
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) ListProgStatusTransitionLogs(w http.ResponseWriter, r *http.Request) {
	pipelineULID := strings.TrimSpace(chi.URLParam(r, "pipeline_ulid"))
	if !requireRequestField(w, pipelineULID, "pipeline_ulid") {
		return
	}

	req := &gprogpb.ListStatusTransitionLogsReq{
		Filters: &gprogpb.StatusTransitionLogFilters{
			PipelineUlid: pipelineULID,
		},
		Cursor:   strings.TrimSpace(r.URL.Query().Get("cursor")),
		PageSize: int32(parseCursorPage(r, 20).PageSize),
	}

	resp, err := h.Gprog.ListStatusTransitionLogs(r.Context(), req)
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	total, err := countCursorAll(r.Context(), func(ctx context.Context, cursor string, limit uint32) (uint32, string, error) {
		resp, err := h.Gprog.GetStatusTransitionLogCount(ctx, &gprogpb.GetStatusTransitionLogCountRequest{
			Filters: req.Filters,
			Limit:   int32(limit),
			Cursor:  cursor,
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

	for _, log := range resp.GetLogs() {
		mapLogSummaryStatuses(log)
	}

	WriteJSON(w, http.StatusOK, map[string]interface{}{
		"logs":        resp.GetLogs(),
		"total":       total.Total,
		"total_label": total.Label(),
		"total_exact": total.Exact,
		"next_cursor": resp.GetNextCursor(),
		"prev_cursor": resp.GetPrevCursor(),
		"has_more":    resp.GetHasMore(),
	})
}

func (h *Handler) GetProgStatusTransitionLogDetail(w http.ResponseWriter, r *http.Request) {
	transitionULID := strings.TrimSpace(chi.URLParam(r, "transition_ulid"))
	if !requireRequestField(w, transitionULID, "transition_ulid") {
		return
	}

	resp, err := h.Gprog.GetStatusTransitionLogDetail(r.Context(), &gprogpb.GetStatusTransitionLogDetailReq{
		TransitionUlid: transitionULID,
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	if resp.GetSummary() != nil {
		mapLogSummaryStatuses(resp.GetSummary())
	}

	WriteJSON(w, http.StatusOK, resp)
}

func mapLogSummaryStatuses(log *gprogpb.StatusTransitionLogSummary) {
	if log == nil {
		return
	}

	mapStatus := func(entityType, statusStr string) string {
		if statusStr == "" {
			return ""
		}
		var val int32
		var ok bool
		switch entityType {
		case "PIPELINE":
			val, ok = gprogpb.PipelineStatus_value[statusStr]
			if !ok {
				val, ok = gprogpb.PipelineStatus_value["PIPELINE_STATUS_"+statusStr]
			}
		case "STAGE":
			val, ok = gprogpb.StageStatus_value[statusStr]
			if !ok {
				val, ok = gprogpb.StageStatus_value["STAGE_STATUS_"+statusStr]
			}
		case "COURSE_UNIT":
			val, ok = gprogpb.CourseUnitStatus_value[statusStr]
			if !ok {
				val, ok = gprogpb.CourseUnitStatus_value["COURSE_UNIT_STATUS_"+statusStr]
			}
		}
		if ok {
			return strconv.Itoa(int(val))
		}
		return statusStr
	}

	log.FromStatus = mapStatus(log.EntityType, log.FromStatus)
	log.ToStatus = mapStatus(log.EntityType, log.ToStatus)
}

func (h *Handler) AdminTerminatePipeline(w http.ResponseWriter, r *http.Request) {
	pipelineULID := strings.TrimSpace(chi.URLParam(r, "pipeline_ulid"))
	if !requireRequestField(w, pipelineULID, "pipeline_ulid") {
		return
	}

	var input struct {
		ReasonMessage string `json:"reason_message"`
	}
	if err := ReadJSON(r, &input); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}
	adminID := AdminID(r)
	if adminID == "" {
		adminID = "adminserver"
	}

	resp, err := h.Gprog.AdminTerminatePipeline(r.Context(), &gprogpb.AdminTerminatePipelineReq{
		PipelineUlid:  pipelineULID,
		AdminUlid:     adminID,
		ReasonMessage: strings.TrimSpace(input.ReasonMessage),
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) GetProgPipelineCertificateViewURL(w http.ResponseWriter, r *http.Request) {
	pipelineULID := strings.TrimSpace(chi.URLParam(r, "pipeline_ulid"))
	candidateULID := strings.TrimSpace(r.URL.Query().Get("candidate_ulid"))
	if !requireRequestFields(w, pipelineULID, "pipeline_ulid", candidateULID, "candidate_ulid") {
		return
	}

	resp, err := h.Gprog.GetPipelineCertificateViewURL(r.Context(), &gprogpb.GetPipelineCertificateViewURLReq{
		CandidateUlid: candidateULID,
		PipelineUlid:  pipelineULID,
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, map[string]string{"view_url": resp.GetViewUrl()})
}

func (h *Handler) ListProgCertificateTasks(w http.ResponseWriter, r *http.Request) {
	candidateULID := strings.TrimSpace(r.URL.Query().Get("candidate_ulid"))
	if !requireRequestField(w, candidateULID, "candidate_ulid") {
		return
	}

	page := parseCursorPage(r, 20)
	req := &gprogpb.ListCertificateTasksReq{
		Filters: &gprogpb.CertificateTaskFilters{
			CandidateUlid: candidateULID,
			PipelineUlid:  strings.TrimSpace(r.URL.Query().Get("pipeline_ulid")),
		},
		Cursor:    page.Cursor,
		PageSize:  int32(page.PageSize),
		SortOrder: gprogpb.SortOrder(page.Sort),
	}

	resp, err := h.Gprog.ListCertificateTasks(r.Context(), req)
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	total, err := countCursorAll(r.Context(), func(ctx context.Context, cursor string, limit uint32) (uint32, string, error) {
		resp, err := h.Gprog.GetCertificateTaskCount(ctx, &gprogpb.GetCertificateTaskCountRequest{
			Filters:   req.Filters,
			Limit:     int32(limit),
			Cursor:    cursor,
			SortOrder: gprogpb.SortOrder(page.Sort),
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

	WriteJSON(w, http.StatusOK, map[string]interface{}{
		"tasks":       resp.GetTasks(),
		"total":       total.Total,
		"total_label": total.Label(),
		"total_exact": total.Exact,
		"next_cursor": resp.GetNextCursor(),
		"prev_cursor": resp.GetPrevCursor(),
		"has_more":    resp.GetHasMore(),
	})
}

func (h *Handler) GetProgCertificateTaskDetail(w http.ResponseWriter, r *http.Request) {
	taskULID := strings.TrimSpace(chi.URLParam(r, "task_ulid"))
	if !requireRequestField(w, taskULID, "task_ulid") {
		return
	}

	resp, err := h.Gprog.GetCertificateTaskDetail(r.Context(), &gprogpb.GetCertificateTaskDetailReq{
		TaskUlid: taskULID,
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) RetryProgCertificateTask(w http.ResponseWriter, r *http.Request) {
	taskULID := strings.TrimSpace(chi.URLParam(r, "task_ulid"))
	adminID := AdminID(r)
	if adminID == "" {
		adminID = "adminserver"
	}
	if !requireRequestFields(w, taskULID, "task_ulid", adminID, "admin_ulid") {
		return
	}

	resp, err := h.Gprog.RetryCertificateTask(r.Context(), &gprogpb.RetryCertificateTaskReq{
		TaskUlid:  taskULID,
		AdminUlid: adminID,
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) AdminForceCourseCompleted(w http.ResponseWriter, r *http.Request) {
	courseUnitULID := strings.TrimSpace(chi.URLParam(r, "course_unit_ulid"))
	if !requireRequestField(w, courseUnitULID, "course_unit_ulid") {
		return
	}

	var input struct {
		ReasonMessage string `json:"reason_message"`
	}
	if err := ReadJSON(r, &input); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}
	adminID := AdminID(r)
	if adminID == "" {
		adminID = "adminserver"
	}

	resp, err := h.Gprog.AdminForceCourseCompleted(r.Context(), &gprogpb.AdminForceCourseCompletedReq{
		CourseUnitUlid: courseUnitULID,
		AdminUlid:      adminID,
		ReasonMessage:  strings.TrimSpace(input.ReasonMessage),
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) AdminForceCourseSignupExam(w http.ResponseWriter, r *http.Request) {
	courseUnitULID := strings.TrimSpace(chi.URLParam(r, "course_unit_ulid"))
	if !requireRequestField(w, courseUnitULID, "course_unit_ulid") {
		return
	}

	var input struct {
		ReasonMessage string `json:"reason_message"`
	}
	if err := ReadJSON(r, &input); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}
	adminID := AdminID(r)
	if adminID == "" {
		adminID = "adminserver"
	}

	resp, err := h.Gprog.AdminForceCourseSignupExam(r.Context(), &gprogpb.AdminForceCourseSignupExamReq{
		CourseUnitUlid: courseUnitULID,
		AdminUlid:      adminID,
		ReasonMessage:  strings.TrimSpace(input.ReasonMessage),
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, resp)
}
