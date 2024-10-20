package models

import (
	"path/filepath"
	"slices"
	"strings"

	"github.com/cloudcopper/swamp/domain/vo"
	"github.com/cloudcopper/swamp/lib/types"
)

type ArtifactFiles []*ArtifactFile
type ArtifactFile struct {
	RepoID     RepoID           `gorm:"primaryKey;not null" validate:"required,validid"`
	ArtifactID ArtifactID       `gorm:"primaryKey;not null" validate:"required,validid"`
	Name       string           `gorm:"primaryKey;not null" validate:"required"`
	Size       types.Size       `validate:"required,ge=0"`
	State      vo.ArtifactState `validate:"min=0,max=1"` // OK(0) or Broken(1)
}

func (files ArtifactFiles) Sort(path string) {
	slices.SortFunc(files, func(a, b *ArtifactFile) int {
		s := []string{
			strings.TrimPrefix(a.Name, path+string(filepath.Separator)),
			strings.TrimPrefix(b.Name, path+string(filepath.Separator)),
		}
		for i := range s {
			if strings.HasPrefix(s[i], "_created") {
				s[i] = "zzzz" + s[i]
			} else if s[i][0] == '_' {
				s[i] = "zzz" + s[i]
			}
			if strings.HasSuffix(s[i], ".md5") {
				s[i] = "zzzzzz" + s[i]
			}
			if strings.HasSuffix(s[i], ".sha256sum") {
				s[i] = "zzzzzzz" + s[i]
			}
		}

		if s[0] > s[1] {
			return 1
		}
		if s[0] < s[1] {
			return -1
		}
		return 0
	})

}
