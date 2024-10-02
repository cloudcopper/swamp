package main

import (
	"embed"
	"log/slog"
	"os"

	"github.com/cloudcopper/swamp"
	"github.com/cloudcopper/swamp/infra/config"
	"github.com/cloudcopper/swamp/lib"
)

const (
	retNoErrorCode      = 0
	retGenericErrorCode = 1
)

//go:embed templates/**
//go:embed static/**
var fs embed.FS

func main() {
	//
	// Use config file name from env SWAMP_REPO_CONFIG
	// or swamp_repos.yml
	// Note the config file might be embedded!!!
	//
	const defaultReposConfigFileName = "swamp_repos.yml"
	config.ReposConfigFileName = lib.GetEnvDefault("SWAMP_REPO_CONFIG", defaultReposConfigFileName)

	// The first filesystem layer location (nothing if empty)
	config.TopRootFileSystemPath = lib.GetEnvDefault("SWAMP_ROOT", "")
	// Second layer is current working dir
	// Third layer is this app embed fs
	// Last layer is the swamp own embed fs

	//
	// Create logger
	//
	log := slog.Default()
	log.Info("starting")

	err := swamp.App(log, fs)

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
