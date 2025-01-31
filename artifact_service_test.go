package swamp

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cloudcopper/random"
	"github.com/cloudcopper/swamp/domain/models"
	"github.com/cloudcopper/swamp/domain/vo"
	"github.com/cloudcopper/swamp/lib"
	"github.com/cloudcopper/swamp/lib/types"
	"github.com/cloudcopper/swamp/ports"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

// TestArtifactServiceScenario1:
//   - Creates 3x repos
//   - Create one artifact
//   - Expire it
//   - Remove expired artifact
func TestArtifactServiceScenario1(t *testing.T) {
	assert := require.New(t)

	testRepoID := "repo2"
	input := "/var/lib/swamp/input/" + testRepoID
	storage := "/var/lib/swamp/storage/" + testRepoID
	dirs := []string{
		"/what/ever", "/what/other",
		input, storage,
		"/dont/care", "/nobody/other",
	}
	// Create in memory filesystem used by this test only
	fs := afero.NewMemMapFs()
	// Create requested directories on fs
	for _, dir := range dirs {
		assert.NoError(fs.MkdirAll(dir, os.ModePerm))
	}

	repos := []*models.Repo{
		{
			RepoID:    "repo1",
			Name:      "Repo1",
			Input:     "/what/ever",
			Storage:   "/what/other",
			Retention: types.Duration(30 * 24 * time.Hour),
		},
		{
			RepoID:    testRepoID,
			Name:      "Repo2",
			Input:     input,
			Storage:   storage,
			Retention: types.Duration(24 * time.Hour),
		},
		{
			RepoID:    "repo3",
			Name:      "Repo3",
			Input:     "/dont/care",
			Storage:   "/nobody/other",
			Retention: types.Duration(365 * 24 * time.Hour),
		},
	}

	testFakeApp(t, fs, repos, func(app *testFakeAppInternals) {
		db, fs, rr, ar, st, as := app.db, app.fs, app.rr, app.ar, app.st, app.as

		//
		// - Create one artifact (in repo2)
		//

		// Create input artifact files - five(5) files - four(4) artifacts + checksum file
		creationTime := time.Now().UTC().Unix()
		assert.NoError(afero.WriteFile(fs, filepath.Join(input, "file1.bin"), random.ByteSlice(32*1024), 0o644))
		assert.NoError(afero.WriteFile(fs, filepath.Join(input, "file2.bin"), random.ByteSlice(64*1024), 0o644))
		assert.NoError(afero.WriteFile(fs, filepath.Join(input, "_export.txt"), []byte(random.Declare(32)), 0o644))
		assert.NoError(afero.WriteFile(fs, filepath.Join(input, "_createdAt.txt"), []byte(fmt.Sprintf("%v", creationTime)), 0o644))
		checksumFileName := sealArtifact(t, fs, input)

		//
		// Check preconditions ...
		//
		// ...repo shall has no artifacts
		repoModel, err := rr.FindByID(testRepoID, ports.WithRelationship(true))
		assert.NoError(err)
		assert.NotNil(repoModel)
		assert.Equal(testRepoID, repoModel.RepoID)
		assert.Equal(input, repoModel.Input)
		assert.Equal(storage, repoModel.Storage)
		assert.Empty(repoModel.Artifacts)
		assert.Zero(repoModel.Size)
		// ...no artifact metas exists
		var metas models.ArtifactMetas
		metas = models.ArtifactMetas{}
		assert.NoError(db.Find(&metas).Error)
		assert.Empty(metas)
		// ...no artifact files exists
		var files models.ArtifactFiles
		assert.NoError(db.Find(&files).Error)
		assert.Empty(files)

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

		// TODO Check access over artifact storage
		_ = st

		// ...repoModel properly updated
		repoModel, err = rr.FindByID(testRepoID, ports.WithRelationship(true))
		assert.NoError(err)
		assert.NotNil(repoModel)
		assert.Equal(testRepoID, repoModel.RepoID)
		assert.NotZero(repoModel.Size)
		assert.Len(repoModel.Artifacts, 1)
		assert.Equal(len(repoModel.Artifacts), repoModel.ArtifactsCount)

		// ...artifactModel propely created
		artifactModel := repoModel.Artifacts[0]
		assert.Equal(artifactModel.Storage, repoModel.Storage)
		assert.Equal(artifactModel.Size, repoModel.Size)
		assert.Equal(vo.ArtifactIsOK, artifactModel.State)
		// ...and has meta from _export.txt
		assert.NotEmpty(artifactModel.Meta)
		// ...artifact has files
		assert.Len(artifactModel.Files, 5)
		// ...artifact metas updated
		metas = models.ArtifactMetas{}
		assert.NoError(db.Find(&metas).Error)
		assert.NotEmpty(metas)
		// ...artifact files updated
		files = models.ArtifactFiles{}
		assert.NoError(db.Find(&files).Error)
		assert.Len(files, 5)

		// ...storage has artifacts
		storedArtifactPath := filepath.Join(storage, artifactModel.ArtifactID)
		assert.True(lib.First(afero.Exists(fs, filepath.Join(storedArtifactPath, "file1.bin"))))
		assert.True(lib.First(afero.Exists(fs, filepath.Join(storedArtifactPath, "file2.bin"))))
		assert.True(lib.First(afero.Exists(fs, filepath.Join(storedArtifactPath, "_export.txt"))))
		assert.True(lib.First(afero.Exists(fs, filepath.Join(storedArtifactPath, "_createdAt.txt"))))
		_, fileName := filepath.Split(checksumFileName)
		assert.True(lib.First(afero.Exists(fs, filepath.Join(storedArtifactPath, fileName))))

		// ...and has five(5) files as test created
		/* TODO Change it!!
		files, err = st.GetArtifactFiles(repoModel.Storage, artifactModel.ArtifactID)
		assert.NoError(err)
		assert.Len(files, 5)
		*/

		//
		// - Expire it
		//
		a, err := ar.FindAllStatusExpired()
		assert.NoError(err)
		assert.Empty(a)

		now := creationTime + int64(repos[1].Retention/1000000000) + 1
		as.markExpiredArtifacts(now)
		a, err = ar.FindAllStatusExpired()
		assert.NoError(err)
		assert.Len(a, 1)
		artifactModel = a[0]
		assert.True(artifactModel.State.IsExpired())

		//
		// - Remove expired artifact
		//
		limit := 1
		as.removeExpiredArtifacts(limit)

		// ...shall has no status expured artifacts
		a, err = ar.FindAllStatusExpired()
		assert.NoError(err)
		assert.Empty(a)
		// ...shall has no artifacts
		a, err = ar.FindAll()
		assert.NoError(err)
		assert.Empty(a)
		// ...repo shall has no artifacts and be zero size
		repoModel, err = rr.FindByID(testRepoID, ports.WithRelationship(true))
		assert.NoError(err)
		assert.Empty(repoModel.Artifacts)
		assert.Zero(repoModel.Size)
		assert.Zero(repoModel.ArtifactsCount)

		// ...artifact meta shall be empty (we removed last artifact)
		metas = models.ArtifactMetas{}
		assert.NoError(db.Find(&metas).Error)
		assert.Empty(metas)
		// ...artifact files shall be empty (we removed last artifact)
		files = models.ArtifactFiles{}
		assert.NoError(db.Find(&files).Error)
		assert.Empty(files)

		// Check files are removed
		exist, err := afero.DirExists(fs, storedArtifactPath)
		assert.NoError(err)
		assert.False(exist)
	})
}
