package main

import (
	"log/slog"
)

func main() {
	//
	// Create logger
	//
	log := slog.Default()
	log.Info("starting")
	defer log.Info("exiting")
	app(log)
}
