package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	gmidpb "github.com/afnandelfin620-star/cftptest/cftp/gmid"
	"github.com/casdoor/casdoor-go-sdk/casdoorsdk"
)

// GetAdminMe GET /api/user/me
func (h *Handler) GetAdminMe(w http.ResponseWriter, r *http.Request) {
	name := AdminName(r)

	fullUser, err := casdoorsdk.GetUser(name)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, ErrInternal, "failed to get user info")
		return
	}

	WriteJSON(w, http.StatusOK, UserMeRsp{
		Name:        fullUser.Name,
		Email:       fullUser.Email,
		DisplayName: fullUser.DisplayName,
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
	name := AdminName(r)

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

	// fullUser.Email = input.Email
	fullUser.DisplayName = input.DisplayName
	fullUser.Affiliation = input.Affiliation
	fullUser.Title = input.Title
	fullUser.RealName = input.RealName
	fullUser.Bio = input.Bio
	fullUser.Gender = input.Gender
	fullUser.Birthday = input.Birthday
	fullUser.Education = input.Education

	if _, err := casdoorsdk.UpdateUser(fullUser); err != nil {
		slog.Error("Failed to update user", "error", err)
		WriteError(w, http.StatusInternalServerError, ErrProfileUpdateFailed, "failed to update user profile")
		return
	}

	WriteJSON(w, http.StatusOK, BaseRsp{Code: 0, Msg: "淇敼鎴愬姛"})
}

// UpdateUserPassword PUT /api/user/password
func (h *Handler) UpdateUserPassword(w http.ResponseWriter, r *http.Request) {
	name := AdminName(r)

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

	WriteJSON(w, http.StatusOK, BaseRsp{Code: 0, Msg: "瀵嗙爜淇敼鎴愬姛"})
}

// ListUsers GET /api/user/list
func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := casdoorsdk.GetUsers()
	if err != nil {
		slog.Error("Failed to list users", "error", err)
		WriteError(w, http.StatusInternalServerError, ErrInternal, "failed to get users")
		return
	}

	var res []map[string]interface{}
	for _, u := range users {
		name := u.DisplayName
		if name == "" {
			name = u.Name
		}

		// 閫氳繃 gmid 鑾峰彇鐪熷疄鐨?ULID
		gmidResp, err := h.Gmid.GetUlidByUUID(r.Context(), &gmidpb.GetUlidByUUIDRequest{
			UserUuid: u.Id,
		})
		if err != nil || gmidResp.UserUlid == "" {
			slog.Warn("Failed to get ULID for Casdoor User, skipping", "uuid", u.Id, "error", err)
			continue
		}

		res = append(res, map[string]interface{}{
			"id":    gmidResp.UserUlid,
			"name":  name,
			"email": u.Email,
		})
	}

	WriteJSON(w, http.StatusOK, map[string]interface{}{
		"users": res,
	})
}
