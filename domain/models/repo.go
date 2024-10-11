package models

import (
	"github.com/cloudcopper/swamp/domain/errors"
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
	Meta        RepoMetas      `gorm:"foreignKey:RepoID" validate:"-"`
	Artifacts   Artifacts      `gorm:"foreignKey:RepoID" yaml:"-" validate:"-"`
}

func (model *Repo) Validate() error {
	err := lib.Validate.Struct(model)
	if err != nil {
		return err
	}

	for _, m := range model.Meta {
		if m.RepoID == "" {
			m.RepoID = model.ID
			continue
		}
		if m.RepoID != model.ID {
			return errors.ErrIncorrectMetaID
		}
	}

	return nil
}
