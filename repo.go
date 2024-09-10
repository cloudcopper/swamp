package main

import "strings"

type Repo struct {
	RepoConfig `xorm:"extends"`
	Artifacts  []*Artifact `xorm:"-"`
}

func NewRepo(cfg RepoConfig) (*Repo, error) {
	r := &Repo{
		RepoConfig: cfg,
	}

	if err := r.Validate(); err != nil {
		return nil, err
	}

	return r, nil
}

func (r *Repo) Validate() error {
	return nil
}

func (r *Repo) IsPathInInput(path string) bool {
	return strings.HasPrefix(path, r.Input)
}
