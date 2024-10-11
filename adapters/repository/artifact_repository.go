package repository

import (
	"fmt"
	"time"

	"github.com/cloudcopper/swamp/domain/models"
	"github.com/cloudcopper/swamp/ports"
	"gorm.io/gorm"
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

func (r *ArtifactRepository) FindByID(repoID models.RepoID, artifactID models.ArtifactID, flags ...interface{}) (*models.Artifact, error) {
	var artifact *models.Artifact
	db := r.db

	for _, flag := range flags {
		switch v := flag.(type) {
		case ports.WithRelationship:
			if v {
				db = db.Preload("Meta", func(db ports.DB) ports.DB {
					return db.Order("key DESC")
				})
			}
		}
	}

	err := db.First(&artifact, models.Artifact{ID: artifactID, RepoID: repoID}).Error
	return artifact, err
}

func (r *ArtifactRepository) Create(model *models.Artifact) error {
	err := r.db.Transaction(func(db *gorm.DB) error {
		if model.CreatedAt == 0 {
			model.CreatedAt = time.Now().UTC().Unix()
		}
		if err := model.Validate(); err != nil {
			return fmt.Errorf("invalid artifact object: %w", err)
		}

		// Create artifact model
		if err := db.Create(model).Error; err != nil {
			return err
		}

		// Modify the Repo.Size
		err := db.Model(&models.Repo{}).Where("id = ?", model.RepoID).Update("size", gorm.Expr("size + ?", model.Size)).Error
		return err
	})
	return err
}
