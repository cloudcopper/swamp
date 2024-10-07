package models

import (
	"github.com/cloudcopper/swamp/domain/vo"
	"github.com/cloudcopper/swamp/lib"
	"github.com/cloudcopper/swamp/lib/types"
)

type ArtifactID = string

const EmptyArtifactID = ArtifactID("")

type Artifact struct {
	ID        ArtifactID       `gorm:"primaryKey;not null" validate:"required,validid"`
	RepoID    RepoID           `gorm:"primaryKey;not null" validate:"required,validid"`
	Size      types.Size       `gorm:"not null" validate:"required,gt=0"`
	State     vo.ArtifactState `gorm:"int" validate:"min=0,max=3"`
	CreatedAt int64            `gorm:"index;column:created_at" validate:"required,gt=0"` // UTC Unix time of creation - equal to ```date +%s```
	Checksum  string           `gorm:"not null" validate:"required,min=8"`
}

func (model *Artifact) Validate() error {
	err := lib.Validate.Struct(model)
	return err
}
