package workflows

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"

	"smartcourse/services/course-service/domain"
)

// TaskQueue is the task queue course-service's worker polls.
const TaskQueue = "course-service"

// Activity names; they equal the PublishActivities method names.
const (
	activityBegin       = "BeginPublishing"
	activityValidate    = "ValidateCourse"
	activityVerifyAsset = "VerifyAssets"
	activitySearchIndex = "UpdateSearchIndex"
	activityRefreshCach = "RefreshCache"
	activityInitAnalyt  = "InitAnalytics"
	activityFinalize    = "FinalizePublishing"
	activityRevert      = "RevertPublishing"
)

// Application error types returned (non-retryably) by the activities.
const (
	errTypeInvalidInput   = "InvalidInput"
	errTypeStatusConflict = "StatusConflict"
	errTypeValidation     = "ValidationFailed"
)

const reasonCancelled = "publishing cancelled"

// PublishCourseInput starts the workflow. PreviousStatus is the status the
// course had before publishing (draft or ready); a failed publish restores it.
type PublishCourseInput struct {
	CourseID       uuid.UUID `json:"course_id"`
	PreviousStatus string    `json:"previous_status"`
}

type PublishCourseResult struct {
	CourseID uuid.UUID `json:"course_id"`
	Status   string    `json:"status"`
}

// PublishCourseWorkflow moves a course to publishing, validates it, runs the
// post-validation steps in parallel, and marks it ready. If any step after
// BeginPublishing fails, the course is restored to its previous status and a
// CoursePublishFailed event is recorded. It only orchestrates; all work happens
// in activities so the workflow code stays deterministic.
func PublishCourseWorkflow(ctx workflow.Context, in PublishCourseInput) (PublishCourseResult, error) {
	if in.PreviousStatus != domain.CourseStatusDraft && in.PreviousStatus != domain.CourseStatusReady {
		return PublishCourseResult{}, temporal.NewNonRetryableApplicationError(
			fmt.Sprintf("previous_status must be %q or %q, got %q",
				domain.CourseStatusDraft, domain.CourseStatusReady, in.PreviousStatus),
			errTypeInvalidInput, nil)
	}

	actCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2,
			MaximumInterval:    10 * time.Second,
			MaximumAttempts:    3,
		},
	})

	// A failure here leaves the course untouched, so there is nothing to undo.
	if err := workflow.ExecuteActivity(actCtx, activityBegin, in).Get(actCtx, nil); err != nil {
		return PublishCourseResult{}, err
	}

	if err := workflow.ExecuteActivity(actCtx, activityValidate, in.CourseID).Get(actCtx, nil); err != nil {
		return PublishCourseResult{}, compensate(ctx, in, failureReason(ctx, err), err)
	}

	// Run every step to completion so the failure reason is deterministic.
	steps := []string{activityVerifyAsset, activitySearchIndex, activityRefreshCach, activityInitAnalyt}
	futures := make([]workflow.Future, len(steps))
	for i, name := range steps {
		futures[i] = workflow.ExecuteActivity(actCtx, name, in.CourseID)
	}
	var stepErrs []error
	for _, f := range futures {
		if err := f.Get(actCtx, nil); err != nil {
			stepErrs = append(stepErrs, err)
		}
	}
	if len(stepErrs) > 0 {
		return PublishCourseResult{}, compensate(ctx, in, failureReason(ctx, stepErrs...), errors.Join(stepErrs...))
	}

	if err := workflow.ExecuteActivity(actCtx, activityFinalize, in.CourseID).Get(actCtx, nil); err != nil {
		return PublishCourseResult{}, compensate(ctx, in, failureReason(ctx, err), err)
	}

	return PublishCourseResult{CourseID: in.CourseID, Status: domain.CourseStatusReady}, nil
}

// compensate restores the previous status on a context that survives workflow
// cancellation, and returns cause (the revert error too, if the revert failed).
func compensate(ctx workflow.Context, in PublishCourseInput, reason string, cause error) error {
	dctx, _ := workflow.NewDisconnectedContext(ctx)
	dctx = workflow.WithActivityOptions(dctx, workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2,
			MaximumInterval:    10 * time.Second,
			MaximumAttempts:    5,
		},
	})
	revert := RevertInput{CourseID: in.CourseID, PreviousStatus: in.PreviousStatus, Reason: reason}
	if err := workflow.ExecuteActivity(dctx, activityRevert, revert).Get(dctx, nil); err != nil {
		return errors.Join(cause, fmt.Errorf("revert publishing: %w", err))
	}
	return cause
}

// failureReason builds the CoursePublishFailed reason: "publishing cancelled"
// on cancellation, otherwise the failure messages joined with "; " in the
// order given.
func failureReason(ctx workflow.Context, errs ...error) string {
	if ctx.Err() != nil {
		return reasonCancelled
	}
	msgs := make([]string, 0, len(errs))
	for _, err := range errs {
		if temporal.IsCanceledError(err) {
			return reasonCancelled
		}
		var appErr *temporal.ApplicationError
		if errors.As(err, &appErr) {
			msgs = append(msgs, appErr.Message())
		} else {
			msgs = append(msgs, err.Error())
		}
	}
	return strings.Join(msgs, "; ")
}
