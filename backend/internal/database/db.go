package database

import (
	"awewe-panel/internal/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Init(path string) error {
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		return err
	}
	if err := db.AutoMigrate(&models.Node{}, &models.Client{}); err != nil {
		return err
	}
	DB = db
	return nil
}
