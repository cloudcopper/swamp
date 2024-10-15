package repository

import (
	"fmt"
	"time"

	"github.com/cloudcopper/swamp/domain/models"
	"github.com/cloudcopper/swamp/domain/vo"
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

func (r *ArtifactRepository) Create(model *models.Artifact) error {
	err := r.db.Transaction(func(db *gorm.DB) error {
		/*
			if model.CreatedAt == 0 {
				model.CreatedAt = time.Now().UTC().Unix()
			}
		*/
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

func (r *ArtifactRepository) Update(model *models.Artifact) error {
	db := r.db
	err := db.Save(model).Error
	return err
}

func (r *ArtifactRepository) Delete(model *models.Artifact) error {
	err := r.db.Transaction(func(db *gorm.DB) error {
		// Modify the Repo.Size
		if err := db.Model(&models.Repo{}).Where("id = ?", model.RepoID).Update("size", gorm.Expr("size - ?", model.Size)).Error; err != nil {
			return err
		}

		err := db.Delete(model).Error
		return err
	})
	return err
}

func (r *ArtifactRepository) FindAll() ([]*models.Artifact, error) {
	var artifacts []*models.Artifact
	db := r.db
	db = db.Order("created_at DESC")
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

// FindAllExpired returns all expired artifacts
// as calculated by fields CreatedAt and ExpiredAt and proper state.
// It will not returns non expireable (CreatedAt == ExpiredAt) artifacts.
func (r *ArtifactRepository) FindAllExpired(flags ...interface{}) ([]*models.Artifact, error) {
	var artifacts []*models.Artifact
	now := time.Now().UTC().Unix()
	db := r.db
	db = db.Order("expired_at ASC")
	db = db.Where("expired_at != created_at")
	db = db.Where("expired_at < ?", now)
	db = db.Where("state & ? == ?", vo.ArtifactIsExpired, vo.ArtifactIsExpired)

	for _, flag := range flags {
		switch v := flag.(type) {
		case ports.Limit:
			db = db.Limit(int(v))
		}
	}

	err := db.Find(&artifacts).Error
	return artifacts, err
}

// FindAllNowExpired returns all now expired artifacts.
// Its artifacts which are expired now but has no proper state.
func (r *ArtifactRepository) FindAllNowExpired() ([]*models.Artifact, error) {
	var artifacts []*models.Artifact
	now := time.Now().UTC().Unix()
	db := r.db
	db = db.Order("expired_at ASC")
	db = db.Where("expired_at != created_at")
	db = db.Where("expired_at < ?", now)
	db = db.Where("state & ? != ?", vo.ArtifactIsExpired, vo.ArtifactIsExpired)
	err := db.Find(&artifacts).Error
	return artifacts, err
}

// FindAllNotExpired returns all not expired artifacts
// as calculated by fields CreatedAt and ExpiredAt.
// It will not returns non expireable (CreatedAt == ExpiredAt) artifacts.
// It does not takes into account State field.
func (r *ArtifactRepository) FindAllNotExpired() ([]*models.Artifact, error) {
	var artifacts []*models.Artifact
	now := time.Now().UTC().Unix()
	db := r.db
	db = db.Order("expired_at ASC")
	db = db.Where("expired_at != created_at")
	db = db.Where("expired_at >= ?", now)
	err := db.Find(&artifacts).Error
	return artifacts, err
}

func (r *ArtifactRepository) IterateAll(callback func(repo *models.Artifact) (bool, error)) error {
	db := r.db
	db = db.Order("created_at DESC")
	return iterateAll[models.Artifact](db, callback)
}
