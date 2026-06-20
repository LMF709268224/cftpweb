package server

import (
	"log/slog"
	"net/http"
	"os"
	"strings"

	"adminbff/config"
	"adminbff/handler"

	gmidpb "github.com/LMF709268224/cftpproto/gmid"

	"github.com/casdoor/casdoor-go-sdk/casdoorsdk"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ContextKey 鐢ㄤ簬鍦?context 涓紶閫掔敤鎴蜂俊鎭?
type ContextKey = handler.ContextKey

const (
	CtxKeyAdminID = handler.CtxKeyAdminID
	CtxKeyEmail   = handler.CtxKeyEmail
	CtxKeyName    = handler.CtxKeyName
	CtxKeyToken   = handler.CtxKeyToken
)

// corsMiddleware 澶勭悊 CORS 璺ㄥ煙璇锋眰
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

// authMiddleware 楠岃瘉 Casdoor JWT 骞跺皢鐢ㄦ埛淇℃伅娉ㄥ叆 context
func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var tokenStr string

		// 浼樺厛浠?Cookie 璇诲彇
		cookie, err := r.Cookie("access_token")
		if err == nil && cookie.Value != "" {
			tokenStr = cookie.Value
		} else {
			// 鍚庨€€绛栫暐锛氫粠 Header 璇诲彇
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

		// 浣跨敤 Casdoor SDK 楠岃瘉 JWT 绛惧悕鍜屾湁鏁堟湡
		claims, err := casdoorsdk.ParseJwtToken(tokenStr)
		if err != nil {
			slog.Warn("JWT validation failed", "error", err)
			handler.WriteError(w, http.StatusUnauthorized, handler.ErrInvalidToken, "invalid or expired token")
			return
		}

		// 楠岃瘉鏄惁鏄鐞嗗憳
		if !handler.IsCftpAdmin(&claims.User) {
			slog.Warn("authMiddleware: user is not an admin", "casdoor_user_id", claims.User.Id)
			handler.WriteError(w, http.StatusForbidden, handler.ErrUnauthorized, "admin privileges required")
			return
		}

		// 璋冪敤 gmid 鏈嶅姟杩涜 UID 瑙ｆ瀽
		resp, err := s.grpcPool.Gmid.GetUlidByUUID(r.Context(), &gmidpb.GetUlidByUUIDRequest{
			UserUuid: claims.User.Id,
		})

		if err != nil {
			if status.Code(err) == codes.NotFound {
				// 鏈湪鍐呴儴寤虹珛鏄犲皠鍏崇郴锛屽彲鑳芥槸鏃犳晥鎴栧皻鏈畬鎴愬垵濮嬪寲
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

		// 娉ㄥ叆 context
		ctx := handler.WithCandidate(r.Context(), candidateID, claims.Email, claims.Name, tokenStr)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
