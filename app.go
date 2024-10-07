package swamp

import (
	"database/sql"
	"embed"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	_ "modernc.org/sqlite" // purego sqlite3 driver

	"github.com/cloudcopper/swamp/adapters"
	"github.com/cloudcopper/swamp/adapters/http"
	"github.com/cloudcopper/swamp/adapters/http/controllers"
	"github.com/cloudcopper/swamp/adapters/repository"
	"github.com/cloudcopper/swamp/domain/models"
	"github.com/cloudcopper/swamp/infra"
	"github.com/cloudcopper/swamp/infra/config"
	"github.com/cloudcopper/swamp/lib"
	"github.com/cloudcopper/swamp/ports"
	slogGorm "github.com/orandin/slog-gorm"
)

// The App return code errors
const (
	retLayerFilesystemError          = 9
	retLoadConfigError               = 10
	retConnectDatabaseError          = 11
	retOpenDatabaseError             = 12
	retMigrateDatabaseError          = 13
	retCreateRepoRepositoryError     = 14
	retCreateArtifactRepositoryError = 15
	retCreateArtifactStorageError    = 16
	retCreateChecksumServiceError    = 17
	retCreateInputWatcherError       = 18
	retCreateRepoRecordError         = 20
	retCreateWebServerError          = 40
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
	// EventBus
	var bus ports.EventBus = infra.NewEventBus()
	defer bus.Shutdown()

	// Create layered filesystem
	fs, err := infra.NewLayerFileSystem(config.TopRootFileSystemPath, os.Getwd, cmdFS, appFS)
	if err != nil {
		log.Error("unable to create layered filesystem!!!", slog.Any("err", err))
		return lib.NewErrorCode(err, retLayerFilesystemError)
	}

	// Load configuration
	config, err := config.LoadConfig(log, fs)
	if err != nil {
		log.Error("unable to load config!!!", slog.Any("err", err))
		return lib.NewErrorCode(err, retLoadConfigError)
	}

	//
	// Open database
	//
	driver := "sqlite"
	source := "file::memory:?cache=shared&_pragma=foreign_keys(1)"
	sqlDB, err := sql.Open(driver, source)
	if err != nil {
		log.Error("unable connect to database", slog.Any("err", err), slog.String("driver", driver), slog.String("source", source))
		return lib.NewErrorCode(err, retConnectDatabaseError)
	}
	defer sqlDB.Close()
	dbLogger := slogGorm.New(slogGorm.WithHandler(log.Handler()))
	db, err := gorm.Open(sqlite.Dialector{Conn: sqlDB}, &gorm.Config{
		Logger: dbLogger,
	})
	if err != nil {
		log.Error("unable open orm", slog.Any("err", err), slog.String("driver", driver), slog.String("source", source))
		return lib.NewErrorCode(err, retOpenDatabaseError)
	}
	// Sync database
	if err := db.AutoMigrate(new(models.Repo), new(models.RepoMeta), new(models.Artifact), new(models.ArtifactMeta)); err != nil {
		log.Error("unable sync database", slog.Any("err", err), slog.String("driver", driver), slog.String("source", source))
		return lib.NewErrorCode(err, retMigrateDatabaseError)
	}

	// Create repositories
	repoRepository, err := repository.NewRepoRepository(db)
	if err != nil {
		log.Error("unable create repo repository", slog.Any("err", err))
		return lib.NewErrorCode(err, retCreateRepoRepositoryError)
	}
	artifactRepository, err := repository.NewArtifactRepository(db)
	if err != nil {
		log.Error("unable create artifact repository", slog.Any("err", err))
		return lib.NewErrorCode(err, retCreateArtifactRepositoryError)
	}
	repositories := repository.NewRepositories(repoRepository, artifactRepository)

	// Create artifact storage
	artifactStorage, err := adapters.NewBasicArtifactStorageAdapter(log, db)
	if err != nil {
		log.Error("unable to create artifact storage", slog.Any("err", err))
		return lib.NewErrorCode(err, retCreateArtifactStorageError)
	}
	defer artifactStorage.Close()
	// Create artifacts service:
	// - create artifacts by new checksum files
	// - checking artifacts in storage
	artifactService, err := NewArtifactService(log, bus, artifactStorage, repositories)
	if err != nil {
		log.Error("unable to create artifact service", slog.Any("err", err))
		return lib.NewErrorCode(err, retCreateChecksumServiceError)
	}
	defer artifactService.Close()
	// Create repo service
	repoService := NewRepoService(log, bus, infra.NewFilepathWalk(), repositories)
	defer repoService.Close()
	// Create filesystem watcher for input files
	inputWatcher, err := infra.NewWatcherService("input", log, bus)
	if err != nil {
		log.Error("unable to create new watcher service", slog.Any("err", err))
		return lib.NewErrorCode(err, retCreateInputWatcherError)
	}
	defer inputWatcher.Close()

	// Perform neccesery startup operations
	if err := startup(log, config, bus, repoRepository); err != nil {
		return err
	}

	// Create router
	router := http.NewRouter(log)
	// Create render object
	// It also loads templates
	render := infra.NewRender(fs)
	// Create controllers
	frontPageController := controllers.NewFrontPageController(log, render, repositories)
	repoContoller := controllers.NewRepoController(log, render, repoRepository)
	artifactController := controllers.NewArtifactController(log, render, artifactRepository)
	// Add routes
	router.Get("/", frontPageController.Index)
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
	addr := ":8080"
	httpServer, err := infra.NewWebServer(log, addr, router)
	if err != nil {
		log.Error("unable create web server", slog.Any("err", err), slog.String("addr", addr))
		return lib.NewErrorCode(err, retCreateWebServerError)
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
