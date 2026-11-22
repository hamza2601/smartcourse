package workflows

import (
	"context"
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// TaskQueue is the task queue course-service's worker polls.
const TaskQueue = "course-service"

// HelloWorkflow greets name through GreetActivity. It only orchestrates: all
// work happens in the activity, so the workflow code stays deterministic.
func HelloWorkflow(ctx workflow.Context, name string) (string, error) {
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2,
			MaximumInterval:    10 * time.Second,
			MaximumAttempts:    3,
		},
	})

	var greeting string
	if err := workflow.ExecuteActivity(ctx, GreetActivity, name).Get(ctx, &greeting); err != nil {
		return "", err
	}
	return greeting, nil
}

func GreetActivity(ctx context.Context, name string) (string, error) {
	return fmt.Sprintf("Hello, %s!", name), nil
}
