package handler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/casdoor/casdoor-go-sdk/casdoorsdk"
)

type fakeProfileUserStore struct {
	current          *casdoorsdk.User
	phoneUser        *casdoorsdk.User
	phoneLookupError error
	lookupPhone      string
	phoneLookupCalls int
	updateCalls      int
}

func (store *fakeProfileUserStore) GetUser(string) (*casdoorsdk.User, error) {
	return store.current, nil
}

func (store *fakeProfileUserStore) GetUserByPhone(phone string) (*casdoorsdk.User, error) {
	store.phoneLookupCalls++
	store.lookupPhone = phone
	return store.phoneUser, store.phoneLookupError
}

func (store *fakeProfileUserStore) UpdateUser(*casdoorsdk.User) (bool, error) {
	store.updateCalls++
	return true, nil
}

func updateProfileRequest(body string) *http.Request {
	req := httptest.NewRequest(http.MethodPut, "/api/user/profile", strings.NewReader(body))
	ctx := WithCandidate(req.Context(), "candidate-ulid", "candidate@example.com", "candidate", "token")
	return req.WithContext(ctx)
}

func TestUpdateUserProfileRejectsPhoneUsedByAnotherAccount(t *testing.T) {
	store := &fakeProfileUserStore{
		current:   &casdoorsdk.User{Owner: "gfi", Name: "candidate", Phone: "+6580000000"},
		phoneUser: &casdoorsdk.User{Owner: "gfi", Name: "another-candidate", Phone: "+6591234567"},
	}
	handler := &Handler{profileUsers: store}
	recorder := httptest.NewRecorder()

	handler.UpdateUserProfile(recorder, updateProfileRequest(`{"phone":"+65 9123-4567"}`))

	if recorder.Code != http.StatusConflict {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"error_code":"PHONE_ALREADY_IN_USE"`) {
		t.Fatalf("body = %s", recorder.Body.String())
	}
	if store.lookupPhone != "+6591234567" {
		t.Fatalf("lookup phone = %q", store.lookupPhone)
	}
	if store.updateCalls != 0 {
		t.Fatalf("update calls = %d", store.updateCalls)
	}
}

func TestUpdateUserProfileAllowsPhoneOwnedByCurrentAccount(t *testing.T) {
	store := &fakeProfileUserStore{
		current:   &casdoorsdk.User{Owner: "gfi", Name: "candidate", Phone: "+6580000000"},
		phoneUser: &casdoorsdk.User{Owner: "gfi", Name: "candidate", Phone: "+6591234567"},
	}
	handler := &Handler{profileUsers: store}
	recorder := httptest.NewRecorder()

	handler.UpdateUserProfile(recorder, updateProfileRequest(`{"phone":"+65 9123-4567"}`))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if store.updateCalls != 1 {
		t.Fatalf("update calls = %d", store.updateCalls)
	}
}

func TestUpdateUserProfileDoesNotSaveWhenPhoneLookupFails(t *testing.T) {
	store := &fakeProfileUserStore{
		current:          &casdoorsdk.User{Owner: "gfi", Name: "candidate", Phone: "+6580000000"},
		phoneLookupError: errors.New("casdoor unavailable"),
	}
	handler := &Handler{profileUsers: store}
	recorder := httptest.NewRecorder()

	handler.UpdateUserProfile(recorder, updateProfileRequest(`{"phone":"+65 9123-4567"}`))

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if store.updateCalls != 0 {
		t.Fatalf("update calls = %d", store.updateCalls)
	}
}

func TestUpdateUserProfileSkipsLookupForUnchangedPhone(t *testing.T) {
	store := &fakeProfileUserStore{
		current: &casdoorsdk.User{Owner: "gfi", Name: "candidate", Phone: "+65 9123-4567"},
	}
	handler := &Handler{profileUsers: store}
	recorder := httptest.NewRecorder()

	handler.UpdateUserProfile(recorder, updateProfileRequest(`{"phone":"+6591234567"}`))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if store.phoneLookupCalls != 0 {
		t.Fatalf("phone lookup calls = %d", store.phoneLookupCalls)
	}
	if store.updateCalls != 1 {
		t.Fatalf("update calls = %d", store.updateCalls)
	}
}
