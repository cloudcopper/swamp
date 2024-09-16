package main

import (
	"log/slog"

	"github.com/cloudcopper/swamp"
)

func main() {
	//
	// Create logger
	//
	log := slog.Default()
	log.Info("starting")
	defer log.Info("exiting")
	swamp.App(log)
}
