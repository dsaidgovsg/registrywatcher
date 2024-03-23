package db

import (
	"registrywatcher/app/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func InitDB(dsn string) (db *gorm.DB) {
	// Add your database initialization code here
	var err error
	if db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{}); err != nil {
		panic("failed to connect database")
	}
	if err = db.AutoMigrate(&models.Repository{}, &models.Tag{}); err != nil {
		panic("failed to migrate database")
	}
	return db
}
