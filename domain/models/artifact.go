package models

import "time"

type ArtifactID string

type Artifact struct {
	ID        ArtifactID `xorm:"pk unique"`
	RepoName  string     `xorm:"index notnull"`
	CreatedAt time.Time
}
