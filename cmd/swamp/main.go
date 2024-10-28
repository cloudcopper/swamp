package main

import (
	"embed"
	"flag"
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
	// Use config file name from env SWAMP_REPO_CONFIG
	// or swamp_repos.yml
	// Note the config file might be embedded!!!
	config.ReposConfigFileName = lib.GetEnvDefault("SWAMP_REPO_CONFIG", config.ReposConfigFileName)

	// The first filesystem layer location (nothing if empty)
	config.TopRootFileSystemPath = lib.GetEnvDefault("SWAMP_ROOT", config.TopRootFileSystemPath)
	// Second layer is current working dir
	// Third layer is this app embed fs
	// Last layer is the swamp own embed fs

	// Handle command line arguments
	flag.StringVar(&config.Listen, "listen", config.Listen, "web server listen address")
	flag.StringVar(&config.ReposConfigFileName, "repos", config.ReposConfigFileName, "repos config file name")
	flag.StringVar(&config.TopRootFileSystemPath, "root", config.TopRootFileSystemPath, "first layer of filesystem (optional)")
	flag.DurationVar(&config.TimerExpiredStart, "exp-start", config.TimerExpiredStart, "expired start timer")
	flag.DurationVar(&config.TimerExpiredInterval, "exp-interval", config.TimerExpiredInterval, "expired check interval")
	flag.IntVar(&config.TimerExpiredLimit, "exp-limit", config.TimerExpiredLimit, "expired check limit")
	flag.DurationVar(&config.TimerBrokenStart, "broken-start", config.TimerBrokenStart, "broken start timer")
	flag.DurationVar(&config.TimerBrokenInterval, "broken-interval", config.TimerBrokenInterval, "broken check interval")
	flag.IntVar(&config.TimerBrokenLimit, "broken-limit", config.TimerBrokenLimit, "broken check limit")
	flag.Parse()

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
