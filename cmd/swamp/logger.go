package main

import (
	"log/slog"
	"os"

	"github.com/phsym/console-slog"
)

const timeFormat string = "2006-01-02 15:04:05.000" // may be time.DateTime

func init() {
	setDefaultLogger(slog.LevelInfo)
}

func setDefaultLogger(level slog.Leveler) {
	handler := console.NewHandler(os.Stdout, &console.HandlerOptions{
		Level:      level,
		TimeFormat: timeFormat,
	})
	log := slog.New(handler)
	slog.SetDefault(log)
}
