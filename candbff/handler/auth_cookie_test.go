package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClearTokenCookiesExpiresBothAuthCookies(t *testing.T) {
	recorder := httptest.NewRecorder()

	clearTokenCookies(recorder)

	cookies := recorder.Result().Cookies()
	if len(cookies) != 2 {
		t.Fatalf("cookie count = %d, want 2", len(cookies))
	}

	byName := make(map[string]*http.Cookie, len(cookies))
	for _, cookie := range cookies {
		byName[cookie.Name] = cookie
	}

	for _, name := range []string{"access_token", "refresh_token"} {
		cookie := byName[name]
		if cookie == nil {
			t.Fatalf("missing %s cookie", name)
		}
		if cookie.Value != "" {
			t.Fatalf("%s value = %q, want empty", name, cookie.Value)
		}
		if cookie.Path != "/" {
			t.Fatalf("%s path = %q, want /", name, cookie.Path)
		}
		if !cookie.HttpOnly || !cookie.Secure {
			t.Fatalf("%s cookie must remain HttpOnly and Secure", name)
		}
		if cookie.SameSite != http.SameSiteStrictMode {
			t.Fatalf("%s SameSite = %v, want Strict", name, cookie.SameSite)
		}
		if !cookie.Expires.Before(time.Now()) {
			t.Fatalf("%s expiry = %v, want a past time", name, cookie.Expires)
		}
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

func TestOAuthStateMismatchPreservesOutstandingStates(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
	req.AddCookie(&http.Cookie{
		Name:  oauthStateCookieName,
		Value: "expected.other",
	})
	rec := httptest.NewRecorder()

	if consumeOAuthState(rec, req, "provided") {
		t.Fatal("consumeOAuthState() = true, want false")
	}
	if cookies := rec.Result().Cookies(); len(cookies) != 0 {
		t.Fatalf("mismatched state changed cookies: %+v", cookies)
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
