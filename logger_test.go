package swamp

import (
	"log/slog"
	"os"
)

func init() {
	level := new(slog.LevelVar)
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	})
	slog.SetDefault(slog.New(handler))
	level.Set(slog.LevelDebug)
}
