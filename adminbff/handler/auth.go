package handler

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"adminbff/config"

	"github.com/casdoor/casdoor-go-sdk/casdoorsdk"
)

const (
	oauthStateCookieName = "admin_oauth_state"
	oauthStateTTL        = 10 * time.Minute
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

// GetLoginURL  GET /api/auth/login-url
// 返回 Casdoor 登录页 URL，前端拿到后 redirect 用户到 Casdoor 完成登录
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

	// 这里如果一开始 init 的是 k8s 内网的地址, 就要重新拼接
	signinUrl := casdoorsdk.GetSigninUrl(redirectSigninURL)

	if h.CasdoorEndpoint != "" {
		// 如果配置了公网地址，就把 SDK 生成的内网 URL 替换为公网可访问的 URL
		if parsedUrl, err := url.Parse(signinUrl); err == nil {
			if parsedPublic, err := url.Parse(h.CasdoorEndpoint); err == nil {
				parsedUrl.Scheme = parsedPublic.Scheme
				parsedUrl.Host = parsedPublic.Host
				signinUrl = parsedUrl.String()
			}
		}
	}

	parsedSigninURL, err := url.Parse(signinUrl)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, ErrInternal, "failed to build login URL")
		return
	}
	query := parsedSigninURL.Query()
	query.Set("state", oauthState)
	parsedSigninURL.RawQuery = query.Encode()
	signinUrl = parsedSigninURL.String()
	setOAuthStateCookie(w, oauthState)

	WriteJSON(w, http.StatusOK, AuthURLRsp{URL: signinUrl})
}

// Login  POST /auth/login
// Casdoor OAuth 回调: 前端拿到 code 后调用此接口换取 JWT
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

	claims, err := casdoorsdk.ParseJwtToken(token.AccessToken)
	if err != nil {
		WriteError(w, http.StatusUnauthorized, ErrInvalidToken, "failed to parse token: "+err.Error())
		return
	}

	if !IsExpectedCasdoorApplication(token.AccessToken, claims, h.CasdoorClientId, h.CasdoorAppName) {
		WriteError(w, http.StatusUnauthorized, ErrInvalidToken, "token was not issued for the admin application")
		return
	}

	if !IsCftpAdmin(&claims.User) {
		WriteError(w, http.StatusForbidden, ErrAuthFailed, "only cftp admins are allowed to login")
		return
	}

	setTokenCookies(w, token.AccessToken, token.RefreshToken, token.Expiry)

	WriteJSON(w, http.StatusOK, LoginRsp{
		User: UserInfo{
			Name: claims.User.Name,
		},
	})
}

// RefreshToken  POST /auth/refresh
// 用 refresh_token 换取新的 access_token
func (h *Handler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	refreshToken := strings.TrimSpace(readRefreshToken(r))
	if refreshToken == "" {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "refresh_token is required")
		return
	}

	token, err := casdoorsdk.RefreshOAuthToken(refreshToken)
	if err != nil {
		WriteError(w, http.StatusUnauthorized, ErrAuthFailed, "failed to refresh token: "+err.Error())
		return
	}

	claims, err := casdoorsdk.ParseJwtToken(token.AccessToken)
	if err != nil {
		WriteError(w, http.StatusUnauthorized, ErrInvalidToken, "failed to parse token: "+err.Error())
		return
	}

	if !IsExpectedCasdoorApplication(token.AccessToken, claims, h.CasdoorClientId, h.CasdoorAppName) {
		WriteError(w, http.StatusUnauthorized, ErrInvalidToken, "token was not issued for the admin application")
		return
	}

	if !IsCftpAdmin(&claims.User) {
		WriteError(w, http.StatusForbidden, ErrAuthFailed, "only cftp admins are allowed to login")
		return
	}

	nextRefreshToken := token.RefreshToken
	if strings.TrimSpace(nextRefreshToken) == "" {
		nextRefreshToken = refreshToken
	}
	setTokenCookies(w, token.AccessToken, nextRefreshToken, token.Expiry)

	WriteJSON(w, http.StatusOK, LoginRsp{
		User: UserInfo{
			Name: claims.User.Name,
		},
	})
}

func readRefreshToken(r *http.Request) string {
	if cookie, err := r.Cookie("refresh_token"); err == nil && strings.TrimSpace(cookie.Value) != "" {
		return cookie.Value
	}
	return ""
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

// Logout  POST /api/auth/logout
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	clearTokenCookies(w)
	clearOAuthStateCookie(w)
	WriteJSON(w, http.StatusOK, BaseRsp{Code: 200, Msg: "logout success"})
}

func newOAuthState() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func setOAuthStateCookie(w http.ResponseWriter, state string) {
	http.SetCookie(w, &http.Cookie{
		Name:     oauthStateCookieName,
		Value:    state,
		Path:     "/api/auth",
		Expires:  time.Now().Add(oauthStateTTL),
		MaxAge:   int(oauthStateTTL.Seconds()),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
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
	clearOAuthStateCookie(w)
	cookie, err := r.Cookie(oauthStateCookieName)
	if err != nil {
		return false
	}
	expected := strings.TrimSpace(cookie.Value)
	provided = strings.TrimSpace(provided)
	if expected == "" || len(expected) != len(provided) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(expected), []byte(provided)) == 1
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

func IsCftpAdmin(user *casdoorsdk.User) bool {
	if user == nil {
		return false
	}

	adminRole := os.Getenv(config.EnvRoleAdminBasic)
	if adminRole == "" {
		adminRole = "role_admin_basic"
	}
	adminRole = strings.ToLower(strings.TrimSpace(adminRole))

	for _, role := range user.Roles {
		if role == nil {
			continue
		}
		if strings.ToLower(strings.TrimSpace(role.Name)) == adminRole {
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
