package ports

import "gorm.io/gorm"

type DB = *gorm.DB

type WithRelationship bool
type Limit int
type LimitArtifacts int

var ErrRecordNotFound = gorm.ErrRecordNotFound
