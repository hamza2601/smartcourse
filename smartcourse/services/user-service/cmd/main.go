package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	apperrors "smartcourse/libs/common/errors"
	"smartcourse/libs/common/httpserver"
	"smartcourse/libs/common/logger"
	"smartcourse/libs/common/outbox"
	"smartcourse/services/user-service/config"
	"smartcourse/services/user-service/database"
	"smartcourse/services/user-service/domain"
	"smartcourse/services/user-service/persistence"
	"smartcourse/services/user-service/repositories"
	"smartcourse/services/user-service/routes"
	"smartcourse/services/user-service/services"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}

	if err := database.RunLiquibaseMigrations(cfg); err != nil {
		return err
	}

	db, err := database.InitDB(cfg)
	if err != nil {
		return err
	}
	defer db.Close()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := seedAdmin(ctx, cfg, services.NewUserService(repositories.NewUserRepository(db.DB))); err != nil {
		return err
	}

	public, internal := routes.SetupUserServiceRoutes(db.DB, cfg)

	lg := logger.New("user-service")
	relay := &outbox.Relay{
		DB:           db.DB,
		Table:        persistence.OutboxTable,
		Publisher:    outbox.LogPublisher{Logger: lg},
		PollInterval: cfg.Outbox.PollInterval,
		BatchSize:    cfg.Outbox.BatchSize,
		Logger:       lg,
	}
	relayDone := make(chan struct{})
	go func() {
		defer close(relayDone)
		relay.Run(ctx)
	}()

	log.Printf("User Service: public API on :%s, internal API on :%s", cfg.Port, cfg.InternalPort)
	err = httpserver.Run(ctx,
		&http.Server{Addr: ":" + cfg.Port, Handler: public},
		&http.Server{Addr: ":" + cfg.InternalPort, Handler: internal},
	)
	// Stop the relay too when a server failed before any shutdown signal,
	// and let it finish its current batch before the DB is closed.
	stop()
	<-relayDone
	return err
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
