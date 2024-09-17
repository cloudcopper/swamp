package domain

import "github.com/cloudcopper/swamp/domain/models"

type ArtifactRepository interface {
	IterateAll(func(*models.Artifact) (bool, error)) error
	FindAll() ([]*models.Artifact, error)
	FindByID(models.RepoID, models.ArtifactID) (*models.Artifact, error)
	Create(*models.Artifact) error
}
