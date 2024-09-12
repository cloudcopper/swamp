package ports

import "github.com/cloudcopper/swamp/domain"

type ArtifactsStorage interface {
	NewArtifacts(*domain.Repo, []string, domain.ArtifactID) (domain.ArtifactID, error)
}
