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
	OpenFile(storage string, artifactID models.ArtifactID, filename string) (File, error)
	RemoveArtifact(storage string, artifactID models.ArtifactID) error
}
