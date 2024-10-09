package main

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/cloudcopper/swamp/adapters/http"
	"github.com/cloudcopper/swamp/adapters/http/controllers"
	"github.com/cloudcopper/swamp/adapters/repository"
	"github.com/cloudcopper/swamp/domain"
	"github.com/cloudcopper/swamp/domain/errors"
	"github.com/cloudcopper/swamp/domain/models"
	"github.com/cloudcopper/swamp/infra"
	"github.com/cloudcopper/swamp/lib"
	"github.com/cloudcopper/swamp/ports"
)

var numRepos = []int{1, 20}
var numArtifacts = []int{30, 600}

func main() {
	log := slog.Default()
	err := app(log)
	_ = err
}

// Massive copy paste from app.go
func app(log *slog.Logger) error {
	// Force development environment
	os.Setenv("GO_ENV", "development")

	// Create layered filesystem
	fs, err := infra.NewLayerFileSystem(os.Getwd)
	if err != nil {
		log.Error("unable to create layered filesystem!!!", slog.Any("err", err))
		return lib.NewErrorCode(err, errors.RetLayerFilesystemError)
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
	if err := db.AutoMigrate(new(models.Repo), new(models.RepoMeta), new(models.Artifact), new(models.ArtifactMeta)); err != nil {
		log.Error("unable sync database", slog.Any("err", err), slog.String("driver", driver), slog.String("source", source))
		return lib.NewErrorCode(err, errors.RetMigrateDatabaseError)
	}
	// Create repositories
	repoRepository, err := repository.NewRepoRepository(db)
	if err != nil {
		log.Error("unable create repo repository", slog.Any("err", err))
		return lib.NewErrorCode(err, errors.RetCreateRepoRepositoryError)
	}
	artifactRepository, err := repository.NewArtifactRepository(db)
	if err != nil {
		log.Error("unable create artifact repository", slog.Any("err", err))
		return lib.NewErrorCode(err, errors.RetCreateArtifactRepositoryError)
	}
	repositories := repository.NewRepositories(repoRepository, artifactRepository)

	// Perform neccesery startup operations
	if err := startup(log, repositories); err != nil {
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
	return nil
}

// Prefill database with random data
func startup(log ports.Logger, repos domain.Repositories) error {

	return nil
}
