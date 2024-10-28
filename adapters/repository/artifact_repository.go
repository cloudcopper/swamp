package repository

import (
	"fmt"

	"github.com/cloudcopper/swamp/domain/models"
	"github.com/cloudcopper/swamp/domain/vo"
	"github.com/cloudcopper/swamp/lib"
	"github.com/cloudcopper/swamp/ports"
	"github.com/go-playground/validator/v10"
	"gorm.io/gorm"
)

type ArtifactRepository struct {
	db        ports.DB
	fs        ports.FS
	validator *validator.Validate
}

func NewArtifactRepository(db ports.DB, f ports.FS) (*ArtifactRepository, error) {
	r := &ArtifactRepository{
		db:        db,
		fs:        f,
		validator: lib.NewValidator(f),
	}
	_, err := r.FindAll()
	return r, err
}

func (r *ArtifactRepository) Create(model *models.Artifact) error {
	err := r.db.Transaction(func(db *gorm.DB) error {
		model.Meta.Secure()

		if err := model.Validate(r.validator); err != nil {
			return fmt.Errorf("invalid artifact object: %w", err)
		}

		// Create artifact model
		if err := db.Create(model).Error; err != nil {
			return err
		}

		// Modify the Repo.Size
		if err := db.Model(&models.Repo{}).Where("repo_id = ?", model.RepoID).Update("size", gorm.Expr("size + ?", model.Size)).Error; err != nil {
			return err
		}
		// Modify the Repo.ArtifactsCount
		err := db.Model(&models.Repo{}).Where("repo_id = ?", model.RepoID).Update("artifacts_count", gorm.Expr("artifacts_count + ?", 1)).Error
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
		if err := db.Model(&models.Repo{}).Where("repo_id = ?", model.RepoID).Update("size", gorm.Expr("size - ?", model.Size)).Error; err != nil {
			return err
		}
		// Modify the Repo.ArtifactsCount
		if err := db.Model(&models.Repo{}).Where("repo_id = ?", model.RepoID).Update("artifacts_count", gorm.Expr("artifacts_count - ?", 1)).Error; err != nil {
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
	db = db.Preload("Meta", func(db ports.DB) ports.DB {
		return db.Order("key DESC")
	})
	err := db.Find(&artifacts).Error
	return artifacts, err
}

func (r *ArtifactRepository) FindByID(repoID models.RepoID, artifactID models.ArtifactID, flags ...interface{}) (*models.Artifact, error) {
	var artifact *models.Artifact
	db := r.db

	for _, flag := range flags {
		switch v := flag.(type) {
		case ports.WithRelationship:
			if !v {
				continue
			}
			db = db.Preload("Meta", func(db ports.DB) ports.DB {
				return db.Order("key DESC")
			})
			db = db.Preload("Files")
		default:
			panic(flag)
		}
	}

	err := db.First(&artifact, models.Artifact{ArtifactID: artifactID, RepoID: repoID}).Error
	artifact.Files.Sort(artifact.Storage)
	return artifact, err
}

// FindAllTimeExpired returns all now expired artifacts.
// Its artifacts which are expired now but has no proper state.
func (r *ArtifactRepository) FindAllTimeExpired(now int64) ([]*models.Artifact, error) {
	var artifacts []*models.Artifact
	db := r.db
	db = db.Order("expired_at ASC")
	db = db.Where("expired_at != created_at")
	db = db.Where("expired_at < ?", now)
	db = db.Where("state & ? != ?", vo.ArtifactIsExpired, vo.ArtifactIsExpired)
	err := db.Find(&artifacts).Error
	return artifacts, err
}

// FindAllStatusExpired returns all expired artifacts
// as calculated by fields CreatedAt and ExpiredAt and proper state.
// It will not returns non expireable (CreatedAt == ExpiredAt) artifacts,
// nor broken expired.
func (r *ArtifactRepository) FindAllStatusExpired(flags ...interface{}) ([]*models.Artifact, error) {
	var artifacts []*models.Artifact
	db := r.db
	db = db.Order("expired_at ASC")
	db = db.Where("expired_at != created_at")
	db = db.Where("state == ?", vo.ArtifactIsExpired)

	for _, flag := range flags {
		switch v := flag.(type) {
		case ports.Limit:
			db = db.Limit(int(v))
		default:
			panic(flag)
		}
	}

	err := db.Find(&artifacts).Error
	return artifacts, err
}

func (r *ArtifactRepository) FindAllStatusNotBroken() ([]*models.Artifact, error) {
	var artifacts []*models.Artifact
	db := r.db
	db = db.Order("created_at ASC")
	db = db.Where("state & ? != ?", vo.ArtifactIsBroken, vo.ArtifactIsBroken)

	err := db.Find(&artifacts).Error
	return artifacts, err
}

func (r *ArtifactRepository) FindAllStatusBroken(flags ...interface{}) ([]*models.Artifact, error) {
	var artifacts []*models.Artifact
	db := r.db
	db = db.Order("created_at ASC")
	db = db.Where("state & ? == ?", vo.ArtifactIsBroken, vo.ArtifactIsBroken)

	for _, flag := range flags {
		switch v := flag.(type) {
		case ports.Limit:
			db = db.Limit(int(v))
		default:
			panic(flag)
		}
	}

	err := db.Find(&artifacts).Error
	return artifacts, err
}

func (r *ArtifactRepository) IterateAll(callback func(repo *models.Artifact) (bool, error)) error {
	db := r.db
	db = db.Order("created_at DESC")
	return iterateAll[models.Artifact](db, callback)
}
