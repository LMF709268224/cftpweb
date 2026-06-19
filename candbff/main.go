package main

import (
	"candbff/config"
	"candbff/server"
	"context"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/urfave/cli/v3"
)

func logLevelEnv() slog.Level {
	switch strings.ToLower(os.Getenv(config.EnvLogLevel)) {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func isLogSourceEnv() bool {
	return strings.ToLower(os.Getenv(config.EnvLogSource)) == "true"
}

func setupLoggerEnv(level slog.Level, addSource bool) {
	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: addSource,
	}
	handler := slog.NewJSONHandler(os.Stdout, opts)
	slog.SetDefault(slog.New(handler))
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	level := logLevelEnv()
	sourceEnabled := isLogSourceEnv()

	setupLoggerEnv(level, sourceEnabled)

	slog.Info("Server starting",
		"level", level.String(),
		"source_enabled", sourceEnabled,
		"pid", os.Getpid(),
	)

	cmd := &cli.Command{
		Name:  "candbff",
		Usage: "HTTP gateway server for candidate web portal",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			s := server.NewServer()

			return s.Run(ctx)
		},
	}

	if err := cmd.Run(ctx, os.Args); err != nil {
		slog.Error("Fatal error", "error", err)
		os.Exit(1)
	}
}
