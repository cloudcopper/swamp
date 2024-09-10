package main

import "time"

type Artifact struct {
	ID        string    `xorm:"pk unique"`
	RepoName  string    `xorm:"index"`
	CreatedAt time.Time `xorm:"created"`
}
