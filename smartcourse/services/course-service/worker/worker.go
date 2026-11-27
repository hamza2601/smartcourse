package worker

import (
	"context"
	"log/slog"
	"time"

	"go.temporal.io/sdk/client"
	tlog "go.temporal.io/sdk/log"
	sdkworker "go.temporal.io/sdk/worker"

	"smartcourse/services/course-service/config"
	"smartcourse/services/course-service/workflows"
)

const (
	// Dialing is retried with a doubling backoff until about dialBudget has
	// passed; each attempt gets at most attemptTimeout.
	dialBudget     = 30 * time.Second
	attemptTimeout = 5 * time.Second
	initialBackoff = time.Second
	maxBackoff     = 8 * time.Second
)

// Run connects to Temporal and serves the course-service task queue until ctx
// is cancelled, then stops the worker gracefully. If Temporal stays unreachable
// it logs an error and returns, so the rest of the service keeps running
// without a worker.
func Run(ctx context.Context, cfg config.TemporalConfig, activities *workflows.PublishActivities, starter *workflows.PublishStarter, lg *slog.Logger) {
	c, err := dial(ctx, cfg, lg)
	if err != nil {
		if ctx.Err() == nil {
			lg.Error("temporal unreachable; continuing without the worker",
				"hostport", cfg.HostPort, "namespace", cfg.Namespace, "error", err)
		}
		return
	}
	defer c.Close()
	starter.SetClient(c)

	w := sdkworker.New(c, workflows.TaskQueue, sdkworker.Options{})
	w.RegisterWorkflow(workflows.PublishCourseWorkflow)
	w.RegisterActivity(activities)

	if err := w.Start(); err != nil {
		lg.Error("temporal worker failed to start", "task_queue", workflows.TaskQueue, "error", err)
		return
	}
	lg.Info("temporal worker started", "hostport", cfg.HostPort, "namespace", cfg.Namespace, "task_queue", workflows.TaskQueue)

	<-ctx.Done()
	w.Stop()
	lg.Info("temporal worker stopped", "task_queue", workflows.TaskQueue)
}

func dial(ctx context.Context, cfg config.TemporalConfig, lg *slog.Logger) (client.Client, error) {
	deadline := time.Now().Add(dialBudget)
	backoff := initialBackoff

	for attempt := 1; ; attempt++ {
		attemptCtx, cancel := context.WithTimeout(ctx, attemptTimeout)
		c, err := client.DialContext(attemptCtx, client.Options{
			HostPort:  cfg.HostPort,
			Namespace: cfg.Namespace,
			Logger:    tlog.NewStructuredLogger(lg),
		})
		cancel()
		if err == nil {
			return c, nil
		}
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if time.Now().Add(backoff).After(deadline) {
			return nil, err
		}

		lg.Warn("temporal not reachable yet; retrying",
			"hostport", cfg.HostPort, "attempt", attempt, "retry_in", backoff.String(), "error", err)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(backoff):
		}
		backoff = min(backoff*2, maxBackoff)
	}
}
