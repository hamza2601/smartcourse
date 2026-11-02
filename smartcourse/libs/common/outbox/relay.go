package outbox

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Publisher delivers an event to the outside world. LogPublisher is the
// current implementation; a Kafka or Temporal publisher can replace it.
type Publisher interface {
	Publish(ctx context.Context, ev Event) error
}

// LogPublisher "publishes" by writing the event to the structured log.
type LogPublisher struct {
	Logger *slog.Logger
}

func (p LogPublisher) Publish(ctx context.Context, ev Event) error {
	p.Logger.InfoContext(ctx, "outbox event published",
		"event_id", ev.ID,
		"event_type", ev.EventType,
		"aggregate_type", ev.AggregateType,
		"aggregate_id", ev.AggregateID,
		"payload", ev.Payload,
	)
	return nil
}

// Relay polls an outbox table and publishes unprocessed events in created_at
// order. Rows are claimed with FOR UPDATE SKIP LOCKED, so several relay
// instances (e.g. service replicas) never hand the same row to the publisher
// at the same time.
type Relay struct {
	DB           *gorm.DB
	Table        string
	Publisher    Publisher
	PollInterval time.Duration
	BatchSize    int
	Logger       *slog.Logger
}

// batchTimeout bounds one batch, which keeps running after ctx is cancelled.
const batchTimeout = 30 * time.Second

// Run polls until ctx is cancelled, then returns once any in-flight batch has
// finished. A full batch is followed immediately by another, so a backlog
// drains without waiting for the ticker.
func (r *Relay) Run(ctx context.Context) {
	ticker := time.NewTicker(r.PollInterval)
	defer ticker.Stop()

	r.Logger.Info("outbox relay started", "table", r.Table, "poll_interval", r.PollInterval.String(), "batch_size", r.BatchSize)
	for {
		for ctx.Err() == nil {
			n, err := r.ProcessBatch(ctx)
			if err != nil {
				r.Logger.Error("outbox relay batch failed", "table", r.Table, "error", err)
				break
			}
			if n < r.BatchSize {
				break
			}
		}

		select {
		case <-ctx.Done():
			r.Logger.Info("outbox relay stopped", "table", r.Table)
			return
		case <-ticker.C:
		}
	}
}

// ProcessBatch publishes up to BatchSize unprocessed events and marks the
// published ones processed, returning how many it marked. If the publisher
// fails, events published before the failure are still marked and the rest
// stay unprocessed for the next poll.
//
// Once started, a batch runs to completion even if ctx is cancelled: aborting
// after events were published would roll back their processed_at marks and
// publish them again on the next run.
func (r *Relay) ProcessBatch(ctx context.Context) (int, error) {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), batchTimeout)
	defer cancel()

	var (
		marked     int
		publishErr error
	)
	err := r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var batch []record
		if err := tx.Table(r.Table).
			Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where("processed_at IS NULL").
			Order("created_at, id").
			Limit(r.BatchSize).
			Find(&batch).Error; err != nil {
			return err
		}

		published := make([]uuid.UUID, 0, len(batch))
		for _, rec := range batch {
			if err := r.Publisher.Publish(ctx, rec.toEvent()); err != nil {
				publishErr = err
				break
			}
			published = append(published, rec.ID)
		}
		if len(published) == 0 {
			return nil
		}

		if err := tx.Table(r.Table).
			Where("id IN ?", published).
			Update("processed_at", time.Now().UTC()).Error; err != nil {
			return err
		}
		marked = len(published)
		return nil
	})
	if err != nil {
		return 0, err
	}
	return marked, publishErr
}
