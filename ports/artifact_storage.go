package ports

import (
	"github.com/cloudcopper/swamp/domain/models"
)

type NewArtifactInfo struct {
	Size      int64
	CreatedAt int64
}

type ArtifactStorage interface {
	NewArtifact(src FS, input string, artifacts []string, storage string, artifactID models.ArtifactID) (*NewArtifactInfo, error)
	OpenFile(storage string, artifactID models.ArtifactID, filename string) (File, error)
	RemoveArtifact(storage string, artifactID models.ArtifactID) error
}
