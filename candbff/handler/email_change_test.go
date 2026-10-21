package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/casdoor/casdoor-go-sdk/casdoorsdk"
	"github.com/redis/go-redis/v9"
)

func TestEmailChangeRequiresBothConfirmations(t *testing.T) {
	cache := miniredis.RunT(t)
	t.Setenv("CANDIDATE_PORTAL_ORIGIN", "http://localhost:5173")
	t.Setenv("EMAIL_SENDER", "no-reply@example.com")
	store := &fakeProfileUserStore{current: &casdoorsdk.User{Owner: "gfi", Name: "candidate", Email: "old@example.com"}}
	h := &Handler{Rdb: redis.NewClient(&redis.Options{Addr: cache.Addr()}), profileUsers: store}
	t.Cleanup(func() { _ = h.Rdb.Close() })
	request := func(path, body string) *http.Request {
		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
		return req.WithContext(WithCandidate(req.Context(), "candidate-id", "old@example.com", "candidate", "token"))
	}
	codeResponse := httptest.NewRecorder()
	h.SendEmailChangeCode(codeResponse, request("/api/user/email/send-code", ""))
	if codeResponse.Code != http.StatusOK || len(store.sentContents) != 1 {
		t.Fatalf("code response: %d %s", codeResponse.Code, codeResponse.Body.String())
	}
	code := regexp.MustCompile(`\b[0-9]{6}\b`).FindString(store.sentContents[0])
	if code == "" {
		t.Fatal("verification code missing from email")
	}
	wrongCode := "000000"
	if code == wrongCode {
		wrongCode = "111111"
	}

	bad := httptest.NewRecorder()
	badBody, _ := json.Marshal(map[string]string{"new_email": "new@example.com", "code": wrongCode})
	h.RequestEmailChange(bad, request("/api/user/email/request", string(badBody)))
	if bad.Code != http.StatusBadRequest {
		t.Fatalf("wrong code accepted: %d", bad.Code)
	}
	cache.Del(emailRequestThrottleKey("candidate"))
	body, _ := json.Marshal(map[string]string{"new_email": "new@example.com", "code": code})
	sent := httptest.NewRecorder()
	h.RequestEmailChange(sent, request("/api/user/email/request", string(body)))
	if sent.Code != http.StatusOK || len(store.sentContents) != 3 {
		t.Fatalf("request response: %d %s", sent.Code, sent.Body.String())
	}
	if store.current.Email != "old@example.com" {
		t.Fatal("email changed before confirmations")
	}

	confirm := func(mailText string) *httptest.ResponseRecorder {
		link := regexp.MustCompile(`http://localhost:5173/email-confirm\?[^\s]+`).FindString(mailText)
		parsed, err := url.Parse(link)
		if err != nil || link == "" {
			t.Fatalf("invalid confirmation link: %q %v", link, err)
		}
		payload, _ := json.Marshal(map[string]string{
			"id": parsed.Query().Get("id"), "side": parsed.Query().Get("side"), "token": parsed.Query().Get("token"),
		})
		response := httptest.NewRecorder()
		h.ConfirmEmailChange(response, request("/api/auth/email/confirm", string(payload)))
		return response
	}
	first := confirm(store.sentContents[1])
	if first.Code != http.StatusOK || !strings.Contains(first.Body.String(), `"pending"`) || store.current.Email != "old@example.com" {
		t.Fatalf("first confirmation: %d %s, email=%q", first.Code, first.Body.String(), store.current.Email)
	}
	second := confirm(store.sentContents[2])
	if second.Code != http.StatusOK || !strings.Contains(second.Body.String(), `"updated"`) || store.current.Email != "new@example.com" || store.updateCalls != 1 {
		t.Fatalf("second confirmation: %d %s, email=%q, updates=%d", second.Code, second.Body.String(), store.current.Email, store.updateCalls)
	}
	if len(second.Result().Cookies()) == 0 {
		t.Fatal("completion did not clear authentication cookies")
	}
	replay := confirm(store.sentContents[2])
	if replay.Code != http.StatusBadRequest || store.updateCalls != 1 {
		t.Fatalf("replay: %d, updates=%d", replay.Code, store.updateCalls)
	}
}

func TestExpiredEmailConfirmationDoesNotRecreateState(t *testing.T) {
	cache := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: cache.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	key := emailRequestKey("request")
	latest := emailLatestKey("candidate")
	ctx := context.Background()
	client.HSet(ctx, key, "old_hash", emailTokenHash("token"))
	client.Expire(ctx, key, time.Second)
	client.Set(ctx, latest, "request", time.Minute)
	cache.FastForward(2 * time.Second)
	result, err := markEmailConfirmed.Run(ctx, client, []string{key, latest}, "request", "old", emailTokenHash("token")).Int()
	if err != nil || result != 0 || cache.Exists(key) {
		t.Fatalf("expired state recreated: result=%d error=%v exists=%v", result, err, cache.Exists(key))
	}
}
