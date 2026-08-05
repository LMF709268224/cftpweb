package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"candbff/config"
)

func TestLoginRejectsMissingAuthorizationResponse(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)

	(&Handler{}).Login(recorder, request)

	assertHandlerAPIError(t, recorder, http.StatusBadRequest, ErrInvalidRequest)
}

func TestRefreshTokenRejectsMissingToken(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", nil)

	(&Handler{}).RefreshToken(recorder, request)

	assertHandlerAPIError(t, recorder, http.StatusUnauthorized, ErrUnauthorized)
}

func TestValidatedAuthCallback(t *testing.T) {
	tests := []struct {
		name       string
		requestURL string
		callback   string
		corsOrigin string
		want       string
		wantError  bool
	}{
		{
			name:       "same request origin",
			requestURL: "https://candidate.example/api/auth/login-url",
			callback:   "https://candidate.example/callback",
			want:       "https://candidate.example/callback",
		},
		{
			name:       "configured origin",
			requestURL: "https://bff.example/api/auth/login-url",
			callback:   "https://candidate.example/callback",
			corsOrigin: "https://candidate.example",
			want:       "https://candidate.example/callback",
		},
		{
			name:       "localhost development origin",
			requestURL: "https://bff.example/api/auth/login-url",
			callback:   "http://localhost:5173/callback",
			want:       "http://localhost:5173/callback",
		},
		{
			name:       "external origin",
			requestURL: "https://candidate.example/api/auth/login-url",
			callback:   "https://evil.example/callback",
			wantError:  true,
		},
		{
			name:       "wrong callback path",
			requestURL: "https://candidate.example/api/auth/login-url",
			callback:   "https://candidate.example/orders",
			wantError:  true,
		},
		{
			name:       "callback query rejected",
			requestURL: "https://candidate.example/api/auth/login-url",
			callback:   "https://candidate.example/callback?next=/orders",
			wantError:  true,
		},
		{
			name:       "non HTTP scheme",
			requestURL: "https://candidate.example/api/auth/login-url",
			callback:   "javascript:alert(1)",
			wantError:  true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv(config.EnvCORSOrigins, test.corsOrigin)
			request := httptest.NewRequest(http.MethodGet, test.requestURL, nil)

			got, err := validatedAuthCallback(request, test.callback)
			if test.wantError {
				if err == nil {
					t.Fatalf("validatedAuthCallback() = %q, want error", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("validatedAuthCallback() error = %v", err)
			}
			if got != test.want {
				t.Fatalf("validatedAuthCallback() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestLogoutClearsAuthenticationAndOAuthState(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "https://candidate.example/api/auth/logout", nil)

	(&Handler{}).Logout(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%q", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	cookies := recorder.Result().Cookies()
	requiredExpiredCookies := map[string]bool{
		accessTokenCookieName:  false,
		refreshTokenCookieName: false,
		oauthStateCookieName:   false,
	}
	for _, cookie := range cookies {
		if _, required := requiredExpiredCookies[cookie.Name]; required && cookie.MaxAge < 0 {
			requiredExpiredCookies[cookie.Name] = true
		}
	}
	for name, expired := range requiredExpiredCookies {
		if !expired {
			t.Errorf("logout did not expire cookie %q", name)
		}
	}
}
