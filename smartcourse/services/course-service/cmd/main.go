package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"smartcourse/libs/common/httpserver"
	"smartcourse/libs/common/logger"
	"smartcourse/libs/common/outbox"
	"smartcourse/services/course-service/config"
	"smartcourse/services/course-service/database"
	"smartcourse/services/course-service/persistence"
	"smartcourse/services/course-service/routes"
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

	public, internal := routes.SetupCourseServiceRoutes(db.DB, cfg)

	lg := logger.New("course-service")
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

	log.Printf("Course Service: public API on :%s, internal API on :%s", cfg.Port, cfg.InternalPort)
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
