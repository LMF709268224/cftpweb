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
	gpaypb "github.com/afnandelfin620-star/cftptest/cftp/gpay"
	"github.com/casdoor/casdoor-go-sdk/casdoorsdk"
)

const (
	adminDashboardSampleLimit = 500
	adminDashboardPageSize    = 100
)

var (
	getDashboardRole = casdoorsdk.GetRole
)

const (
	defaultDashboardAdminRole       = "role_admin_basic"
	defaultDashboardStudentRole     = "role_student_basic"
	defaultDashboardMembershipRoles = "role_membership_affiliate,role_membership_associate,role_membership_charterholder"
)

type opsDashboardResponse struct {
	CandidateTotal         int64                       `json:"candidate_total"`
	UserStats              opsDashboardUserStats       `json:"user_stats"`
	UserRoleStats          []opsDashboardRoleStat      `json:"user_role_stats"`
	Users                  []opsDashboardUser          `json:"users"`
	UserTotal              int                         `json:"user_total"`
	UserPage               int                         `json:"user_page"`
	UserPageSize           int                         `json:"user_page_size"`
	TodayRevenue           []opsDashboardRevenue       `json:"today_revenue"`
	TodayRevenueExact      bool                        `json:"today_revenue_exact"`
	Revenue48h             []opsDashboardRevenueWindow `json:"revenue_48h"`
	NewCandidates48h       *int64                      `json:"new_candidates_48h"`
	UnverifiedCandidates   int64                       `json:"unverified_candidates"`
	AggregationSampleLimit int                         `json:"aggregation_sample_limit"`
	GeneratedAt            string                      `json:"generated_at"`
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
	UserGroup     string   `json:"user_group"`
	IsMember      bool     `json:"is_member"`
	Status        string   `json:"status"`
	EmailVerified bool     `json:"email_verified"`
	MFAEnabled    bool     `json:"mfa_enabled"`
	LastLoginAt   string   `json:"last_login_at"`
	CreatedAt     string   `json:"created_at"`
}

type opsDashboardRevenue struct {
	Currency    string `json:"currency"`
	AmountMinor int64  `json:"amount_minor"`
	OrderCount  int64  `json:"order_count"`
}

type opsDashboardRevenueWindow struct {
	Currency            string `json:"currency"`
	PreviousAmountMinor int64  `json:"previous_amount_minor"`
	CurrentAmountMinor  int64  `json:"current_amount_minor"`
}

type opsDashboardRoleConfig struct {
	admin       string
	student     string
	memberships []string
}

func (h *Handler) OpsDashboard(w http.ResponseWriter, r *http.Request) {
	users, err := h.CandidateProfiles.UsersOrRefresh(r.Context())
	if err != nil {
		slog.Error("Failed to initialize dashboard identity cache", "error", err)
		WriteError(w, http.StatusServiceUnavailable, ErrServiceUnavailable, "user identity cache is not ready")
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
	candidateByUserID := h.dashboardCandidateULIDs(cftpUsers)
	filteredUsers := filterOpsDashboardUsers(cftpUsers, candidateByUserID, roleConfig, roleDefinitions, opsDashboardUserFilterFromRequest(r))
	pageUsers := paginateOpsDashboardUsers(filteredUsers, userPage, userPageSize)
	userSummary := h.buildOpsUserSummary(cftpUsers, pageUsers, candidateByUserID, roleConfig, roleDefinitions)

	now := time.Now().UTC()
	orders, todayRevenueExact, err := h.listDashboardPaidOrders(r.Context(), now)
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	revenue, revenue48h := summarizeDashboardRevenue(orders, now)
	newCandidates48h, unverifiedCandidates := dashboardCandidateActivity(cftpUsers, roleConfig, roleDefinitions, now)

	WriteJSON(w, http.StatusOK, opsDashboardResponse{
		CandidateTotal:         userSummary.candidateTotal,
		UserStats:              userSummary.stats,
		UserRoleStats:          userSummary.roleStats,
		Users:                  userSummary.users,
		UserTotal:              len(filteredUsers),
		UserPage:               userPage,
		UserPageSize:           userPageSize,
		TodayRevenue:           revenue,
		TodayRevenueExact:      todayRevenueExact,
		Revenue48h:             revenue48h,
		NewCandidates48h:       newCandidates48h,
		UnverifiedCandidates:   unverifiedCandidates,
		AggregationSampleLimit: 0,
		GeneratedAt:            now.Format(time.RFC3339),
	})
}

func summarizeDashboardRevenue(orders []*gpaypb.OrderSummary, now time.Time) ([]opsDashboardRevenue, []opsDashboardRevenueWindow) {
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	previousStart := now.Add(-48 * time.Hour)
	currentStart := now.Add(-24 * time.Hour)
	today := make(map[string]*opsDashboardRevenue)
	windows := make(map[string]*opsDashboardRevenueWindow)
	for _, order := range orders {
		if order == nil || order.GetPaidAt() <= 0 {
			continue
		}
		paidAt := time.Unix(order.GetPaidAt(), 0).UTC()
		if paidAt.After(now) || paidAt.Before(previousStart) {
			continue
		}
		currency := strings.ToUpper(strings.TrimSpace(order.GetCurrency()))
		if currency == "" {
			currency = "UNKNOWN"
		}
		if !paidAt.Before(todayStart) {
			item := today[currency]
			if item == nil {
				item = &opsDashboardRevenue{Currency: currency}
				today[currency] = item
			}
			item.AmountMinor += order.GetAmount()
			item.OrderCount++
		}
		window := windows[currency]
		if window == nil {
			window = &opsDashboardRevenueWindow{Currency: currency}
			windows[currency] = window
		}
		if paidAt.Before(currentStart) {
			window.PreviousAmountMinor += order.GetAmount()
		} else {
			window.CurrentAmountMinor += order.GetAmount()
		}
	}
	dayResult := make([]opsDashboardRevenue, 0, len(today))
	windowResult := make([]opsDashboardRevenueWindow, 0, len(windows))
	for _, item := range today {
		dayResult = append(dayResult, *item)
	}
	for _, item := range windows {
		windowResult = append(windowResult, *item)
	}
	sort.Slice(dayResult, func(i, j int) bool { return dayResult[i].Currency < dayResult[j].Currency })
	sort.Slice(windowResult, func(i, j int) bool { return windowResult[i].Currency < windowResult[j].Currency })
	return dayResult, windowResult
}

func dashboardCandidateActivity(users []*casdoorsdk.User, roleConfig opsDashboardRoleConfig, roleDefinitions map[string]*casdoorsdk.Role, now time.Time) (*int64, int64) {
	var newCandidates, unverified int64
	completeDates := true
	for _, user := range users {
		if user == nil || !dashboardRoleFlags(user, roleNames(user), roleConfig, roleDefinitions)["candidate"] {
			continue
		}
		if !user.EmailVerified {
			unverified++
		}
		createdAt, err := time.Parse(time.RFC3339, user.CreatedTime)
		if err != nil {
			completeDates = false
			continue
		}
		if !createdAt.After(now) && !createdAt.Before(now.Add(-48*time.Hour)) {
			newCandidates++
		}
	}
	if !completeDates {
		return nil, unverified
	}
	return &newCandidates, unverified
}

func (h *Handler) listDashboardPaidOrders(ctx context.Context, now time.Time) ([]*gpaypb.OrderSummary, bool, error) {
	from := now.Add(-48 * time.Hour).Unix()
	to := now.Unix() + 1
	items := make([]*gpaypb.OrderSummary, 0, adminDashboardPageSize)
	cursor := ""
	seen := make(map[string]struct{})
	for {
		resp, err := h.Gpay.ListOrders(ctx, &gpaypb.ListOrdersRequest{
			Filters: &gpaypb.OrderAdminFilters{
				StatusFilter: gpaypb.OrderStatus_ORDER_STATUS_COMPLETED,
				PaidAtFrom:   &from,
				PaidAtTo:     &to,
			},
			Cursor:    cursor,
			PageSize:  adminDashboardPageSize,
			SortOrder: gpaypb.SortOrder_SORT_ORDER_DESC,
		})
		if err != nil {
			return nil, false, err
		}
		if resp == nil {
			return nil, false, fmt.Errorf("dashboard order response is missing")
		}
		if resp.GetHasMore() && len(resp.GetOrders()) == 0 {
			return nil, false, fmt.Errorf("dashboard order page is empty before the end")
		}
		for _, order := range resp.GetOrders() {
			if order == nil || order.GetPaidAt() < from || order.GetPaidAt() >= to {
				return nil, false, fmt.Errorf("dashboard order is outside the requested paid_at range")
			}
		}
		items = append(items, resp.GetOrders()...)
		if !resp.GetHasMore() {
			return items, true, nil
		}
		nextCursor := strings.TrimSpace(resp.GetNextCursor())
		if nextCursor == "" || nextCursor == cursor {
			return nil, false, fmt.Errorf("dashboard order cursor did not advance")
		}
		if _, ok := seen[nextCursor]; ok {
			return nil, false, fmt.Errorf("dashboard order cursor loop detected")
		}
		seen[nextCursor] = struct{}{}
		cursor = nextCursor
	}
}

type opsDashboardUserSummary struct {
	candidateTotal int64
	stats          opsDashboardUserStats
	roleStats      []opsDashboardRoleStat
	users          []opsDashboardUser
}

type opsDashboardUserFilter struct {
	keyword string
	role    string
	group   string
	status  string
}

func opsDashboardUserFilterFromRequest(r *http.Request) opsDashboardUserFilter {
	return opsDashboardUserFilter{
		keyword: strings.ToLower(strings.TrimSpace(r.URL.Query().Get("user_keyword"))),
		role:    strings.ToLower(strings.TrimSpace(r.URL.Query().Get("user_role"))),
		group:   strings.ToLower(strings.TrimSpace(r.URL.Query().Get("user_group"))),
		status:  strings.ToLower(strings.TrimSpace(r.URL.Query().Get("user_status"))),
	}
}

func (h *Handler) dashboardCandidateULIDs(users []*casdoorsdk.User) map[string]string {
	candidateByUserID := make(map[string]string, len(users))
	if h.CandidateProfiles == nil || !h.CandidateProfiles.Ready() {
		return candidateByUserID
	}
	for _, user := range users {
		if user == nil || strings.TrimSpace(user.Id) == "" {
			continue
		}
		candidateULID, ok := h.CandidateProfiles.ULIDForUUID(user.Id)
		if ok && candidateULID != "" {
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
	seen := make(map[string]bool, len(users))
	for _, user := range users {
		if user == nil || strings.TrimSpace(user.Id) == "" || seen[user.Id] {
			continue
		}
		roles := roleNames(user)
		roleFlags := dashboardRoleFlags(user, roles, roleConfig, roleDefinitions)
		if roleFlags["staff"] || roleFlags["candidate"] {
			filtered = append(filtered, user)
			seen[user.Id] = true
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
		if filter.group != "" && filter.group != "all" && !roleFlags[filter.group] {
			continue
		}
		if filter.status != "" && filter.status != "all" && status != filter.status {
			continue
		}
		if filter.role != "" && filter.role != "all" {
			matchesRole := roleFlags[filter.role]
			if filter.role == "student" {
				matchesRole = roleFlags["candidate"]
			} else if filter.role == "member" {
				matchesRole = roleFlags["candidate"] && roleFlags["member"]
			}
			if !matchesRole {
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
		roleFlags := dashboardRoleFlags(user, roles, roleConfig, roleDefinitions)
		if roleFlags["staff"] {
			roleCounts["admin"]++
		}
		if roleFlags["candidate"] {
			roleCounts["student"]++
			if roleFlags["member"] {
				roleCounts["member"]++
			}
		}
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
			UserGroup:     dashboardUserGroup(roleFlags),
			IsMember:      roleFlags["candidate"] && roleFlags["member"],
			Status:        dashboardUserStatus(user),
			EmailVerified: user.EmailVerified,
			// Match Casdoor's IsMfaEnabled; never expose MFA secrets or recovery codes.
			MFAEnabled:  user.PreferredMfaType != "",
			LastLoginAt: user.LastSigninTime,
			CreatedAt:   user.CreatedTime,
		})
	}

	return opsDashboardUserSummary{
		candidateTotal: statsForStudents(roleCounts),
		stats:          stats,
		roleStats: []opsDashboardRoleStat{
			{Key: "admin", Label: "Admin", Count: roleCounts["admin"]},
			{Key: "student", Label: "Student", Count: roleCounts["student"]},
			{Key: "member", Label: "Member", Count: roleCounts["member"]},
		},
		users: pageItems,
	}
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
	if roleFlags["candidate"] {
		return "student"
	}
	if len(roles) > 0 {
		return roles[0]
	}
	return ""
}

func dashboardUserGroup(roleFlags map[string]bool) string {
	if roleFlags["staff"] {
		return "staff"
	}
	return "candidate"
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

		role, err := getDashboardRole(roleName)
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
	flags := make(map[string]bool, 5)
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
	flags["staff"] = flags["admin"]
	flags["candidate"] = !flags["admin"] && (flags["student"] || flags["member"])
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
