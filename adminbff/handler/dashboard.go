package handler

import (
	"net/http"
	"strings"
	"time"

	mallpb "github.com/afnandelfin620-star/cftptest/cftp/gmall"
	gmidpb "github.com/afnandelfin620-star/cftptest/cftp/gmid"
	gprogpb "github.com/afnandelfin620-star/cftptest/cftp/gprog"
	"github.com/casdoor/casdoor-go-sdk/casdoorsdk"
)

const adminDashboardSampleLimit int32 = 500

type opsDashboardResponse struct {
	CandidateTotal int64                     `json:"candidate_total"`
	StageBuckets   []opsDashboardStageBucket `json:"stage_buckets"`
	TodayRevenue   []opsDashboardRevenue     `json:"today_revenue"`
	GeneratedAt    string                    `json:"generated_at"`
}

type opsDashboardStageBucket struct {
	StageID string `json:"stage_id"`
	Status  string `json:"status"`
	Count   int64  `json:"count"`
}

type opsDashboardRevenue struct {
	Currency    string `json:"currency"`
	AmountMinor int64  `json:"amount_minor"`
	OrderCount  int64  `json:"order_count"`
}

func (h *Handler) OpsDashboard(w http.ResponseWriter, r *http.Request) {
	candidateTotal, err := h.countCandidates(r)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, ErrInternal, "failed to count candidates")
		return
	}

	pipelines, err := h.Gprog.ListPipelines(r.Context(), &gprogpb.ListPipelinesReq{
		Limit:  adminDashboardSampleLimit,
		Offset: 0,
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	stageCounts := make(map[string]int64)
	for _, pipeline := range pipelines.GetPipelines() {
		if pipeline == nil {
			continue
		}
		key := strings.TrimSpace(pipeline.GetCurrentStageUlid())
		if key == "" {
			key = "未进入阶段"
		}
		status := pipeline.GetStatus().String()
		stageCounts[key+"|"+status]++
	}

	stageBuckets := make([]opsDashboardStageBucket, 0, len(stageCounts))
	for key, count := range stageCounts {
		parts := strings.SplitN(key, "|", 2)
		status := ""
		if len(parts) == 2 {
			status = parts[1]
		}
		stageBuckets = append(stageBuckets, opsDashboardStageBucket{
			StageID: parts[0],
			Status:  status,
			Count:   count,
		})
	}

	orders, err := h.Mall.ListOrders(r.Context(), &mallpb.ListOrdersRequest{
		PaymentStatus: "PAID",
		Limit:         adminDashboardSampleLimit,
		Offset:        0,
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	revenueByCurrency := make(map[string]*opsDashboardRevenue)
	for _, order := range orders.GetItems() {
		if order == nil || !isSameLocalDay(order.GetCreatedAt(), startOfDay) {
			continue
		}
		currency := strings.ToUpper(strings.TrimSpace(order.GetCurrencyCode()))
		if currency == "" {
			currency = "UNKNOWN"
		}
		item := revenueByCurrency[currency]
		if item == nil {
			item = &opsDashboardRevenue{Currency: currency}
			revenueByCurrency[currency] = item
		}
		item.AmountMinor += order.GetAmountMinor()
		item.OrderCount++
	}

	revenue := make([]opsDashboardRevenue, 0, len(revenueByCurrency))
	for _, item := range revenueByCurrency {
		revenue = append(revenue, *item)
	}

	WriteJSON(w, http.StatusOK, opsDashboardResponse{
		CandidateTotal: candidateTotal,
		StageBuckets:   stageBuckets,
		TodayRevenue:   revenue,
		GeneratedAt:    now.Format(time.RFC3339),
	})
}

func (h *Handler) countCandidates(r *http.Request) (int64, error) {
	users, err := casdoorsdk.GetUsers()
	if err != nil {
		return 0, err
	}

	var total int64
	for _, user := range users {
		if user == nil || user.Id == "" {
			continue
		}
		resp, err := h.Gmid.GetUlidByUUID(r.Context(), &gmidpb.GetUlidByUUIDRequest{UserUuid: user.Id})
		if err != nil {
			return 0, err
		}
		if strings.TrimSpace(resp.GetUserUlid()) != "" {
			total++
		}
	}
	return total, nil
}

func isSameLocalDay(raw string, startOfDay time.Time) bool {
	createdAt, err := time.Parse(time.RFC3339, strings.TrimSpace(raw))
	if err != nil {
		return false
	}
	local := createdAt.In(startOfDay.Location())
	return !local.Before(startOfDay) && local.Before(startOfDay.Add(24*time.Hour))
}
