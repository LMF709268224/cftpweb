package handler

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/casdoor/casdoor-go-sdk/casdoorsdk"
)

const (
	userPropWorkPhone  = "work_phone"
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
		Name:        fullUser.Name,
		Email:       fullUser.Email,
		DisplayName: fullUser.DisplayName,
		FirstName:   fullUser.FirstName,
		LastName:    fullUser.LastName,
		Phone:       fullUser.Phone,
		HomePhone:   getUserProperty(fullUser, "home_phone"),
		WorkPhone:   getUserProperty(fullUser, userPropWorkPhone),
		Country:     fullUser.Region,
		Province:    province,
		City:        fullUser.Location,
		Region:      fullUser.Region,
		Location:    fullUser.Location,
		Address:     fullUser.Address,
		AddressText: addressText,
		PostalCode:  getUserProperty(fullUser, userPropPostalCode),
		Affiliation: fullUser.Affiliation,
		Title:       fullUser.Title,
		RealName:    userRealName(fullUser),
		Bio:         fullUser.Bio,
		Gender:      fullUser.Gender,
		Birthday:    fullUser.Birthday,
		Education:   fullUser.Education,
	})
}

// UpdateUserProfile PUT /api/user/profile
func (h *Handler) UpdateUserProfile(w http.ResponseWriter, r *http.Request) {
	name := CandidateName(r)

	var input UserProfileInput
	if err := ReadJSON(r, &input); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}
	if err := normalizeAndValidateUserProfileInput(&input); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, err.Error())
		return
	}

	fullUser, err := casdoorsdk.GetUser(name)
	if err != nil {
		slog.Error("Failed to get full user", "error", err)
		WriteError(w, http.StatusInternalServerError, ErrInternal, "failed to get user info")
		return
	}

	if strings.TrimSpace(fullUser.Email) == "" {
		fullUser.Email = strings.TrimSpace(input.Email)
	}
	fullUser.DisplayName = input.DisplayName
	fullUser.FirstName = input.FirstName
	fullUser.LastName = input.LastName
	fullUser.Phone = firstNonEmpty(input.HomePhone, input.Phone)
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
	setUserProperty(fullUser, userPropWorkPhone, input.WorkPhone)
	setUserProperty(fullUser, userPropProvince, input.Province)
	setUserProperty(fullUser, userPropPostalCode, input.PostalCode)
	setUserProperty(fullUser, userPropRealName, input.RealName)
	setUserProperty(fullUser, userPropRealNameV2, input.RealName)

	if _, err := casdoorsdk.UpdateUser(fullUser); err != nil {
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
		WriteError(w, http.StatusBadRequest, ErrPasswordIncorrect, "failed to change password: "+err.Error())
		return
	}

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
