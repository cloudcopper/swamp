package domain

import "time"

type ArtifactID string

type Artifact struct {
	ID        ArtifactID `xorm:"pk unique"`
	RepoName  string     `xorm:"index"`
	CreatedAt time.Time  `xorm:"created"`
}
