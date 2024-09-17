package repository

import (
	"fmt"
	"time"

	"github.com/cloudcopper/swamp/domain/models"
	"github.com/cloudcopper/swamp/ports"
)

type ArtifactRepository struct {
	db ports.DB
}

func NewArtifactRepository(db ports.DB) (*ArtifactRepository, error) {
	r := &ArtifactRepository{
		db: db,
	}
	_, err := r.FindAll()
	return r, err
}

func (r *ArtifactRepository) IterateAll(callback func(repo *models.Artifact) (bool, error)) error {
	db := r.db.Order("created_at DESC")
	return iterateAll[models.Artifact](db, callback)
}

func (r *ArtifactRepository) FindAll() ([]*models.Artifact, error) {
	var artifacts []*models.Artifact
	db := r.db.Order("created_at DESC")
	err := db.Find(&artifacts).Error
	return artifacts, err
}

func (r *ArtifactRepository) FindByID(repoID models.RepoID, artifactID models.ArtifactID) (*models.Artifact, error) {
	var artifact *models.Artifact
	db := r.db
	err := db.Find(&artifact, models.Artifact{ID: artifactID, RepoID: repoID}).Error
	return artifact, err
}

func (r *ArtifactRepository) Create(model *models.Artifact) error {
	if model.CreatedAt == 0 {
		model.CreatedAt = time.Now().UTC().Unix()
	}
	if err := model.Validate(); err != nil {
		return fmt.Errorf("invalid repo object: %w", err)
	}
	err := r.db.Create(model).Error
	return err
}
