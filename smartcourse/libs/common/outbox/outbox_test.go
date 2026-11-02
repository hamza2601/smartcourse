package outbox

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// These tests need a Postgres, e.g. the docker-compose database:
//
//	SMARTCOURSE_TEST_DSN="host=localhost port=5432 user=postgres password=password dbname=smartcourse_db sslmode=disable"
//
// Each test gets its own throwaway schema with an outbox_events table of the
// same shape the services' Liquibase changelogs create.
func testTable(t *testing.T) (*gorm.DB, string) {
	t.Helper()
	dsn := os.Getenv("SMARTCOURSE_TEST_DSN")
	if dsn == "" {
		t.Skip("SMARTCOURSE_TEST_DSN not set; skipping database test")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	schema := "outbox_test_" + strings.ReplaceAll(uuid.NewString()[:8], "-", "")
	table := schema + ".outbox_events"
	ddl := fmt.Sprintf(`
		CREATE SCHEMA %[1]s;
		CREATE TABLE %[2]s (
			id UUID PRIMARY KEY,
			aggregate_type VARCHAR(100) NOT NULL,
			aggregate_id UUID NOT NULL,
			event_type VARCHAR(255) NOT NULL,
			payload JSONB NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT now(),
			processed_at TIMESTAMP NULL
		);
		CREATE INDEX ON %[2]s (created_at) WHERE processed_at IS NULL;`, schema, table)
	if err := db.Exec(ddl).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		db.Exec("DROP SCHEMA " + schema + " CASCADE")
		if sqlDB, err := db.DB(); err == nil {
			sqlDB.Close()
		}
	})
	return db, table
}

func count(t *testing.T, db *gorm.DB, table, where string) int64 {
	t.Helper()
	var n int64
	if err := db.Table(table).Where(where).Count(&n).Error; err != nil {
		t.Fatal(err)
	}
	return n
}

func event(t *testing.T, n int) Event {
	t.Helper()
	ev, err := NewEvent("thing", uuid.New(), "ThingHappened", map[string]int{"n": n})
	if err != nil {
		t.Fatal(err)
	}
	// Distinct, ordered timestamps make the publish order deterministic.
	ev.CreatedAt = time.Date(2026, 1, 1, 0, 0, n, 0, time.UTC)
	return ev
}

type recordingPublisher struct {
	mu     sync.Mutex
	events []Event
	failOn int // 1-based call number that fails; 0 = never
}

func (p *recordingPublisher) Publish(_ context.Context, ev Event) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.failOn != 0 && len(p.events)+1 == p.failOn {
		p.failOn = 0
		return errors.New("broker unavailable")
	}
	p.events = append(p.events, ev)
	return nil
}

func (p *recordingPublisher) count() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.events)
}

func newRelay(db *gorm.DB, table string, pub Publisher, batch int) *Relay {
	return &Relay{
		DB: db, Table: table, Publisher: pub,
		PollInterval: 20 * time.Millisecond, BatchSize: batch,
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
}

func TestInsertCommitsWithTransaction(t *testing.T) {
	db, table := testTable(t)
	err := db.Transaction(func(tx *gorm.DB) error { return Insert(tx, table, event(t, 1)) })
	if err != nil {
		t.Fatal(err)
	}
	if n := count(t, db, table, "processed_at IS NULL"); n != 1 {
		t.Fatalf("outbox rows = %d, want 1", n)
	}
}

func TestRolledBackTransactionLeavesNoOutboxRow(t *testing.T) {
	db, table := testTable(t)
	businessErr := errors.New("business write failed")

	err := db.Transaction(func(tx *gorm.DB) error {
		if err := Insert(tx, table, event(t, 1)); err != nil {
			return err
		}
		return businessErr // e.g. the seat reservation failed after the event was staged
	})
	if !errors.Is(err, businessErr) {
		t.Fatalf("want business error, got %v", err)
	}
	if n := count(t, db, table, "1=1"); n != 0 {
		t.Fatalf("outbox rows after rollback = %d, want 0", n)
	}
}

func TestRelayMarksRowsProcessedInOrder(t *testing.T) {
	db, table := testTable(t)
	// Inserted out of order on purpose; the relay must publish by created_at.
	for _, n := range []int{3, 1, 2} {
		if err := Insert(db, table, event(t, n)); err != nil {
			t.Fatal(err)
		}
	}
	pub := &recordingPublisher{}
	relay := newRelay(db, table, pub, 2)

	if got, err := relay.ProcessBatch(context.Background()); err != nil || got != 2 {
		t.Fatalf("first batch: %d, %v; want 2", got, err)
	}
	if got, err := relay.ProcessBatch(context.Background()); err != nil || got != 1 {
		t.Fatalf("second batch: %d, %v; want 1", got, err)
	}

	if n := count(t, db, table, "processed_at IS NULL"); n != 0 {
		t.Fatalf("unprocessed rows = %d, want 0", n)
	}
	for i, ev := range pub.events {
		var p struct{ N int }
		if err := json.Unmarshal(ev.Payload, &p); err != nil {
			t.Fatal(err)
		}
		if p.N != i+1 {
			t.Fatalf("publish #%d has n=%d, want %d (created_at order)", i+1, p.N, i+1)
		}
	}
}

func TestRelayDoesNotReprocessProcessedRows(t *testing.T) {
	db, table := testTable(t)
	for n := 1; n <= 3; n++ {
		if err := Insert(db, table, event(t, n)); err != nil {
			t.Fatal(err)
		}
	}
	pub := &recordingPublisher{}
	relay := newRelay(db, table, pub, 10)

	if _, err := relay.ProcessBatch(context.Background()); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		if got, err := relay.ProcessBatch(context.Background()); err != nil || got != 0 {
			t.Fatalf("re-run %d processed %d (err %v), want 0", i+1, got, err)
		}
	}
	if pub.count() != 3 {
		t.Fatalf("published %d events, want exactly 3", pub.count())
	}
}

func TestRelayPublishFailureLeavesRestForNextPoll(t *testing.T) {
	db, table := testTable(t)
	for n := 1; n <= 3; n++ {
		if err := Insert(db, table, event(t, n)); err != nil {
			t.Fatal(err)
		}
	}
	pub := &recordingPublisher{failOn: 2}
	relay := newRelay(db, table, pub, 10)

	got, err := relay.ProcessBatch(context.Background())
	if err == nil || got != 1 {
		t.Fatalf("got %d, %v; want 1 marked and a publish error", got, err)
	}
	if n := count(t, db, table, "processed_at IS NULL"); n != 2 {
		t.Fatalf("unprocessed = %d, want 2", n)
	}
	if got, err := relay.ProcessBatch(context.Background()); err != nil || got != 2 {
		t.Fatalf("retry: %d, %v; want 2", got, err)
	}
}

func TestConcurrentRelaysPublishEachEventOnce(t *testing.T) {
	db, table := testTable(t)
	const total = 60
	for n := 1; n <= total; n++ {
		if err := Insert(db, table, event(t, n)); err != nil {
			t.Fatal(err)
		}
	}
	pub := &recordingPublisher{}

	var wg sync.WaitGroup
	for w := 0; w < 3; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			relay := newRelay(db, table, pub, 5)
			for {
				n, err := relay.ProcessBatch(context.Background())
				if err != nil {
					t.Error(err)
					return
				}
				if n == 0 {
					return
				}
			}
		}()
	}
	wg.Wait()

	seen := map[uuid.UUID]int{}
	for _, ev := range pub.events {
		seen[ev.ID]++
	}
	if len(pub.events) != total || len(seen) != total {
		t.Fatalf("published %d events (%d distinct), want %d each exactly once", len(pub.events), len(seen), total)
	}
}

func TestRelayRunPublishesAndStopsOnCancel(t *testing.T) {
	db, table := testTable(t)
	pub := &recordingPublisher{}
	relay := newRelay(db, table, pub, 10)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { relay.Run(ctx); close(done) }()

	if err := Insert(db, table, event(t, 1)); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for pub.count() == 0 {
		if time.Now().After(deadline) {
			t.Fatal("relay never published the event")
		}
		time.Sleep(10 * time.Millisecond)
	}

	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not return after context cancel")
	}
	if n := count(t, db, table, "processed_at IS NULL"); n != 0 {
		t.Fatalf("unprocessed rows = %d, want 0", n)
	}
}
