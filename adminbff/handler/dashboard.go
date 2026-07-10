package handler

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"adminbff/config"
	mallpb "github.com/afnandelfin620-star/cftptest/cftp/gmall"
	gmidpb "github.com/afnandelfin620-star/cftptest/cftp/gmid"
	gprogpb "github.com/afnandelfin620-star/cftptest/cftp/gprog"
	"github.com/casdoor/casdoor-go-sdk/casdoorsdk"
)

const (
	adminDashboardSampleLimit = 500
	adminDashboardPageSize    = 100
)

const (
	defaultDashboardAdminRole       = "role_admin_basic"
	defaultDashboardStudentRole     = "role_student_basic"
	defaultDashboardMembershipRoles = "role_membership_affiliate,role_membership_associate,role_membership_charterholder"
)

type opsDashboardResponse struct {
	CandidateTotal           int64                     `json:"candidate_total"`
	UserStats                opsDashboardUserStats     `json:"user_stats"`
	UserRoleStats            []opsDashboardRoleStat    `json:"user_role_stats"`
	ProfileCompletionPercent int64                     `json:"profile_completion_percent"`
	Users                    []opsDashboardUser        `json:"users"`
	UserTotal                int                       `json:"user_total"`
	UserPage                 int                       `json:"user_page"`
	UserPageSize             int                       `json:"user_page_size"`
	StageBuckets             []opsDashboardStageBucket `json:"stage_buckets"`
	TodayRevenue             []opsDashboardRevenue     `json:"today_revenue"`
	GeneratedAt              string                    `json:"generated_at"`
}

type opsDashboardUserStats struct {
	Total         int64 `json:"total"`
	Active        int64 `json:"active"`
	Inactive      int64 `json:"inactive"`
	Admins        int64 `json:"admins"`
	Members       int64 `json:"members"`
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

type opsDashboardRoleConfig struct {
	admin       string
	student     string
	memberships []string
}

func (h *Handler) OpsDashboard(w http.ResponseWriter, r *http.Request) {
	users, err := casdoorsdk.GetUsers()
	if err != nil {
		WriteError(w, http.StatusInternalServerError, ErrInternal, "failed to count candidates")
		return
	}

	userPage := int(int32Query(r, "user_page", 1))
	if userPage < 1 {
		userPage = 1
	}
	userPageSize := int(int32Query(r, "user_page_size", 10))
	if userPageSize < 1 {
		userPageSize = 10
	}
	if userPageSize > 100 {
		userPageSize = 100
	}

	roleConfig := dashboardRoleConfigFromEnv()
	roleDefinitions := dashboardRoleDefinitions(roleConfig)
	cftpUsers := filterCFTPDashboardUsers(users, roleConfig, roleDefinitions)
	candidateByUserID := h.dashboardCandidateULIDs(r, cftpUsers)
	filteredUsers := filterOpsDashboardUsers(cftpUsers, candidateByUserID, roleConfig, roleDefinitions, opsDashboardUserFilterFromRequest(r))
	pageUsers := paginateOpsDashboardUsers(filteredUsers, userPage, userPageSize)
	userSummary := h.buildOpsUserSummary(cftpUsers, pageUsers, candidateByUserID, roleConfig, roleDefinitions)

	pipelines, err := h.listDashboardPipelines(r.Context())
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	stageCounts := make(map[string]int64)
	for _, pipeline := range pipelines {
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

	orders, err := h.listDashboardPaidOrders(r.Context())
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	revenueByCurrency := make(map[string]*opsDashboardRevenue)
	for _, order := range orders {
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
		UserTotal:                len(filteredUsers),
		UserPage:                 userPage,
		UserPageSize:             userPageSize,
		StageBuckets:             stageBuckets,
		TodayRevenue:             revenue,
		GeneratedAt:              now.Format(time.RFC3339),
	})
}

func (h *Handler) listDashboardPipelines(ctx context.Context) ([]*gprogpb.PipelineSummary, error) {
	items := make([]*gprogpb.PipelineSummary, 0, adminDashboardSampleLimit)
	cursor := ""
	seen := make(map[string]struct{})

	for page := 0; page < adminDashboardSampleLimit/adminDashboardPageSize && len(items) < adminDashboardSampleLimit; page++ {
		pageSize := adminDashboardPageSize
		if remaining := adminDashboardSampleLimit - len(items); remaining < pageSize {
			pageSize = remaining
		}
		resp, err := h.Gprog.ListPipelines(ctx, &gprogpb.ListPipelinesReq{
			Cursor:   cursor,
			PageSize: int32(pageSize),
		})
		if err != nil {
			return nil, err
		}
		items = append(items, resp.GetPipelines()...)
		if !resp.GetHasMore() {
			break
		}
		nextCursor := strings.TrimSpace(resp.GetNextCursor())
		if nextCursor == "" || nextCursor == cursor {
			return nil, fmt.Errorf("dashboard pipeline cursor did not advance")
		}
		if _, ok := seen[nextCursor]; ok {
			return nil, fmt.Errorf("dashboard pipeline cursor loop detected")
		}
		seen[nextCursor] = struct{}{}
		cursor = nextCursor
	}

	if len(items) > adminDashboardSampleLimit {
		items = items[:adminDashboardSampleLimit]
	}
	return items, nil
}

func (h *Handler) listDashboardPaidOrders(ctx context.Context) ([]*mallpb.OrderSummary, error) {
	items := make([]*mallpb.OrderSummary, 0, adminDashboardSampleLimit)
	cursor := ""
	seen := make(map[string]struct{})

	for page := 0; page < adminDashboardSampleLimit/adminDashboardPageSize && len(items) < adminDashboardSampleLimit; page++ {
		pageSize := adminDashboardPageSize
		if remaining := adminDashboardSampleLimit - len(items); remaining < pageSize {
			pageSize = remaining
		}
		resp, err := h.Mall.ListOrders(ctx, &mallpb.ListOrdersRequest{
			Filters: &mallpb.OrderFilters{
				PaymentStatus: "PAID",
			},
			Cursor:   cursor,
			PageSize: uint32(pageSize),
		})
		if err != nil {
			return nil, err
		}
		items = append(items, resp.GetItems()...)
		if !resp.GetHasMore() {
			break
		}
		nextCursor := strings.TrimSpace(resp.GetNextCursor())
		if nextCursor == "" || nextCursor == cursor {
			return nil, fmt.Errorf("dashboard order cursor did not advance")
		}
		if _, ok := seen[nextCursor]; ok {
			return nil, fmt.Errorf("dashboard order cursor loop detected")
		}
		seen[nextCursor] = struct{}{}
		cursor = nextCursor
	}

	if len(items) > adminDashboardSampleLimit {
		items = items[:adminDashboardSampleLimit]
	}
	return items, nil
}

type opsDashboardUserSummary struct {
	candidateTotal           int64
	stats                    opsDashboardUserStats
	roleStats                []opsDashboardRoleStat
	profileCompletionPercent int64
	users                    []opsDashboardUser
}

type opsDashboardUserFilter struct {
	keyword string
	role    string
	status  string
}

func opsDashboardUserFilterFromRequest(r *http.Request) opsDashboardUserFilter {
	return opsDashboardUserFilter{
		keyword: strings.ToLower(strings.TrimSpace(r.URL.Query().Get("user_keyword"))),
		role:    strings.ToLower(strings.TrimSpace(r.URL.Query().Get("user_role"))),
		status:  strings.ToLower(strings.TrimSpace(r.URL.Query().Get("user_status"))),
	}
}

func (h *Handler) dashboardCandidateULIDs(r *http.Request, users []*casdoorsdk.User) map[string]string {
	candidateByUserID := make(map[string]string, len(users))
	for _, user := range users {
		if user == nil || strings.TrimSpace(user.Id) == "" {
			continue
		}
		candidateULID := h.dashboardCandidateULID(r, user.Id)
		if candidateULID != "" {
			candidateByUserID[user.Id] = candidateULID
		}
	}
	return candidateByUserID
}

func filterCFTPDashboardUsers(
	users []*casdoorsdk.User,
	roleConfig opsDashboardRoleConfig,
	roleDefinitions map[string]*casdoorsdk.Role,
) []*casdoorsdk.User {
	filtered := make([]*casdoorsdk.User, 0, len(users))
	for _, user := range users {
		if user == nil || strings.TrimSpace(user.Id) == "" {
			continue
		}
		roles := roleNames(user)
		roleFlags := dashboardRoleFlags(user, roles, roleConfig, roleDefinitions)
		if roleFlags["admin"] || roleFlags["student"] {
			filtered = append(filtered, user)
		}
	}
	return filtered
}

func filterOpsDashboardUsers(
	users []*casdoorsdk.User,
	candidateByUserID map[string]string,
	roleConfig opsDashboardRoleConfig,
	roleDefinitions map[string]*casdoorsdk.Role,
	filter opsDashboardUserFilter,
) []*casdoorsdk.User {
	filtered := make([]*casdoorsdk.User, 0, len(users))
	for _, user := range users {
		if user == nil || strings.TrimSpace(user.Id) == "" {
			continue
		}
		candidateULID := candidateByUserID[user.Id]
		roles := roleNames(user)
		roleFlags := dashboardRoleFlags(user, roles, roleConfig, roleDefinitions)
		roleLabel := dashboardPrimaryRole(roles, roleFlags)
		status := strings.ToLower(dashboardUserStatus(user))
		if filter.status != "" && filter.status != "all" && status != filter.status {
			continue
		}
		if filter.role != "" && filter.role != "all" {
			if !roleFlags[strings.ToLower(filter.role)] {
				continue
			}
		}
		if filter.keyword != "" {
			haystack := strings.ToLower(strings.Join([]string{
				dashboardUserName(user),
				user.Email,
				user.Phone,
				dashboardLocation(user),
				roleLabel,
				strings.Join(roles, " "),
				candidateULID,
				user.Id,
			}, " "))
			if !strings.Contains(haystack, filter.keyword) {
				continue
			}
		}
		filtered = append(filtered, user)
	}
	return filtered
}

func paginateOpsDashboardUsers(users []*casdoorsdk.User, page, pageSize int) []*casdoorsdk.User {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	start := (page - 1) * pageSize
	if start >= len(users) {
		return []*casdoorsdk.User{}
	}
	end := start + pageSize
	if end > len(users) {
		end = len(users)
	}
	return users[start:end]
}

func (h *Handler) buildOpsUserSummary(
	users []*casdoorsdk.User,
	pageUsers []*casdoorsdk.User,
	candidateByUserID map[string]string,
	roleConfig opsDashboardRoleConfig,
	roleDefinitions map[string]*casdoorsdk.Role,
) opsDashboardUserSummary {
	roleCounts := map[string]int64{
		"admin":   0,
		"student": 0,
		"member":  0,
	}
	stats := opsDashboardUserStats{}
	var profileScore int64

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

		roles := roleNames(user)
		for key := range dashboardRoleFlags(user, roles, roleConfig, roleDefinitions) {
			roleCounts[key]++
		}
		profileScore += int64(profileCompletionPercent(user))
	}
	stats.Admins = roleCounts["admin"]
	stats.Members = roleCounts["member"]

	pageItems := make([]opsDashboardUser, 0, len(pageUsers))
	for _, user := range pageUsers {
		if user == nil || user.Id == "" {
			continue
		}
		roles := roleNames(user)
		candidateULID := candidateByUserID[user.Id]
		roleFlags := dashboardRoleFlags(user, roles, roleConfig, roleDefinitions)
		pageItems = append(pageItems, opsDashboardUser{
			ID:            user.Id,
			CandidateULID: candidateULID,
			Name:          dashboardUserName(user),
			Email:         strings.TrimSpace(user.Email),
			Phone:         strings.TrimSpace(user.Phone),
			Location:      dashboardLocation(user),
			Roles:         roles,
			RoleLabel:     dashboardPrimaryRole(roles, roleFlags),
			Status:        dashboardUserStatus(user),
			EmailVerified: user.EmailVerified,
			CreatedAt:     user.CreatedTime,
		})
	}

	var completion int64
	if stats.Total > 0 {
		completion = profileScore / stats.Total
	}

	return opsDashboardUserSummary{
		candidateTotal: statsForStudents(roleCounts),
		stats:          stats,
		roleStats: []opsDashboardRoleStat{
			{Key: "admin", Label: "Admin", Count: roleCounts["admin"]},
			{Key: "student", Label: "Student", Count: roleCounts["student"]},
			{Key: "member", Label: "Member", Count: roleCounts["member"]},
		},
		profileCompletionPercent: completion,
		users:                    pageItems,
	}
}

func (h *Handler) dashboardCandidateULID(r *http.Request, userID string) string {
	if strings.TrimSpace(userID) == "" {
		return ""
	}
	resp, err := h.Gmid.GetUlidByUUID(r.Context(), &gmidpb.GetUlidByUUIDRequest{UserUuid: userID})
	if err != nil {
		return ""
	}
	return strings.TrimSpace(resp.GetUserUlid())
}

func statsForStudents(roleCounts map[string]int64) int64 {
	return roleCounts["student"]
}

func roleNames(user *casdoorsdk.User) []string {
	roles := make([]string, 0, len(user.Roles))
	for _, role := range user.Roles {
		if role == nil {
			continue
		}
		name := strings.TrimSpace(role.Name)
		if name == "" {
			name = strings.TrimSpace(role.DisplayName)
		}
		if name != "" {
			roles = append(roles, name)
		}
	}
	sort.Strings(roles)
	return roles
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

func dashboardPrimaryRole(roles []string, roleFlags map[string]bool) string {
	if roleFlags["admin"] {
		return "admin"
	}
	if roleFlags["member"] {
		return "member"
	}
	if roleFlags["student"] {
		return "student"
	}
	if len(roles) > 0 {
		return roles[0]
	}
	return "member"
}

func dashboardRoleConfigFromEnv() opsDashboardRoleConfig {
	return opsDashboardRoleConfig{
		admin:       envOrDefault(config.EnvRoleAdminBasic, defaultDashboardAdminRole),
		student:     envOrDefault(config.EnvRoleStudentBasic, defaultDashboardStudentRole),
		memberships: splitEnvList(envOrDefault(config.EnvRoleMembershipRoles, defaultDashboardMembershipRoles)),
	}
}

func dashboardRoleDefinitions(roleConfig opsDashboardRoleConfig) map[string]*casdoorsdk.Role {
	roleNames := make([]string, 0, 2+len(roleConfig.memberships))
	roleNames = append(roleNames, roleConfig.admin, roleConfig.student)
	roleNames = append(roleNames, roleConfig.memberships...)

	definitions := make(map[string]*casdoorsdk.Role, len(roleNames))
	seen := make(map[string]bool, len(roleNames))
	for _, roleName := range roleNames {
		roleName = strings.TrimSpace(roleName)
		normalized := strings.ToLower(roleName)
		if roleName == "" || seen[normalized] {
			continue
		}
		seen[normalized] = true

		role, err := casdoorsdk.GetRole(roleName)
		if err != nil {
			slog.Warn("dashboard role definition load failed", "role", roleName, "err", err)
			continue
		}
		if role != nil {
			definitions[normalized] = role
		}
	}
	return definitions
}

func dashboardRoleFlags(user *casdoorsdk.User, roles []string, roleConfig opsDashboardRoleConfig, roleDefinitions map[string]*casdoorsdk.Role) map[string]bool {
	flags := make(map[string]bool, 3)
	if dashboardUserMatchesRole(user, roles, roleConfig.admin, roleDefinitions) {
		flags["admin"] = true
	}
	if dashboardUserMatchesRole(user, roles, roleConfig.student, roleDefinitions) {
		flags["student"] = true
	}
	for _, roleName := range roleConfig.memberships {
		if dashboardUserMatchesRole(user, roles, roleName, roleDefinitions) {
			flags["member"] = true
			break
		}
	}
	return flags
}

func dashboardUserMatchesRole(user *casdoorsdk.User, roles []string, roleName string, roleDefinitions map[string]*casdoorsdk.Role) bool {
	roleName = strings.TrimSpace(roleName)
	if user == nil || roleName == "" {
		return false
	}
	normalizedRole := strings.ToLower(roleName)
	for _, role := range roles {
		if strings.EqualFold(role, roleName) || strings.EqualFold(dashboardRoleNamePart(role), roleName) {
			return true
		}
	}

	roleDefinition := roleDefinitions[normalizedRole]
	if roleDefinition == nil {
		return false
	}

	userKeys := dashboardUserKeys(user)
	for _, roleUser := range roleDefinition.Users {
		if userKeys[strings.ToLower(strings.TrimSpace(roleUser))] {
			return true
		}
		if userKeys[strings.ToLower(dashboardRoleNamePart(roleUser))] {
			return true
		}
	}

	groupKeys := dashboardUserGroupKeys(user)
	for _, roleGroup := range roleDefinition.Groups {
		if groupKeys[strings.ToLower(strings.TrimSpace(roleGroup))] {
			return true
		}
		if groupKeys[strings.ToLower(dashboardRoleNamePart(roleGroup))] {
			return true
		}
	}

	for _, inheritedRole := range roleDefinition.Roles {
		for _, userRole := range roles {
			if strings.EqualFold(userRole, inheritedRole) || strings.EqualFold(dashboardRoleNamePart(userRole), dashboardRoleNamePart(inheritedRole)) {
				return true
			}
		}
	}
	return false
}

func dashboardUserKeys(user *casdoorsdk.User) map[string]bool {
	keys := make(map[string]bool)
	addKey := func(value string) {
		value = strings.TrimSpace(value)
		if value != "" {
			keys[strings.ToLower(value)] = true
		}
	}
	addKey(user.Id)
	addKey(user.Name)
	addKey(user.Email)
	addKey(dashboardCasdoorPath(user.Owner, user.Id))
	addKey(dashboardCasdoorPath(user.Owner, user.Name))
	return keys
}

func dashboardUserGroupKeys(user *casdoorsdk.User) map[string]bool {
	keys := make(map[string]bool)
	for _, group := range user.Groups {
		group = strings.TrimSpace(group)
		if group == "" {
			continue
		}
		keys[strings.ToLower(group)] = true
		keys[strings.ToLower(dashboardRoleNamePart(group))] = true
	}
	return keys
}

func dashboardCasdoorPath(owner string, name string) string {
	owner = strings.TrimSpace(owner)
	name = strings.TrimSpace(name)
	if owner == "" || name == "" {
		return ""
	}
	return owner + "/" + name
}

func dashboardRoleNamePart(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	parts := strings.Split(value, "/")
	return strings.TrimSpace(parts[len(parts)-1])
}

func envOrDefault(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func splitEnvList(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
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
