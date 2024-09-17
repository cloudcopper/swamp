package domain

import "github.com/cloudcopper/swamp/domain/models"

type RepoRepository interface {
	Create(repo *models.Repo) error
	FindAll() ([]*models.Repo, error)
	FindAllWithRelations() ([]*models.Repo, error)
	IterateAll(func(*models.Repo) (bool, error)) error
	FindByID(models.RepoID) (*models.Repo, error)
	FindAllByID(models.RepoID) ([]*models.Repo, error)
}
