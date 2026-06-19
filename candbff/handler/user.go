package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/casdoor/casdoor-go-sdk/casdoorsdk"
)

const (
	userPropWorkPhone  = "work_phone"
	userPropProvince   = "province"
	userPropPostalCode = "postal_code"
)

// GetUserMe GET /api/user/me
func (h *Handler) GetUserMe(w http.ResponseWriter, r *http.Request) {
	name := CandidateName(r)

	fullUser, err := casdoorsdk.GetUser(name)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, ErrInternal, "failed to get user info")
		return
	}

	addressText := strings.Join(fullUser.Address, ", ")

	WriteJSON(w, http.StatusOK, UserMeRsp{
		Name:        fullUser.Name,
		Email:       fullUser.Email,
		DisplayName: fullUser.DisplayName,
		FirstName:   fullUser.FirstName,
		LastName:    fullUser.LastName,
		Phone:       fullUser.Phone,
		HomePhone:   fullUser.Phone,
		WorkPhone:   getUserProperty(fullUser, userPropWorkPhone),
		Country:     fullUser.Region,
		Province:    getUserProperty(fullUser, userPropProvince),
		City:        fullUser.Location,
		Region:      fullUser.Region,
		Location:    fullUser.Location,
		Address:     fullUser.Address,
		AddressText: addressText,
		PostalCode:  getUserProperty(fullUser, userPropPostalCode),
		Affiliation: fullUser.Affiliation,
		Title:       fullUser.Title,
		RealName:    fullUser.RealName,
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
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
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
	fullUser.Address = addressFromText(input.Address)
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

	if _, err := casdoorsdk.UpdateUser(fullUser); err != nil {
		slog.Error("Failed to update user", "error", err)
		WriteError(w, http.StatusInternalServerError, ErrProfileUpdateFailed, "failed to update user profile")
		return
	}

	WriteJSON(w, http.StatusOK, BaseRsp{Code: 0, Msg: "修改成功"})
}

// UpdateUserPassword PUT /api/user/password
func (h *Handler) UpdateUserPassword(w http.ResponseWriter, r *http.Request) {
	name := CandidateName(r)

	var input UserPasswordInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
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

	WriteJSON(w, http.StatusOK, BaseRsp{Code: 0, Msg: "密码修改成功"})
}

func getUserProperty(user *casdoorsdk.User, key string) string {
	if user == nil || user.Properties == nil {
		return ""
	}
	return user.Properties[key]
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

func addressFromText(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return []string{value}
}
