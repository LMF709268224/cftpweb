package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"adminbff/config"

	"github.com/casdoor/casdoor-go-sdk/casdoorsdk"
)

func TestGetLoginURLBindsStateCookie(t *testing.T) {
	casdoorsdk.InitConfig(
		"https://casdoor.internal",
		"client",
		"secret",
		"certificate",
		"organization",
		"admin-app",
	)
	h := &Handler{CasdoorEndpoint: "https://login.example.com"}
	req := httptest.NewRequest(
		http.MethodGet,
		"https://admin.example.com/api/auth/login-url?callback=https%3A%2F%2Fadmin.example.com%2Fcallback",
		nil,
	)
	rec := httptest.NewRecorder()

	h.GetLoginURL(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GetLoginURL() status = %d, body = %s", rec.Code, rec.Body.String())
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
	if signinURL.Host != "login.example.com" {
		t.Fatalf("signin host = %q, want login.example.com", signinURL.Host)
	}

	state := signinURL.Query().Get("state")
	if state == "" {
		t.Fatal("signin URL does not contain state")
	}
	if redirectURI := signinURL.Query().Get("redirect_uri"); redirectURI != "https://admin.example.com/callback" {
		t.Fatalf("redirect_uri = %q", redirectURI)
	}

	cookies := rec.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != oauthStateCookieName || cookies[0].Value != state {
		t.Fatalf("OAuth state cookie = %+v, URL state = %q", cookies, state)
	}
}

func TestValidatedAuthCallback(t *testing.T) {
	t.Run("accepts same origin callback", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "https://admin.example.com/api/auth/login-url", nil)

		got, err := validatedAuthCallback(req, "https://admin.example.com/callback")
		if err != nil {
			t.Fatalf("validatedAuthCallback() error = %v", err)
		}
		if got != "https://admin.example.com/callback" {
			t.Fatalf("validatedAuthCallback() = %q", got)
		}
	})

	t.Run("accepts explicitly configured origin", func(t *testing.T) {
		t.Setenv(config.EnvCORSOrigins, "https://admin.example.com")
		req := httptest.NewRequest(http.MethodGet, "http://adminbff.internal/api/auth/login-url", nil)

		if _, err := validatedAuthCallback(req, "https://admin.example.com/callback"); err != nil {
			t.Fatalf("validatedAuthCallback() error = %v", err)
		}
	})

	t.Run("rejects untrusted origin", func(t *testing.T) {
		t.Setenv(config.EnvCORSOrigins, "https://admin.example.com")
		req := httptest.NewRequest(http.MethodGet, "https://admin.example.com/api/auth/login-url", nil)

		if _, err := validatedAuthCallback(req, "https://evil.example/callback"); err == nil {
			t.Fatal("validatedAuthCallback() error = nil, want untrusted origin error")
		}
	})

	t.Run("rejects unexpected callback path", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "https://admin.example.com/api/auth/login-url", nil)

		if _, err := validatedAuthCallback(req, "https://admin.example.com/other"); err == nil {
			t.Fatal("validatedAuthCallback() error = nil, want callback path error")
		}
	})
}

func TestOAuthStateIsRandomAndSingleUse(t *testing.T) {
	first, err := newOAuthState()
	if err != nil {
		t.Fatalf("newOAuthState() error = %v", err)
	}
	second, err := newOAuthState()
	if err != nil {
		t.Fatalf("newOAuthState() second error = %v", err)
	}
	if first == "" || first == second {
		t.Fatalf("OAuth states are not unique: %q and %q", first, second)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
	req.AddCookie(&http.Cookie{Name: oauthStateCookieName, Value: first})
	rec := httptest.NewRecorder()
	if !consumeOAuthState(rec, req, first) {
		t.Fatal("consumeOAuthState() = false, want true")
	}

	cookies := rec.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != oauthStateCookieName || cookies[0].MaxAge != -1 {
		t.Fatalf("state deletion cookie = %+v", cookies)
	}
}

func TestOAuthStateRejectsMismatch(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
	req.AddCookie(&http.Cookie{Name: oauthStateCookieName, Value: "expected"})
	rec := httptest.NewRecorder()

	if consumeOAuthState(rec, req, "provided") {
		t.Fatal("consumeOAuthState() = true, want false")
	}
	if cookies := rec.Result().Cookies(); len(cookies) != 0 {
		t.Fatalf("mismatched state changed cookies: %+v", cookies)
	}
}

func TestOAuthStateCookieSupportsConcurrentLogins(t *testing.T) {
	first, err := newOAuthState()
	if err != nil {
		t.Fatalf("newOAuthState() error = %v", err)
	}
	second, err := newOAuthState()
	if err != nil {
		t.Fatalf("newOAuthState() second error = %v", err)
	}

	firstReq := httptest.NewRequest(http.MethodGet, "/api/auth/login-url", nil)
	firstRec := httptest.NewRecorder()
	setOAuthStateCookie(firstRec, firstReq, first)
	firstCookie := oauthStateCookieFromRecorder(t, firstRec)

	secondReq := httptest.NewRequest(http.MethodGet, "/api/auth/login-url", nil)
	secondReq.AddCookie(firstCookie)
	secondRec := httptest.NewRecorder()
	setOAuthStateCookie(secondRec, secondReq, second)
	combinedCookie := oauthStateCookieFromRecorder(t, secondRec)

	firstCallback := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
	firstCallback.AddCookie(combinedCookie)
	firstCallbackRec := httptest.NewRecorder()
	if !consumeOAuthState(firstCallbackRec, firstCallback, first) {
		t.Fatal("consumeOAuthState(first) = false, want true")
	}

	remainingCookie := oauthStateCookieFromRecorder(t, firstCallbackRec)
	remainingReq := httptest.NewRequest(http.MethodGet, "/api/auth/login-url", nil)
	remainingReq.AddCookie(remainingCookie)
	remainingStates := readOAuthStates(remainingReq)
	if len(remainingStates) != 1 || remainingStates[0] != second {
		t.Fatalf("remaining states = %q, want only second state", remainingStates)
	}

	secondCallback := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
	secondCallback.AddCookie(remainingCookie)
	secondCallbackRec := httptest.NewRecorder()
	if !consumeOAuthState(secondCallbackRec, secondCallback, second) {
		t.Fatal("consumeOAuthState(second) = false, want true")
	}
	cleared := oauthStateCookieFromRecorder(t, secondCallbackRec)
	if cleared.MaxAge != -1 {
		t.Fatalf("state cookie MaxAge = %d, want -1", cleared.MaxAge)
	}
}

func TestOAuthStateCookieCapsOutstandingStates(t *testing.T) {
	var current *http.Cookie
	for i := 0; i < maxOutstandingOAuthStates+1; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/auth/login-url", nil)
		if current != nil {
			req.AddCookie(current)
		}
		rec := httptest.NewRecorder()
		setOAuthStateCookie(rec, req, string(rune('a'+i)))
		current = oauthStateCookieFromRecorder(t, rec)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/auth/login-url", nil)
	req.AddCookie(current)
	states := readOAuthStates(req)
	if len(states) != maxOutstandingOAuthStates {
		t.Fatalf("state count = %d, want %d", len(states), maxOutstandingOAuthStates)
	}
	if states[0] != "b" || states[len(states)-1] != "f" {
		t.Fatalf("retained states = %q, want newest states b through f", states)
	}
}

func oauthStateCookieFromRecorder(t *testing.T, recorder *httptest.ResponseRecorder) *http.Cookie {
	t.Helper()
	for _, cookie := range recorder.Result().Cookies() {
		if cookie.Name == oauthStateCookieName {
			return cookie
		}
	}
	t.Fatalf("missing %s cookie", oauthStateCookieName)
	return nil
}

func TestLoginResponseDoesNotExposeToken(t *testing.T) {
	body, err := json.Marshal(LoginRsp{User: UserInfo{Name: "Admin"}})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if strings.Contains(string(body), "token") {
		t.Fatalf("login response exposes token: %s", body)
	}
}
