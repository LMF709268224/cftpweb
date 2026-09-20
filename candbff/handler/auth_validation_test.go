package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"candbff/config"
	"github.com/casdoor/casdoor-go-sdk/casdoorsdk"
)

func TestGetLoginURLDisablesCaching(t *testing.T) {
	casdoorsdk.InitConfig(
		"https://casdoor.internal",
		"client",
		"secret",
		"certificate",
		"organization",
		"candidate-app",
	)
	h := &Handler{CasdoorEndpoint: "https://login.example.com"}
	req := httptest.NewRequest(
		http.MethodGet,
		"https://candidate.example/api/auth/login-url?callback=https%3A%2F%2Fcandidate.example%2Fcallback",
		nil,
	)
	rec := httptest.NewRecorder()

	h.GetLoginURL(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GetLoginURL() status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control = %q, want no-store", got)
	}
	if got := rec.Header().Get("Pragma"); got != "no-cache" {
		t.Fatalf("Pragma = %q, want no-cache", got)
	}

	var response struct {
		Data AuthURLRsp `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode login response: %v", err)
	}
	signinURL, err := url.Parse(response.Data.URL)
	if err != nil {
		t.Fatalf("parse signin URL: %v", err)
	}
	state := strings.TrimSpace(signinURL.Query().Get("state"))
	if state == "" {
		t.Fatal("signin URL does not contain state")
	}
	if cookies := rec.Result().Cookies(); len(cookies) != 1 || cookies[0].Name != oauthStateCookieName || cookies[0].Value != state {
		t.Fatalf("OAuth state cookie = %+v, URL state = %q", cookies, state)
	}
}

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
