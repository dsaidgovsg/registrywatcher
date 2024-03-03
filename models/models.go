package models

import (
	"time"

	"gorm.io/gorm"
)

type Repository struct {
	gorm.Model
	Name       string `gorm:"unique"`
	PinnedTag  string
	AutoDeploy bool
}

type Tag struct {
	gorm.Model
	RepositoryID uint
	Name         string
	Digest       string
	LastPushedAt time.Time
}
