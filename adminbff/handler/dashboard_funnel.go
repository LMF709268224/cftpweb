package handler

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	gprogpb "github.com/afnandelfin620-star/cftptest/cftp/gprog"
)

type dashboardFunnelResponse struct {
	CandidateAccounts      int  `json:"candidate_accounts"`
	UnlinkedAccounts       int  `json:"unlinked_accounts"`
	WithPipeline           int  `json:"with_pipeline"`
	CompletedPipeline      int  `json:"completed_pipeline"`
	Exact                  bool `json:"exact"`
	AggregationSampleLimit int  `json:"aggregation_sample_limit"`
}

func (h *Handler) OpsDashboardFunnel(w http.ResponseWriter, r *http.Request) {
	users, err := h.CandidateProfiles.UsersOrRefresh(r.Context())
	if err != nil {
		slog.Error("Failed to initialize dashboard identity cache", "error", err)
		WriteError(w, http.StatusServiceUnavailable, ErrServiceUnavailable, "user identity cache is not ready")
		return
	}
	roleConfig := dashboardRoleConfigFromEnv()
	roleDefinitions := dashboardRoleDefinitions(roleConfig)
	cftpUsers := filterCFTPDashboardUsers(users, roleConfig, roleDefinitions)
	candidateAccounts := 0
	unlinkedAccounts := 0
	candidateIDs := make(map[string]struct{})
	candidateByUserID := h.dashboardCandidateULIDs(cftpUsers)
	for _, user := range cftpUsers {
		if dashboardRoleFlags(user, roleNames(user), roleConfig, roleDefinitions)["candidate"] {
			candidateAccounts++
			if candidateID := candidateByUserID[user.Id]; candidateID != "" {
				candidateIDs[candidateID] = struct{}{}
			} else {
				unlinkedAccounts++
			}
		}
	}
	withPipeline, completedPipeline, exact, err := h.dashboardPipelineCandidates(r.Context(), candidateIDs)
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, dashboardFunnelResponse{
		CandidateAccounts:      candidateAccounts,
		UnlinkedAccounts:       unlinkedAccounts,
		WithPipeline:           withPipeline,
		CompletedPipeline:      completedPipeline,
		Exact:                  exact && unlinkedAccounts == 0,
		AggregationSampleLimit: adminDashboardSampleLimit,
	})
}

func (h *Handler) dashboardPipelineCandidates(ctx context.Context, candidateIDs map[string]struct{}) (int, int, bool, error) {
	withPipeline := make(map[string]struct{})
	completedPipeline := make(map[string]struct{})
	seenCursors := make(map[string]struct{})
	cursor := ""
	for page := 0; page < adminDashboardSampleLimit/adminDashboardPageSize; page++ {
		resp, err := h.Gprog.ListPipelines(ctx, &gprogpb.ListPipelinesReq{
			Filters:   &gprogpb.PipelineFilters{},
			Cursor:    cursor,
			PageSize:  adminDashboardPageSize,
			SortOrder: gprogpb.SortOrder_SORT_ORDER_DESC,
		})
		if err != nil {
			return 0, 0, false, err
		}
		if resp == nil {
			return 0, 0, false, fmt.Errorf("dashboard pipeline response is missing")
		}
		for _, pipeline := range resp.GetPipelines() {
			if pipeline == nil || strings.TrimSpace(pipeline.GetCandidateUlid()) == "" {
				return 0, 0, false, fmt.Errorf("dashboard pipeline candidate is missing")
			}
			candidateID := pipeline.GetCandidateUlid()
			if _, known := candidateIDs[candidateID]; !known {
				continue
			}
			withPipeline[candidateID] = struct{}{}
			if pipeline.GetStatus() == gprogpb.PipelineStatus_PIPELINE_STATUS_COMPLETED {
				completedPipeline[candidateID] = struct{}{}
			}
		}
		if !resp.GetHasMore() {
			return len(withPipeline), len(completedPipeline), true, nil
		}
		nextCursor := strings.TrimSpace(resp.GetNextCursor())
		if nextCursor == "" || nextCursor == cursor {
			return 0, 0, false, fmt.Errorf("dashboard pipeline cursor did not advance")
		}
		if _, seen := seenCursors[nextCursor]; seen {
			return 0, 0, false, fmt.Errorf("dashboard pipeline cursor loop detected")
		}
		seenCursors[nextCursor] = struct{}{}
		cursor = nextCursor
	}
	return len(withPipeline), len(completedPipeline), false, nil
}
