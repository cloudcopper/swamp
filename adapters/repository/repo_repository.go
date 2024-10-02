package repository

import (
	"fmt"

	"github.com/cloudcopper/swamp/domain/models"
	"github.com/cloudcopper/swamp/ports"
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
	if err := model.Validate(); err != nil {
		return fmt.Errorf("invalid repo object: %w", err)
	}
	if err := r.db.Save(model).Error; err != nil {
		return fmt.Errorf("unable to save repo object: %w", err)
	}
	return nil
}

func (r *RepoRepository) FindAll(flags ...interface{}) ([]*models.Repo, error) {
	var repos []*models.Repo
	db := r.db.Order("name ASC")

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

	err := db.Find(&repos).Error

	// TODO ATM the Repo.Size is not in the DB
	// and we populate it only here.
	// How can it be maintaned well with respect to add/remove artifacts dynamically?
	// Can it be in DB and then we update it when checking dangling repos, adding new or remove expired or broken?
	for _, r := range repos {
		for _, a := range r.Artifacts {
			r.Size += a.Size
		}
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

	err := db.Find(&repo, models.Repo{ID: id}).Error
	return repo, err
}

func (r *RepoRepository) IterateAll(callback func(repo *models.Repo) (bool, error)) error {
	db := r.db.Order("name ASC")
	return iterateAll[models.Repo](db, callback)
}
