package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	mallpb "github.com/afnandelfin620-star/cftptest/cftp/gmall"
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

func (h *Handler) AdminPurgeProgPipelineTestData(w http.ResponseWriter, r *http.Request) {
	pipelineULID := strings.TrimSpace(chi.URLParam(r, "pipeline_ulid"))
	if !requireRequestField(w, pipelineULID, "pipeline_ulid") {
		return
	}

	detail, err := h.Gprog.GetPipelineDetail(r.Context(), &gprogpb.GetPipelineDetailReq{
		PipelineUlid: pipelineULID,
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	pipeline := detail.GetPipeline()
	if pipeline == nil {
		WriteError(w, http.StatusNotFound, ErrNotFound, "pipeline not found")
		return
	}

	candidateULID := strings.TrimSpace(pipeline.GetCandidateUlid())
	pipelineCcULID := strings.TrimSpace(pipeline.GetPipelineCcUlid())
	if !requireRequestFields(w, candidateULID, "candidate_ulid", pipelineCcULID, "pipeline_cc_ulid") {
		return
	}

	bundleOrderULID, err := h.findBundleOrderForProgPipeline(r, candidateULID, pipelineCcULID)
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	if bundleOrderULID == "" {
		WriteError(w, http.StatusNotFound, ErrNotFound, "no matching bundle order was found for pipeline")
		return
	}

	adminID := AdminID(r)
	if adminID == "" {
		adminID = "adminserver"
	}
	resp, err := h.Mall.AdminPurgeCandidateBundle(r.Context(), &mallpb.AdminPurgeCandidateBundleRequest{
		CandidateUlid:   candidateULID,
		BundleOrderUlid: bundleOrderULID,
		AdminUlid:       adminID,
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, map[string]interface{}{
		"success":           resp.GetSuccess(),
		"message":           resp.GetMessage(),
		"candidate_ulid":    candidateULID,
		"pipeline_ulid":     pipelineULID,
		"pipeline_cc_ulid":  pipelineCcULID,
		"bundle_order_ulid": bundleOrderULID,
	})
}

func (h *Handler) findBundleOrderForProgPipeline(r *http.Request, candidateULID string, pipelineCcULID string) (string, error) {
	if candidateULID == "" || pipelineCcULID == "" {
		return "", nil
	}

	direct, err := h.Mall.ListBundleOrders(r.Context(), &mallpb.ListBundleOrdersRequest{
		CandidateUlid: candidateULID,
		BundleUlid:    pipelineCcULID,
		Limit:         20,
	})
	if err != nil {
		return "", err
	}
	for _, order := range direct.GetItems() {
		if order.GetCandidateUlid() == candidateULID && order.GetBundleOrderUlid() != "" {
			return order.GetBundleOrderUlid(), nil
		}
	}

	orders, err := h.Mall.ListBundleOrders(r.Context(), &mallpb.ListBundleOrdersRequest{
		CandidateUlid: candidateULID,
		Limit:         100,
	})
	if err != nil {
		return "", err
	}
	for _, order := range orders.GetItems() {
		if order.GetCandidateUlid() != candidateULID || order.GetBundleOrderUlid() == "" {
			continue
		}
		if order.GetBundleUlid() == pipelineCcULID {
			return order.GetBundleOrderUlid(), nil
		}
		detail, err := h.Mall.GetBundleOrderDetail(r.Context(), &mallpb.GetBundleOrderDetailRequest{
			BundleOrderUlid: order.GetBundleOrderUlid(),
		})
		if err != nil {
			continue
		}
		if strings.Contains(detail.GetDetail().GetItemsSnapshotJson(), pipelineCcULID) {
			return order.GetBundleOrderUlid(), nil
		}
	}

	return "", nil
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
