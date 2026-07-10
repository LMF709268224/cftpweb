package handler

import (
	"net/http"

	gcredspb "github.com/afnandelfin620-star/cftptest/cftp/gcreds"
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

	// 1. 获取最近管线 (首页只展示前 3 个)
	pipelinesResp, err := h.Gprog.ListPipelines(ctx, &gprogpb.ListPipelinesReq{
		Filters: &gprogpb.PipelineFilters{
			CandidateUlid: candidateID,
		},
		PageSize: 3,
	})
	if err == nil {
		for _, p := range pipelinesResp.GetPipelines() {
			out.RecentPipelines = append(out.RecentPipelines, toPipelineSummary(p))
		}
	}

	// 2. 获取统计数据
	// 已获得认证数
	credCountResp, err := h.Creds.GetCandidateCredentialCount(ctx, &gcredspb.GetCandidateCredentialCountRequest{
		CandidateUlid: candidateID,
		Limit:         1000,
	})
	if err == nil {
		out.Stats.CertificationsEarned = int(credCountResp.GetCount())
	}

	// 学习中的课程数 (RUNNING, WAIT_FINAL_ELIG, ISSUING_CERT)
	inProgressStatuses := []gprogpb.PipelineStatus{
		gprogpb.PipelineStatus_PIPELINE_STATUS_RUNNING,
		gprogpb.PipelineStatus_PIPELINE_STATUS_WAIT_FINAL_ELIG,
		gprogpb.PipelineStatus_PIPELINE_STATUS_ISSUING_CERT,
	}
	for _, status := range inProgressStatuses {
		progCountResp, err := h.Gprog.GetPipelineCount(ctx, &gprogpb.GetPipelineCountRequest{
			Filters: &gprogpb.PipelineFilters{
				CandidateUlid: candidateID,
				Status:        status,
			},
			Limit: 1000,
		})
		if err == nil {
			out.Stats.CoursesInProgress += int(progCountResp.GetCount())
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

	// List recent in-progress pipelines, fetch max 10 to filter locally if needed,
	// though ListPipelines doesn't support NOT IN status filter directly.
	// Since GetDashboardLearning just needs some recent pipelines that aren't completed,
	// we will fetch up to a reasonable limit and filter.
	pipelinesResp, err := h.Gprog.ListPipelines(ctx, &gprogpb.ListPipelinesReq{
		Filters: &gprogpb.PipelineFilters{
			CandidateUlid: candidateID,
		},
		PageSize: 10,
	})
	if err == nil {
		for _, p := range pipelinesResp.GetPipelines() {
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
