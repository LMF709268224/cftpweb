package handler

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	gmidpb "github.com/afnandelfin620-star/cftptest/cftp/gmid"

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
	// 这里如果一开始init的是k8s内网的地址,就要重新拼接
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

	_, err = h.resolveCandidateId(r, claims.User.Id, token.AccessToken)
	if err != nil {
		slog.Error("Failed to resolve user ULID", "error", err)
		WriteError(w, http.StatusInternalServerError, ErrInternal, "internal error")
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

	_, err = h.resolveCandidateId(r, claims.User.Id, token.AccessToken)
	if err != nil {
		slog.Error("Failed to resolve user ULID", "error", err)
		WriteError(w, http.StatusInternalServerError, ErrInternal, "internal error")
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

func (h *Handler) resolveCandidateId(r *http.Request, casdoorUserId string, accessToken string) (string, error) {
	slog.Info("resolveCandidateId: starting ID resolution", "casdoor_user_id", casdoorUserId)

	// 1. 尝试从 gmid 查询
	resp, err := h.Gmid.GetUlidByUUID(r.Context(), &gmidpb.GetUlidByUUIDRequest{
		UserUuid: casdoorUserId,
	})

	if err != nil {
		slog.Warn("resolveCandidateId: mapping not found in gmid, attempting to fetch from Casdoor API", "casdoor_user_id", casdoorUserId)
		slog.Warn("resolveCandidateId: gRPC call to GetCandidateByUser failed", "error", err)
		// 2. 如果没找到，从 Casdoor API 抓取 UID
		// 由于现在不在前端返回，后端静默进行第一次抓取并持久化
		data, getErr := getUserIds(h.CasdoorEndpoint, casdoorUserId, UserIdType_UUID, h.CasdoorClientId, h.CasdoorClientSecret)
		if getErr != nil {
			slog.Error("resolveCandidateId: failed to fetch user IDs from Casdoor", "error", getErr)
			return "", getErr
		}
		ulid := data.Ulid
		slog.Info("resolveCandidateId: successfully fetched IDs from Casdoor", "ulid", ulid, "uuid", data.Uuid, "id", data.Id)

		if ulid == "" {
			slog.Warn("resolveCandidateId: Casdoor returned empty ULID!", "casdoor_user_id", casdoorUserId)
		}

		slog.Info("resolveCandidateId: successfully linked user in gmid", "casdoor_user_id", casdoorUserId, "ulid", ulid)

		// 添加默认角色

		return ulid, nil
	}

	slog.Info("resolveCandidateId: found existing mapping in gmid", "casdoor_user_id", casdoorUserId, "candidate_id", resp.UserUlid)
	return resp.UserUlid, nil
}

type UserIdType string

const (
	UserIdType_ULID UserIdType = "ulid"
	UserIdType_UUID UserIdType = "uuid"
	UserIdType_ID   UserIdType = "id"
)

type CasdoorGetIdsData struct {
	Ulid string `json:"ulid"`
	Uuid string `json:"uuid"`
	Id   string `json:"id"`
}

type CasdoorGetIdsResponse struct {
	Status string            `json:"status"`
	Msg    string            `json:"msg"`
	Data   CasdoorGetIdsData `json:"data"`
}

func getUserIds(endpoint, idValue string, idType UserIdType, clientId, clientSecret string) (CasdoorGetIdsData, error) {
	// 构造 Query 参数
	params := url.Values{}
	switch idType {
	case UserIdType_ID:
		params.Set("byId", idValue)
	case UserIdType_ULID:
		params.Set("byUid", idValue)
	case UserIdType_UUID:
		params.Set("byUUID", idValue)
	default:
		return CasdoorGetIdsData{}, fmt.Errorf("invalid id type: %s", idType)
	}

	// 拼接完整的 URL
	fullURL := fmt.Sprintf("%s/api/get-user-ids?%s", endpoint, params.Encode())

	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return CasdoorGetIdsData{}, fmt.Errorf("failed to create request: %w", err)
	}

	// 注入 Bearer Token 鉴权头 (auth 变量传入的是 accessToken，必须用 Bearer)
	// 如果你的 Casdoor 接口确实需要 clientId:clientSecret，请恢复 base64 的 Basic auth 代码。
	auth := base64.StdEncoding.EncodeToString([]byte(clientId + ":" + clientSecret))
	req.Header.Set("Authorization", "Basic "+auth)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return CasdoorGetIdsData{}, fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		slog.Error("getUserIds: Casdoor API returned non-200 status", "status", resp.StatusCode)
		return CasdoorGetIdsData{}, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var response CasdoorGetIdsResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		slog.Error("getUserIds: failed to decode Casdoor response", "error", err)
		return CasdoorGetIdsData{}, fmt.Errorf("failed to decode response: %w", err)
	}

	// 校验 Casdoor 的业务状态码
	if response.Status != "ok" {
		slog.Error("getUserIds: Casdoor returned business error", "msg", response.Msg)
		return CasdoorGetIdsData{}, fmt.Errorf("casdoor error: %s", response.Msg)
	}

	slog.Info("getUserIds: successfully parsed Casdoor response", "data", response.Data)
	return response.Data, nil
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
