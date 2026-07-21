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
