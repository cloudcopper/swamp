package domain

import "github.com/cloudcopper/swamp/domain/models"

type RepoRepository interface {
	Create(model *models.Repo) error
	FindAll(flags ...interface{}) ([]*models.Repo, error)
	FindByID(id models.RepoID, flags ...interface{}) (*models.Repo, error)
	IterateAll(func(*models.Repo) (bool, error)) error
}
