package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/casdoor/casdoor-go-sdk/casdoorsdk"
)

func TestEmailChangeUsesCasdoorCodesInOrder(t *testing.T) {
	store := &fakeProfileUserStore{current: &casdoorsdk.User{Owner: "gfi", Name: "candidate", Email: "old@example.com"}}
	var calls []string
	casdoor := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer candidate-token" {
			t.Errorf("Authorization = %q", got)
		}
		calls = append(calls, r.URL.Path)
		switch r.URL.Path {
		case "/api/send-verification-code":
			if err := r.ParseForm(); err != nil {
				t.Error(err)
				return
			}
			if r.Form.Get("method") != "reset" || r.Form.Get("type") != "email" || r.Form.Get("applicationId") != "admin/app-candidate" || r.Form.Get("captchaType") != "none" {
				t.Errorf("unexpected code request: %v", r.Form)
			}
			if dest := r.Form.Get("dest"); dest != "old@example.com" && dest != "new@example.com" {
				t.Errorf("unexpected destination: %q", dest)
			}
		case "/api/verify-code":
			var proof map[string]string
			if err := json.NewDecoder(r.Body).Decode(&proof); err != nil {
				t.Error(err)
				return
			}
			if proof["organization"] != "gfi" || proof["name"] != "candidate" || proof["username"] != "old@example.com" || proof["code"] != "123456" {
				t.Errorf("unexpected old email proof: %v", proof)
			}
		case "/api/reset-email-or-phone":
			if err := r.ParseForm(); err != nil {
				t.Error(err)
				return
			}
			if r.Form.Get("type") != "email" || r.Form.Get("dest") != "new@example.com" || r.Form.Get("code") != "654321" {
				t.Errorf("unexpected email reset: %v", r.Form)
			}
		default:
			t.Errorf("unexpected Casdoor path: %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer casdoor.Close()
	h := &Handler{CasdoorAPIEndpoint: casdoor.URL, CasdoorApplicationID: "admin/app-candidate", profileUsers: store}
	request := func(path, body string) *http.Request {
		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
		return req.WithContext(WithCandidate(req.Context(), "candidate-id", "old@example.com", "candidate", "candidate-token"))
	}
	old := httptest.NewRecorder()
	h.SendEmailChangeCode(old, request("/api/user/email/send-code", ""))
	if old.Code != http.StatusOK {
		t.Fatalf("send old code: %d %s", old.Code, old.Body.String())
	}
	newCode := httptest.NewRecorder()
	h.SendNewEmailChangeCode(newCode, request("/api/user/email/send-new-code", `{"new_email":"new@example.com"}`))
	if newCode.Code != http.StatusOK {
		t.Fatalf("send new code: %d %s", newCode.Code, newCode.Body.String())
	}
	confirmed := httptest.NewRecorder()
	h.ConfirmEmailChange(confirmed, request("/api/user/email/confirm", `{"new_email":"new@example.com","old_code":"123456","new_code":"654321"}`))
	if confirmed.Code != http.StatusOK || !strings.Contains(confirmed.Body.String(), `"updated"`) {
		t.Fatalf("confirm email: %d %s", confirmed.Code, confirmed.Body.String())
	}
	if len(confirmed.Result().Cookies()) == 0 {
		t.Fatal("email change did not clear local auth cookies")
	}
	want := []string{"/api/send-verification-code", "/api/send-verification-code", "/api/verify-code", "/api/reset-email-or-phone"}
	if len(calls) != len(want) {
		t.Fatalf("Casdoor calls = %v, want %v", calls, want)
	}
	for i := range want {
		if calls[i] != want[i] {
			t.Fatalf("Casdoor calls = %v, want %v", calls, want)
		}
	}
}

func TestEmailChangeRejectsOldCodeBeforeReset(t *testing.T) {
	var resetCalled bool
	casdoor := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/verify-code" {
			_, _ = w.Write([]byte(`{"status":"error","msg":"Wrong verification code!"}`))
			return
		}
		resetCalled = true
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer casdoor.Close()
	h := &Handler{CasdoorAPIEndpoint: casdoor.URL, profileUsers: &fakeProfileUserStore{current: &casdoorsdk.User{Owner: "gfi", Name: "candidate", Email: "old@example.com"}}}
	req := httptest.NewRequest(http.MethodPost, "/api/user/email/confirm", strings.NewReader(`{"new_email":"new@example.com","old_code":"123456","new_code":"654321"}`))
	req = req.WithContext(WithCandidate(req.Context(), "candidate-id", "old@example.com", "candidate", "candidate-token"))
	response := httptest.NewRecorder()
	h.ConfirmEmailChange(response, req)
	if response.Code != http.StatusBadRequest || resetCalled {
		t.Fatalf("old code failure: status=%d, resetCalled=%v", response.Code, resetCalled)
	}
}

func TestEmailChangeRejectsSameAddressBeforeSendingCode(t *testing.T) {
	store := &fakeProfileUserStore{current: &casdoorsdk.User{Owner: "gfi", Name: "candidate", Email: "old@example.com"}}
	h := &Handler{profileUsers: store}
	req := httptest.NewRequest(http.MethodPost, "/api/user/email/send-new-code", strings.NewReader(`{"new_email":"OLD@example.com"}`))
	req = req.WithContext(WithCandidate(req.Context(), "candidate-id", "old@example.com", "candidate", "candidate-token"))
	response := httptest.NewRecorder()
	h.SendNewEmailChangeCode(response, req)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("same address accepted: %d", response.Code)
	}
}
