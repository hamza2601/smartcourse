package workflows

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
)

var (
	// ErrPublishUnavailable: there is no Temporal connection (yet).
	ErrPublishUnavailable = errors.New("publishing service unavailable")
	// ErrPublishInProgress: a publish for this course is already running.
	ErrPublishInProgress = errors.New("publishing already in progress")
)

// PublishWorkflowID is the workflow ID of a course's publish workflow: one per
// course, so a second concurrent publish of the same course is rejected.
func PublishWorkflowID(courseID uuid.UUID) string {
	return "publish-course-" + courseID.String()
}

// PublishStarter starts PublishCourseWorkflow. The Temporal client is set by
// the worker once it has connected, so it is held behind a mutex; the zero
// value is ready to use and reports ErrPublishUnavailable until then.
type PublishStarter struct {
	mu     sync.RWMutex
	client client.Client
}

func (s *PublishStarter) SetClient(c client.Client) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.client = c
}

// StartPublish starts the workflow without waiting for it to finish.
func (s *PublishStarter) StartPublish(ctx context.Context, courseID uuid.UUID, previousStatus string) error {
	s.mu.RLock()
	c := s.client
	s.mu.RUnlock()
	if c == nil {
		return ErrPublishUnavailable
	}

	_, err := c.ExecuteWorkflow(ctx, client.StartWorkflowOptions{
		ID:                       PublishWorkflowID(courseID),
		TaskQueue:                TaskQueue,
		WorkflowExecutionTimeout: 10 * time.Minute,
		// A running publish of the same course is rejected; a finished one may be re-run.
		WorkflowIDConflictPolicy: enumspb.WORKFLOW_ID_CONFLICT_POLICY_FAIL,
		WorkflowIDReusePolicy:    enumspb.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE,
		// Without this the SDK hides the already-started error and returns the
		// running workflow instead, so a duplicate publish would look successful.
		WorkflowExecutionErrorWhenAlreadyStarted: true,
	}, PublishCourseWorkflow, PublishCourseInput{CourseID: courseID, PreviousStatus: previousStatus})
	if err != nil {
		var started *serviceerror.WorkflowExecutionAlreadyStarted
		if errors.As(err, &started) {
			return ErrPublishInProgress
		}
		return err
	}
	return nil
}
