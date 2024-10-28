package viewmodels

import (
	"github.com/cloudcopper/swamp/domain/models"
	"github.com/cloudcopper/swamp/lib/types"
)

type Repo struct {
	RepoID         models.RepoID
	Name           string
	Description    string
	Input          string
	Storage        string
	Broken         string
	Retention      types.Duration
	Size           types.Size
	ArtifactsCount int
	Meta           models.RepoMetas
	Artifacts      []*Artifact
}

func NewRepo(repo *models.Repo) *Repo {
	r := &Repo{
		RepoID:         repo.RepoID,
		Name:           repo.Name,
		Description:    repo.Description,
		Input:          repo.Input,
		Storage:        repo.Storage,
		Broken:         repo.Broken,
		Retention:      repo.Retention,
		Size:           repo.Size,
		ArtifactsCount: repo.ArtifactsCount,
		Meta:           repo.Meta,
	}

	for _, a := range repo.Artifacts {
		r.Artifacts = append(r.Artifacts, NewArtifact(a))
	}
	return r
}

func NewRepos(repos []*models.Repo) []*Repo {
	a := []*Repo{}
	for _, repo := range repos {
		a = append(a, NewRepo(repo))
	}
	return a
}
