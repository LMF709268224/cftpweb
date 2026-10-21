package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/mail"
	"net/url"
	"strings"
	"time"

	"github.com/casdoor/casdoor-go-sdk/casdoorsdk"
)

var casdoorEmailHTTPClient = &http.Client{Timeout: 10 * time.Second}

type casdoorEmailResponse struct {
	Status string `json:"status"`
	Msg    string `json:"msg"`
}

type casdoorEmailError struct{ message string }

func (e casdoorEmailError) Error() string { return e.message }

func validEmailAddress(value string) bool {
	address, err := mail.ParseAddress(value)
	return err == nil && address.Address == value && len(value) <= 254
}

func (h *Handler) emailApplicationID() (string, error) {
	if h.CasdoorApplicationID != "" {
		return h.CasdoorApplicationID, nil
	}
	application, err := casdoorsdk.GetApplication(h.CasdoorAppName)
	if err != nil {
		return "", err
	}
	if application == nil || application.Owner == "" || application.Name == "" {
		return "", errors.New("Casdoor application is unavailable")
	}
	return application.Owner + "/" + application.Name, nil
}

func (h *Handler) callCasdoorEmailAPI(r *http.Request, path string, body io.Reader, contentType string) error {
	token, _ := r.Context().Value(CtxKeyToken).(string)
	if h.CasdoorAPIEndpoint == "" || token == "" {
		return errors.New("Casdoor email API is not configured")
	}
	endpoint := strings.TrimRight(h.CasdoorAPIEndpoint, "/") + path
	req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, endpoint, body)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", contentType)
	if language := r.Header.Get("Accept-Language"); language != "" {
		req.Header.Set("Accept-Language", language)
	}
	client := h.CasdoorEmailHTTPClient
	if client == nil {
		client = casdoorEmailHTTPClient
	}
	response, err := client.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("Casdoor returned HTTP %d", response.StatusCode)
	}
	var result casdoorEmailResponse
	if err := json.NewDecoder(io.LimitReader(response.Body, 8192)).Decode(&result); err != nil {
		return err
	}
	if result.Status != "ok" {
		if result.Msg == "" {
			result.Msg = "Casdoor rejected the request"
		}
		return casdoorEmailError{message: result.Msg}
	}
	return nil
}

func (h *Handler) sendEmailVerificationCode(w http.ResponseWriter, r *http.Request, dest string) {
	applicationID, err := h.emailApplicationID()
	if err != nil {
		slog.Error("Failed to resolve Casdoor application", "error", err)
		WriteError(w, http.StatusServiceUnavailable, ErrServiceUnavailable, "email verification is unavailable")
		return
	}
	form := url.Values{
		"method":        {"reset"},
		"type":          {"email"},
		"dest":          {dest},
		"applicationId": {applicationID},
		"captchaType":   {"none"},
	}
	err = h.callCasdoorEmailAPI(r, "/api/send-verification-code", strings.NewReader(form.Encode()), "application/x-www-form-urlencoded")
	if err != nil {
		var rejected casdoorEmailError
		if errors.As(err, &rejected) {
			WriteError(w, http.StatusBadRequest, ErrPrecondition, rejected.message)
			return
		}
		slog.Error("Failed to send Casdoor email verification code", "error", err)
		WriteError(w, http.StatusBadGateway, ErrServiceUnavailable, "failed to send verification email")
		return
	}
	WriteJSON(w, http.StatusOK, map[string]string{"email": dest})
}

// SendEmailChangeCode sends a Casdoor code to the currently registered address.
func (h *Handler) SendEmailChangeCode(w http.ResponseWriter, r *http.Request) {
	user, err := h.getProfileUserStore().GetUser(CandidateName(r))
	if err != nil || user == nil || !validEmailAddress(user.Email) {
		WriteError(w, http.StatusServiceUnavailable, ErrServiceUnavailable, "current email is unavailable")
		return
	}
	h.sendEmailVerificationCode(w, r, user.Email)
}

// SendNewEmailChangeCode sends a Casdoor code to the proposed address.
func (h *Handler) SendNewEmailChangeCode(w http.ResponseWriter, r *http.Request) {
	var input struct {
		NewEmail string `json:"new_email"`
	}
	if err := ReadJSON(r, &input); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}
	newEmail := strings.ToLower(strings.TrimSpace(input.NewEmail))
	if !validEmailAddress(newEmail) {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "valid new email is required")
		return
	}
	users := h.getProfileUserStore()
	user, err := users.GetUser(CandidateName(r))
	if err != nil || user == nil || !validEmailAddress(user.Email) {
		WriteError(w, http.StatusServiceUnavailable, ErrServiceUnavailable, "current email is unavailable")
		return
	}
	if strings.EqualFold(newEmail, user.Email) {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "new email must differ from current email")
		return
	}
	existing, err := users.GetUserByEmail(newEmail)
	if err != nil {
		WriteError(w, http.StatusServiceUnavailable, ErrServiceUnavailable, "failed to check new email")
		return
	}
	if existing != nil {
		WriteError(w, http.StatusConflict, ErrPrecondition, "new email is unavailable")
		return
	}
	h.sendEmailVerificationCode(w, r, newEmail)
}

func (h *Handler) ConfirmEmailChange(w http.ResponseWriter, r *http.Request) {
	var input struct {
		NewEmail string `json:"new_email"`
		OldCode  string `json:"old_code"`
		NewCode  string `json:"new_code"`
	}
	if err := ReadJSON(r, &input); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}
	newEmail := strings.ToLower(strings.TrimSpace(input.NewEmail))
	if !validEmailAddress(newEmail) || !sixDigitCode(input.OldCode) || !sixDigitCode(input.NewCode) {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "valid email and two six-digit codes are required")
		return
	}
	user, err := h.getProfileUserStore().GetUser(CandidateName(r))
	if err != nil || user == nil || !validEmailAddress(user.Email) {
		WriteError(w, http.StatusServiceUnavailable, ErrServiceUnavailable, "current email is unavailable")
		return
	}
	if strings.EqualFold(newEmail, user.Email) {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "new email must differ from current email")
		return
	}
	oldProof, err := json.Marshal(map[string]string{
		"organization": user.Owner,
		"name":         user.Name,
		"username":     user.Email,
		"code":         input.OldCode,
	})
	if err != nil {
		WriteError(w, http.StatusInternalServerError, ErrInternal, "failed to verify current email")
		return
	}
	if err := h.callCasdoorEmailAPI(r, "/api/verify-code", bytes.NewReader(oldProof), "application/json"); err != nil {
		h.writeEmailVerificationError(w, err, "failed to verify current email")
		return
	}
	form := url.Values{"type": {"email"}, "dest": {newEmail}, "code": {input.NewCode}}
	if err := h.callCasdoorEmailAPI(r, "/api/reset-email-or-phone", strings.NewReader(form.Encode()), "application/x-www-form-urlencoded"); err != nil {
		h.writeEmailVerificationError(w, err, "failed to change email")
		return
	}
	slog.Info("Candidate email changed after Casdoor verification", "user", user.Name, "old_email", user.Email, "new_email", newEmail)
	clearTokenCookies(w, r)
	WriteJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func sixDigitCode(code string) bool {
	if len(code) != 6 {
		return false
	}
	for _, digit := range code {
		if digit < '0' || digit > '9' {
			return false
		}
	}
	return true
}

func (h *Handler) writeEmailVerificationError(w http.ResponseWriter, err error, fallback string) {
	var rejected casdoorEmailError
	if errors.As(err, &rejected) {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, rejected.message)
		return
	}
	slog.Error("Casdoor email verification request failed", "error", err)
	WriteError(w, http.StatusBadGateway, ErrServiceUnavailable, fallback)
}
