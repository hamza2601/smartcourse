package main

import (
	"log"
	"os"

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

	router := routes.SetupCourseServiceRoutes(db.DB, cfg)

	port := os.Getenv("COURSE_SERVICE_PORT")
	if port == "" {
		port = "8082"
	}
	log.Printf("Course Service starting on :%s\n", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
