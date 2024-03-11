package logging

import (
	"log/slog"
	"os"
)

func NewDefaultLogger(def *slog.Logger) *slog.Logger {
	if def != nil {
		return def
	}

	level := &slog.LevelVar{}

	level.Set(10_000)

	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	}))
}
