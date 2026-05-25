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
	msgResp, err := h.Gmsg.ListMessages(ctx, &gmsgpb.ListMessagesRequest{
		UserId: candidateID,
		Status: gmsgpb.MessageStatus_UNREAD.Enum(),
		Limit:  99,
	})
	if err == nil {
		out.UnreadMessagesCount = uint32(len(msgResp.GetMessages()))
	}

	WriteJSON(w, http.StatusOK, out)
}

// GetDashboardStats  GET /api/dashboard/stats
func (h *Handler) GetDashboardStats(w http.ResponseWriter, r *http.Request) {
	// 复用 Dashboard 里的统计逻辑或返回精简版
	h.Dashboard(w, r)
}

// GetDashboardTodos  GET /api/dashboard/todos
func (h *Handler) GetDashboardTodos(w http.ResponseWriter, r *http.Request) {
	WriteJSON(w, http.StatusOK, []PendingItem{
		{Type: "exam_signup", Count: 1, Description: "You have 1 exam ready for signup."},
	})
}

// GetDashboardLearning  GET /api/dashboard/learning
func (h *Handler) GetDashboardLearning(w http.ResponseWriter, r *http.Request) {
	// 返回最近学习的内容，可以从 Gprog 聚合
	h.Dashboard(w, r)
}
