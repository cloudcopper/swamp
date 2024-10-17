package swamp

import (
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cloudcopper/swamp/domain/models"
	"github.com/cloudcopper/swamp/domain/vo"
	"github.com/cloudcopper/swamp/infra"
	"github.com/cloudcopper/swamp/lib"
	"github.com/cloudcopper/swamp/lib/random"
	"github.com/cloudcopper/swamp/ports"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestArtifactSericeCreateArtifactWithMeta(t *testing.T) {
	assert := assert.New(t)
	noErr := func(err error) {
		assert.NoError(err)
		if err != nil {
			t.FailNow()
		}
	}

	testRepoID := "repo2"
	input := filepath.Join("/var/lib/swamp/input", testRepoID)
	storage := filepath.Join("/var/lib/swamp/storage", testRepoID)
	dirs := []string{
		"/what/ever", "/what/other",
		input, storage,
		"/dont/care", "/nobody/other",
	}
	// Create in memory filesystem used by this test only
	fs := afero.NewMemMapFs()
	// Create requested directories on fs
	for _, dir := range dirs {
		noErr(fs.MkdirAll(dir, os.ModePerm))
	}

	repos := []*models.Repo{
		{
			ID:      "repo1",
			Name:    "Repo1",
			Input:   "/what/ever",
			Storage: "/what/other",
		},
		{
			ID:      testRepoID,
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

	testFakeApp(t, fs, repos, func(app *testFakeAppInternals) {
		fs, rr, st, as := app.fs, app.rr, app.st, app.as

		//
		// Create artifact for repo2
		//

		// Create input artifacts
		noErr(afero.WriteFile(fs, filepath.Join(input, "file1.bin"), random.ByteSlice(32*1024), 0644))
		noErr(afero.WriteFile(fs, filepath.Join(input, "file2.bin"), random.ByteSlice(64*1024), 0644))
		noErr(afero.WriteFile(fs, filepath.Join(input, "_export.txt"), []byte(random.Declare(32)), 0644))
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
		// Check preconditions ...
		//
		// ...repo shall has no artifacts
		repoModel, err := rr.FindByID(testRepoID, ports.WithRelationship(true))
		noErr(err)
		assert.NotNil(repoModel)
		assert.Equal(testRepoID, repoModel.ID)
		assert.Equal(input, repoModel.Input)
		assert.Equal(storage, repoModel.Storage)
		assert.Empty(repoModel.Artifacts)
		assert.Zero(repoModel.Size)

		//
		// Signal to artifact serivce to check the checksum file
		//
		as.checkInputFile(repos, fs, checksumFileName)

		//
		// Now check the artifact is well created in repo2...
		//

		// ...input artifacts shall be removed by artifact service
		assert.False(lib.First(afero.Exists(fs, filepath.Join(input, "file1.bin"))))
		assert.False(lib.First(afero.Exists(fs, filepath.Join(input, "file2.bin"))))
		assert.False(lib.First(afero.Exists(fs, filepath.Join(input, "_export.txt"))))
		assert.False(lib.First(afero.Exists(fs, filepath.Join(input, "_createdAt.txt"))))
		assert.False(lib.First(afero.Exists(fs, checksumFileName)))

		// ...repoModel properly updated
		repoModel, err = rr.FindByID(testRepoID, ports.WithRelationship(true))
		noErr(err)
		assert.NotNil(repoModel)
		assert.Equal(testRepoID, repoModel.ID)
		assert.Len(repoModel.Artifacts, 1)
		assert.NotZero(repoModel.Size)

		// ...artifactModel propely created
		artifactModel := repoModel.Artifacts[0]
		assert.Equal(artifactModel.Storage, repoModel.Storage)
		assert.Equal(artifactModel.Size, repoModel.Size)
		assert.Equal(vo.ArtifactIsOK, artifactModel.State)
		// ...and has meta from _export.txt
		assert.NotEmpty(artifactModel.Meta)

		// ...storage has artifacts
		assert.True(lib.First(afero.Exists(fs, filepath.Join(storage, artifactModel.ID, "file1.bin"))))
		assert.True(lib.First(afero.Exists(fs, filepath.Join(storage, artifactModel.ID, "file2.bin"))))
		assert.True(lib.First(afero.Exists(fs, filepath.Join(storage, artifactModel.ID, "_export.txt"))))
		assert.True(lib.First(afero.Exists(fs, filepath.Join(storage, artifactModel.ID, "_createdAt.txt"))))
		_, fileName := filepath.Split(checksumFileName)
		assert.True(lib.First(afero.Exists(fs, filepath.Join(storage, artifactModel.ID, fileName))))

		// ...and has five(5) files as test created
		files, err := st.GetArtifactFiles(repoModel.Storage, artifactModel.ID)
		noErr(err)
		assert.Len(files, 5)
	})
}
