package repository

import (
	"fmt"

	"github.com/cloudcopper/swamp/domain/models"
	"github.com/cloudcopper/swamp/lib"
	"github.com/cloudcopper/swamp/ports"
	"gorm.io/gorm"
)

type RepoRepository struct {
	db ports.DB
}

func NewRepoRepository(db ports.DB) (*RepoRepository, error) {
	r := &RepoRepository{
		db: db,
	}
	_, err := r.FindAll()
	return r, err
}

func (r *RepoRepository) Create(model *models.Repo) error {
	err := r.db.Transaction(func(db *gorm.DB) error {
		if err := model.Validate(); err != nil {
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
	db := r.db.Order("name ASC")

	for _, flag := range flags {
		switch v := flag.(type) {
		case ports.WithRelationship:
			if v {
				db = db.Preload("Meta", func(db ports.DB) ports.DB {
					return db.Order("key ASC")
				})
				db = db.Preload("Artifacts", func(db ports.DB) ports.DB {
					return db.Order("created_at DESC")
				})
			}
		}
	}

	err := db.Find(&repos).Error
	for _, r := range repos {
		lib.Assert(len(r.Artifacts) == 0 || r.Size > 0)
	}

	return repos, err
}

func (r *RepoRepository) FindByID(id models.RepoID, flags ...interface{}) (*models.Repo, error) {
	var repo *models.Repo
	db := r.db

	for _, flag := range flags {
		switch v := flag.(type) {
		case ports.WithRelationship:
			if v {
				db = db.Preload("Artifacts", func(db ports.DB) ports.DB {
					return db.Order("created_at DESC")
				})
			}
		}
	}

	err := db.First(&repo, models.Repo{ID: id}).Error
	return repo, err
}

func (r *RepoRepository) IterateAll(callback func(repo *models.Repo) (bool, error)) error {
	db := r.db.Order("name ASC")
	return iterateAll[models.Repo](db, callback)
}
