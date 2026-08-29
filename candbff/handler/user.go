package handler

import (
	"crypto/rand"
	"fmt"
	"log/slog"
	"math/big"
	"net/http"
	"strings"
	"time"

	gmailpb "github.com/afnandelfin620-star/cftptest/cftp/gmail"
	"github.com/afnandelfin620-star/cftptest/cftp/util"
	"github.com/casdoor/casdoor-go-sdk/casdoorsdk"
	"github.com/redis/go-redis/v9"
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
	emailVerificationTTL          = 5 * time.Minute
	emailVerificationSendCooldown = time.Minute
	maxEmailVerificationAttempts  = 5
)

var verifyEmailCodeScript = redis.NewScript(`
local stored = redis.call("GET", KEYS[1])
if not stored then
  return 0
end
if stored == ARGV[1] then
  redis.call("DEL", KEYS[1], KEYS[2])
  return 1
end
local attempts = redis.call("INCR", KEYS[2])
local ttl = redis.call("TTL", KEYS[1])
if ttl > 0 then
  redis.call("EXPIRE", KEYS[2], ttl)
else
  redis.call("EXPIRE", KEYS[2], ARGV[3])
end
if attempts >= tonumber(ARGV[2]) then
  redis.call("DEL", KEYS[1])
  return -2
end
return -1
`)

const (
	userPropProvince   = "province"
	userPropPostalCode = "postal_code"
	userPropRealName   = "realName"
	userPropRealNameV2 = "real_name"
)

// GetUserMe GET /api/user/me
func (h *Handler) GetUserMe(w http.ResponseWriter, r *http.Request) {
	name := CandidateName(r)

	fullUser, err := casdoorsdk.GetUser(name)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, ErrInternal, "failed to get user info")
		return
	}

	addressText := addressLine(fullUser.Address, 0)
	province := firstNonEmpty(addressLine(fullUser.Address, 1), getUserProperty(fullUser, userPropProvince))

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
	})
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

// SendEmailCode POST /api/user/profile/email/send-code
func (h *Handler) SendEmailCode(w http.ResponseWriter, r *http.Request) {
	name := CandidateName(r)

	var input EmailSendCodeInput
	if err := ReadJSON(r, &input); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}
	input.Email = strings.TrimSpace(input.Email)
	if input.Email == "" {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "email required")
		return
	}
	if h.Rdb == nil {
		WriteError(w, http.StatusServiceUnavailable, ErrServiceUnavailable, "email verification is unavailable")
		return
	}

	cooldownKey := "candbff:email_verification:send:" + name
	allowed, err := h.Rdb.SetNX(r.Context(), cooldownKey, "1", emailVerificationSendCooldown).Result()
	if err != nil {
		slog.Error("Failed to apply email verification send limit", "error", err)
		WriteError(w, http.StatusServiceUnavailable, ErrServiceUnavailable, "email verification is unavailable")
		return
	}
	if !allowed {
		WriteError(w, http.StatusTooManyRequests, ErrRateLimited, "verification code was requested too recently")
		return
	}
	keepCooldown := false
	defer func() {
		if !keepCooldown {
			_ = h.Rdb.Del(r.Context(), cooldownKey).Err()
		}
	}()

	fullUser, err := casdoorsdk.GetUser(name)
	if err != nil {
		slog.Error("Failed to get full user", "error", err)
		WriteError(w, http.StatusInternalServerError, ErrInternal, "failed to get user info")
		return
	}
	if input.Email == fullUser.Email {
		msg := "This is already your current email"
		if input.Lang == "zh" {
			msg = "这已经是你当前的邮箱了"
		}
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, msg)
		return
	}

	existingUser, err := casdoorsdk.GetUserByEmail(input.Email)
	if err != nil {
		slog.Error("Failed to check whether email is registered", "error", err)
		WriteError(w, http.StatusInternalServerError, ErrInternal, "failed to check email availability")
		return
	}
	if existingUser != nil {
		msg := "This email is already registered by another user"
		if input.Lang == "zh" {
			msg = "该邮箱已被其他用户注册"
		}
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, msg)
		return
	}

	code, err := newEmailVerificationCode()
	if err != nil {
		slog.Error("Failed to generate email verification code", "error", err)
		WriteError(w, http.StatusInternalServerError, ErrInternal, "failed to send email")
		return
	}
	cacheKey := "candbff:email_verification:" + name
	cacheValue := input.Email + ":" + code
	if _, err := h.Rdb.TxPipelined(r.Context(), func(pipe redis.Pipeliner) error {
		pipe.Set(r.Context(), cacheKey, cacheValue, emailVerificationTTL)
		pipe.Del(r.Context(), cacheKey+":attempts")
		return nil
	}); err != nil {
		slog.Error("Failed to cache verification code in Redis", "error", err)
		WriteError(w, http.StatusInternalServerError, ErrInternal, "failed to send email")
		return
	}

	content := fmt.Sprintf("Your verification code is: %s. It will expire in 5 minutes.", code)
	if input.Lang == "zh" {
		content = fmt.Sprintf("您的验证码是：%s。该验证码将在5分钟后过期。", code)
	}

	_, err = h.Gmail.CreateMailRaw(r.Context(), &gmailpb.CreateMailRawRequest{
		MailUlid:     util.NewULID(),
		BusinessUnit: "candbff",
		ToEmail:      input.Email,
		ToName:       fullUser.DisplayName,
		Subject:      "Verification Code",
		HtmlBody:     content,
	})
	if err != nil {
		_ = h.Rdb.Del(r.Context(), cacheKey).Err()
		slog.Error("Failed to send verification code", "error", err)
		WriteError(w, http.StatusInternalServerError, ErrInternal, "failed to send email")
		return
	}

	keepCooldown = true
	WriteJSON(w, http.StatusOK, BaseRsp{Code: 0, Msg: "success"})
}

func newEmailVerificationCode() (string, error) {
	value, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", value.Int64()), nil
}

// UpdateUserEmail PUT /api/user/profile/email
func (h *Handler) UpdateUserEmail(w http.ResponseWriter, r *http.Request) {
	name := CandidateName(r)

	var input EmailUpdateInput
	if err := ReadJSON(r, &input); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}
	input.Email = strings.TrimSpace(input.Email)
	input.VerificationCode = strings.TrimSpace(input.VerificationCode)
	if input.Email == "" || input.VerificationCode == "" {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "email and verification code are required")
		return
	}
	if h.Rdb == nil {
		WriteError(w, http.StatusServiceUnavailable, ErrServiceUnavailable, "email verification is unavailable")
		return
	}

	cacheKey := "candbff:email_verification:" + name
	attemptsKey := cacheKey + ":attempts"
	expectedValue := input.Email + ":" + input.VerificationCode
	verificationResult, err := verifyEmailCodeScript.Run(
		r.Context(),
		h.Rdb,
		[]string{cacheKey, attemptsKey},
		expectedValue,
		maxEmailVerificationAttempts,
		int(emailVerificationTTL.Seconds()),
	).Int64()
	if err != nil {
		slog.Error("Failed to verify email code", "error", err)
		WriteError(w, http.StatusServiceUnavailable, ErrServiceUnavailable, "email verification is unavailable")
		return
	}
	if verificationResult == -2 {
		WriteError(w, http.StatusTooManyRequests, ErrRateLimited, "too many invalid verification attempts")
		return
	}
	if verificationResult != 1 {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid or expired verification code")
		return
	}

	fullUser, err := casdoorsdk.GetUser(name)
	if err != nil {
		slog.Error("Failed to get full user", "error", err)
		WriteError(w, http.StatusInternalServerError, ErrInternal, "failed to get user info")
		return
	}

	fullUser.Email = input.Email

	_, err = casdoorsdk.UpdateUser(fullUser)
	if err != nil {
		slog.Error("Failed to update user email", "error", err)
		WriteError(w, http.StatusInternalServerError, ErrInternal, "failed to update user email")
		return
	}

	WriteJSON(w, http.StatusOK, BaseRsp{Code: 0, Msg: "success"})
}
