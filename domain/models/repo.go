package models

import (
	"github.com/cloudcopper/swamp/lib"
	"github.com/cloudcopper/swamp/lib/types"
)

type RepoID = string

const EmptyRepoID = RepoID("")

type Repo struct {
	ID          RepoID         `gorm:"primaryKey;not null" validate:"required,validid"`
	Name        string         `gorm:"uniqueIndex;not null;column:name" validate:"required"`
	Description string         `gorm:"string"`
	Input       string         `gorm:"index" validate:"required,min=3,dir,abspath"`
	Storage     string         `gorm:"uniqueIndex;not null" validate:"required,min=3,dir,abspath,nefield=Input"`
	Retention   types.Duration `gorm:"int64" validate:"min=0"`
	Broken      string         `gorm:"string" validate:"omitempty,min=3,eq=/dev/null|dir,abspath,nefield=Input,nefield=Storage"`
	Size        types.Size     `gorm:"int64" validate:"min=0"`
	Artifacts   []*Artifact    `gorm:"foreignKey:RepoID" yaml:"-" validate:"-"`
}

func (model *Repo) Validate() error {
	err := lib.Validate.Struct(model)
	return err
}
