package models

import (
	"github.com/cloudcopper/swamp/domain/vo"
	"github.com/cloudcopper/swamp/lib/types"
)

type File struct {
	Name  string           `validate:"required"`
	Size  types.Size       `validate:"required,gt=0"`
	State vo.ArtifactState `validate:"min=0,max=1"` // OK(0) or Broken(1)
}
