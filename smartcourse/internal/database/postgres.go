package database

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"smartcourse/internal/config"
)

type Database struct {
	DB *gorm.DB
}

func InitDB(config *config.Config) (*Database, error) {
	db, err := gorm.Open(postgres.Open(config.GetDSN()), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	return &Database{DB: db}, nil
}

func (d *Database) Close() error {
	sqlDB, err := d.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
