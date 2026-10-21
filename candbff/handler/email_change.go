package handler

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"log/slog"
	"math/big"
	"net/http"
	"net/mail"
	"net/url"
	"os"
	"strings"
	"time"

	"candbff/config"
	"github.com/redis/go-redis/v9"
)

const (
	emailCodeTTL    = 10 * time.Minute
	emailRequestTTL = 30 * time.Minute
	emailSendDelay  = time.Minute
)

var markEmailConfirmed = redis.NewScript(`
if redis.call('GET', KEYS[2]) ~= ARGV[1] then return 0 end
if redis.call('HGET', KEYS[1], ARGV[2] .. '_hash') ~= ARGV[3] then return 0 end
redis.call('HSET', KEYS[1], ARGV[2] .. '_done', '1')
if redis.call('HGET', KEYS[1], 'old_done') == '1' and redis.call('HGET', KEYS[1], 'new_done') == '1' then return 2 end
return 1
`)

func emailCodeKey(name string) string            { return "candidate:email:code:" + name }
func emailCodeThrottleKey(name string) string    { return "candidate:email:code:wait:" + name }
func emailRequestThrottleKey(name string) string { return "candidate:email:request:wait:" + name }
func emailLatestKey(name string) string          { return "candidate:email:latest:" + name }
func emailRequestKey(id string) string           { return "candidate:email:request:" + id }

func randomEmailToken() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func emailTokenHash(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func validEmailAddress(value string) bool {
	address, err := mail.ParseAddress(value)
	return err == nil && address.Address == value && len(value) <= 254
}

func emailChangeConfig() (string, string, bool) {
	origin := strings.TrimRight(strings.TrimSpace(os.Getenv(config.EnvCandidatePortalOrigin)), "/")
	sender := strings.TrimSpace(os.Getenv(config.EnvEmailSender))
	parsed, err := url.Parse(origin)
	if err != nil || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.Path != "" ||
		(parsed.Scheme != "https" && !(parsed.Scheme == "http" && isLocalDevOrigin(origin))) || !validEmailAddress(sender) {
		return "", "", false
	}
	return origin, sender, true
}

func (h *Handler) emailChangeAvailable(w http.ResponseWriter) (string, string, bool) {
	origin, sender, valid := emailChangeConfig()
	if h.Rdb == nil || !valid {
		WriteError(w, http.StatusServiceUnavailable, ErrServiceUnavailable, "email change is not configured")
		return "", "", false
	}
	return origin, sender, true
}

// SendEmailChangeCode sends a short-lived reauthentication code to the current address.
func (h *Handler) SendEmailChangeCode(w http.ResponseWriter, r *http.Request) {
	_, sender, ok := h.emailChangeAvailable(w)
	if !ok {
		return
	}
	name := CandidateName(r)
	user, err := h.getProfileUserStore().GetUser(name)
	if err != nil || user == nil || !validEmailAddress(user.Email) {
		WriteError(w, http.StatusServiceUnavailable, ErrServiceUnavailable, "current email is unavailable")
		return
	}
	allowed, err := h.Rdb.SetNX(r.Context(), emailCodeThrottleKey(name), "1", emailSendDelay).Result()
	if err != nil {
		WriteError(w, http.StatusServiceUnavailable, ErrServiceUnavailable, "verification is unavailable")
		return
	}
	if !allowed {
		WriteError(w, http.StatusTooManyRequests, ErrPrecondition, "Please wait before requesting another code.")
		return
	}
	number, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		h.Rdb.Del(r.Context(), emailCodeThrottleKey(name))
		WriteError(w, http.StatusInternalServerError, ErrInternal, "failed to create code")
		return
	}
	code := fmt.Sprintf("%06d", number.Int64())
	if err := h.Rdb.Set(r.Context(), emailCodeKey(name), emailTokenHash(code), emailCodeTTL).Err(); err != nil {
		h.Rdb.Del(r.Context(), emailCodeThrottleKey(name))
		WriteError(w, http.StatusServiceUnavailable, ErrServiceUnavailable, "verification is unavailable")
		return
	}
	if err := h.getProfileUserStore().SendEmail("Confirm your email change", "Your verification code is "+code+". It expires in 10 minutes.", sender, user.Email); err != nil {
		h.Rdb.Del(r.Context(), emailCodeKey(name), emailCodeThrottleKey(name))
		slog.Error("Failed to send email change code", "error", err, "user", name)
		WriteError(w, http.StatusBadGateway, ErrServiceUnavailable, "failed to send verification email")
		return
	}
	WriteJSON(w, http.StatusOK, map[string]string{"email": user.Email})
}

func (h *Handler) RequestEmailChange(w http.ResponseWriter, r *http.Request) {
	origin, sender, ok := h.emailChangeAvailable(w)
	if !ok {
		return
	}
	var input struct {
		NewEmail string `json:"new_email"`
		Code     string `json:"code"`
	}
	if err := ReadJSON(r, &input); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}
	newEmail := strings.ToLower(strings.TrimSpace(input.NewEmail))
	if !validEmailAddress(newEmail) || len(input.Code) != 6 {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "valid email and six-digit code are required")
		return
	}
	name := CandidateName(r)
	users := h.getProfileUserStore()
	user, err := users.GetUser(name)
	if err != nil || user == nil || !validEmailAddress(user.Email) {
		WriteError(w, http.StatusServiceUnavailable, ErrServiceUnavailable, "failed to load user")
		return
	}
	if strings.EqualFold(newEmail, user.Email) {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "new email must differ from current email")
		return
	}
	allowed, err := h.Rdb.SetNX(r.Context(), emailRequestThrottleKey(name), "1", emailSendDelay).Result()
	if err != nil {
		WriteError(w, http.StatusServiceUnavailable, ErrServiceUnavailable, "verification is unavailable")
		return
	}
	if !allowed {
		WriteError(w, http.StatusTooManyRequests, ErrPrecondition, "Please wait before trying again.")
		return
	}
	codeHash, err := h.Rdb.Get(r.Context(), emailCodeKey(name)).Result()
	if err != nil || subtle.ConstantTimeCompare([]byte(codeHash), []byte(emailTokenHash(input.Code))) != 1 {
		attempts := h.Rdb.Incr(r.Context(), emailCodeKey(name)+":attempts").Val()
		h.Rdb.Expire(r.Context(), emailCodeKey(name)+":attempts", emailCodeTTL)
		if attempts >= 5 {
			h.Rdb.Del(r.Context(), emailCodeKey(name))
		}
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "verification code is invalid or expired")
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
	id, err := randomEmailToken()
	if err != nil {
		WriteError(w, http.StatusInternalServerError, ErrInternal, "failed to create request")
		return
	}
	oldToken, err := randomEmailToken()
	if err != nil {
		WriteError(w, http.StatusInternalServerError, ErrInternal, "failed to create request")
		return
	}
	newToken, err := randomEmailToken()
	if err != nil {
		WriteError(w, http.StatusInternalServerError, ErrInternal, "failed to create request")
		return
	}
	key := emailRequestKey(id)
	fields := map[string]interface{}{"user": name, "old_email": user.Email, "new_email": newEmail,
		"old_hash": emailTokenHash(oldToken), "new_hash": emailTokenHash(newToken), "old_done": "0", "new_done": "0"}
	create := h.Rdb.TxPipeline()
	create.HSet(r.Context(), key, fields)
	create.Expire(r.Context(), key, emailRequestTTL)
	if _, err := create.Exec(r.Context()); err != nil {
		WriteError(w, http.StatusServiceUnavailable, ErrServiceUnavailable, "verification is unavailable")
		return
	}
	link := func(side, token string) string {
		q := url.Values{"id": {id}, "side": {side}, "token": {token}}
		return origin + "/email-confirm?" + q.Encode()
	}
	if err := users.SendEmail("Confirm your email change", "Confirm your current email address: "+link("old", oldToken)+"\nThis link expires in 30 minutes.", sender, user.Email); err != nil {
		h.Rdb.Del(r.Context(), key)
		slog.Error("Failed to send old-email confirmation", "error", err, "user", name)
		WriteError(w, http.StatusBadGateway, ErrServiceUnavailable, "failed to send confirmation email")
		return
	}
	if err := users.SendEmail("Confirm your new email", "Confirm your new email address: "+link("new", newToken)+"\nThis link expires in 30 minutes.", sender, newEmail); err != nil {
		h.Rdb.Del(r.Context(), key)
		slog.Error("Failed to send new-email confirmation", "error", err, "user", name)
		WriteError(w, http.StatusBadGateway, ErrServiceUnavailable, "failed to send confirmation email")
		return
	}
	pipe := h.Rdb.TxPipeline()
	pipe.Set(r.Context(), emailLatestKey(name), id, emailRequestTTL)
	pipe.Del(r.Context(), emailCodeKey(name), emailCodeKey(name)+":attempts")
	if _, err := pipe.Exec(r.Context()); err != nil {
		WriteError(w, http.StatusServiceUnavailable, ErrServiceUnavailable, "verification is unavailable")
		return
	}
	WriteJSON(w, http.StatusOK, map[string]string{"status": "sent"})
}

func (h *Handler) ConfirmEmailChange(w http.ResponseWriter, r *http.Request) {
	if h.Rdb == nil {
		WriteError(w, http.StatusServiceUnavailable, ErrServiceUnavailable, "verification is unavailable")
		return
	}
	var input struct {
		ID    string `json:"id"`
		Side  string `json:"side"`
		Token string `json:"token"`
	}
	if err := ReadJSON(r, &input); err != nil || len(input.ID) != 43 || len(input.Token) != 43 || (input.Side != "old" && input.Side != "new") {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid confirmation link")
		return
	}
	key := emailRequestKey(input.ID)
	state, err := h.Rdb.HGetAll(r.Context(), key).Result()
	if err != nil || len(state) == 0 {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "confirmation link has expired")
		return
	}
	if subtle.ConstantTimeCompare([]byte(state[input.Side+"_hash"]), []byte(emailTokenHash(input.Token))) != 1 {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid confirmation link")
		return
	}
	marked, err := markEmailConfirmed.Run(r.Context(), h.Rdb,
		[]string{key, emailLatestKey(state["user"])}, input.ID, input.Side, emailTokenHash(input.Token)).Int()
	if err != nil {
		WriteError(w, http.StatusServiceUnavailable, ErrServiceUnavailable, "verification is unavailable")
		return
	}
	if marked == 0 {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "confirmation link has expired")
		return
	}
	if marked == 1 {
		WriteJSON(w, http.StatusOK, map[string]string{"status": "pending"})
		return
	}
	locked, err := h.Rdb.SetNX(r.Context(), key+":updating", "1", time.Minute).Result()
	if err != nil {
		WriteError(w, http.StatusServiceUnavailable, ErrServiceUnavailable, "verification is unavailable")
		return
	}
	if !locked {
		WriteJSON(w, http.StatusOK, map[string]string{"status": "pending"})
		return
	}
	defer h.Rdb.Del(context.Background(), key+":updating")
	users := h.getProfileUserStore()
	user, err := users.GetUser(state["user"])
	if err != nil || user == nil || !strings.EqualFold(user.Email, state["old_email"]) {
		WriteError(w, http.StatusConflict, ErrPrecondition, "account email changed; start again")
		return
	}
	existing, err := users.GetUserByEmail(state["new_email"])
	if err != nil {
		WriteError(w, http.StatusServiceUnavailable, ErrServiceUnavailable, "failed to check new email")
		return
	}
	if existing != nil {
		WriteError(w, http.StatusConflict, ErrPrecondition, "new email is unavailable")
		return
	}
	user.Email = state["new_email"]
	updated, err := users.UpdateUser(user)
	if err != nil || !updated {
		slog.Error("Failed to update confirmed email", "error", err, "user", state["user"])
		WriteError(w, http.StatusBadGateway, ErrProfileUpdateFailed, "failed to update email")
		return
	}
	h.Rdb.Del(r.Context(), key, emailLatestKey(state["user"]))
	slog.Info("Candidate email changed after dual confirmation", "user", state["user"], "old_email", state["old_email"], "new_email", state["new_email"])
	clearTokenCookies(w, r)
	WriteJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}
