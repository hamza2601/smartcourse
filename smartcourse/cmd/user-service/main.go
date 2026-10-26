package main

import (
	"context"
	"errors"
	"log"
	"os"

	"smartcourse/internal/config"
	"smartcourse/internal/database"
	"smartcourse/internal/domain"
	apperrors "smartcourse/internal/errors"
	"smartcourse/internal/repositories"
	"smartcourse/internal/routes"
	"smartcourse/internal/services"
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

	if err := seedAdmin(context.Background(), cfg, services.NewUserService(repositories.NewUserRepository(db.DB))); err != nil {
		log.Fatalf("failed to seed admin user: %v", err)
	}

	router := routes.SetupUserServiceRoutes(db.DB, cfg)

	port := os.Getenv("USER_SERVICE_PORT")
	if port == "" {
		port = "8081"
	}
	log.Printf("User Service starting on :%s\n", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}

// seedAdmin creates the first admin, since only admins can create users.
// It is a no-op unless ADMIN_EMAIL and ADMIN_PASSWORD are both set.
func seedAdmin(ctx context.Context, cfg *config.Config, userService *services.UserService) error {
	if cfg.AdminEmail == "" || cfg.AdminPassword == "" {
		return nil
	}
	_, err := userService.RegisterUser(ctx, "Admin", cfg.AdminEmail, domain.RoleAdmin, cfg.AdminPassword)
	var exists *apperrors.UserAlreadyExistsError
	if err != nil && !errors.As(err, &exists) {
		return err
	}
	log.Println("Admin user ready")
	return nil
}
