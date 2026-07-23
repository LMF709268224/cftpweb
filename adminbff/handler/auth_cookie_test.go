package handler

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestClearTokenCookiesExpiresAllAuthCookieParts(t *testing.T) {
	recorder := httptest.NewRecorder()

	clearTokenCookies(recorder, httptest.NewRequest(http.MethodPost, "https://example.com/api/auth/logout", nil))

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
			if cookie.Value != "" || cookie.MaxAge != -1 || !cookie.Expires.Before(time.Now()) {
				t.Fatalf("%s was not expired", name)
			}
			if cookie.Path != "/" || !cookie.HttpOnly || !cookie.Secure || cookie.SameSite != http.SameSiteStrictMode {
				t.Fatalf("%s has invalid security attributes: %+v", name, cookie)
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

	authenticatedRequest := httptest.NewRequest(http.MethodGet, "https://example.com/api/admin/me", nil)
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
		t.Fatal("access token changed during cookie round trip")
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
	request := httptest.NewRequest(http.MethodGet, "https://example.com/api/admin/me", nil)
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
	request := httptest.NewRequest(http.MethodGet, "https://example.com/api/admin/me", nil)
	request.AddCookie(&http.Cookie{Name: accessTokenCookieName, Value: tokenCookieEncodingPrefix + ".2.first"})

	if _, err := ReadAccessTokenCookie(request); err == nil {
		t.Fatal("ReadAccessTokenCookie() error = nil, want missing chunk error")
	}
}

func TestRefreshTokenWithoutCookieReturnsUnauthorized(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "https://example.com/api/auth/refresh", nil)

	(&Handler{}).RefreshToken(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
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
