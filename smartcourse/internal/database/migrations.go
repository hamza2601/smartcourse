package database

import (
	"smartcourse/internal/models"
)

func (d *Database) RunMigrations() error {
	return d.DB.AutoMigrate(
		&models.User{},
		&models.Course{},
		&models.Enrollment{},
		&models.Progress{},
	)
}
