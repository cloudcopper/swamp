package ports

import (
	"github.com/cloudcopper/swamp/domain/models"
)

type ArtifactStorage interface {
	NewArtifact(*models.Repo, models.ArtifactID, []string) (models.ArtifactID, int64, int64, error)
	GetArtifactFiles(models.RepoID, models.ArtifactID) (models.ArtifactFiles, error)
}
