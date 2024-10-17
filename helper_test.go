package swamp

import (
	"log/slog"
	"testing"

	"github.com/cloudcopper/swamp/adapters"
	"github.com/cloudcopper/swamp/adapters/repository"
	"github.com/cloudcopper/swamp/domain"
	"github.com/cloudcopper/swamp/domain/models"
	"github.com/cloudcopper/swamp/infra"
	"github.com/cloudcopper/swamp/ports"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

type testFakeAppInternals struct {
	fs ports.FS
	rr domain.RepoRepository
	ar domain.ArtifactRepository
	st *adapters.BasicArtifactStorageAdapter
	as *ArtifactService
}

func testFakeApp(t *testing.T, fs afero.Fs, repos []*models.Repo, callback func(*testFakeAppInternals)) {
	var err error
	assert := assert.New(t)
	noErr := func(err error) {
		assert.NoError(err)
		if err != nil {
			t.FailNow()
		}
	}

	// Create logger
	log := slog.Default()
	// Create eventbus
	var bus ports.EventBus = infra.NewEventBus()
	defer bus.Shutdown()
	// Create artifact storage adapter
	artifactStorage, err := adapters.NewBasicArtifactStorageAdapter(log, fs)
	noErr(err)
	defer artifactStorage.Close()
	// Create database
	driver := infra.DriverSqlite
	source := infra.SourceSqliteInMemory
	db, closeDb, err := infra.NewDatabase(log, driver, source)
	noErr(err)
	defer closeDb()
	noErr(db.AutoMigrate(new(models.Repo), new(models.RepoMeta), new(models.Artifact), new(models.ArtifactMeta)))
	// Create repos repository
	repoRepository, err := repository.NewRepoRepository(db, fs)
	noErr(err)
	// Create artifacts repository
	artifactRepository, err := repository.NewArtifactRepository(db, fs)
	noErr(err)
	// Create artifact service
	artifactService := &ArtifactService{
		log:             log,
		bus:             bus,
		artifactStorage: artifactStorage,
		repositories:    repository.NewRepositories(repoRepository, artifactRepository),
	}
	assert.True(artifactService != nil)

	// Create requested repos
	for _, repo := range repos {
		noErr(repoRepository.Create(repo))
	}

	// Call the callback to continue test
	app := &testFakeAppInternals{
		fs: fs,
		rr: repoRepository,
		ar: artifactRepository,
		st: artifactStorage,
		as: artifactService,
	}

	callback(app)
}
