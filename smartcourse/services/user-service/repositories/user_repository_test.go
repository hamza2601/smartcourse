package repositories

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	apperrors "smartcourse/libs/common/errors"
	"smartcourse/libs/common/outbox"
	"smartcourse/services/user-service/domain"
	"smartcourse/services/user-service/persistence"
)

// Needs SMARTCOURSE_TEST_DSN pointing at a database with user_service
// migrated. Each test runs in a rolled-back transaction.
func testRepo(t *testing.T) (*UserRepository, *gorm.DB) {
	t.Helper()
	dsn := os.Getenv("SMARTCOURSE_TEST_DSN")
	if dsn == "" {
		t.Skip("SMARTCOURSE_TEST_DSN not set; skipping database test")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	tx := db.Begin()
	t.Cleanup(func() {
		tx.Rollback()
		if sqlDB, err := db.DB(); err == nil {
			sqlDB.Close()
		}
	})
	return NewUserRepository(tx), tx
}

func registered(t *testing.T, u *domain.User) outbox.Event {
	t.Helper()
	ev, err := outbox.NewEvent(domain.AggregateTypeUser, u.ID, domain.EventUserRegistered, domain.UserRegisteredPayload{UserID: u.ID, Email: u.Email})
	if err != nil {
		t.Fatal(err)
	}
	return ev
}

func countEvents(t *testing.T, tx *gorm.DB, userID uuid.UUID) int64 {
	t.Helper()
	var n int64
	if err := tx.Table(persistence.OutboxTable).Where("event_type = ? AND aggregate_id = ?", domain.EventUserRegistered, userID).Count(&n).Error; err != nil {
		t.Fatal(err)
	}
	return n
}

func TestCreateUserWritesUserRegisteredEvent(t *testing.T) {
	repo, tx := testRepo(t)
	u := &domain.User{ID: uuid.New(), Name: "A", Email: uuid.NewString() + "@test.local", Role: domain.RoleStudent}

	if err := repo.CreateUser(context.Background(), u, registered(t, u)); err != nil {
		t.Fatal(err)
	}
	if got := countEvents(t, tx, u.ID); got != 1 {
		t.Fatalf("UserRegistered rows = %d, want 1", got)
	}
}

func TestDuplicateUserLeavesNoEvent(t *testing.T) {
	repo, tx := testRepo(t)
	email := uuid.NewString() + "@test.local"
	first := &domain.User{ID: uuid.New(), Name: "A", Email: email, Role: domain.RoleStudent}
	if err := repo.CreateUser(context.Background(), first, registered(t, first)); err != nil {
		t.Fatal(err)
	}

	dup := &domain.User{ID: uuid.New(), Name: "B", Email: email, Role: domain.RoleStudent}
	err := repo.CreateUser(context.Background(), dup, registered(t, dup))
	var exists *apperrors.UserAlreadyExistsError
	if !errors.As(err, &exists) {
		t.Fatalf("want UserAlreadyExistsError, got %v", err)
	}
	if got := countEvents(t, tx, dup.ID); got != 0 {
		t.Fatalf("failed registration left %d outbox rows, want 0 (rollback)", got)
	}
}
