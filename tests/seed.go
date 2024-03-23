package testutils

import (
	"registrywatcher/app/models"

	"gorm.io/gorm"
)

type DataSeed struct {
	Repos []models.Repository
	Tags  []models.Tag
}

func (seed *DataSeed) Seed(db *gorm.DB) {
	for _, repo := range seed.Repos {
		db.Create(&repo)
	}
	for _, tag := range seed.Tags {
		db.Create(&tag)
	}
}
