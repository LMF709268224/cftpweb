package server

import (
	"context"
	"errors"
	"sync"
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

func TestUserULIDCacheSharesInFlightLookup(t *testing.T) {
	cache := newUserULIDCache(time.Minute, 10)
	started := make(chan struct{})
	finish := make(chan struct{})
	var calls atomic.Int32
	load := func(context.Context) (string, error) {
		calls.Add(1)
		close(started)
		<-finish
		return "candidate-1", nil
	}

	var wg sync.WaitGroup
	results := make(chan string, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			value, err := cache.resolve(context.Background(), "user-1", load)
			if err != nil {
				t.Errorf("resolve() error = %v", err)
				return
			}
			results <- value
		}()
	}
	<-started
	close(finish)
	wg.Wait()
	close(results)
	if got := calls.Load(); got != 1 {
		t.Fatalf("loader calls = %d, want 1", got)
	}
	for value := range results {
		if value != "candidate-1" {
			t.Errorf("cached value = %q", value)
		}
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
