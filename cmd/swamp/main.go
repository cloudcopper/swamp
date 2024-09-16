package main

import (
	"log/slog"
	"os"

	"github.com/cloudcopper/swamp"
)

func main() {
	//
	// Create logger
	//
	log := slog.Default()
	log.Info("starting")
	defer log.Info("exiting")
	code := swamp.App(log)
	os.Exit(code)
}
