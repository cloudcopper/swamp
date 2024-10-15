package ports

import "gorm.io/gorm"

type DB = *gorm.DB

type WithRelationship bool
type Limit int

var ErrRecordNotFound = gorm.ErrRecordNotFound
