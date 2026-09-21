package server

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestUserULIDCacheCachesSuccessfulLookups(t *testing.T) {
	cache := newUserULIDCache(time.Minute, 10)
	var calls atomic.Int32
	load := func(context.Context) (string, error) {
		calls.Add(1)
		return "candidate-1", nil
	}

	for i := 0; i < 2; i++ {
		value, err := cache.resolve(context.Background(), "user-1", load)
		if err != nil || value != "candidate-1" {
			t.Fatalf("resolve() = %q, %v", value, err)
		}
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("loader calls = %d, want 1", got)
	}
}

func TestUserULIDCacheExpiresEntries(t *testing.T) {
	cache := newUserULIDCache(5*time.Millisecond, 10)
	var calls atomic.Int32
	load := func(context.Context) (string, error) {
		calls.Add(1)
		return "candidate-1", nil
	}

	if _, err := cache.resolve(context.Background(), "user-1", load); err != nil {
		t.Fatal(err)
	}
	time.Sleep(10 * time.Millisecond)
	if _, err := cache.resolve(context.Background(), "user-1", load); err != nil {
		t.Fatal(err)
	}
	if got := calls.Load(); got != 2 {
		t.Fatalf("loader calls = %d, want 2 after expiry", got)
	}
}

func TestUserULIDCacheDoesNotCacheErrors(t *testing.T) {
	cache := newUserULIDCache(time.Minute, 10)
	wantErr := errors.New("gmid unavailable")
	var calls atomic.Int32
	load := func(context.Context) (string, error) {
		calls.Add(1)
		return "", wantErr
	}

	for i := 0; i < 2; i++ {
		if _, err := cache.resolve(context.Background(), "user-1", load); !errors.Is(err, wantErr) {
			t.Fatalf("resolve() error = %v, want %v", err, wantErr)
		}
	}
	if got := calls.Load(); got != 2 {
		t.Fatalf("loader calls = %d, want 2 for uncached errors", got)
	}
}
