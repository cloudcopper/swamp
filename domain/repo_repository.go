package domain

import "github.com/cloudcopper/swamp/domain/models"

type RepoRepository interface {
	FindAll() ([]*models.Repo, error)
	IterateAll(func(*models.Repo) (bool, error)) error
}
