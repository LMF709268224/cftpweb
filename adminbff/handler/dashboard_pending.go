package handler

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	gcredspb "github.com/afnandelfin620-star/cftptest/cftp/gcreds"
)

type dashboardPendingAgeResponse struct {
	CreatedOlder48h        int  `json:"created_older_48h"`
	Exact                  bool `json:"exact"`
	AggregationSampleLimit int  `json:"aggregation_sample_limit"`
}

func (h *Handler) OpsDashboardPendingAge(w http.ResponseWriter, r *http.Request) {
	count, exact, err := h.dashboardPendingCreatedOlder48h(r.Context(), time.Now().UTC())
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, dashboardPendingAgeResponse{
		CreatedOlder48h:        count,
		Exact:                  exact,
		AggregationSampleLimit: adminDashboardSampleLimit,
	})
}

func (h *Handler) dashboardPendingCreatedOlder48h(ctx context.Context, now time.Time) (int, bool, error) {
	cutoff := now.Add(-48 * time.Hour)
	cursor := ""
	seenCursors := make(map[string]struct{})
	count := 0
	for page := 0; page < adminDashboardSampleLimit/adminDashboardPageSize; page++ {
		resp, err := h.Creds.ListApplications(ctx, &gcredspb.ListApplicationsRequest{
			Filters:   &gcredspb.ApplicationFilters{Statuses: []string{"Pending"}},
			Cursor:    cursor,
			PageSize:  adminDashboardPageSize,
			SortOrder: gcredspb.SortOrder_SORT_ORDER_DESC,
		})
		if err != nil {
			return 0, false, err
		}
		if resp == nil {
			return 0, false, fmt.Errorf("dashboard pending applications response is missing")
		}
		for _, application := range resp.GetApplications() {
			if application == nil || !strings.EqualFold(application.GetStatus(), "Pending") {
				return 0, false, fmt.Errorf("dashboard pending application status is missing or unexpected")
			}
			createdAt, err := time.Parse(time.RFC3339, application.GetCreatedAt())
			if err != nil {
				return 0, false, fmt.Errorf("dashboard pending application creation time: %w", err)
			}
			if createdAt.Before(cutoff) {
				count++
			}
		}
		if !resp.GetHasMore() {
			return count, true, nil
		}
		nextCursor := strings.TrimSpace(resp.GetNextCursor())
		if nextCursor == "" || nextCursor == cursor {
			return 0, false, fmt.Errorf("dashboard pending application cursor did not advance")
		}
		if _, seen := seenCursors[nextCursor]; seen {
			return 0, false, fmt.Errorf("dashboard pending application cursor loop detected")
		}
		seenCursors[nextCursor] = struct{}{}
		cursor = nextCursor
	}
	return count, false, nil
}
