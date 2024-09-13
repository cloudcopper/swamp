package domain

import (
	"time"
)

type Repo struct {
	Name      string `xorm:"pk unique"`
	Input     string `xorm:"unique"`
	Meta      string
	Storage   string `xorm:"unique"`
	Retention *time.Duration
	Broken    string
	CreatedAt time.Time   `xorm:"created" yaml:"-"`
	Artifacts []*Artifact `xorm:"-" yaml:"-"`
}

func NewRepo(repo Repo) (*Repo, error) {
	r := &Repo{}
	*r = repo

	if err := r.Validate(); err != nil {
		return nil, err
	}

	return r, nil
}

func (r *Repo) Validate() error {
	return nil
}
