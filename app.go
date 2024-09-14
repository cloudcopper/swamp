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
	"xorm.io/xorm/names"

	"github.com/cloudcopper/swamp/adapters"
	"github.com/cloudcopper/swamp/adapters/http/controllers"
	"github.com/cloudcopper/swamp/adapters/repository"
	"github.com/cloudcopper/swamp/domain/models"
	"github.com/cloudcopper/swamp/infra"
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
		os.Exit(10)
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
		os.Exit(11)
	}
	defer engine.Close()
	//
	// Sync database
	//
	engine.SetMapper(names.GonicMapper{})
	if err := engine.Sync2(new(models.Repo), new(models.Artifact)); err != nil {
		log.Error("unable sync database", slog.Any("err", err), slog.String("driver", driver), slog.String("source", source))
		os.Exit(12)
	}
	//
	// Create repositories
	//
	repoRepository, err := repository.NewRepoRepository(engine)
	if err != nil {
		log.Error("unable create repo repository", slog.Any("err", err))
		os.Exit(13)
	}
	artifactRepository, err := repository.NewArtifactRepository(engine)
	if err != nil {
		log.Error("unable create artifact repository", slog.Any("err", err))
		os.Exit(14)
	}
	repositories := repository.NewRepositories(repoRepository, artifactRepository)

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
		repo, err := models.NewRepo(v.Repo)
		if err != nil {
			log.Error("unable create repo object", slog.Any("err", err))
			continue
		}
		if _, err := session.Insert(repo); err != nil {
			log.Error("unable insert repo record", slog.Any("err", err))
			os.Exit(15)
		}
	}
	session.Commit()
	session.Close()

	//
	// Create artifact storage
	//
	artifactStorage, err := adapters.NewBasicArtifactStorageAdapter(log, engine)
	if err != nil {
		log.Error("unable to create artifact storage", slog.Any("err", err))
		os.Exit(16)
	}
	defer artifactStorage.Close()

	//
	// Create filesystem watcher for input files
	//
	inputWatcher, err := NewWatcherService(log, "input")
	if err != nil {
		log.Error("unable to create new watcher service", slog.Any("err", err))
		os.Exit(17)
	}
	defer inputWatcher.Close()

	//
	// Create service reacting to new checksum files
	//
	checksumService, err := NewChecksumService(log, inputWatcher, artifactStorage, repositories)
	if err != nil {
		log.Error("unable to create checksum service", slog.Any("err", err))
		os.Exit(18)
	}
	defer checksumService.Close()

	// TODO Traversal all repos and "bind" "services". Order is undecided atm
	// TODO We could rescan added dir during AddDir to generate events to binded
	// TODO service for processing artifacts stored prior
	err = repoRepository.IterateAll(func(repo *models.Repo) (bool, error) {
		err := inputWatcher.AddDir(repo.Input)
		if err != nil {
			log.Error("unable add dir to input watcher", slog.Any("err", err), slog.String("input", repo.Input))
		}
		return true, nil
	})
	if err != nil {
		log.Error("failure during adding dir to input watcher", slog.Any("err", err))
		os.Exit(19)
	}

	go func() { // DEBUG this is debug purpose only function
		ch := inputWatcher.GetChanRemoved()
		for name := range ch {
			log.Info("chan removed", slog.String("path", name))
		}
		log.Info("chan rm go done")
	}()

	// TODO Create artifacts validator/mover to trash (might need to be done early due to
	//      dynamic repos already created before)

	//
	// Create router
	//
	router := adapters.NewRouter(log)
	//
	// Create controllers
	//
	frontPageController := controllers.NewFrontPageController(log, repoRepository, artifactRepository)
	repoContoller := controllers.NewRepoController(log, repoRepository)
	artifactController := controllers.NewArtifactController(log, artifactRepository)
	//
	// Add routes
	//
	router.Get("/", frontPageController.Index)
	router.Get("/repos", repoContoller.Index)
	router.Route("/repo", func(router ports.Router) {
		router.Get("/", repoContoller.Index)
		router.Get("/{repoName}", repoContoller.Get)
	})
	router.Get("/artifacts", artifactController.Index)
	router.Route("/artifact", func(router ports.Router) {
		router.Get("/", artifactController.Index)
		router.Get("/{artifactID}", artifactController.Get)
	})
	// Create http server
	// The router must has all routes already
	// It will start server in separate goroutine
	addr := ":8080"
	httpServer, err := infra.NewWebServer(log, addr, router)
	if err != nil {
		log.Error("unable create web server", slog.Any("err", err), slog.String("addr", addr))
		os.Exit(20)
	}

	//
	// Add ctrl-c shutdown
	//
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	log.Info("press ctrl-c to exit")
	<-c

	// Close http server
	httpServer.Close()
	// Close watcher by ctrl-c
	inputWatcher.Close()

	// Dump whole db to test file
	dumpFile := "./swamp_db.txt"
	err = engine.DumpAllToFile(dumpFile)
	log.Error("dump whole db to file", slog.String("dumpFile", dumpFile), slog.Any("err", err))
}
