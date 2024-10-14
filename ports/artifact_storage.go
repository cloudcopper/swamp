package ports

import (
	"github.com/cloudcopper/swamp/domain/models"
)

type ArtifactStorage interface {
	NewArtifact(input string, storage string, artifactID models.ArtifactID, artifacts []string) (models.ArtifactID, int64, int64, error)
	GetArtifactFiles(storage string, artifactID models.ArtifactID) (models.ArtifactFiles, error)
}
