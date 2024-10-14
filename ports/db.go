package ports

import "gorm.io/gorm"

type DB = *gorm.DB

type WithRelationship bool

var ErrRecordNotFound = gorm.ErrRecordNotFound
