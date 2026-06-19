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
// 杩斿洖 Casdoor 鐧诲綍椤?URL锛屽墠绔嬁鍒板悗 redirect 鐢ㄦ埛鍒?Casdoor 瀹屾垚鐧诲綍
func (h *Handler) GetLoginURL(w http.ResponseWriter, r *http.Request) {
	// 榛樿涓嶉渶瑕佹彁渚?callback url锛孋asdoor 浼氫娇鐢ㄥ湪搴旂敤閰嶇疆涓～鍐欑殑 redirect_uri
	// 涔熷彲浠ヤ粠鍙傛暟鑾峰彇 callback 浼犲叆
	redirectSigninURL := r.URL.Query().Get("callback")
	// 杩欓噷濡傛灉涓€寮€濮媔nit鐨勬槸k8s鍐呯綉鐨勫湴鍧€,灏辫閲嶆柊鎷兼帴
	signinUrl := casdoorsdk.GetSigninUrl(redirectSigninURL)

	if h.CasdoorEndpoint != "" {
		// 濡傛灉閰嶇疆浜嗗叕缃戝湴鍧€锛屽氨鎶?SDK 鐢熸垚鐨勫唴缃?URL 鏇挎崲涓哄叕缃戝彲璁块棶鐨?URL
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
// Casdoor OAuth 鍥炶皟: 鍓嶇鎷垮埌 code 鍚庤皟鐢ㄦ鎺ュ彛鎹㈠彇 JWT
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
// 鐢?refresh_token 鎹㈠彇鏂扮殑 access_token
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
