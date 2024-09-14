package ports

import (
	"time"

	"github.com/cloudcopper/swamp/domain/models"
)

type ArtifactStorage interface {
	NewArtifact(*models.Repo, []string, models.ArtifactID) (models.ArtifactID, time.Time, error)
}
