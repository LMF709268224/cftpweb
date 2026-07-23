package handler

import (
	"crypto/rand"
	"crypto/subtle"
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

const (
	oauthStateCookieName = "cand_oauth_state"
	// newOAuthState returns base64url values, so a dot safely separates entries.
	oauthStateCookieSeparator = "."
	oauthStateTTL             = 10 * time.Minute
	maxOutstandingOAuthStates = 5
)

func setTokenCookies(w http.ResponseWriter, accessToken, refreshToken string, expiresAt time.Time) {
	if expiresAt.IsZero() || expiresAt.Before(time.Now()) {
		expiresAt = time.Now().Add(24 * time.Hour)
	}
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
	redirectSigninURL, err := validatedAuthCallback(r, r.URL.Query().Get("callback"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, err.Error())
		return
	}

	oauthState, err := newOAuthState()
	if err != nil {
		WriteError(w, http.StatusInternalServerError, ErrInternal, "failed to initialize login")
		return
	}

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

	parsedSigninURL, err := url.Parse(signinURL)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, ErrInternal, "failed to build login URL")
		return
	}
	query := parsedSigninURL.Query()
	query.Set("state", oauthState)
	parsedSigninURL.RawQuery = query.Encode()
	signinURL = parsedSigninURL.String()
	setOAuthStateCookie(w, r, oauthState)

	WriteJSON(w, http.StatusOK, AuthURLRsp{URL: signinURL})
}

// Login handles POST /auth/login.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var input LoginInput
	if err := ReadJSON(r, &input); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid request body: "+err.Error())
		return
	}

	input.Code = strings.TrimSpace(input.Code)
	input.State = strings.TrimSpace(input.State)
	if input.Code == "" || input.State == "" {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "code and state are required")
		return
	}
	if !consumeOAuthState(w, r, input.State) {
		WriteError(w, http.StatusUnauthorized, ErrAuthFailed, "invalid or expired login state")
		return
	}

	token, err := casdoorsdk.GetOAuthToken(input.Code, input.State)
	if err != nil {
		WriteError(w, http.StatusUnauthorized, ErrAuthFailed, "failed to exchange token: "+err.Error())
		return
	}

	h.handleTokenExchange(w, r, token, "")
}

// RefreshToken handles POST /auth/refresh.
func (h *Handler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	refreshToken := ""
	if cookie, err := r.Cookie("refresh_token"); err == nil {
		refreshToken = strings.TrimSpace(cookie.Value)
	}
	if refreshToken == "" {
		var input struct {
			RefreshToken string `json:"refresh_token"`
		}
		if err := ReadJSON(r, &input); err != nil {
			WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid request body: "+err.Error())
			return
		}
		refreshToken = strings.TrimSpace(input.RefreshToken)
	}
	if refreshToken == "" {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "refresh_token is required")
		return
	}

	token, err := casdoorsdk.RefreshOAuthToken(refreshToken)
	if err != nil {
		WriteError(w, http.StatusUnauthorized, ErrAuthFailed, "failed to refresh token: "+err.Error())
		return
	}

	h.handleTokenExchange(w, r, token, refreshToken)
}

// handleTokenExchange is shared by Login and RefreshToken: parses claims, resolves the candidate
// ULID, sets auth cookies, and writes the success response.
func (h *Handler) handleTokenExchange(w http.ResponseWriter, r *http.Request, token *oauth2.Token, currentRefreshToken string) {
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

	setTokenCookies(w, token.AccessToken, refreshTokenForCookie(token, currentRefreshToken), token.Expiry)
	WriteJSON(w, http.StatusOK, LoginRsp{
		User: UserInfo{Name: claims.User.Name},
	})
}

func refreshTokenForCookie(token *oauth2.Token, currentRefreshToken string) string {
	if token != nil {
		if refreshed := strings.TrimSpace(token.RefreshToken); refreshed != "" {
			return refreshed
		}
	}
	return strings.TrimSpace(currentRefreshToken)
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
	clearOAuthStateCookie(w)
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

func newOAuthState() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func setOAuthStateCookie(w http.ResponseWriter, r *http.Request, state string) {
	state = strings.TrimSpace(state)
	if state == "" {
		return
	}

	states := readOAuthStates(r)
	for _, existing := range states {
		if sameOAuthState(existing, state) {
			writeOAuthStateCookie(w, states)
			return
		}
	}

	states = append(states, state)
	if len(states) > maxOutstandingOAuthStates {
		states = states[len(states)-maxOutstandingOAuthStates:]
	}
	writeOAuthStateCookie(w, states)
}

func writeOAuthStateCookie(w http.ResponseWriter, states []string) {
	http.SetCookie(w, &http.Cookie{
		Name:     oauthStateCookieName,
		Value:    strings.Join(states, oauthStateCookieSeparator),
		Path:     "/api/auth",
		Expires:  time.Now().Add(oauthStateTTL),
		MaxAge:   int(oauthStateTTL.Seconds()),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
}

func readOAuthStates(r *http.Request) []string {
	cookie, err := r.Cookie(oauthStateCookieName)
	if err != nil {
		return nil
	}

	rawStates := strings.Split(strings.TrimSpace(cookie.Value), oauthStateCookieSeparator)
	states := make([]string, 0, min(len(rawStates), maxOutstandingOAuthStates))
	for _, state := range rawStates {
		state = strings.TrimSpace(state)
		if state == "" {
			continue
		}
		states = append(states, state)
	}
	if len(states) > maxOutstandingOAuthStates {
		states = states[len(states)-maxOutstandingOAuthStates:]
	}
	return states
}

func clearOAuthStateCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     oauthStateCookieName,
		Value:    "",
		Path:     "/api/auth",
		Expires:  time.Now().Add(-time.Hour),
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
}

func consumeOAuthState(w http.ResponseWriter, r *http.Request, provided string) bool {
	provided = strings.TrimSpace(provided)
	if provided == "" {
		return false
	}

	states := readOAuthStates(r)
	matched := false
	remaining := states[:0]
	for _, expected := range states {
		if sameOAuthState(expected, provided) {
			matched = true
			continue
		}
		remaining = append(remaining, expected)
	}
	if !matched {
		return false
	}

	if len(remaining) == 0 {
		clearOAuthStateCookie(w)
	} else {
		writeOAuthStateCookie(w, remaining)
	}
	return true
}

func sameOAuthState(expected, provided string) bool {
	return len(expected) == len(provided) &&
		subtle.ConstantTimeCompare([]byte(expected), []byte(provided)) == 1
}

func validatedAuthCallback(r *http.Request, raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	callback, err := url.ParseRequestURI(raw)
	if err != nil || !callback.IsAbs() || callback.User != nil {
		return "", fmt.Errorf("callback must be an absolute HTTP(S) URL")
	}
	if callback.Scheme != "http" && callback.Scheme != "https" {
		return "", fmt.Errorf("callback must use HTTP or HTTPS")
	}
	if callback.Host == "" || callback.Path != "/callback" || callback.RawQuery != "" || callback.Fragment != "" {
		return "", fmt.Errorf("callback must target the configured /callback page")
	}

	callbackOrigin := (&url.URL{Scheme: callback.Scheme, Host: callback.Host}).String()
	if sameAuthOrigin(callbackOrigin, requestOrigin(r)) || configuredAuthOrigin(callbackOrigin) || isLocalDevOrigin(callbackOrigin) {
		return callback.String(), nil
	}
	return "", fmt.Errorf("callback origin is not allowed")
}

func isLocalDevOrigin(origin string) bool {
	parsed, err := url.Parse(origin)
	if err != nil {
		return false
	}
	hostname := strings.ToLower(parsed.Hostname())
	return hostname == "localhost" || hostname == "127.0.0.1"
}

func requestOrigin(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if forwarded := strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-Proto"), ",")[0]); forwarded == "http" || forwarded == "https" {
		scheme = forwarded
	}
	return (&url.URL{Scheme: scheme, Host: strings.TrimSpace(r.Host)}).String()
}

func configuredAuthOrigin(origin string) bool {
	for _, allowed := range strings.Split(os.Getenv(config.EnvCORSOrigins), ",") {
		allowed = strings.TrimSpace(allowed)
		if allowed == "" || allowed == "*" {
			continue
		}
		parsed, err := url.Parse(allowed)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" {
			continue
		}
		allowedOrigin := (&url.URL{Scheme: parsed.Scheme, Host: parsed.Host}).String()
		if sameAuthOrigin(origin, allowedOrigin) {
			return true
		}
	}
	return false
}

func sameAuthOrigin(left, right string) bool {
	return strings.EqualFold(strings.TrimRight(left, "/"), strings.TrimRight(right, "/"))
}
