package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	gprogpb "github.com/afnandelfin620-star/cftptest/cftp/gprog"
	"github.com/go-chi/chi/v5"
)

func (h *Handler) ListProgPipelines(w http.ResponseWriter, r *http.Request) {
	req := &gprogpb.ListPipelinesReq{
		CandidateUlid:  strings.TrimSpace(r.URL.Query().Get("candidate_ulid")),
		PipelineCcUlid: strings.TrimSpace(r.URL.Query().Get("pipeline_cc_ulid")),
		Status:         gprogpb.PipelineStatus(parseEnumQuery(r, "status")),
		Limit:          int32(parseUint32Query(r, "limit")),
		Offset:         int32(parseUint32Query(r, "offset")),
	}

	resp, err := h.Gprog.ListPipelines(r.Context(), req)
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, resp)
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

	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) AdminTriggerProgNextStage(w http.ResponseWriter, r *http.Request) {
	pipelineULID := strings.TrimSpace(chi.URLParam(r, "pipeline_ulid"))
	if !requireRequestField(w, pipelineULID, "pipeline_ulid") {
		return
	}

	var input struct {
		ReasonMessage string `json:"reason_message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
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
		PipelineUlid: pipelineULID,
		Limit:        int32(parseUint32Query(r, "limit")),
		Offset:       int32(parseUint32Query(r, "offset")),
	}

	resp, err := h.Gprog.ListStatusTransitionLogs(r.Context(), req)
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	for _, log := range resp.GetLogs() {
		mapLogSummaryStatuses(log)
	}

	WriteJSON(w, http.StatusOK, resp)
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
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
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

func (h *Handler) AdminForceCourseCompleted(w http.ResponseWriter, r *http.Request) {
	courseUnitULID := strings.TrimSpace(chi.URLParam(r, "course_unit_ulid"))
	if !requireRequestField(w, courseUnitULID, "course_unit_ulid") {
		return
	}

	var input struct {
		ReasonMessage string `json:"reason_message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
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
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
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
