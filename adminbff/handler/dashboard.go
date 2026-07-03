package handler

import (
	"net/http"
	"sort"
	"strings"
	"time"

	mallpb "github.com/afnandelfin620-star/cftptest/cftp/gmall"
	gmidpb "github.com/afnandelfin620-star/cftptest/cftp/gmid"
	gprogpb "github.com/afnandelfin620-star/cftptest/cftp/gprog"
	"github.com/casdoor/casdoor-go-sdk/casdoorsdk"
)

const adminDashboardSampleLimit int32 = 500

type opsDashboardResponse struct {
	CandidateTotal           int64                     `json:"candidate_total"`
	UserStats                opsDashboardUserStats     `json:"user_stats"`
	UserRoleStats            []opsDashboardRoleStat    `json:"user_role_stats"`
	ProfileCompletionPercent int64                     `json:"profile_completion_percent"`
	Users                    []opsDashboardUser        `json:"users"`
	StageBuckets             []opsDashboardStageBucket `json:"stage_buckets"`
	TodayRevenue             []opsDashboardRevenue     `json:"today_revenue"`
	GeneratedAt              string                    `json:"generated_at"`
}

type opsDashboardUserStats struct {
	Total         int64 `json:"total"`
	Active        int64 `json:"active"`
	Inactive      int64 `json:"inactive"`
	Admins        int64 `json:"admins"`
	EmailVerified int64 `json:"email_verified"`
}

type opsDashboardRoleStat struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Count int64  `json:"count"`
}

type opsDashboardUser struct {
	ID            string   `json:"id"`
	CandidateULID string   `json:"candidate_ulid,omitempty"`
	Name          string   `json:"name"`
	Email         string   `json:"email"`
	Phone         string   `json:"phone"`
	Location      string   `json:"location"`
	Roles         []string `json:"roles"`
	RoleLabel     string   `json:"role_label"`
	Status        string   `json:"status"`
	EmailVerified bool     `json:"email_verified"`
	CreatedAt     string   `json:"created_at"`
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
	users, err := casdoorsdk.GetUsers()
	if err != nil {
		WriteError(w, http.StatusInternalServerError, ErrInternal, "failed to count candidates")
		return
	}
	userSummary := h.buildOpsUserSummary(r, users)

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
		CandidateTotal:           userSummary.candidateTotal,
		UserStats:                userSummary.stats,
		UserRoleStats:            userSummary.roleStats,
		ProfileCompletionPercent: userSummary.profileCompletionPercent,
		Users:                    userSummary.users,
		StageBuckets:             stageBuckets,
		TodayRevenue:             revenue,
		GeneratedAt:              now.Format(time.RFC3339),
	})
}

type opsDashboardUserSummary struct {
	candidateTotal           int64
	stats                    opsDashboardUserStats
	roleStats                []opsDashboardRoleStat
	profileCompletionPercent int64
	users                    []opsDashboardUser
}

func (h *Handler) buildOpsUserSummary(r *http.Request, users []*casdoorsdk.User) opsDashboardUserSummary {
	roleCounts := map[string]int64{
		"students":  0,
		"markers":   0,
		"marketing": 0,
		"corporate": 0,
		"board":     0,
		"partners":  0,
	}
	stats := opsDashboardUserStats{}
	var profileScore int64
	preview := make([]opsDashboardUser, 0, 10)

	for _, user := range users {
		if user == nil || user.Id == "" {
			continue
		}
		stats.Total++
		if user.IsForbidden || user.IsDeleted {
			stats.Inactive++
		} else {
			stats.Active++
		}
		if user.EmailVerified {
			stats.EmailVerified++
		}
		if IsCftpAdmin(user) {
			stats.Admins++
		}

		roles := roleNames(user)
		for key := range classifyDashboardRoles(roles) {
			roleCounts[key]++
		}
		profileScore += int64(profileCompletionPercent(user))

		candidateULID := ""
		resp, err := h.Gmid.GetUlidByUUID(r.Context(), &gmidpb.GetUlidByUUIDRequest{UserUuid: user.Id})
		if err == nil && strings.TrimSpace(resp.GetUserUlid()) != "" {
			candidateULID = strings.TrimSpace(resp.GetUserUlid())
			roleCounts["students"]++
		}

		if len(preview) < 10 {
			preview = append(preview, opsDashboardUser{
				ID:            user.Id,
				CandidateULID: candidateULID,
				Name:          dashboardUserName(user),
				Email:         strings.TrimSpace(user.Email),
				Phone:         strings.TrimSpace(user.Phone),
				Location:      dashboardLocation(user),
				Roles:         roles,
				RoleLabel:     dashboardPrimaryRole(user, roles, candidateULID),
				Status:        dashboardUserStatus(user),
				EmailVerified: user.EmailVerified,
				CreatedAt:     user.CreatedTime,
			})
		}
	}

	var completion int64
	if stats.Total > 0 {
		completion = profileScore / stats.Total
	}

	return opsDashboardUserSummary{
		candidateTotal: statsForStudents(roleCounts),
		stats:          stats,
		roleStats: []opsDashboardRoleStat{
			{Key: "students", Label: "Students", Count: roleCounts["students"]},
			{Key: "markers", Label: "Markers", Count: roleCounts["markers"]},
			{Key: "marketing", Label: "Marketing", Count: roleCounts["marketing"]},
			{Key: "corporate", Label: "Corporate", Count: roleCounts["corporate"]},
			{Key: "board", Label: "Board", Count: roleCounts["board"]},
			{Key: "partners", Label: "Partners", Count: roleCounts["partners"]},
		},
		profileCompletionPercent: completion,
		users:                    preview,
	}
}

func statsForStudents(roleCounts map[string]int64) int64 {
	return roleCounts["students"]
}

func roleNames(user *casdoorsdk.User) []string {
	roles := make([]string, 0, len(user.Roles))
	for _, role := range user.Roles {
		if role == nil {
			continue
		}
		name := strings.TrimSpace(role.DisplayName)
		if name == "" {
			name = strings.TrimSpace(role.Name)
		}
		if name != "" {
			roles = append(roles, name)
		}
	}
	sort.Strings(roles)
	return roles
}

func classifyDashboardRoles(roles []string) map[string]bool {
	result := make(map[string]bool)
	for _, role := range roles {
		normalized := strings.ToLower(role)
		switch {
		case strings.Contains(normalized, "marker"):
			result["markers"] = true
		case strings.Contains(normalized, "marketing"):
			result["marketing"] = true
		case strings.Contains(normalized, "corporate"):
			result["corporate"] = true
		case strings.Contains(normalized, "board"):
			result["board"] = true
		case strings.Contains(normalized, "partner"):
			result["partners"] = true
		}
	}
	return result
}

func dashboardUserName(user *casdoorsdk.User) string {
	name := strings.TrimSpace(user.DisplayName)
	if name == "" {
		name = strings.TrimSpace(user.RealName)
	}
	if name == "" {
		name = strings.TrimSpace(strings.TrimSpace(user.FirstName + " " + user.LastName))
	}
	if name == "" {
		name = strings.TrimSpace(user.Name)
	}
	return name
}

func dashboardLocation(user *casdoorsdk.User) string {
	parts := make([]string, 0, 2)
	if strings.TrimSpace(user.Location) != "" {
		parts = append(parts, strings.TrimSpace(user.Location))
	}
	if strings.TrimSpace(user.Region) != "" && !strings.EqualFold(strings.TrimSpace(user.Region), strings.TrimSpace(user.Location)) {
		parts = append(parts, strings.TrimSpace(user.Region))
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, ", ")
}

func dashboardPrimaryRole(user *casdoorsdk.User, roles []string, candidateULID string) string {
	if IsCftpAdmin(user) {
		return "admin"
	}
	if candidateULID != "" {
		return "student"
	}
	if len(roles) > 0 {
		return roles[0]
	}
	return "member"
}

func dashboardUserStatus(user *casdoorsdk.User) string {
	if user.IsDeleted {
		return "Deleted"
	}
	if user.IsForbidden {
		return "Inactive"
	}
	return "Active"
}

func profileCompletionPercent(user *casdoorsdk.User) int {
	fields := []string{
		user.Name,
		user.DisplayName,
		user.Email,
		user.Phone,
		user.Region,
		user.Location,
		user.RealName,
		user.Gender,
		user.Birthday,
		user.Education,
		user.Affiliation,
		user.Title,
	}
	completed := 0
	for _, field := range fields {
		if strings.TrimSpace(field) != "" {
			completed++
		}
	}
	if len(fields) == 0 {
		return 0
	}
	return completed * 100 / len(fields)
}

func isSameLocalDay(raw string, startOfDay time.Time) bool {
	createdAt, err := time.Parse(time.RFC3339, strings.TrimSpace(raw))
	if err != nil {
		return false
	}
	local := createdAt.In(startOfDay.Location())
	return !local.Before(startOfDay) && local.Before(startOfDay.Add(24*time.Hour))
}
