package repository

import "github.com/cloudcopper/swamp/domain"

type Repositories struct {
	repo     domain.RepoRepository
	artifact domain.ArtifactRepository
}

func NewRepositories(repo domain.RepoRepository, artifact domain.ArtifactRepository) *Repositories {
	r := &Repositories{
		repo:     repo,
		artifact: artifact,
	}

	return r
}

func (r *Repositories) Repo() domain.RepoRepository {
	return r.repo
}

func (r *Repositories) Artifact() domain.ArtifactRepository {
	return r.artifact
}
