package handler

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"sync"

	gccpb "github.com/afnandelfin620-star/cftptest/cftp/gcc"
	gcredspb "github.com/afnandelfin620-star/cftptest/cftp/gcreds"
	gmbrpb "github.com/afnandelfin620-star/cftptest/cftp/gmbr"
	gprogpb "github.com/afnandelfin620-star/cftptest/cftp/gprog"
	"github.com/casdoor/casdoor-go-sdk/casdoorsdk"
)

type profileUserStore interface {
	GetUser(name string) (*casdoorsdk.User, error)
	GetUserByPhone(phone string) (*casdoorsdk.User, error)
	UpdateUser(user *casdoorsdk.User) (bool, error)
}

type casdoorProfileUserStore struct{}

func (casdoorProfileUserStore) GetUser(name string) (*casdoorsdk.User, error) {
	return casdoorsdk.GetUser(name)
}

func (casdoorProfileUserStore) GetUserByPhone(phone string) (*casdoorsdk.User, error) {
	return casdoorsdk.GetUserByPhone(phone)
}

func (casdoorProfileUserStore) UpdateUser(user *casdoorsdk.User) (bool, error) {
	return casdoorsdk.UpdateUser(user)
}

func (h *Handler) getProfileUserStore() profileUserStore {
	if h.profileUsers != nil {
		return h.profileUsers
	}
	return casdoorProfileUserStore{}
}

const (
	userPropProvince   = "province"
	userPropPostalCode = "postal_code"
	userPropRealName   = "realName"
	userPropRealNameV2 = "real_name"
)

// GetUserMe GET /api/user/me
func (h *Handler) GetUserMe(w http.ResponseWriter, r *http.Request) {
	name := CandidateName(r)

	fullUser, err := h.getProfileUserStore().GetUser(name)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, ErrInternal, "failed to get user info")
		return
	}
	if fullUser == nil {
		WriteError(w, http.StatusNotFound, ErrNotFound, "user not found")
		return
	}

	addressText := addressLine(fullUser.Address, 0)
	province := firstNonEmpty(addressLine(fullUser.Address, 1), getUserProperty(fullUser, userPropProvince))
	accountStatus := h.loadUserAccountStatus(r.Context(), CandidateID(r), requestLocale(r))

	WriteJSON(w, http.StatusOK, UserMeRsp{
		Name:             fullUser.Name,
		Email:            fullUser.Email,
		DisplayName:      fullUser.DisplayName,
		FirstName:        fullUser.FirstName,
		LastName:         fullUser.LastName,
		PhoneCountryCode: fullUser.CountryCode,
		Phone:            fullUser.Phone,
		HomePhone:        getUserProperty(fullUser, "home_phone"),
		Country:          fullUser.Region,
		Province:         province,
		City:             fullUser.Location,
		Region:           fullUser.Region,
		Location:         fullUser.Location,
		Address:          fullUser.Address,
		AddressText:      addressText,
		PostalCode:       getUserProperty(fullUser, userPropPostalCode),
		Affiliation:      fullUser.Affiliation,
		Title:            fullUser.Title,
		RealName:         userRealName(fullUser),
		Bio:              fullUser.Bio,
		Gender:           fullUser.Gender,
		Birthday:         fullUser.Birthday,
		Education:        fullUser.Education,
		AccountStatus:    accountStatus,
	})
}

func (h *Handler) loadUserAccountStatus(ctx context.Context, candidateID, locale string) UserAccountStatusRsp {
	var membership UserMembershipStatusRsp
	var certification UserCertificationStatusRsp
	var qualification UserQualificationStatusRsp

	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		defer wg.Done()
		membership = h.loadUserMembershipStatus(ctx, candidateID, locale)
	}()
	go func() {
		defer wg.Done()
		certification = h.loadUserCertificationStatus(ctx, candidateID, locale)
	}()
	go func() {
		defer wg.Done()
		qualification = h.loadUserQualificationStatus(ctx, candidateID)
	}()
	wg.Wait()

	return UserAccountStatusRsp{
		Membership:    membership,
		Certification: certification,
		Qualification: qualification,
	}
}

func (h *Handler) loadUserMembershipStatus(ctx context.Context, candidateID, locale string) UserMembershipStatusRsp {
	out := UserMembershipStatusRsp{}
	if h.Gmbr == nil || strings.TrimSpace(candidateID) == "" {
		return out
	}

	resp, err := h.Gmbr.GetActiveMembership(ctx, &gmbrpb.GetActiveMembershipRequest{CandidateUlid: candidateID})
	if err != nil {
		slog.Warn("failed to load current user membership status", "candidate_id", candidateID, "error", err)
		return out
	}
	out.Available = true
	record := resp.GetMembership()
	if record == nil {
		return out
	}

	out.IsMember = true
	out.MembershipRecordULID = strings.TrimSpace(record.GetMembershipRecordUlid())
	out.MembershipULID = strings.TrimSpace(record.GetMembershipUlid())
	out.MembershipGpath = strings.TrimSpace(record.GetMembershipGpath())
	out.Status = strings.TrimSpace(record.GetStatus())
	out.ExpiresAt = strings.TrimSpace(record.GetExpiresAt())
	if out.MembershipULID == "" {
		return out
	}

	plan, err := h.Gmbr.GetMembership(ctx, &gmbrpb.GetMembershipRequest{MembershipUlid: out.MembershipULID})
	if err != nil {
		slog.Warn("failed to load current user membership plan", "candidate_id", candidateID, "membership_ulid", out.MembershipULID, "error", err)
		return out
	}
	plan = h.localizedMembership(ctx, plan, locale)
	out.PlanName = strings.TrimSpace(plan.GetName())
	out.TierLevel = plan.GetTierLevel()
	if out.MembershipGpath == "" {
		out.MembershipGpath = strings.TrimSpace(plan.GetMembershipGpath())
	}
	return out
}

func (h *Handler) loadUserCertificationStatus(ctx context.Context, candidateID, locale string) UserCertificationStatusRsp {
	out := UserCertificationStatusRsp{Programs: []UserCertificationProgram{}}
	if h.Gprog == nil || strings.TrimSpace(candidateID) == "" {
		return out
	}

	resp, err := h.Gprog.ListCandidatePipelines(ctx, &gprogpb.ListCandidatePipelinesReq{CandidateUlid: candidateID})
	if err != nil {
		slog.Warn("failed to load current user certification status", "candidate_id", candidateID, "error", err)
		return out
	}
	out.Available = true

	for _, pipeline := range resp.GetPipelines() {
		if pipeline == nil {
			continue
		}
		program := UserCertificationProgram{
			PipelineULID:       strings.TrimSpace(pipeline.GetPipelineUlid()),
			PipelineConfigULID: strings.TrimSpace(pipeline.GetPipelineCcUlid()),
			Status:             pipeline.GetStatus().String(),
		}
		if h.Gcc != nil && program.PipelineConfigULID != "" {
			config, configErr := h.Gcc.GetPipeline(ctx, &gccpb.GetPipelineRequest{
				Query: &gccpb.GetPipelineRequest_PipelineUlid{PipelineUlid: program.PipelineConfigULID},
			})
			if configErr != nil {
				slog.Warn("failed to load current user certification config", "candidate_id", candidateID, "pipeline_config_ulid", program.PipelineConfigULID, "error", configErr)
			} else {
				config = h.localizedPipeline(ctx, config, locale)
				program.PipelineGpath = strings.TrimSpace(config.GetPipelineGpath())
				program.Name = strings.TrimSpace(config.GetName())
			}
		}
		out.Programs = append(out.Programs, program)
	}
	out.PurchaseCount = len(out.Programs)
	out.IsCandidate = out.PurchaseCount > 0
	return out
}

func (h *Handler) loadUserQualificationStatus(ctx context.Context, candidateID string) UserQualificationStatusRsp {
	out := UserQualificationStatusRsp{}
	if h.Creds == nil || strings.TrimSpace(candidateID) == "" {
		return out
	}

	resp, err := h.Creds.GetCandidateCredentialCount(ctx, &gcredspb.GetCandidateCredentialCountRequest{
		CandidateUlid: candidateID,
		Limit:         1000,
	})
	if err != nil {
		slog.Warn("failed to load current user qualification status", "candidate_id", candidateID, "error", err)
		return out
	}
	out.Available = true
	out.CredentialCount = int(resp.GetCount())
	out.HasQualification = out.CredentialCount > 0
	return out
}

// UpdateUserProfile PUT /api/user/profile
func (h *Handler) UpdateUserProfile(w http.ResponseWriter, r *http.Request) {
	name := CandidateName(r)
	users := h.getProfileUserStore()

	var input UserProfileInput
	if err := ReadJSON(r, &input); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}
	if err := normalizeAndValidateUserProfileInput(&input); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, err.Error())
		return
	}

	fullUser, err := users.GetUser(name)
	if err != nil {
		slog.Error("Failed to get full user", "error", err)
		WriteError(w, http.StatusInternalServerError, ErrInternal, "failed to get user info")
		return
	}
	if fullUser == nil {
		WriteError(w, http.StatusNotFound, ErrNotFound, "user not found")
		return
	}

	if input.Phone != "" && input.Phone != normalizeProfilePhone(fullUser.Phone) {
		phoneUser, lookupErr := users.GetUserByPhone(input.Phone)
		if lookupErr != nil {
			slog.Error("Failed to check phone availability", "error", lookupErr)
			WriteError(w, http.StatusServiceUnavailable, ErrServiceUnavailable, "failed to check phone availability")
			return
		}
		if phoneUser != nil && (phoneUser.Owner != fullUser.Owner || phoneUser.Name != fullUser.Name) {
			WriteError(w, http.StatusConflict, ErrPhoneAlreadyInUse, "phone number is already in use")
			return
		}
	}

	// We no longer update email through this general profile endpoint.
	// Email updates have a dedicated endpoint with verification.

	fullUser.DisplayName = input.DisplayName
	fullUser.FirstName = input.FirstName
	fullUser.LastName = input.LastName
	fullUser.CountryCode = input.PhoneCountryCode
	fullUser.Phone = input.Phone
	fullUser.Region = input.Country
	fullUser.Location = input.City
	fullUser.Address = addressFromProfile(input.Address, input.Province)
	fullUser.Affiliation = input.Affiliation
	fullUser.Title = input.Title
	fullUser.RealName = input.RealName
	fullUser.Bio = input.Bio
	fullUser.Gender = input.Gender
	fullUser.Birthday = input.Birthday
	fullUser.Education = input.Education

	setUserProperty(fullUser, userPropProvince, input.Province)
	setUserProperty(fullUser, userPropPostalCode, input.PostalCode)
	setUserProperty(fullUser, userPropRealName, input.RealName)
	setUserProperty(fullUser, userPropRealNameV2, input.RealName)

	if _, err := users.UpdateUser(fullUser); err != nil {
		slog.Error("Failed to update user", "error", err)
		WriteError(w, http.StatusInternalServerError, ErrProfileUpdateFailed, "failed to update user profile")
		return
	}

	WriteJSON(w, http.StatusOK, BaseRsp{Code: 0, Msg: "success"})
}

// UpdateUserPassword PUT /api/user/password
func (h *Handler) UpdateUserPassword(w http.ResponseWriter, r *http.Request) {
	name := CandidateName(r)

	var input UserPasswordInput
	if err := ReadJSON(r, &input); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}
	if input.OldPassword == "" || input.NewPassword == "" {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "old_password and new_password are required")
		return
	}

	fullUser, err := casdoorsdk.GetUser(name)
	if err != nil {
		slog.Error("Failed to get full user", "error", err)
		WriteError(w, http.StatusInternalServerError, ErrInternal, "failed to get user info")
		return
	}

	owner := fullUser.Owner

	_, err = casdoorsdk.SetPassword(owner, name, input.OldPassword, input.NewPassword)
	if err != nil {
		slog.Error("Failed to set password", "error", err)
		WriteError(w, http.StatusBadRequest, ErrPasswordIncorrect, "current password is incorrect or the password change was rejected")
		return
	}

	clearTokenCookies(w, r)
	WriteJSON(w, http.StatusOK, BaseRsp{Code: 0, Msg: "success"})
}

func getUserProperty(user *casdoorsdk.User, key string) string {
	if user == nil || user.Properties == nil {
		return ""
	}
	return user.Properties[key]
}

func userRealName(user *casdoorsdk.User) string {
	if user == nil {
		return ""
	}
	return firstNonEmpty(user.RealName, getUserProperty(user, userPropRealName), getUserProperty(user, userPropRealNameV2))
}

func setUserProperty(user *casdoorsdk.User, key string, value string) {
	if user.Properties == nil {
		user.Properties = map[string]string{}
	}
	value = strings.TrimSpace(value)
	if value == "" {
		delete(user.Properties, key)
		return
	}
	user.Properties[key] = value
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func addressFromProfile(address string, province string) []string {
	address = strings.TrimSpace(address)
	province = strings.TrimSpace(province)
	if address == "" && province == "" {
		return nil
	}
	values := []string{address}
	if province != "" {
		values = append(values, province)
	}
	return values
}

func addressLine(address []string, index int) string {
	if index < 0 || index >= len(address) {
		return ""
	}
	return strings.TrimSpace(address[index])
}
