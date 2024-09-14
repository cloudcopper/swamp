package domain

type Repositories interface {
	Repo() RepoRepository
	Artifact() ArtifactRepository
}
