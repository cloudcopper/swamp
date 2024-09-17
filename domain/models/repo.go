package models

import (
	"time"

	"github.com/cloudcopper/swamp/lib"
)

type RepoID = string

const EmptyRepoID = RepoID("")

type Repo struct {
	ID          RepoID `gorm:"primaryKey;not null" validate:"required,validid"`
	Name        string `gorm:"uniqueIndex;not null;column:name" validate:"required"`
	Description string
	Input       string `gorm:"index" validate:"required,min=3,dir,abspath"`
	Storage     string `gorm:"uniqueIndex;not null" validate:"required,min=3,dir,abspath,nefield=Input"`
	Retention   time.Duration
	Broken      string      `validate:"omitempty,min=3,eq=/dev/null|dir,abspath,nefield=Input,nefield=Storage"`
	Artifacts   []*Artifact `gorm:"foreignKey:RepoID" yaml:"-" validate:"-"`
}

func (model *Repo) Validate() error {
	err := lib.Validate.Struct(model)
	return err
}
