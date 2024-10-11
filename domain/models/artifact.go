package models

import (
	"github.com/cloudcopper/swamp/domain/errors"
	"github.com/cloudcopper/swamp/domain/vo"
	"github.com/cloudcopper/swamp/lib"
	"github.com/cloudcopper/swamp/lib/types"
)

type ArtifactID = string

const EmptyArtifactID = ArtifactID("")

type Artifacts []*Artifact

type Artifact struct {
	RepoID    RepoID           `gorm:"primaryKey;not null" validate:"required,validid"`
	ID        ArtifactID       `gorm:"primaryKey;not null" validate:"required,validid"`
	Size      types.Size       `gorm:"not null" validate:"required,gt=0"`
	State     vo.ArtifactState `gorm:"int" validate:"min=0,max=3"`
	CreatedAt int64            `gorm:"index;column:created_at" validate:"required,gt=0"` // UTC Unix time of creation - equal to ```date +%s```
	Checksum  string           `gorm:"not null" validate:"required,min=8"`
	Meta      ArtifactMetas    `gorm:"foreignKey:RepoID,ArtifactID;constraint:OnDelete:CASCADE;" validate:"-"`
	Files     ArtifactFiles    `gorm:"-" valudate:"-"`
}

func (model *Artifact) Validate() error {
	err := lib.Validate.Struct(model)
	if err != nil {
		return err
	}

	for _, m := range model.Meta {
		if m.ArtifactID == "" {
			m.ArtifactID = model.ID
			continue
		}
		if m.ArtifactID != model.ID {
			return errors.ErrIncorrectMetaID
		}
	}

	return nil
}
