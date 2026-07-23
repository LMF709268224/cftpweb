package handler

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

func TestClearTokenCookiesExpiresBothAuthCookies(t *testing.T) {
	recorder := httptest.NewRecorder()

	clearTokenCookies(recorder, httptest.NewRequest("GET", "https://example.com/", nil))

	cookies := recorder.Result().Cookies()
	if len(cookies) != 2*maxTokenCookieChunks {
		t.Fatalf("cookie count = %d, want %d", len(cookies), 2*maxTokenCookieChunks)
	}

	byName := make(map[string]*http.Cookie, len(cookies))
	for _, cookie := range cookies {
		byName[cookie.Name] = cookie
	}

	for _, baseName := range []string{accessTokenCookieName, refreshTokenCookieName} {
		for i := 0; i < maxTokenCookieChunks; i++ {
			name := tokenCookiePartName(baseName, i)
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
			if cookie.MaxAge != -1 || !cookie.Expires.Before(time.Now()) {
				t.Fatalf("%s must be expired, MaxAge = %d, Expires = %v", name, cookie.MaxAge, cookie.Expires)
			}
		}
	}
}

func TestTokenCookiesRoundTripLargeAccessToken(t *testing.T) {
	accessToken := deterministicToken(16 * 1024)
	refreshToken := "refresh-token"
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "https://example.com/api/auth/login", nil)

	if err := setTokenCookies(recorder, request, accessToken, refreshToken, time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("setTokenCookies() error = %v", err)
	}

	authenticatedRequest := httptest.NewRequest(http.MethodGet, "https://example.com/api/user/me", nil)
	accessCookieCount := 0
	for _, cookie := range recorder.Result().Cookies() {
		if cookie.Value == "" || cookie.MaxAge < 0 {
			continue
		}
		if len(cookie.String()) > 4096 {
			t.Fatalf("%s serialized size = %d, exceeds browser cookie limit", cookie.Name, len(cookie.String()))
		}
		authenticatedRequest.AddCookie(cookie)
		if cookie.Name == accessTokenCookieName || strings.HasPrefix(cookie.Name, accessTokenCookieName+"_") {
			accessCookieCount++
		}
	}
	if accessCookieCount < 2 {
		t.Fatalf("access token cookie count = %d, want multiple chunks", accessCookieCount)
	}

	gotAccessToken, err := ReadAccessTokenCookie(authenticatedRequest)
	if err != nil {
		t.Fatalf("ReadAccessTokenCookie() error = %v", err)
	}
	if gotAccessToken != accessToken {
		t.Fatalf("access token changed during cookie round trip")
	}

	gotRefreshToken, err := readRefreshTokenCookie(authenticatedRequest)
	if err != nil {
		t.Fatalf("readRefreshTokenCookie() error = %v", err)
	}
	if gotRefreshToken != refreshToken {
		t.Fatalf("refresh token = %q, want %q", gotRefreshToken, refreshToken)
	}
}

func TestReadAccessTokenCookieSupportsLegacyRawCookie(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "https://example.com/api/user/me", nil)
	request.AddCookie(&http.Cookie{Name: accessTokenCookieName, Value: "legacy.jwt.token"})

	got, err := ReadAccessTokenCookie(request)
	if err != nil {
		t.Fatalf("ReadAccessTokenCookie() error = %v", err)
	}
	if got != "legacy.jwt.token" {
		t.Fatalf("ReadAccessTokenCookie() = %q, want legacy.jwt.token", got)
	}
}

func TestReadAccessTokenCookieRejectsMissingChunk(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "https://example.com/api/user/me", nil)
	request.AddCookie(&http.Cookie{Name: accessTokenCookieName, Value: tokenCookieEncodingPrefix + ".2.first"})

	if _, err := ReadAccessTokenCookie(request); err == nil {
		t.Fatal("ReadAccessTokenCookie() error = nil, want missing chunk error")
	}
}

func deterministicToken(size int) string {
	raw := make([]byte, size)
	state := uint32(1)
	for i := range raw {
		state = state*1664525 + 1013904223
		raw[i] = byte(state >> 24)
	}
	return base64.RawURLEncoding.EncodeToString(raw)
}

func TestRefreshTokenForCookiePreservesCurrentTokenWhenNotRotated(t *testing.T) {
	token := &oauth2.Token{AccessToken: "new-access-token"}

	if got := refreshTokenForCookie(token, " current-refresh-token "); got != "current-refresh-token" {
		t.Fatalf("refreshTokenForCookie() = %q, want current-refresh-token", got)
	}
}

func TestRefreshTokenForCookiePrefersRotatedToken(t *testing.T) {
	token := &oauth2.Token{RefreshToken: " rotated-refresh-token "}

	if got := refreshTokenForCookie(token, "current-refresh-token"); got != "rotated-refresh-token" {
		t.Fatalf("refreshTokenForCookie() = %q, want rotated-refresh-token", got)
	}
}

func TestLoginResponseDoesNotExposeAccessToken(t *testing.T) {
	payload, err := json.Marshal(LoginRsp{User: UserInfo{Name: "candidate"}})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if strings.Contains(string(payload), "token") {
		t.Fatalf("login response exposes token field: %s", payload)
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
