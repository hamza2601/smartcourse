package database

import (
	"smartcourse/internal/models"
)

func (d *Database) RunMigrations() error {
	if err := d.CreateSchemas(); err != nil {
		return err
	}
	return d.DB.AutoMigrate(
		&models.User{},
		&models.Course{},
		&models.Enrollment{},
		&models.Progress{},
		&models.UserServiceOutboxEvent{},
		&models.CourseServiceOutboxEvent{},
		&models.EnrollmentServiceOutboxEvent{},
	)
}
