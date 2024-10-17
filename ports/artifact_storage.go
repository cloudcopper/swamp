package ports

import (
	"github.com/cloudcopper/swamp/domain/models"
)

type NewArtifactInfo struct {
	ID        models.ArtifactID
	Size      int64
	CreatedAt int64
}

type ArtifactStorage interface {
	NewArtifact(input string, storage string, artifactID models.ArtifactID, artifacts []string) (*NewArtifactInfo, error)
	GetArtifactFiles(storage string, artifactID models.ArtifactID) (models.ArtifactFiles, error)
}
