package handler

import (
	"bytes"
	"image"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/casdoor/casdoor-go-sdk/casdoorsdk"
)

func TestUpdateUserLanguagePersistsCasdoorField(t *testing.T) {
	store := &fakeProfileUserStore{current: &casdoorsdk.User{Name: "candidate", Language: "en"}}
	h := &Handler{profileUsers: store}
	req := httptest.NewRequest(http.MethodPut, "/api/user/language", strings.NewReader(`{"language":"zh"}`))
	req = req.WithContext(WithCandidate(req.Context(), "candidate-id", "candidate@example.com", "candidate", "token"))
	response := httptest.NewRecorder()
	h.UpdateUserLanguage(response, req)
	if response.Code != http.StatusOK || store.current.Language != "zh" || store.updateCalls != 1 {
		t.Fatalf("status=%d language=%q updates=%d", response.Code, store.current.Language, store.updateCalls)
	}
}

func TestUploadUserAvatarValidatesAndPersists(t *testing.T) {
	store := &fakeProfileUserStore{current: &casdoorsdk.User{Name: "candidate"}}
	h := &Handler{profileUsers: store}
	request := func(data []byte) *http.Request {
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		part, err := writer.CreateFormFile("avatar", "avatar.png")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := part.Write(data); err != nil {
			t.Fatal(err)
		}
		if err := writer.Close(); err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest(http.MethodPost, "/api/user/avatar", &body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		return req.WithContext(WithCandidate(req.Context(), "candidate-id", "candidate@example.com", "candidate", "token"))
	}
	bad := httptest.NewRecorder()
	h.UploadUserAvatar(bad, request([]byte("not an image")))
	if bad.Code != http.StatusBadRequest || store.avatarUploads != 0 {
		t.Fatalf("invalid upload: %d, uploads=%d", bad.Code, store.avatarUploads)
	}
	var picture bytes.Buffer
	if err := png.Encode(&picture, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatal(err)
	}
	good := httptest.NewRecorder()
	h.UploadUserAvatar(good, request(picture.Bytes()))
	if good.Code != http.StatusOK || store.current.Avatar != "https://example.com/avatar.png" || store.avatarUploads != 1 || store.updateCalls != 1 {
		t.Fatalf("valid upload: %d, avatar=%q, uploads=%d, updates=%d", good.Code, store.current.Avatar, store.avatarUploads, store.updateCalls)
	}
}

func TestUpdateUserProfilePreservesIdentityFields(t *testing.T) {
	store := &fakeProfileUserStore{current: &casdoorsdk.User{
		Name: "candidate", Email: "original@example.com", DisplayName: "Existing display", RealName: "Existing real",
	}}
	h := &Handler{profileUsers: store}
	response := httptest.NewRecorder()
	h.UpdateUserProfile(response, updateProfileRequest(`{"email":"other@example.com","display_name":"Other","real_name":"Other","first_name":"New"}`))
	if response.Code != http.StatusOK || store.current.Email != "original@example.com" || store.current.DisplayName != "Existing display" || store.current.RealName != "Existing real" {
		t.Fatalf("identity fields changed: status=%d user=%+v", response.Code, store.current)
	}
}
