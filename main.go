package main

import (
	"log/slog"
	"os"
	"strings"

	"github.com/davecgh/go-spew/spew"
)

func main() {
	log := slog.Default()
	log.Info("starting")
	defer log.Info("exiting")

	repoConfigs, err := LoadRepoConfigs(log, repoConfigsFileName)
	if err != nil {
		log.Error("unable to load repo config!!!", slog.Any("err", err), slog.String("repoConfigsFileName", repoConfigsFileName))
		os.Exit(1)
	}

	repoConfigs = LoadRepoConfigsDefaults(log, repoConfigs)
	log.Info(spew.Sdump(repoConfigs))

	// TODO
	// - create entity reacting to new seals
	// - create filesystem watcher for seals/repos

	log.Info("create filesystem watcher")
	watcher, err := newWatcher()
	if err != nil {
		log.Error("unable to create new watcher", slog.Any("err", err))
		os.Exit(2)
	}
	defer watcher.Close()

	for k, v := range repoConfigs {
		log := log.With(slog.String("config", k), slog.String("name", v.Name))
		if strings.Contains(v.Name, specialRepoName) || strings.Contains(v.Input, specialRepoName) || strings.Contains(v.Storage, specialRepoName) {
			log.Info("wildcard repos are not supported", slog.String("input", v.Input), slog.String("storage", v.Storage))
			continue
		}
		assert(v.Name != "" && !strings.Contains(v.Name, specialRepoName)) // NOTE We are not supporting whildcard/dynamic repo creations atm
		assert(v.Input != "")                                              // NOTE We are not supporting read-only repo atm
		assert(v.Storage != "")                                            // at least storage must be define

		if !isDirectoryExist(v.Input) {
			log.Error("input directory does not exists", slog.String("input", v.Input))
			continue
		}
		if !isDirectoryExist(v.Storage) {
			log.Error("storage directory does not exists", slog.String("storage", v.Storage))
			continue
		}
	}
	// - traversal all repos

	// TODO Create input web
	// TODO Create read web

	// TODO Add ctrl-c shutdown
	// - close watcher by ctrl-c

	// TODO Create artifacts validator/mover to trash (might need to be done early due to
	//      dynamic repos already created before)
}
