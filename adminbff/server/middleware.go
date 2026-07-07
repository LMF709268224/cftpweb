package server

import (
	"log/slog"
	"net/http"
	"os"
	"strings"

	"adminbff/config"
	"adminbff/handler"

	gmidpb "github.com/afnandelfin620-star/cftptest/cftp/gmid"

	"github.com/casdoor/casdoor-go-sdk/casdoorsdk"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ContextKey 用于在 context 中传递用户信息
type ContextKey = handler.ContextKey

const (
	CtxKeyAdminID = handler.CtxKeyAdminID
	CtxKeyEmail   = handler.CtxKeyEmail
	CtxKeyName    = handler.CtxKeyName
	CtxKeyToken   = handler.CtxKeyToken
)

// corsMiddleware 处理 CORS 跨域请求
func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		rawOrigins := os.Getenv(config.EnvCORSOrigins)

		if origin != "" && isAllowedOrigin(origin, rawOrigins) {
			if rawOrigins == "*" {
				w.Header().Set("Access-Control-Allow-Origin", "*")
			} else {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Max-Age", "86400")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func isAllowedOrigin(origin, raw string) bool {
	if raw == "" {
		return false
	}
	if raw == "*" {
		return true
	}
	for _, o := range strings.Split(raw, ",") {
		if strings.TrimSpace(o) == origin {
			return true
		}
	}
	return false
}

// authMiddleware 验证 Casdoor JWT 并将用户信息注入 context
func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var tokenStr string

		// 优先从 Cookie 读取
		cookie, err := r.Cookie("access_token")
		if err == nil && cookie.Value != "" {
			tokenStr = cookie.Value
		} else {
			// 后退策略：从 Header 读取
			authHeader := r.Header.Get("Authorization")
			if authHeader != "" {
				tokenStr = strings.TrimPrefix(authHeader, "Bearer ")
				if tokenStr == authHeader {
					tokenStr = ""
				}
			}
		}

		if tokenStr == "" {
			slog.Warn("authMiddleware: missing authorization token", "path", r.URL.Path)
			handler.WriteError(w, http.StatusUnauthorized, handler.ErrUnauthorized, "missing authorization token")
			return
		}

		slog.Info("authMiddleware: token found, parsing JWT...", "path", r.URL.Path)

		// 使用 Casdoor SDK 验证 JWT 签名和有效期
		claims, err := casdoorsdk.ParseJwtToken(tokenStr)
		if err != nil {
			slog.Warn("JWT validation failed", "error", err)
			handler.WriteError(w, http.StatusUnauthorized, handler.ErrInvalidToken, "invalid or expired token")
			return
		}

		casdoorCfg := s.config.SecretConfig.Casdoor
		if !handler.IsExpectedCasdoorApplication(tokenStr, claims, casdoorCfg.ClientID, casdoorCfg.AppName) {
			slog.Warn("authMiddleware: token was not issued for admin application", "casdoor_user_id", claims.User.Id, "path", r.URL.Path)
			handler.WriteError(w, http.StatusUnauthorized, handler.ErrInvalidToken, "token was not issued for the admin application")
			return
		}

		// 验证是否是管理员
		if !handler.IsCftpAdmin(&claims.User) {
			slog.Warn("authMiddleware: user is not an admin", "casdoor_user_id", claims.User.Id)
			handler.WriteError(w, http.StatusForbidden, handler.ErrUnauthorized, "admin privileges required")
			return
		}

		// 调用 gmid 服务进行 UID 解析
		resp, err := s.grpcPool.Gmid.GetUlidByUUID(r.Context(), &gmidpb.GetUlidByUUIDRequest{
			UserUuid: claims.User.Id,
		})

		if err != nil {
			if status.Code(err) == codes.NotFound {
				// 未在内部建立映射关系，可能是无效或尚未完成初始化
				slog.Error("authMiddleware: Candidate ULID not found in gmid for user", "casdoor_user_id", claims.User.Id, "user_name", claims.User.Name)
				handler.WriteError(w, http.StatusUnauthorized, handler.ErrUnauthorized, "user mapping not found")
				return
			}
			slog.Error("authMiddleware: Failed to resolve user ULID via gmid", "casdoor_user_id", claims.User.Id, "error", err)
			handler.WriteError(w, http.StatusInternalServerError, handler.ErrInternal, "internal error resolving id")
			return
		}

		candidateID := resp.UserUlid
		slog.Info("authMiddleware: Authentication successful", "candidate_id", candidateID, "user_name", claims.Name, "path", r.URL.Path)

		// 注入 context
		ctx := handler.WithCandidate(r.Context(), candidateID, claims.Email, claims.Name, tokenStr)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
