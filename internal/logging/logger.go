// Package logging provides a structured logger based on stdlib log/slog.
package logging

import (
	"io"
	"log/slog"
	"os"
	"strings"
)

// Config controls logger construction.
type Config struct {
	Level  string    // "debug" | "info" | "warn" | "error" (default "info")
	Format string    // "json" | "text" (default "json")
	Output io.Writer // defaults to os.Stderr when nil
}

// New returns a *slog.Logger configured per cfg.
func New(cfg Config) *slog.Logger {
	out := cfg.Output
	if out == nil {
		out = os.Stderr
	}
	opts := &slog.HandlerOptions{Level: parseLevel(cfg.Level)}
	var h slog.Handler
	if strings.EqualFold(cfg.Format, "text") {
		h = slog.NewTextHandler(out, opts)
	} else {
		h = slog.NewJSONHandler(out, opts)
	}
	return slog.New(h)
}

func parseLevel(s string) slog.Level {
	switch strings.ToLower(s) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
