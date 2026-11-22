package workflows

import (
	"testing"

	"go.temporal.io/sdk/testsuite"
)

func TestHelloWorkflow(t *testing.T) {
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	env.RegisterActivity(GreetActivity)

	env.ExecuteWorkflow(HelloWorkflow, "World")

	if !env.IsWorkflowCompleted() {
		t.Fatal("workflow did not complete")
	}
	if err := env.GetWorkflowError(); err != nil {
		t.Fatalf("workflow failed: %v", err)
	}
	var result string
	if err := env.GetWorkflowResult(&result); err != nil {
		t.Fatal(err)
	}
	if want := "Hello, World!"; result != want {
		t.Fatalf("result = %q, want %q", result, want)
	}
}
