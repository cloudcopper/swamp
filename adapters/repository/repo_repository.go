package repository

import (
	"fmt"

	"github.com/cloudcopper/swamp/domain/models"
	"github.com/cloudcopper/swamp/lib"
	"github.com/cloudcopper/swamp/ports"
	"github.com/go-playground/validator/v10"
	"gorm.io/gorm"
)

type RepoRepository struct {
	db        ports.DB
	fs        ports.FS
	validator *validator.Validate
}

func NewRepoRepository(db ports.DB, f ports.FS) (*RepoRepository, error) {
	r := &RepoRepository{
		db:        db,
		fs:        f,
		validator: lib.NewValidator(f),
	}
	_, err := r.FindAll()
	return r, err
}

func (r *RepoRepository) Create(model *models.Repo) error {
	err := r.db.Transaction(func(db *gorm.DB) error {
		model.Meta.Secure()

		if err := model.Validate(r.validator); err != nil {
			return fmt.Errorf("invalid repo object: %w", err)
		}

		if err := db.Create(model).Error; err != nil {
			return fmt.Errorf("unable to save repo object: %w", err)
		}
		return nil
	})
	return err
}

func (r *RepoRepository) FindAll(flags ...interface{}) ([]*models.Repo, error) {
	var repos []*models.Repo
	err := r.db.Transaction(func(tx *gorm.DB) error {
		db := tx.Order("name ASC")

		limitArtifacts := -1 // -1 - no limit
		withRelationship := false
		for _, flag := range flags {
			switch v := flag.(type) {
			case ports.LimitArtifacts:
				limitArtifacts = int(v)
			case ports.WithRelationship:
				withRelationship = bool(v)
			default:
				panic(flag)
			}
		}

		if withRelationship && limitArtifacts == -1 {
			db = db.Preload("Meta", func(db ports.DB) ports.DB {
				return db.Order("key ASC")
			})
			if limitArtifacts != -1 { // We have to use alternative way to obtain limited number of related models
				db = db.Preload("Artifacts", func(db ports.DB) ports.DB {
					db = db.Order("created_at DESC")
					return db
				})
				db = db.Preload("Artifacts.Meta", func(db ports.DB) ports.DB {
					return db.Order("key ASC")
				})
			}
		}

		if err := db.Find(&repos).Error; err != nil {
			return err
		}

		if withRelationship && limitArtifacts != -1 {
			for _, r := range repos {
				db := tx.Order("created_at DESC")
				db = db.Limit(limitArtifacts)
				db = db.Preload("Meta", func(db ports.DB) ports.DB {
					return db.Order("key ASC")
				})
				if err := db.Where("repo_id = ?", r.RepoID).Find(&r.Artifacts).Error; err != nil {
					return err
				}
			}
		}

		for _, r := range repos {
			lib.Assert((r.ArtifactsCount == 0 && r.Size == 0) || (r.ArtifactsCount > 0 && r.Size > 0))
		}

		return nil
	})

	return repos, err
}

func (r *RepoRepository) FindByID(id models.RepoID, flags ...interface{}) (*models.Repo, error) {
	var repo *models.Repo
	db := r.db

	for _, flag := range flags {
		switch v := flag.(type) {
		case ports.WithRelationship:
			if !v {
				continue
			}
			db = db.Preload("Meta", func(db ports.DB) ports.DB {
				return db.Order("key ASC")
			})
			db = db.Preload("Artifacts", func(db ports.DB) ports.DB {
				return db.Order("created_at DESC")
			})
			db = db.Preload("Artifacts.Meta", func(db ports.DB) ports.DB {
				return db.Order("key ASC")
			})
			db = db.Preload("Artifacts.Files", func(db ports.DB) ports.DB {
				return db.Order("name ASC")
			})
		default:
			panic(flag)
		}
	}

	err := db.First(&repo, models.Repo{RepoID: id}).Error
	lib.Assert(len(repo.Artifacts) == 0 || repo.Size > 0)
	return repo, err
}

func (r *RepoRepository) IterateAll(callback func(repo *models.Repo) (bool, error)) error {
	db := r.db.Order("name ASC")
	return iterateAll[models.Repo](db, callback)
}
