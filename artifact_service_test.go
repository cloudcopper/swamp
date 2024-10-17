package swamp

import (
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cloudcopper/swamp/adapters"
	"github.com/cloudcopper/swamp/adapters/repository"
	"github.com/cloudcopper/swamp/domain/models"
	"github.com/cloudcopper/swamp/infra"
	"github.com/cloudcopper/swamp/lib/random"
	"github.com/cloudcopper/swamp/ports"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestArtifactSericeCreateArtifactWithMeta(t *testing.T) {
	var err error
	assert := assert.New(t)
	noErr := func(err error) {
		assert.NoError(err)
		if err != nil {
			t.FailNow()
		}
	}

	// Create in memory filesystem used by this test only
	fs := afero.NewMemMapFs()

	input := "/var/lib/swamp/input/repo2"
	storage := "/var/lib/swamp/storage/repo2"
	noErr(fs.MkdirAll(input, os.ModePerm))
	noErr(fs.MkdirAll(storage, os.ModePerm))

	repos := []*models.Repo{
		{
			ID:      "repo1",
			Name:    "Repo1",
			Input:   "/what/ever",
			Storage: "/what/other",
		},
		{
			ID:      "repo2",
			Name:    "Repo2",
			Input:   input,
			Storage: storage,
		},
		{
			ID:      "repo3",
			Name:    "Repo3",
			Input:   "/dont/care",
			Storage: "/nobody/other",
		},
	}

	// Create input artifact
	noErr(afero.WriteFile(fs, filepath.Join(input, "file1.bin"), random.ByteSlice(32*1024), 0644))
	noErr(afero.WriteFile(fs, filepath.Join(input, "file2.bin"), random.ByteSlice(64*1024), 0644))
	noErr(afero.WriteFile(fs, filepath.Join(input, "export.txt"), []byte(random.Declare(32)), 0644))
	noErr(afero.WriteFile(fs, filepath.Join(input, "_createdAt.txt"), []byte(fmt.Sprintf("%v", time.Now().UTC().Unix())), 0644))

	// Create checksum file
	checksum := ""
	sha256 := &infra.Sha256{}
	info, err := afero.ReadDir(fs, input)
	noErr(err)
	for _, i := range info {
		name := i.Name()
		sum, err := sha256.Sum(fs, filepath.Join(input, name))
		noErr(err)
		checksum += fmt.Sprintf("%v  %s\n", hex.EncodeToString(sum), name)
	}
	noErr(afero.WriteFile(fs, filepath.Join(input, "xxxxxxxx.xxx"), []byte(checksum), 0644))
	sum, err := sha256.Sum(fs, filepath.Join(input, "xxxxxxxx.xxx"))
	noErr(err)
	checksumFileName := filepath.Join(input, fmt.Sprintf("%v.sha256sum", hex.EncodeToString(sum)))
	noErr(fs.Rename(filepath.Join(input, "xxxxxxxx.xxx"), checksumFileName))

	//
	log := slog.Default()
	var bus ports.EventBus = infra.NewEventBus()
	defer bus.Shutdown()
	//
	artifactStorage, err := adapters.NewBasicArtifactStorageAdapter(log, fs)
	noErr(err)
	defer artifactStorage.Close()
	//
	driver := infra.DriverSqlite
	source := infra.SourceSqliteInMemory
	db, closeDb, err := infra.NewDatabase(log, driver, source)
	noErr(err)
	defer closeDb()
	noErr(db.AutoMigrate(new(models.Repo), new(models.RepoMeta), new(models.Artifact), new(models.ArtifactMeta)))
	repoRepository, err := repository.NewRepoRepository(db, fs)
	noErr(err)
	artifactRepository, err := repository.NewArtifactRepository(db, fs)
	noErr(err)

	//
	as := &ArtifactService{
		log:             log,
		bus:             bus,
		artifactStorage: artifactStorage,
		repositories:    repository.NewRepositories(repoRepository, artifactRepository),
	}
	assert.True(as != nil)

	//
	for _, repo := range repos {
		noErr(fs.MkdirAll(repo.Input, os.ModePerm))
		noErr(fs.MkdirAll(repo.Storage, os.ModePerm))
		noErr(repoRepository.Create(repo))
	}

	as.checkInputFile(repos, fs, checksumFileName)
}
