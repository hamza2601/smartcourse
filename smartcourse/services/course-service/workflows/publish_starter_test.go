package workflows

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/mocks"
)

func TestPublishStarterStartsWorkflowWithExpectedOptions(t *testing.T) {
	c := &mocks.Client{}
	var opts client.StartWorkflowOptions
	var input any
	c.On("ExecuteWorkflow", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Run(func(a mock.Arguments) {
			opts = a.Get(1).(client.StartWorkflowOptions)
			input = a.Get(3)
		}).Return(&mocks.WorkflowRun{}, nil)
	s := &PublishStarter{}
	s.SetClient(c)
	id := uuid.New()

	if err := s.StartPublish(context.Background(), id, "ready"); err != nil {
		t.Fatal(err)
	}

	if opts.ID != "publish-course-"+id.String() {
		t.Errorf("workflow ID = %q", opts.ID)
	}
	if opts.TaskQueue != TaskQueue {
		t.Errorf("task queue = %q", opts.TaskQueue)
	}
	if opts.WorkflowExecutionTimeout != 10*time.Minute {
		t.Errorf("execution timeout = %v", opts.WorkflowExecutionTimeout)
	}
	if opts.WorkflowIDConflictPolicy != enumspb.WORKFLOW_ID_CONFLICT_POLICY_FAIL {
		t.Errorf("conflict policy = %v", opts.WorkflowIDConflictPolicy)
	}
	if !opts.WorkflowExecutionErrorWhenAlreadyStarted {
		t.Error("WorkflowExecutionErrorWhenAlreadyStarted must be set, or the SDK hides the conflict")
	}
	if opts.WorkflowIDReusePolicy != enumspb.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE {
		t.Errorf("reuse policy = %v", opts.WorkflowIDReusePolicy)
	}
	if input != (PublishCourseInput{CourseID: id, PreviousStatus: "ready"}) {
		t.Errorf("workflow input = %+v", input)
	}
}

func TestPublishStarterNotConnected(t *testing.T) {
	s := &PublishStarter{}
	if err := s.StartPublish(context.Background(), uuid.New(), "draft"); !errors.Is(err, ErrPublishUnavailable) {
		t.Fatalf("err = %v, want ErrPublishUnavailable", err)
	}
}

func TestPublishStarterMapsAlreadyStarted(t *testing.T) {
	c := &mocks.Client{}
	c.On("ExecuteWorkflow", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil, &serviceerror.WorkflowExecutionAlreadyStarted{})
	s := &PublishStarter{}
	s.SetClient(c)

	if err := s.StartPublish(context.Background(), uuid.New(), "draft"); !errors.Is(err, ErrPublishInProgress) {
		t.Fatalf("err = %v, want ErrPublishInProgress", err)
	}
}

func TestPublishStarterReturnsOtherErrors(t *testing.T) {
	boom := errors.New("boom")
	c := &mocks.Client{}
	c.On("ExecuteWorkflow", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, boom)
	s := &PublishStarter{}
	s.SetClient(c)

	err := s.StartPublish(context.Background(), uuid.New(), "draft")
	if !errors.Is(err, boom) || errors.Is(err, ErrPublishInProgress) {
		t.Fatalf("err = %v, want the original error", err)
	}
}
