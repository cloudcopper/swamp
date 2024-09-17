package main

import (
	"log/slog"
	"os"

	"github.com/cloudcopper/swamp"
	"github.com/cloudcopper/swamp/lib"
)

const (
	retNoErrorCode      = 0
	retGenericErrorCode = 1
)

func main() {
	//
	// Create logger
	//
	log := slog.Default()
	log.Info("starting")

	err := swamp.App(log)

	code := retNoErrorCode
	if err != nil {
		code = retGenericErrorCode
		if i, ok := err.(lib.ErrorCode); ok {
			code = i.Code()
		}
		log.Error("exit", slog.Int("code", code), slog.Any("err", err))
	} else {
		log.Info("exit")
	}

	os.Exit(code)
}
