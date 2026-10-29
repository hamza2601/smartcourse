package main

import (
	"log"
	"os"

	"smartcourse/services/enrollment-service/config"
	"smartcourse/services/enrollment-service/database"
	"smartcourse/services/enrollment-service/routes"
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

	router := routes.SetupEnrollmentServiceRoutes(db.DB, cfg)

	port := os.Getenv("ENROLLMENT_SERVICE_PORT")
	if port == "" {
		port = "8083"
	}
	log.Printf("Enrollment Service starting on :%s\n", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
