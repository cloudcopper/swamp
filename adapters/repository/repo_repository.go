package repository

import (
	"github.com/cloudcopper/swamp/domain/models"
	"xorm.io/xorm"
)

type RepoRepository struct {
	engine *xorm.Engine
}

func NewRepoRepository(engine *xorm.Engine) (*RepoRepository, error) {
	r := &RepoRepository{
		engine: engine,
	}
	_, err := r.FindAll()
	return r, err
}

func (r *RepoRepository) FindAll() ([]*models.Repo, error) {
	v, err := findAll[models.Repo](r.engine)
	return v, err
}

func (r *RepoRepository) IterateAll(callback func(repo *models.Repo) (bool, error)) error {
	return iterateAll[models.Repo](r.engine, callback)
}
