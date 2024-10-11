package models

import "github.com/cloudcopper/swamp/lib"

type ArtifactMetas []*ArtifactMeta

type ArtifactMeta struct {
	RepoID     RepoID     `gorm:"primaryKey;not null" validate:"required,validid"`
	ArtifactID ArtifactID `gorm:"primaryKey;not null" validate:"required,validid"`
	Key        string     `gorm:"primaryKey;not null" validate:"required"`
	Value      string
}

func (model *ArtifactMeta) Validate() error {
	err := lib.Validate.Struct(model)
	return err
}
