package domain

import "github.com/cloudcopper/swamp/domain/models"

type ArtifactRepository interface {
	Create(model *models.Artifact) error
	Update(model *models.Artifact) error
	Delete(model *models.Artifact) error
	FindAll() ([]*models.Artifact, error)
	FindAllTimeExpired(now int64) ([]*models.Artifact, error)
	FindAllStatusExpired(flags ...interface{}) ([]*models.Artifact, error)
	FindAllStatusNotBroken() ([]*models.Artifact, error)
	FindAllStatusBroken(flags ...interface{}) ([]*models.Artifact, error)
	FindByID(repoID models.RepoID, artifactID models.ArtifactID, flags ...interface{}) (*models.Artifact, error)
	IterateAll(func(*models.Artifact) (bool, error)) error
}
