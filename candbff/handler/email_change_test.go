package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/casdoor/casdoor-go-sdk/casdoorsdk"
	"github.com/golang-jwt/jwt/v4"
)

func TestEmailChangeUsesCasdoorCodesInOrder(t *testing.T) {
	store := &fakeProfileUserStore{current: &casdoorsdk.User{Owner: "gfi", Name: "candidate", Email: "old@example.com"}}
	var calls []string
	resetAttempts := 0
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
			resetAttempts++
			if err := r.ParseForm(); err != nil {
				t.Error(err)
				return
			}
			if r.Form.Get("type") != "email" || r.Form.Get("dest") != "new@example.com" {
				t.Errorf("unexpected email reset: %v", r.Form)
			}
			if resetAttempts == 1 {
				if r.Form.Get("code") != "111111" {
					t.Errorf("first new code = %q", r.Form.Get("code"))
				}
				_, _ = w.Write([]byte(`{"status":"error","msg":"Wrong verification code!"}`))
				return
			}
			if r.Form.Get("code") != "654321" {
				t.Errorf("second new code = %q", r.Form.Get("code"))
			}
		default:
			t.Errorf("unexpected Casdoor path: %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer casdoor.Close()
	h := &Handler{CasdoorAPIEndpoint: casdoor.URL, CasdoorApplicationID: "admin/app-candidate", CasdoorClientSecret: "test-email-proof-secret", profileUsers: store}
	request := func(path, body string) *http.Request {
		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
		return req.WithContext(WithCandidate(req.Context(), "candidate-id", "old@example.com", "candidate", "candidate-token"))
	}
	old := httptest.NewRecorder()
	h.SendEmailChangeCode(old, request("/api/user/email/send-code", ""))
	if old.Code != http.StatusOK {
		t.Fatalf("send old code: %d %s", old.Code, old.Body.String())
	}
	verified := httptest.NewRecorder()
	h.VerifyCurrentEmailChangeCode(verified, request("/api/user/email/verify-current", `{"old_code":"123456"}`))
	if verified.Code != http.StatusOK {
		t.Fatalf("verify old code: %d %s", verified.Code, verified.Body.String())
	}
	var proofResponse struct {
		Data struct {
			Proof string `json:"proof"`
		} `json:"data"`
	}
	if err := json.Unmarshal(verified.Body.Bytes(), &proofResponse); err != nil || proofResponse.Data.Proof == "" {
		t.Fatalf("missing old email proof: %v %s", err, verified.Body.String())
	}
	proof := proofResponse.Data.Proof
	newCodeBody, _ := json.Marshal(map[string]string{"new_email": "new@example.com", "proof": proof})
	newCode := httptest.NewRecorder()
	h.SendNewEmailChangeCode(newCode, request("/api/user/email/send-new-code", string(newCodeBody)))
	if newCode.Code != http.StatusOK {
		t.Fatalf("send new code: %d %s", newCode.Code, newCode.Body.String())
	}
	wrongBody, _ := json.Marshal(map[string]string{"new_email": "new@example.com", "new_code": "111111", "proof": proof})
	wrong := httptest.NewRecorder()
	h.ConfirmEmailChange(wrong, request("/api/user/email/confirm", string(wrongBody)))
	if wrong.Code != http.StatusBadRequest {
		t.Fatalf("wrong new code: %d %s", wrong.Code, wrong.Body.String())
	}
	correctBody, _ := json.Marshal(map[string]string{"new_email": "new@example.com", "new_code": "654321", "proof": proof})
	confirmed := httptest.NewRecorder()
	h.ConfirmEmailChange(confirmed, request("/api/user/email/confirm", string(correctBody)))
	if confirmed.Code != http.StatusOK || !strings.Contains(confirmed.Body.String(), `"updated"`) {
		t.Fatalf("confirm email: %d %s", confirmed.Code, confirmed.Body.String())
	}
	for _, name := range []string{accessTokenCookieName, refreshTokenCookieName} {
		cleared := false
		for _, cookie := range confirmed.Result().Cookies() {
			if cookie.Name == name && cookie.MaxAge < 0 {
				cleared = true
				break
			}
		}
		if !cleared {
			t.Errorf("email change did not expire %s cookie", name)
		}
	}
	want := []string{"/api/send-verification-code", "/api/verify-code", "/api/send-verification-code", "/api/reset-email-or-phone", "/api/reset-email-or-phone"}
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
	h := &Handler{CasdoorAPIEndpoint: casdoor.URL, CasdoorClientSecret: "test-email-proof-secret", profileUsers: &fakeProfileUserStore{current: &casdoorsdk.User{Owner: "gfi", Name: "candidate", Email: "old@example.com"}}}
	req := httptest.NewRequest(http.MethodPost, "/api/user/email/verify-current", strings.NewReader(`{"old_code":"123456"}`))
	req = req.WithContext(WithCandidate(req.Context(), "candidate-id", "old@example.com", "candidate", "candidate-token"))
	response := httptest.NewRecorder()
	h.VerifyCurrentEmailChangeCode(response, req)
	if response.Code != http.StatusBadRequest || resetCalled || !strings.Contains(response.Body.String(), "current email verification code is incorrect or expired") {
		t.Fatalf("old code failure: status=%d, resetCalled=%v", response.Code, resetCalled)
	}
}

func TestEmailChangeRequiresCurrentEmailProof(t *testing.T) {
	casdoorCalled := false
	casdoor := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		casdoorCalled = true
	}))
	defer casdoor.Close()
	h := &Handler{CasdoorAPIEndpoint: casdoor.URL, CasdoorClientSecret: "test-email-proof-secret", profileUsers: &fakeProfileUserStore{current: &casdoorsdk.User{Owner: "gfi", Name: "candidate", Email: "old@example.com"}}}
	for _, tc := range []struct {
		path string
		body string
		call func(http.ResponseWriter, *http.Request)
	}{
		{"/api/user/email/send-new-code", `{"new_email":"new@example.com"}`, h.SendNewEmailChangeCode},
		{"/api/user/email/confirm", `{"new_email":"new@example.com","new_code":"654321"}`, h.ConfirmEmailChange},
	} {
		req := httptest.NewRequest(http.MethodPost, tc.path, strings.NewReader(tc.body))
		req = req.WithContext(WithCandidate(req.Context(), "candidate-id", "old@example.com", "candidate", "candidate-token"))
		response := httptest.NewRecorder()
		tc.call(response, req)
		if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), "current email verification is missing or expired") {
			t.Errorf("%s without proof: %d %s", tc.path, response.Code, response.Body.String())
		}
	}
	if casdoorCalled {
		t.Fatal("Casdoor was called without proof of current email ownership")
	}
}

func TestEmailChangeProofIsBoundToUserAndCurrentAddress(t *testing.T) {
	user := &casdoorsdk.User{Owner: "gfi", Name: "candidate", Email: "old@example.com"}
	h := &Handler{CasdoorClientSecret: "test-email-proof-secret"}
	req := httptest.NewRequest(http.MethodPost, "/api/user/email/send-new-code", nil)
	req = req.WithContext(WithCandidate(req.Context(), "candidate-id", user.Email, user.Name, "candidate-token"))
	proof, err := h.issueEmailChangeProof(req, user)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name  string
		user  *casdoorsdk.User
		proof string
	}{
		{"valid", user, proof},
		{"different user", &casdoorsdk.User{Owner: "gfi", Name: "other", Email: user.Email}, proof},
		{"changed current email", &casdoorsdk.User{Owner: "gfi", Name: user.Name, Email: "changed@example.com"}, proof},
		{"tampered", user, proof + "x"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			valid, err := h.validEmailChangeProof(req, tc.user, tc.proof)
			if err != nil || valid != (tc.name == "valid") {
				t.Fatalf("valid=%v err=%v", valid, err)
			}
		})
	}
	key, _ := h.emailChangeProofKey()
	expired, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, emailChangeProofClaims{
		Owner: user.Owner, Name: user.Name, CurrentEmail: user.Email,
		StandardClaims: jwt.StandardClaims{Issuer: "candbff-email-change", Subject: "candidate-id", ExpiresAt: time.Now().Add(-time.Minute).Unix()},
	}).SignedString(key)
	if valid, _ := h.validEmailChangeProof(req, user, expired); valid {
		t.Fatal("expired proof accepted")
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
