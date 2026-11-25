package workflows

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
)

func newEnv(t *testing.T) (*testsuite.TestWorkflowEnvironment, *PublishActivities) {
	t.Helper()
	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()
	acts := &PublishActivities{}
	env.RegisterActivity(acts)
	return env, acts
}

func okAll(env *testsuite.TestWorkflowEnvironment, a *PublishActivities) {
	env.OnActivity(a.BeginPublishing, mock.Anything, mock.Anything).Return(nil)
	env.OnActivity(a.ValidateCourse, mock.Anything, mock.Anything).Return(nil)
	env.OnActivity(a.VerifyAssets, mock.Anything, mock.Anything).Return(nil)
	env.OnActivity(a.UpdateSearchIndex, mock.Anything, mock.Anything).Return(nil)
	env.OnActivity(a.RefreshCache, mock.Anything, mock.Anything).Return(nil)
	env.OnActivity(a.InitAnalytics, mock.Anything, mock.Anything).Return(nil)
	env.OnActivity(a.FinalizePublishing, mock.Anything, mock.Anything).Return(nil)
}

func nonRetryable(msg string) error {
	return temporal.NewNonRetryableApplicationError(msg, "Test", nil)
}

func TestPublishCourseWorkflowHappyPath(t *testing.T) {
	env, a := newEnv(t)
	okAll(env, a)
	in := PublishCourseInput{CourseID: uuid.New(), PreviousStatus: "draft"}

	env.ExecuteWorkflow(PublishCourseWorkflow, in)

	if !env.IsWorkflowCompleted() || env.GetWorkflowError() != nil {
		t.Fatalf("workflow error: %v", env.GetWorkflowError())
	}
	var res PublishCourseResult
	if err := env.GetWorkflowResult(&res); err != nil {
		t.Fatal(err)
	}
	if res.CourseID != in.CourseID || res.Status != "ready" {
		t.Fatalf("result = %+v", res)
	}
	env.AssertExpectations(t)
	env.AssertNotCalled(t, "RevertPublishing", mock.Anything, mock.Anything)
}

func TestPublishCourseWorkflowValidationFailureReverts(t *testing.T) {
	env, a := newEnv(t)
	env.OnActivity(a.BeginPublishing, mock.Anything, mock.Anything).Return(nil)
	env.OnActivity(a.ValidateCourse, mock.Anything, mock.Anything).Return(nonRetryable("course description is empty"))
	var got RevertInput
	env.OnActivity(a.RevertPublishing, mock.Anything, mock.Anything).Return(
		func(_ context.Context, in RevertInput) error { got = in; return nil })
	in := PublishCourseInput{CourseID: uuid.New(), PreviousStatus: "ready"}

	env.ExecuteWorkflow(PublishCourseWorkflow, in)

	if env.GetWorkflowError() == nil {
		t.Fatal("workflow should fail")
	}
	want := RevertInput{CourseID: in.CourseID, PreviousStatus: "ready", Reason: "course description is empty"}
	if got != want {
		t.Fatalf("revert input = %+v, want %+v", got, want)
	}
	env.AssertNotCalled(t, "VerifyAssets", mock.Anything, mock.Anything)
	env.AssertNotCalled(t, "FinalizePublishing", mock.Anything, mock.Anything)
}

func TestPublishCourseWorkflowFanOutFailureReverts(t *testing.T) {
	env, a := newEnv(t)
	env.OnActivity(a.BeginPublishing, mock.Anything, mock.Anything).Return(nil)
	env.OnActivity(a.ValidateCourse, mock.Anything, mock.Anything).Return(nil)
	env.OnActivity(a.VerifyAssets, mock.Anything, mock.Anything).Return(nonRetryable("lessons with empty content: \"L1\""))
	env.OnActivity(a.UpdateSearchIndex, mock.Anything, mock.Anything).Return(nil)
	env.OnActivity(a.RefreshCache, mock.Anything, mock.Anything).Return(nil)
	env.OnActivity(a.InitAnalytics, mock.Anything, mock.Anything).Return(nonRetryable("analytics down"))
	var got RevertInput
	env.OnActivity(a.RevertPublishing, mock.Anything, mock.Anything).Return(
		func(_ context.Context, in RevertInput) error { got = in; return nil })
	in := PublishCourseInput{CourseID: uuid.New(), PreviousStatus: "draft"}

	env.ExecuteWorkflow(PublishCourseWorkflow, in)

	if env.GetWorkflowError() == nil {
		t.Fatal("workflow should fail")
	}
	wantReason := "lessons with empty content: \"L1\"; analytics down"
	if got.Reason != wantReason || got.PreviousStatus != "draft" || got.CourseID != in.CourseID {
		t.Fatalf("revert input = %+v, want reason %q", got, wantReason)
	}
	env.AssertNotCalled(t, "FinalizePublishing", mock.Anything, mock.Anything)
}

func TestPublishCourseWorkflowFinalizeFailureReverts(t *testing.T) {
	env, a := newEnv(t)
	env.OnActivity(a.BeginPublishing, mock.Anything, mock.Anything).Return(nil)
	env.OnActivity(a.ValidateCourse, mock.Anything, mock.Anything).Return(nil)
	env.OnActivity(a.VerifyAssets, mock.Anything, mock.Anything).Return(nil)
	env.OnActivity(a.UpdateSearchIndex, mock.Anything, mock.Anything).Return(nil)
	env.OnActivity(a.RefreshCache, mock.Anything, mock.Anything).Return(nil)
	env.OnActivity(a.InitAnalytics, mock.Anything, mock.Anything).Return(nil)
	env.OnActivity(a.FinalizePublishing, mock.Anything, mock.Anything).Return(nonRetryable("cannot finalize"))
	env.OnActivity(a.RevertPublishing, mock.Anything, mock.Anything).Return(nil).Once()

	env.ExecuteWorkflow(PublishCourseWorkflow, PublishCourseInput{CourseID: uuid.New(), PreviousStatus: "draft"})

	if env.GetWorkflowError() == nil {
		t.Fatal("workflow should fail")
	}
	env.AssertExpectations(t)
}

func TestPublishCourseWorkflowBeginFailureDoesNotRevert(t *testing.T) {
	env, a := newEnv(t)
	env.OnActivity(a.BeginPublishing, mock.Anything, mock.Anything).Return(nonRetryable("course is not draft"))

	env.ExecuteWorkflow(PublishCourseWorkflow, PublishCourseInput{CourseID: uuid.New(), PreviousStatus: "draft"})

	if env.GetWorkflowError() == nil {
		t.Fatal("workflow should fail")
	}
	env.AssertNotCalled(t, "ValidateCourse", mock.Anything, mock.Anything)
	env.AssertNotCalled(t, "RevertPublishing", mock.Anything, mock.Anything)
}

func TestPublishCourseWorkflowRejectsBadPreviousStatus(t *testing.T) {
	env, _ := newEnv(t)

	env.ExecuteWorkflow(PublishCourseWorkflow, PublishCourseInput{CourseID: uuid.New(), PreviousStatus: "archived"})

	err := env.GetWorkflowError()
	if err == nil {
		t.Fatal("workflow should fail")
	}
	var appErr *temporal.ApplicationError
	if !errors.As(err, &appErr) || !appErr.NonRetryable() {
		t.Fatalf("want a non-retryable application error, got %v", err)
	}
	env.AssertNotCalled(t, "BeginPublishing", mock.Anything, mock.Anything)
}
