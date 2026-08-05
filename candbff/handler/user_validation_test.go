package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/casdoor/casdoor-go-sdk/casdoorsdk"
)

func TestUserHandlersRejectInvalidRequestsBeforeCallingCasdoor(t *testing.T) {
	handler := &Handler{}

	profileRecorder := httptest.NewRecorder()
	handler.UpdateUserProfile(
		profileRecorder,
		newCandidateHandlerRequest(
			http.MethodPut,
			"/api/user/profile",
			`{"phone":"123"}`,
			"candidate-1",
			nil,
		),
	)
	assertHandlerAPIError(t, profileRecorder, http.StatusBadRequest, ErrInvalidRequest)

	passwordRecorder := httptest.NewRecorder()
	handler.UpdateUserPassword(
		passwordRecorder,
		newCandidateHandlerRequest(
			http.MethodPut,
			"/api/user/password",
			`{"old_password":"","new_password":""}`,
			"candidate-1",
			nil,
		),
	)
	assertHandlerAPIError(t, passwordRecorder, http.StatusBadRequest, ErrInvalidRequest)

	sendCodeRecorder := httptest.NewRecorder()
	handler.SendEmailCode(
		sendCodeRecorder,
		newCandidateHandlerRequest(
			http.MethodPost,
			"/api/user/profile/email/send-code",
			`{"email":"   "}`,
			"candidate-1",
			nil,
		),
	)
	assertHandlerAPIError(t, sendCodeRecorder, http.StatusBadRequest, ErrInvalidRequest)

	updateEmailRecorder := httptest.NewRecorder()
	handler.UpdateUserEmail(
		updateEmailRecorder,
		newCandidateHandlerRequest(
			http.MethodPut,
			"/api/user/profile/email",
			`{"email":" candidate@example.com ","verification_code":"   "}`,
			"candidate-1",
			nil,
		),
	)
	assertHandlerAPIError(t, updateEmailRecorder, http.StatusBadRequest, ErrInvalidRequest)
}

func TestNormalizeAndValidateUserProfileInput(t *testing.T) {
	input := UserProfileInput{
		DisplayName: "  Candidate Name  ",
		Phone:       " +65 9123-4567 ",
		HomePhone:   " +65 6123 4567 ",
		PostalCode:  " 123 456 ",
		Gender:      " Female ",
		Province:    " Singapore ",
	}

	if err := normalizeAndValidateUserProfileInput(&input); err != nil {
		t.Fatalf("normalize profile: %v", err)
	}
	if input.DisplayName != "Candidate Name" {
		t.Fatalf("display_name = %q", input.DisplayName)
	}
	if input.Phone != "+6591234567" || input.HomePhone != "+6561234567" {
		t.Fatalf("phones = (%q, %q)", input.Phone, input.HomePhone)
	}
	if input.PostalCode != "123 456" {
		t.Fatalf("postal_code = %q", input.PostalCode)
	}
	if input.Gender != "Female" {
		t.Fatalf("gender = %q", input.Gender)
	}
	if input.Province != "Singapore" {
		t.Fatalf("province = %q", input.Province)
	}
}

func TestNormalizeAndValidateUserProfileInputRejectsInvalidAndOversizedValues(t *testing.T) {
	tests := []struct {
		name  string
		input UserProfileInput
	}{
		{
			name:  "invalid gender",
			input: UserProfileInput{Gender: "unknown"},
		},
		{
			name:  "invalid postal code",
			input: UserProfileInput{PostalCode: "***"},
		},
		{
			name:  "oversized display name",
			input: UserProfileInput{DisplayName: strings.Repeat("a", profileNameTextMax+1)},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := normalizeAndValidateUserProfileInput(&test.input); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestUserProfilePropertyAndAddressHelpers(t *testing.T) {
	user := &casdoorsdk.User{}
	setUserProperty(user, userPropProvince, " Singapore ")
	setUserProperty(user, userPropRealName, " Candidate Name ")

	if getUserProperty(user, userPropProvince) != "Singapore" {
		t.Fatalf("province = %q", getUserProperty(user, userPropProvince))
	}
	if userRealName(user) != "Candidate Name" {
		t.Fatalf("real name = %q", userRealName(user))
	}

	address := addressFromProfile(" 1 Main Street ", " Singapore ")
	if len(address) != 2 || addressLine(address, 0) != "1 Main Street" || addressLine(address, 1) != "Singapore" {
		t.Fatalf("address = %#v", address)
	}

	setUserProperty(user, userPropProvince, " ")
	if getUserProperty(user, userPropProvince) != "" {
		t.Fatalf("empty province was not removed: %#v", user.Properties)
	}
}
