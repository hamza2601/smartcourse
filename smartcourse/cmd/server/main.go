package main

import (
	"log"

	"smartcourse/internal/config"
	"smartcourse/internal/database"
	"smartcourse/internal/routes"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	if err := database.RunLiquibaseMigrations(cfg); err != nil {
		log.Fatalf("failed to run liquibase migrations: %v", err)
	}

	db, err := database.InitDB(cfg)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	router := routes.SetupRoutes(db.DB)

	log.Printf("Starting server on :%s\n", cfg.ServerPort)
	if err := router.Run(":" + cfg.ServerPort); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
