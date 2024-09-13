package ports

import (
	"time"

	"github.com/cloudcopper/swamp/domain"
)

type ArtifactsStorage interface {
	NewArtifacts(*domain.Repo, []string, domain.ArtifactID) (domain.ArtifactID, time.Time, error)
}
