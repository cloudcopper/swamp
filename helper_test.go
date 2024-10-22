package swamp

import (
	"encoding/hex"
	"fmt"
	"log/slog"
	"path/filepath"
	"testing"

	"github.com/cloudcopper/swamp/adapters"
	"github.com/cloudcopper/swamp/adapters/repository"
	"github.com/cloudcopper/swamp/domain"
	"github.com/cloudcopper/swamp/domain/models"
	"github.com/cloudcopper/swamp/infra"
	"github.com/cloudcopper/swamp/ports"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type testFakeAppInternals struct {
	db *gorm.DB
	fs ports.FS
	rr domain.RepoRepository
	ar domain.ArtifactRepository
	st *adapters.BasicArtifactStorageAdapter
	as *ArtifactService
}

func testFakeApp(t *testing.T, fs afero.Fs, repos []*models.Repo, callback func(*testFakeAppInternals)) {
	var err error
	assert := require.New(t)
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
	noErr(db.AutoMigrate(new(models.Repo), new(models.RepoMeta), new(models.Artifact), new(models.ArtifactMeta), new(models.ArtifactFiles)))
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
		db: db,
		fs: fs,
		rr: repoRepository,
		ar: artifactRepository,
		st: artifactStorage,
		as: artifactService,
	}

	callback(app)
}

func sealArtifact(t *testing.T, fs afero.Fs, input string) string {
	var err error
	assert := require.New(t)
	// Create checksum file
	checksum := ""
	sha256 := &infra.Sha256{}
	info, err := afero.ReadDir(fs, input)
	assert.NoError(err)
	for _, i := range info {
		name := i.Name()
		sum, err := sha256.Sum(fs, filepath.Join(input, name))
		assert.NoError(err)
		checksum += fmt.Sprintf("%v  %s\n", hex.EncodeToString(sum), name)
	}
	assert.NoError(afero.WriteFile(fs, filepath.Join(input, "xxxxxxxx.xxx"), []byte(checksum), 0o644))
	sum, err := sha256.Sum(fs, filepath.Join(input, "xxxxxxxx.xxx"))
	assert.NoError(err)
	checksumFileName := filepath.Join(input, fmt.Sprintf("%v.sha256sum", hex.EncodeToString(sum)))
	assert.NoError(fs.Rename(filepath.Join(input, "xxxxxxxx.xxx"), checksumFileName))

	return checksumFileName
}
