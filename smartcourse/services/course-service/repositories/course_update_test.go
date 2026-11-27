package repositories

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	apperrors "smartcourse/libs/common/errors"
	"smartcourse/services/course-service/domain"
)

func TestUpdateCourseOnlyWhileDraftOrReady(t *testing.T) {
	repo, _ := testRepo(t)
	ctx := context.Background()
	name, desc, limit := "New name", "New description", 25

	for _, tc := range []struct {
		status   string
		editable bool
	}{
		{domain.CourseStatusDraft, true},
		{domain.CourseStatusReady, true},
		{domain.CourseStatusPublishing, false},
		{domain.CourseStatusArchived, false},
	} {
		course := &domain.Course{ID: uuid.New(), Name: "Old", Description: "Old desc", InstructorID: uuid.New(), Status: tc.status}
		if err := repo.CreateCourse(ctx, course, newEvent(t, domain.EventCourseCreated, course.ID)); err != nil {
			t.Fatal(err)
		}

		got, err := repo.UpdateCourse(ctx, course.ID, &name, &desc, &limit)

		if tc.editable {
			if err != nil {
				t.Fatalf("%s: %v", tc.status, err)
			}
			if got.Name != name || got.Description != desc || got.EnrollmentLimit == nil || *got.EnrollmentLimit != limit || got.Status != tc.status {
				t.Fatalf("%s: unexpected course %+v", tc.status, got)
			}
			continue
		}
		var notEditable *apperrors.CourseNotEditableError
		if !errors.As(err, &notEditable) || notEditable.Status != tc.status {
			t.Fatalf("%s: want CourseNotEditableError, got %v", tc.status, err)
		}
		unchanged, err := repo.GetCourseByID(ctx, course.ID)
		if err != nil || unchanged.Name != "Old" || unchanged.Description != "Old desc" || unchanged.EnrollmentLimit != nil {
			t.Fatalf("%s: course changed: %+v, %v", tc.status, unchanged, err)
		}
	}
}

func TestUpdateCourseLeavesOmittedFieldsAndReportsMissing(t *testing.T) {
	repo, _ := testRepo(t)
	ctx := context.Background()
	limit := 5
	course := &domain.Course{ID: uuid.New(), Name: "Keep", Description: "Keep desc", InstructorID: uuid.New(), Status: domain.CourseStatusDraft}
	if err := repo.CreateCourse(ctx, course, newEvent(t, domain.EventCourseCreated, course.ID)); err != nil {
		t.Fatal(err)
	}

	got, err := repo.UpdateCourse(ctx, course.ID, nil, nil, &limit)
	if err != nil || got.Name != "Keep" || got.Description != "Keep desc" || *got.EnrollmentLimit != 5 {
		t.Fatalf("got %+v, %v", got, err)
	}

	_, err = repo.UpdateCourse(ctx, uuid.New(), nil, nil, &limit)
	var notFound *apperrors.CourseNotFoundError
	if !errors.As(err, &notFound) {
		t.Fatalf("want CourseNotFoundError, got %v", err)
	}
}
