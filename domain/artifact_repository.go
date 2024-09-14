package domain

import "github.com/cloudcopper/swamp/domain/models"

type ArtifactRepository interface {
	FindAll() ([]*models.Artifact, error)
	IterateAll(func(*models.Artifact) (bool, error)) error
	Insert(*models.Artifact) error
}
