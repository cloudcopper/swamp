package viewmodels

import (
	"path/filepath"
	"strings"
	"time"

	"github.com/cloudcopper/swamp/domain/models"
	"github.com/cloudcopper/swamp/domain/vo"
	"github.com/cloudcopper/swamp/lib/types"
)

type Artifact struct {
	RepoID    models.RepoID
	ID        models.ArtifactID
	Size      types.Size
	State     vo.ArtifactState
	CreatedAt time.Time
	ExpiredAt expiredTime
	Checksum  string
	Meta      models.ArtifactMetas
	Files     models.ArtifactFiles
}

type expiredTime time.Time

func (e expiredTime) String() string {
	t := time.Time(e)
	if t.IsZero() {
		return "-"
	}
	return t.String()
}

func NewArtifact(artifact *models.Artifact) *Artifact {
	var expiredAt expiredTime
	if artifact.CreatedAt != artifact.ExpiredAt && artifact.ExpiredAt != 0 {
		expiredAt = expiredTime(time.Unix(artifact.ExpiredAt, 0))
	}
	a := &Artifact{
		RepoID:    artifact.RepoID,
		ID:        artifact.ID,
		Size:      artifact.Size,
		State:     artifact.State,
		CreatedAt: time.Unix(artifact.CreatedAt, 0),
		ExpiredAt: expiredAt,
		Checksum:  artifact.Checksum,
		Meta:      artifact.Meta,
	}
	for _, f := range artifact.Files {
		f.Name = strings.TrimPrefix(f.Name, filepath.Join(artifact.Storage, artifact.ID)+string(filepath.Separator))
		base := filepath.Base(f.Name)
		if base[0] == '_' || base[0] == '.' { // skip "hidden" files
			continue
		}
		a.Files = append(a.Files, f)
	}
	return a
}
