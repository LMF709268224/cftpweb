package handler

import (
	"net/http"
	gprog "github.com/afnandelfin620-star/cftptest/cftp/gprog"
)

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

	for _, p := range resp.GetPipelines() {
		out.List = append(out.List, toPipelineSummary(p))
	}

	WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) ListMaterials(w http.ResponseWriter, r *http.Request) {
	WriteError(w, http.StatusNotImplemented, ErrInternal, "Under construction due to LMS refactoring")
}

func (h *Handler) GetAccessURL(w http.ResponseWriter, r *http.Request) {
	WriteError(w, http.StatusNotImplemented, ErrInternal, "Under construction due to LMS refactoring")
}

func (h *Handler) ReportProgress(w http.ResponseWriter, r *http.Request) {
	WriteError(w, http.StatusNotImplemented, ErrInternal, "Under construction due to LMS refactoring")
}

func (h *Handler) GetProgress(w http.ResponseWriter, r *http.Request) {
	WriteError(w, http.StatusNotImplemented, ErrInternal, "Under construction due to LMS refactoring")
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
		StartedAt:        p.StartedAt,
		CompletedAt:      p.CompletedAt,
		CreatedAt:        p.CreatedAt,
	}
}