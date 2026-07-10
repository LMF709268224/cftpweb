package handler

import (
	"net/http"

	gmsgpb "github.com/afnandelfin620-star/cftptest/cftp/gmsg"
	gprogpb "github.com/afnandelfin620-star/cftptest/cftp/gprog"
)

// Dashboard  GET /api/dashboard
func (h *Handler) Dashboard(w http.ResponseWriter, r *http.Request) {
	candidateID := CandidateID(r)
	ctx := r.Context()

	out := DashboardRsp{
		CandidateName: CandidateName(r),
		Stats: DashboardStats{
			MembershipLevel: "Standard",
		},
	}

	// 1. 获取管线进度并统计
	progResp, err := h.Gprog.ListCandidatePipelines(ctx, &gprogpb.ListCandidatePipelinesReq{
		CandidateUlid: candidateID,
	})
	if err == nil {
		for _, p := range progResp.GetPipelines() {
			if p.Status == gprogpb.PipelineStatus_PIPELINE_STATUS_COMPLETED {
				out.Stats.CertificationsEarned++
			} else {
				out.Stats.CoursesInProgress++
			}
			// 只在首页展示前 3 个管线
			if len(out.RecentPipelines) < 3 {
				out.RecentPipelines = append(out.RecentPipelines, toPipelineSummary(p))
			}
		}
	}

	// 2. 获取未读消息数
	msgResp, err := h.Gmsg.GetMessageCount(ctx, &gmsgpb.GetMessageCountRequest{
		Filters: &gmsgpb.MessageFilters{
			UserUlid: candidateID,
			Status:   gmsgpb.MessageStatus_UNREAD.Enum(),
		},
		Limit: 99,
	})
	if err == nil {
		out.UnreadMessagesCount = msgResp.GetCount()
	}

	WriteJSON(w, http.StatusOK, out)
}

// GetDashboardStats  GET /api/dashboard/stats — returns the same aggregated data as Dashboard.
func (h *Handler) GetDashboardStats(w http.ResponseWriter, r *http.Request) {
	h.Dashboard(w, r)
}

// GetDashboardTodos  GET /api/dashboard/todos — returns pending action items for the candidate.
// TODO: implement real logic once the backend aggregation service is available.
func (h *Handler) GetDashboardTodos(w http.ResponseWriter, r *http.Request) {
	WriteJSON(w, http.StatusOK, []PendingItem{})
}

// GetDashboardLearning  GET /api/dashboard/learning — returns in-progress learning items.
func (h *Handler) GetDashboardLearning(w http.ResponseWriter, r *http.Request) {
	candidateID := CandidateID(r)
	ctx := r.Context()

	out := struct {
		RecentPipelines []PipelineSummary `json:"recent_pipelines"`
	}{}

	progResp, err := h.Gprog.ListCandidatePipelines(ctx, &gprogpb.ListCandidatePipelinesReq{
		CandidateUlid: candidateID,
	})
	if err == nil {
		for _, p := range progResp.GetPipelines() {
			if p.Status != gprogpb.PipelineStatus_PIPELINE_STATUS_COMPLETED {
				out.RecentPipelines = append(out.RecentPipelines, toPipelineSummary(p))
			}
			if len(out.RecentPipelines) >= 5 {
				break
			}
		}
	}
	if out.RecentPipelines == nil {
		out.RecentPipelines = []PipelineSummary{}
	}

	WriteJSON(w, http.StatusOK, out)
}
