package models

import (
	"fmt"
	"time"

	"github.com/cloudcopper/swamp/domain/errors"
	"github.com/cloudcopper/swamp/domain/vo"
	"github.com/cloudcopper/swamp/lib/types"
	"github.com/go-playground/validator/v10"
)

type ArtifactID = string

const EmptyArtifactID = ArtifactID("")

type Artifacts []*Artifact

type Artifact struct {
	RepoID     RepoID           `gorm:"primaryKey;not null" validate:"required,validid"`
	ArtifactID ArtifactID       `gorm:"primaryKey;not null" validate:"required,validid"`
	Storage    string           `gorm:"not null" validate:"required,min=3,dir,abspath"`
	Size       types.Size       `gorm:"not null" validate:"required,gt=0"`
	State      vo.ArtifactState `gorm:"int" validate:"min=0,max=3"`
	CreatedAt  int64            `gorm:"index;column:created_at" validate:"required,gt=0"` // UTC Unix time of creation - equal to ```date +%s```
	ExpiredAt  int64            `gorm:"index;column:expired_at" validate:"required,gt=0"` // UTC Unix time at which the artifacts expires
	Checksum   string           `gorm:"not null" validate:"required,min=8"`
	Meta       ArtifactMetas    `gorm:"foreignKey:RepoID,ArtifactID;constraint:OnDelete:CASCADE;" validate:"-"`
	Files      ArtifactFiles    `gorm:"foreignKey:RepoID,ArtifactID;constraint:OnDelete:CASCADE;" valudate:"-"`
}

func (model *Artifact) Validate(val *validator.Validate) error {
	err := val.Struct(model)
	if err != nil {
		return err
	}

	// Extra check CreatedAt, ExpiredAt and State.IsExpired()
	now := time.Now().UTC().Unix()
	if model.CreatedAt == model.ExpiredAt && model.State.IsExpired() {
		return fmt.Errorf("artifact has false expired state")
	}
	if model.CreatedAt != model.ExpiredAt && model.State.IsExpired() != (model.ExpiredAt < now) {
		return fmt.Errorf("artifact has wrong expired state")
	}

	for _, m := range model.Meta {
		if m.RepoID == "" {
			m.RepoID = model.RepoID
		}
		if m.RepoID != model.RepoID {
			return errors.ErrIncorrectMetaID
		}
		if m.ArtifactID == "" {
			m.ArtifactID = model.ArtifactID
		}
		if m.ArtifactID != model.ArtifactID {
			return errors.ErrIncorrectMetaID
		}
	}
	for _, f := range model.Files {
		if f.RepoID == "" {
			f.RepoID = model.RepoID
		}
		if f.RepoID != model.RepoID {
			return errors.ErrIncorrectFileID
		}
		if f.ArtifactID == "" {
			f.ArtifactID = model.ArtifactID
		}
		if f.ArtifactID != model.ArtifactID {
			return errors.ErrIncorrectFileID
		}
		if f.State.IsBroken() {
			model.State |= vo.ArtifactIsBroken
		}
	}

	return nil
}

func (a Artifacts) HasArtifactID(artifactID ArtifactID) bool {
	for _, artifact := range a {
		if artifact.ArtifactID == artifactID {
			return true
		}
	}
	return false
}
