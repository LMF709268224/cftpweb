package handler

import (
	"net/http"
	"net/url"
	"os"
	"time"

	"adminbff/config"

	"github.com/casdoor/casdoor-go-sdk/casdoorsdk"
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

// GetLoginURL  GET /api/auth/login-url
// 返回 Casdoor 登录页 URL，前端拿到后 redirect 用户到 Casdoor 完成登录
func (h *Handler) GetLoginURL(w http.ResponseWriter, r *http.Request) {
	// 默认不需要提供 callback url，Casdoor 会使用在应用配置中填写的 redirect_uri
	// 也可以从参数获取 callback 传入
	redirectSigninURL := r.URL.Query().Get("callback")
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

	if input.Code == "" || input.State == "" {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "code and state are required")
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

	if !IsCftpAdmin(&claims.User) {
		WriteError(w, http.StatusForbidden, ErrAuthFailed, "only cftp admins are allowed to login")
		return
	}

	setTokenCookies(w, token.AccessToken, token.RefreshToken, token.Expiry)

	WriteJSON(w, http.StatusOK, LoginRsp{
		Token: token.AccessToken,
		User: UserInfo{
			Name: claims.User.Name,
		},
	})
}

// RefreshToken  POST /auth/refresh
// 用 refresh_token 换取新的 access_token
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

	claims, err := casdoorsdk.ParseJwtToken(token.AccessToken)
	if err != nil {
		WriteError(w, http.StatusUnauthorized, ErrInvalidToken, "failed to parse token: "+err.Error())
		return
	}

	if !IsCftpAdmin(&claims.User) {
		WriteError(w, http.StatusForbidden, ErrAuthFailed, "only cftp admins are allowed to login")
		return
	}

	setTokenCookies(w, token.AccessToken, token.RefreshToken, token.Expiry)

	WriteJSON(w, http.StatusOK, LoginRsp{
		Token: token.AccessToken,
		User: UserInfo{
			Name: claims.User.Name,
		},
	})
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
	WriteJSON(w, http.StatusOK, BaseRsp{Code: 200, Msg: "logout success"})
}

func IsCftpAdmin(user *casdoorsdk.User) bool {
	if user == nil {
		return false
	}

	adminRole := os.Getenv(config.EnvRoleAdminBasic)
	if adminRole == "" {
		adminRole = "role_admin_basic"
	}

	for _, role := range user.Roles {
		if role.Name == adminRole {
			return true
		}
	}
	return false
}
