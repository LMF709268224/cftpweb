package handler

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestNewEmailVerificationCodeUsesSixDigits(t *testing.T) {
	digits := regexp.MustCompile(`^[0-9]{6}$`)
	seen := make(map[string]struct{})
	for i := 0; i < 32; i++ {
		code, err := newEmailVerificationCode()
		if err != nil {
			t.Fatalf("newEmailVerificationCode: %v", err)
		}
		if !digits.MatchString(code) {
			t.Fatalf("code = %q, want six digits", code)
		}
		seen[code] = struct{}{}
	}
	if len(seen) == 1 {
		t.Fatal("all generated verification codes were identical")
	}
}

func TestVerifyEmailCodeScriptLimitsInvalidAttempts(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	ctx := context.Background()
	codeKey := "candbff:email_verification:candidate-1"
	attemptsKey := codeKey + ":attempts"
	expected := "candidate@example.com:123456"
	if err := client.Set(ctx, codeKey, expected, emailVerificationTTL).Err(); err != nil {
		t.Fatalf("seed code: %v", err)
	}

	for attempt := 1; attempt < maxEmailVerificationAttempts; attempt++ {
		result, err := verifyEmailCodeScript.Run(
			ctx, client, []string{codeKey, attemptsKey},
			"candidate@example.com:000000", maxEmailVerificationAttempts, int(emailVerificationTTL.Seconds()),
		).Int64()
		if err != nil {
			t.Fatalf("attempt %d: %v", attempt, err)
		}
		if result != -1 {
			t.Fatalf("attempt %d result = %d, want -1", attempt, result)
		}
	}

	result, err := verifyEmailCodeScript.Run(
		ctx, client, []string{codeKey, attemptsKey},
		"candidate@example.com:000000", maxEmailVerificationAttempts, int(emailVerificationTTL.Seconds()),
	).Int64()
	if err != nil {
		t.Fatalf("locked attempt: %v", err)
	}
	if result != -2 {
		t.Fatalf("locked attempt result = %d, want -2", result)
	}
	if client.Exists(ctx, codeKey).Val() != 0 {
		t.Fatal("verification code still exists after too many invalid attempts")
	}

	if err := client.Set(ctx, codeKey, expected, time.Minute).Err(); err != nil {
		t.Fatalf("seed replacement code: %v", err)
	}
	result, err = verifyEmailCodeScript.Run(
		ctx, client, []string{codeKey, attemptsKey},
		expected, maxEmailVerificationAttempts, int(emailVerificationTTL.Seconds()),
	).Int64()
	if err != nil {
		t.Fatalf("correct code: %v", err)
	}
	if result != 1 {
		t.Fatalf("correct code result = %d, want 1", result)
	}
	if client.Exists(ctx, codeKey, attemptsKey).Val() != 0 {
		t.Fatal("successful verification did not consume code and attempt state")
	}
}
