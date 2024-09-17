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

func (r *RepoRepository) IterateAll(callback func(repo *models.Repo) (bool, error)) error {
	db := r.db.Order("name ASC")
	return iterateAll[models.Repo](db, callback)
}

func (r *RepoRepository) FindAll() ([]*models.Repo, error) {
	var repos []*models.Repo
	db := r.db.Order("name ASC")
	err := db.Find(&repos).Error
	return repos, err
}

func (r *RepoRepository) FindAllWithRelations() ([]*models.Repo, error) {
	var repos []*models.Repo
	db := r.db.Order("name ASC").Preload("Artifacts", func(db ports.DB) ports.DB {
		return db.Order("created_at DESC")
	})
	err := db.Find(&repos).Error
	return repos, err
}

func (r *RepoRepository) FindAllByID(id models.RepoID) ([]*models.Repo, error) {
	var repos []*models.Repo
	db := r.db.Order("name ASC").Preload("Artifacts", func(db ports.DB) ports.DB {
		return db.Order("created_at DESC")
	})
	err := db.Find(&repos, models.Repo{ID: id}).Error
	return repos, err
}

func (r *RepoRepository) FindByID(id models.RepoID) (*models.Repo, error) {
	var repo *models.Repo
	db := r.db
	err := db.Find(&repo, models.Repo{ID: id}).Error
	return repo, err
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
