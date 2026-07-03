package handler

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"candbff/config"
	gmidpb "github.com/afnandelfin620-star/cftptest/cftp/gmid"

	"github.com/casdoor/casdoor-go-sdk/casdoorsdk"
	"golang.org/x/oauth2"
)

func setTokenCookies(w http.ResponseWriter, accessToken, refreshToken string, expiresAt time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    accessToken,
		Expires:  expiresAt,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		Expires:  expiresAt.AddDate(0, 1, 0), // arbitrarily 1 month
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
	})
}

// GetLoginURL handles GET /api/auth/login-url.
func (h *Handler) GetLoginURL(w http.ResponseWriter, r *http.Request) {
	redirectSigninURL := r.URL.Query().Get("callback")
	signinURL := casdoorsdk.GetSigninUrl(redirectSigninURL)

	if h.CasdoorEndpoint != "" {
		if parsedURL, err := url.Parse(signinURL); err == nil {
			if parsedPublic, err := url.Parse(h.CasdoorEndpoint); err == nil {
				parsedURL.Scheme = parsedPublic.Scheme
				parsedURL.Host = parsedPublic.Host
				signinURL = parsedURL.String()
			}
		}
	}

	WriteJSON(w, http.StatusOK, AuthURLRsp{URL: signinURL})
}

// Login handles POST /auth/login.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var input LoginInput
	if err := ReadJSON(r, &input); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid request body: "+err.Error())
		return
	}

	if input.Code == "" || input.State == "" {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "code and state are required")
		return
	}

	token, err := casdoorsdk.GetOAuthToken(input.Code, input.State)
	if err != nil {
		WriteError(w, http.StatusUnauthorized, ErrAuthFailed, "failed to exchange token: "+err.Error())
		return
	}

	h.handleTokenExchange(w, r, token)
}

// RefreshToken handles POST /auth/refresh.
func (h *Handler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	var input struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := ReadJSON(r, &input); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid request body: "+err.Error())
		return
	}

	if input.RefreshToken == "" {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "refresh_token is required")
		return
	}

	token, err := casdoorsdk.RefreshOAuthToken(input.RefreshToken)
	if err != nil {
		WriteError(w, http.StatusUnauthorized, ErrAuthFailed, "failed to refresh token: "+err.Error())
		return
	}

	h.handleTokenExchange(w, r, token)
}

// handleTokenExchange is shared by Login and RefreshToken: parses claims, resolves the candidate
// ULID, sets auth cookies, and writes the success response.
func (h *Handler) handleTokenExchange(w http.ResponseWriter, r *http.Request, token *oauth2.Token) {
	claims, err := casdoorsdk.ParseJwtToken(token.AccessToken)
	if err != nil {
		WriteError(w, http.StatusUnauthorized, ErrInvalidToken, "failed to parse token: "+err.Error())
		return
	}

	if !IsExpectedCasdoorApplication(token.AccessToken, claims, h.CasdoorClientId, h.CasdoorAppName) {
		WriteError(w, http.StatusUnauthorized, ErrInvalidToken, "token was not issued for the candidate application")
		return
	}

	if !IsCftpStudent(&claims.User) {
		WriteError(w, http.StatusForbidden, ErrNotStudent, "only cftp students are allowed to login")
		return
	}

	if _, err = h.resolveCandidateUlid(r, claims.User.Id); err != nil {
		slog.Error("Failed to resolve user ULID", "error", err)
		WriteError(w, http.StatusInternalServerError, ErrInternal, "internal error")
		return
	}

	setTokenCookies(w, token.AccessToken, token.RefreshToken, token.Expiry)
	WriteJSON(w, http.StatusOK, LoginRsp{
		Token: token.AccessToken,
		User:  UserInfo{Name: claims.User.Name},
	})
}

func (h *Handler) resolveCandidateUlid(r *http.Request, casdoorUserUlid string) (string, error) {
	if casdoorUserUlid == "" {
		return "", fmt.Errorf("casdoor user id is required")
	}

	slog.Info("resolveCandidateUlid: starting ID resolution", "casdoor_user_id", casdoorUserUlid)
	resp, err := h.Gmid.GetUlidByUUID(r.Context(), &gmidpb.GetUlidByUUIDRequest{
		UserUuid: casdoorUserUlid,
	})
	if err != nil {
		slog.Warn("resolveCandidateUlid: failed to resolve user ULID from gmid", "casdoor_user_id", casdoorUserUlid, "error", err)
		return "", err
	}

	if resp.UserUlid == "" {
		err := fmt.Errorf("gmid returned empty user_ulid for user_uuid %q", casdoorUserUlid)
		slog.Warn("resolveCandidateUlid: empty user ULID from gmid", "casdoor_user_id", casdoorUserUlid)
		return "", err
	}

	slog.Info("resolveCandidateUlid: found existing mapping in gmid", "casdoor_user_id", casdoorUserUlid, "candidate_id", resp.UserUlid)
	return resp.UserUlid, nil
}

func clearTokenCookies(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    "",
		Expires:  time.Now().Add(-1 * time.Hour),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Expires:  time.Now().Add(-1 * time.Hour),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
	})
}

// Logout handles POST /api/auth/logout.
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	clearTokenCookies(w)
	WriteJSON(w, http.StatusOK, BaseRsp{Code: 200, Msg: "logout success"})
}

func IsCftpStudent(user *casdoorsdk.User) bool {
	if user == nil {
		return false
	}

	studentRole := os.Getenv(config.EnvRoleStudentBasic)
	if studentRole == "" {
		studentRole = "role_student_basic"
	}

	for _, role := range user.Roles {
		if role == nil {
			continue
		}
		if strings.EqualFold(role.Name, studentRole) {
			return true
		}
	}
	return false
}

func IsExpectedCasdoorApplication(tokenStr string, claims *casdoorsdk.Claims, expectedClientID, expectedAppName string) bool {
	expected := expectedCasdoorAudiences(expectedClientID, expectedAppName)
	if len(expected) == 0 {
		return false
	}

	for _, aud := range tokenAudiences(tokenStr, claims) {
		if _, ok := expected[strings.ToLower(strings.TrimSpace(aud))]; ok {
			return true
		}
	}
	return false
}

func expectedCasdoorAudiences(expectedClientID, expectedAppName string) map[string]struct{} {
	expected := make(map[string]struct{}, 2)
	for _, value := range []string{expectedClientID, expectedAppName} {
		value = strings.ToLower(strings.TrimSpace(value))
		if value != "" {
			expected[value] = struct{}{}
		}
	}
	return expected
}

func tokenAudiences(tokenStr string, claims *casdoorsdk.Claims) []string {
	audiences := make([]string, 0, 4)
	if claims != nil {
		for _, aud := range claims.RegisteredClaims.Audience {
			audiences = append(audiences, aud)
		}
	}

	payload, ok := decodeJWTPayload(tokenStr)
	if !ok {
		return audiences
	}

	for _, key := range []string{"aud", "azp", "client_id", "clientId", "application", "app", "appName"} {
		audiences = appendJSONClaimValues(audiences, payload[key])
	}
	return audiences
}

func decodeJWTPayload(tokenStr string) (map[string]interface{}, bool) {
	parts := strings.Split(tokenStr, ".")
	if len(parts) < 2 {
		return nil, false
	}

	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, false
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, false
	}
	return payload, true
}

func appendJSONClaimValues(values []string, raw interface{}) []string {
	switch v := raw.(type) {
	case string:
		return append(values, v)
	case []interface{}:
		for _, item := range v {
			if s, ok := item.(string); ok {
				values = append(values, s)
			}
		}
	}
	return values
}
