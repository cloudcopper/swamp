package repository

import (
	"github.com/cloudcopper/swamp/domain/models"
	"xorm.io/xorm"
)

type ArtifactRepository struct {
	engine *xorm.Engine
}

func NewArtifactRepository(engine *xorm.Engine) (*ArtifactRepository, error) {
	r := &ArtifactRepository{
		engine: engine,
	}
	_, err := r.FindAll()
	return r, err
}

func (r *ArtifactRepository) FindAll() ([]*models.Artifact, error) {
	v, err := findAll[models.Artifact](r.engine)
	return v, err
}

func (r *ArtifactRepository) IterateAll(callback func(repo *models.Artifact) (bool, error)) error {
	return iterateAll[models.Artifact](r.engine, callback)
}

func (r *ArtifactRepository) Insert(model *models.Artifact) error {
	_, err := r.engine.Insert(model)
	return err
}
