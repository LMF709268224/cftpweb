package server

import (
	"context"
	"errors"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestGracefulShutdownClosesRedisClient(t *testing.T) {
	redisServer := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	server := &Server{rdb: client}

	server.gracefulShutdown()
	if server.rdb != nil {
		t.Fatal("Redis client reference was not cleared")
	}
	if err := client.Ping(context.Background()).Err(); !errors.Is(err, redis.ErrClosed) {
		t.Fatalf("Ping after shutdown error = %v, want redis.ErrClosed", err)
	}

	server.gracefulShutdown()
}
