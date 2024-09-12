package swamp

import (
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	_ "modernc.org/sqlite" // purego sqlite3 driver
	"xorm.io/xorm"

	"github.com/cloudcopper/swamp/adapters"
	"github.com/cloudcopper/swamp/domain"
	"github.com/cloudcopper/swamp/lib"
	"github.com/cloudcopper/swamp/ports"
	"github.com/davecgh/go-spew/spew"
)

func App(log *ports.Logger) {
	//
	// Load configuration
	//
	repoConfigs, err := LoadRepoConfigs(log, repoConfigsFileName)
	if err != nil {
		log.Error("unable to load repo config!!!", slog.Any("err", err), slog.String("repoConfigsFileName", repoConfigsFileName))
		os.Exit(2)
	}
	repoConfigs = LoadRepoConfigsDefaults(log, repoConfigs)
	log.Info(spew.Sdump(repoConfigs))

	//
	// Open database
	//
	driver := "sqlite"
	source := "file::memory:?cache=shared"
	engine, err := xorm.NewEngine(driver, source) // using modernc.org/sqlite
	if err != nil {
		log.Error("unable connect to database", slog.Any("err", err), slog.String("driver", driver), slog.String("source", source))
		os.Exit(3)
	}
	defer engine.Close()

	//
	// Sync database
	//
	if err := engine.Sync(new(domain.Repo)); err != nil {
		log.Error("unable sync database", slog.Any("err", err), slog.String("driver", driver), slog.String("source", source))
		os.Exit(4)
	}

	//
	// Create repos
	// TODO By hexarch that shall be in separate function in main.go using services (as accessing db and domain)
	// TODO Some of lib.Asserts shall be part of Validate()
	//
	session := engine.NewSession()
	session.Begin()
	for k, v := range repoConfigs {
		log := log.With(slog.String("config", k), slog.String("name", v.Name))
		if strings.Contains(v.Name, specialRepoName) || strings.Contains(v.Input, specialRepoName) || strings.Contains(v.Storage, specialRepoName) {
			log.Warn("wildcard repos are not supported", slog.String("input", v.Input), slog.String("storage", v.Storage))
			continue
		}
		lib.Assert(v.Name != "" && !strings.Contains(v.Name, specialRepoName)) // NOTE We are not supporting wildcard/dynamic repo creations atm
		lib.Assert(v.Input != "")                                              // NOTE We are not supporting read-only repo atm
		lib.Assert(v.Storage != "")                                            // at least storage must be define

		// Check directory as is (potentially relative)
		if !lib.IsDirectoryExist(v.Input) {
			log.Error("input directory does not exists", slog.String("input", v.Input))
			continue
		}
		if !lib.IsDirectoryExist(v.Storage) {
			if err := os.MkdirAll(v.Storage, os.ModePerm); err != nil { // TODO Shall it has more strick permission?
				log.Error("storage directory can not be created", slog.Any("err", err), slog.String("storage", v.Storage))
				continue
			}
		}

		// Convert to abs
		abspath, err := filepath.Abs(v.Input)
		if err != nil {
			log.Error("input directory can not be converted to abspath", slog.Any("err", err), slog.String("input", v.Input))
			continue
		}
		if abspath != v.Input {
			log.Debug("input directory converted to abspath", slog.String("input", v.Input), slog.String("abspath", abspath))
			v.Input = abspath
		}

		// Add repo
		repo, err := domain.NewRepo(v.Repo)
		if err != nil {
			log.Error("unable create repo object", slog.Any("err", err))
			continue
		}
		if _, err := session.Insert(repo); err != nil {
			log.Error("unable insert repo record", slog.Any("err", err))
			os.Exit(5)
		}
	}
	session.Commit()
	session.Close()

	//
	// Create artifacts storage
	//
	artifactsStorage, err := adapters.NewBasicArtifactsStorageAdapter(log, engine)
	if err != nil {
		log.Error("unable to create artifacts storage", slog.Any("err", err))
		os.Exit(6)
	}
	defer artifactsStorage.Close()

	//
	// Create filesystem watcher for input files
	//
	inputWatcher, err := NewWatcherService(log, "input")
	if err != nil {
		log.Error("unable to create new watcher service", slog.Any("err", err))
		os.Exit(7)
	}
	defer inputWatcher.Close()

	//
	// Create service reacting to new checksum files
	//
	checksumService, err := NewChecksumService(log, engine, inputWatcher, artifactsStorage)
	if err != nil {
		log.Error("unable to create checksum service", slog.Any("err", err))
		os.Exit(8)
	}
	defer checksumService.Close()

	// TODO Traversal all repos and "bind" "services". Order is undecided atm
	// TODO We could rescan added dir during AddDir to generate events to binded
	// TODO service for processing artifacts stored prior
	err = Iterate(engine, func(repo *domain.Repo) (bool, error) {
		err := inputWatcher.AddDir(repo.Input)
		if err != nil {
			log.Error("unable add dir to input watcher", slog.Any("err", err), slog.String("input", repo.Input))
		}
		return true, nil
	})
	if err != nil {
		log.Error("failure during adding dir to input watcher", slog.Any("err", err))
		os.Exit(9)
	}

	go func() { // DEBUG this is debug purpose only function
		ch := inputWatcher.GetChanRemoved()
		for name := range ch {
			log.Info("chan removed", slog.String("path", name))
		}
		log.Info("chan rm go done")
	}()

	// TODO Create input web
	// TODO Create read web
	// TODO Create artifacts validator/mover to trash (might need to be done early due to
	//      dynamic repos already created before)

	//
	// Add ctrl-c shutdown
	//
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	log.Info("press ctrl-c to exit")
	<-c
	// Close watcher by ctrl-c
	inputWatcher.Close()
}
