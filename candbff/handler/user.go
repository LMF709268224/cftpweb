package handler

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
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
	GetUserByEmail(email string) (*casdoorsdk.User, error)
	GetUserByPhone(phone string) (*casdoorsdk.User, error)
	UpdateUser(user *casdoorsdk.User) (bool, error)
	UploadResource(user, tag, parent, path string, data []byte) (string, string, error)
	SendEmail(title, content, sender string, receivers ...string) error
}

type casdoorProfileUserStore struct{}

func (casdoorProfileUserStore) GetUser(name string) (*casdoorsdk.User, error) {
	return casdoorsdk.GetUser(name)
}

func (casdoorProfileUserStore) GetUserByEmail(email string) (*casdoorsdk.User, error) {
	return casdoorsdk.GetUserByEmail(email)
}

func (casdoorProfileUserStore) GetUserByPhone(phone string) (*casdoorsdk.User, error) {
	return casdoorsdk.GetUserByPhone(phone)
}

func (casdoorProfileUserStore) UpdateUser(user *casdoorsdk.User) (bool, error) {
	return casdoorsdk.UpdateUser(user)
}

func (casdoorProfileUserStore) UploadResource(user, tag, parent, path string, data []byte) (string, string, error) {
	return casdoorsdk.UploadResource(user, tag, parent, path, data)
}

func (casdoorProfileUserStore) SendEmail(title, content, sender string, receivers ...string) error {
	return casdoorsdk.SendEmail(title, content, sender, receivers...)
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
		Avatar:           fullUser.Avatar,
		Language:         fullUser.Language,
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

	fullUser.FirstName = input.FirstName
	fullUser.LastName = input.LastName
	fullUser.CountryCode = input.PhoneCountryCode
	fullUser.Phone = input.Phone
	fullUser.Region = input.Country
	fullUser.Location = input.City
	fullUser.Address = addressFromProfile(input.Address, input.Province)
	fullUser.Affiliation = input.Affiliation
	fullUser.Title = input.Title
	fullUser.Bio = input.Bio
	fullUser.Gender = input.Gender
	fullUser.Birthday = input.Birthday
	fullUser.Education = input.Education

	setUserProperty(fullUser, userPropProvince, input.Province)
	setUserProperty(fullUser, userPropPostalCode, input.PostalCode)
	if updated, err := users.UpdateUser(fullUser); err != nil || !updated {
		slog.Error("Failed to update user", "error", err)
		WriteError(w, http.StatusInternalServerError, ErrProfileUpdateFailed, "failed to update user profile")
		return
	}

	WriteJSON(w, http.StatusOK, BaseRsp{Code: 0, Msg: "success"})
}

// UpdateUserLanguage PUT /api/user/language
func (h *Handler) UpdateUserLanguage(w http.ResponseWriter, r *http.Request) {
	var input UserLanguageInput
	if err := ReadJSON(r, &input); err != nil || (input.Language != "zh" && input.Language != "en") {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "language must be zh or en")
		return
	}
	users := h.getProfileUserStore()
	fullUser, err := users.GetUser(CandidateName(r))
	if err != nil || fullUser == nil {
		WriteError(w, http.StatusServiceUnavailable, ErrServiceUnavailable, "failed to load user")
		return
	}
	fullUser.Language = input.Language
	if updated, err := users.UpdateUser(fullUser); err != nil || !updated {
		WriteError(w, http.StatusBadGateway, ErrProfileUpdateFailed, "failed to update language")
		return
	}
	WriteJSON(w, http.StatusOK, BaseRsp{Code: 0, Msg: "success"})
}

const maxAvatarBytes = 2 << 20

// UploadUserAvatar POST /api/user/avatar
func (h *Handler) UploadUserAvatar(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxAvatarBytes+1024)
	if err := r.ParseMultipartForm(maxAvatarBytes); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "avatar must be a JPEG or PNG under 2 MB")
		return
	}
	file, _, err := r.FormFile("avatar")
	if err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "avatar file is required")
		return
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxAvatarBytes+1))
	if err != nil || len(data) == 0 || len(data) > maxAvatarBytes {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "avatar must be under 2 MB")
		return
	}
	contentType := http.DetectContentType(data)
	if contentType != "image/jpeg" && contentType != "image/png" {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "avatar must be a JPEG or PNG")
		return
	}
	config, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || config.Width < 1 || config.Height < 1 || config.Width > 4096 || config.Height > 4096 || (format != "jpeg" && format != "png") {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid avatar image")
		return
	}
	users := h.getProfileUserStore()
	fullUser, err := users.GetUser(CandidateName(r))
	if err != nil || fullUser == nil {
		WriteError(w, http.StatusServiceUnavailable, ErrServiceUnavailable, "failed to load user")
		return
	}
	random := make([]byte, 12)
	if _, err := rand.Read(random); err != nil {
		WriteError(w, http.StatusInternalServerError, ErrInternal, "failed to prepare avatar")
		return
	}
	extension := ".jpg"
	if format == "png" {
		extension = ".png"
	}
	path := filepath.ToSlash(filepath.Join("avatars", hex.EncodeToString(random)+extension))
	avatarURL, _, err := users.UploadResource(fullUser.Name, "avatar", "", path, data)
	if err != nil || avatarURL == "" {
		slog.Error("Failed to upload avatar", "error", err)
		WriteError(w, http.StatusBadGateway, ErrProfileUpdateFailed, "failed to upload avatar")
		return
	}
	fullUser.Avatar = avatarURL
	if updated, err := users.UpdateUser(fullUser); err != nil || !updated {
		WriteError(w, http.StatusBadGateway, ErrProfileUpdateFailed, "failed to update avatar")
		return
	}
	WriteJSON(w, http.StatusOK, map[string]string{"avatar": avatarURL})
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
