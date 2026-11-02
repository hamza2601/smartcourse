package config

import (
	"fmt"
	"strconv"
	"time"
)

// OutboxConfig tunes the outbox relay that publishes this service's events.
type OutboxConfig struct {
	PollInterval time.Duration
	BatchSize    int
}

func loadOutboxConfig() (OutboxConfig, error) {
	interval, err := time.ParseDuration(getEnv("OUTBOX_POLL_INTERVAL", "1s"))
	if err != nil || interval <= 0 {
		return OutboxConfig{}, fmt.Errorf("OUTBOX_POLL_INTERVAL must be a positive duration (e.g. 1s)")
	}
	batch, err := strconv.Atoi(getEnv("OUTBOX_BATCH_SIZE", "100"))
	if err != nil || batch <= 0 {
		return OutboxConfig{}, fmt.Errorf("OUTBOX_BATCH_SIZE must be a positive integer")
	}
	return OutboxConfig{PollInterval: interval, BatchSize: batch}, nil
}
