package repositories

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"smartcourse/services/course-service/persistence"
)

// Proves migration 004's CHECK: the four current statuses are accepted and the
// retired 'published' is rejected. Needs SMARTCOURSE_TEST_DSN with course_service
// migrated through 004; it runs in the same rolled-back transaction as the
// other repository tests.
func TestCourseStatusCheckConstraint(t *testing.T) {
	_, tx := testRepo(t)

	insert := func(status string) error {
		// A nested transaction is a savepoint, so a rejected insert does not
		// abort the surrounding transaction.
		return tx.Transaction(func(tx *gorm.DB) error {
			return tx.Create(&persistence.CourseDB{
				ID: uuid.New(), Name: "status check", InstructorID: uuid.New(), Status: status,
			}).Error
		})
	}

	for _, status := range []string{"draft", "publishing", "ready", "archived"} {
		if err := insert(status); err != nil {
			t.Errorf("status %q should be accepted, got %v", status, err)
		}
	}

	err := insert("published")
	if err == nil {
		t.Fatal("status 'published' should be rejected by chk_courses_status")
	}
	if !strings.Contains(err.Error(), "chk_courses_status") {
		t.Fatalf("rejected, but not by chk_courses_status: %v", err)
	}
}
