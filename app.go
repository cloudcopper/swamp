package swamp

import (
	"embed"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/cloudcopper/swamp/adapters"
	"github.com/cloudcopper/swamp/adapters/http"
	"github.com/cloudcopper/swamp/adapters/http/controllers"
	"github.com/cloudcopper/swamp/adapters/repository"
	"github.com/cloudcopper/swamp/domain/errors"
	"github.com/cloudcopper/swamp/domain/models"
	"github.com/cloudcopper/swamp/infra"
	"github.com/cloudcopper/swamp/infra/config"
	"github.com/cloudcopper/swamp/infra/disk"
	"github.com/cloudcopper/swamp/lib"
	"github.com/cloudcopper/swamp/ports"
	"github.com/spf13/afero"
)

// App execute application and returns error, when complete by ctrl-c.
// The application reads config(s), templates and static web files
// from layered filesystem.
// Layered filesystem consists of next layers:
//   - ./ of ${SWAMP_ROOT} (optional)
//   - ./ of current working directory
//   - embed.fs given as parameter (cmdFS)
//   - package own embed.fs (appFS)
func App(log ports.Logger, cmdFS embed.FS) error {
	var realFS ports.FS = afero.NewOsFs()

	// EventBus
	var bus ports.EventBus = infra.NewEventBus()
	defer bus.Shutdown()

	// Create layered filesystem
	fs, err := infra.NewLayerFileSystem(config.TopRootFileSystemPath, os.Getwd, cmdFS, appFS)
	if err != nil {
		log.Error("unable to create layered filesystem!!!", slog.Any("err", err))
		return lib.NewErrorCode(err, errors.RetLayerFilesystemError)
	}

	// Load configuration
	cfg, err := config.LoadConfig(log, fs)
	if err != nil {
		log.Error("unable to load config!!!", slog.Any("err", err))
		return lib.NewErrorCode(err, errors.RetLoadConfigError)
	}

	// Open database
	driver := infra.DriverSqlite
	source := infra.SourceSqliteInMemory
	db, closeDb, err := infra.NewDatabase(log, driver, source)
	if err != nil {
		log.Error("unable to create database", slog.Any("err", err), slog.String("driver", driver), slog.String("source", source))
		return lib.NewErrorCode(err, errors.RetCreateDatabaseError)
	}
	defer closeDb()
	// Sync database
	if err := db.AutoMigrate(new(models.Repo), new(models.RepoMeta), new(models.Artifact), new(models.ArtifactMeta), new(models.ArtifactFile)); err != nil {
		log.Error("unable sync database", slog.Any("err", err), slog.String("driver", driver), slog.String("source", source))
		return lib.NewErrorCode(err, errors.RetMigrateDatabaseError)
	}
	// Create repositories
	repoRepository, err := repository.NewRepoRepository(db, realFS)
	if err != nil {
		log.Error("unable create repo repository", slog.Any("err", err))
		return lib.NewErrorCode(err, errors.RetCreateRepoRepositoryError)
	}
	artifactRepository, err := repository.NewArtifactRepository(db, realFS)
	if err != nil {
		log.Error("unable create artifact repository", slog.Any("err", err))
		return lib.NewErrorCode(err, errors.RetCreateArtifactRepositoryError)
	}
	repositories := repository.NewRepositories(repoRepository, artifactRepository)

	// Create artifact storage
	artifactStorage, err := adapters.NewBasicArtifactStorageAdapter(log, realFS)
	if err != nil {
		log.Error("unable to create artifact storage", slog.Any("err", err))
		return lib.NewErrorCode(err, errors.RetCreateArtifactStorageError)
	}
	defer artifactStorage.Close()
	// Create artifacts service:
	// - create artifacts by new checksum files
	// - checking artifacts in storage
	artifactService, err := NewArtifactService(log, bus, artifactStorage, repositories)
	if err != nil {
		log.Error("unable to create artifact service", slog.Any("err", err))
		return lib.NewErrorCode(err, errors.RetCreateChecksumServiceError)
	}
	defer artifactService.Close()
	// Create repo service
	// - signal dangling artifacts at startup/repo update
	// - handling artifacts retention
	repoService := NewRepoService(log, bus, disk.NewFilepathWalk(realFS), repoRepository)
	defer repoService.Close()
	// Create filesystem watcher for input files
	inputWatcher, err := infra.NewWatcherService("input", log, bus)
	if err != nil {
		log.Error("unable to create new watcher service", slog.Any("err", err))
		return lib.NewErrorCode(err, errors.RetCreateInputWatcherError)
	}
	defer inputWatcher.Close()

	// Perform neccesery startup operations
	if err := startup(log, cfg, bus, repoRepository); err != nil {
		return err
	}

	// Create router
	router := http.NewRouter(log)
	// Create render object
	// It also loads templates
	render := infra.NewRender(fs, "layout")
	// Create controllers
	frontPageController := controllers.NewFrontPageController(log, render, repositories)
	repoContoller := controllers.NewRepoController(log, render, repoRepository)
	artifactController := controllers.NewArtifactController(log, render, artifactRepository, artifactStorage)
	aboutPageController := controllers.NewAboutPageController(log, render)
	// Add routes
	router.Get("/", frontPageController.Index)
	router.Get("/about", aboutPageController.Index)
	router.Get("/repo/{repoID}/artifact/{artifactID}/file/*", artifactController.DownloadSingleFile)
	// WARN Next two routes are more like documentation as those are not working
	// Please see https://github.com/go-chi/chi/issues/758 and related
	// So the artifactController.GetHandler would handle calling proper handled
	// base on suffix
	router.Get("/repo/{repoID}/artifact/{artifactID}.tar.gz", artifactController.DownloadGzip)
	router.Get("/repo/{repoID}/artifact/{artifactID}.zip", artifactController.DownloadZip)
	router.Get("/repo/{repoID}/artifact/{artifactID}", artifactController.Get)
	router.Get("/repo/{repoID}", repoContoller.Get)
	// Static file handler
	fileServer := http.FileServer(http.FS(fs))
	router.Handle("/static/*", fileServer)
	// 404 handler
	router.NotFound(frontPageController.NotFound)
	// Create http server
	// The router must has all routes already
	// It will start server in separate goroutine
	addr := config.Listen
	httpServer, err := infra.NewWebServer(log, addr, router)
	if err != nil {
		log.Error("unable create web server", slog.Any("err", err), slog.String("addr", addr))
		return lib.NewErrorCode(err, errors.RetCreateWebServerError)
	}

	// Add ctrl-c shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	log.Info("press ctrl-c to exit")
	// Wait for ctrl-c
	<-c

	// Close http server
	httpServer.Close()

	// Close watcher by ctrl-c
	inputWatcher.Close()
	// TODO Optionally dump whole db to debug file ?
	return nil
}
