package repositories

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"gorm.io/gorm"

	apperrors "smartcourse/libs/common/errors"
	"smartcourse/services/course-service/domain"
	"smartcourse/services/course-service/persistence"
)

// Same setup as the course repository tests: needs SMARTCOURSE_TEST_DSN, and
// everything runs inside a transaction that is rolled back.
func newContentRepo(t *testing.T) (*ContentRepository, *gorm.DB, uuid.UUID) {
	t.Helper()
	_, tx := testRepo(t)
	return NewContentRepository(tx), tx, insertCourse(t, tx)
}

func insertCourse(t *testing.T, tx *gorm.DB) uuid.UUID {
	t.Helper()
	course := &persistence.CourseDB{ID: uuid.New(), Name: "Go", InstructorID: uuid.New(), Status: domain.CourseStatusDraft}
	if err := tx.Create(course).Error; err != nil {
		t.Fatal(err)
	}
	return course.ID
}

// addModule creates a module; position 0 means "auto".
func addModule(t *testing.T, repo *ContentRepository, courseID uuid.UUID, title string, position int) *domain.Module {
	t.Helper()
	m := &domain.Module{ID: uuid.New(), CourseID: courseID, Title: title, Position: position}
	if err := repo.CreateModule(context.Background(), m); err != nil {
		t.Fatalf("create module %q: %v", title, err)
	}
	return m
}

func addLesson(t *testing.T, repo *ContentRepository, moduleID uuid.UUID, title string, position int) *domain.Lesson {
	t.Helper()
	l := &domain.Lesson{ID: uuid.New(), ModuleID: moduleID, Title: title, Position: position}
	if err := repo.CreateLesson(context.Background(), l); err != nil {
		t.Fatalf("create lesson %q: %v", title, err)
	}
	return l
}

func TestCreateModuleAssignsNextPosition(t *testing.T) {
	repo, tx, courseID := newContentRepo(t)

	for want := 1; want <= 3; want++ {
		if m := addModule(t, repo, courseID, "auto", 0); m.Position != want {
			t.Fatalf("auto position = %d, want %d", m.Position, want)
		}
	}
	if m := addModule(t, repo, courseID, "explicit", 10); m.Position != 10 {
		t.Fatalf("explicit position = %d, want 10", m.Position)
	}
	if m := addModule(t, repo, courseID, "after explicit", 0); m.Position != 11 {
		t.Fatalf("auto after explicit = %d, want 11", m.Position)
	}
	// Positions are per course.
	if m := addModule(t, repo, insertCourse(t, tx), "other course", 0); m.Position != 1 {
		t.Fatalf("first module of another course = %d, want 1", m.Position)
	}
}

func TestCreateLessonAssignsNextPosition(t *testing.T) {
	repo, tx, courseID := newContentRepo(t)
	module := addModule(t, repo, courseID, "M", 0)

	for want := 1; want <= 3; want++ {
		if l := addLesson(t, repo, module.ID, "auto", 0); l.Position != want {
			t.Fatalf("auto position = %d, want %d", l.Position, want)
		}
	}
	if l := addLesson(t, repo, module.ID, "explicit", 7); l.Position != 7 {
		t.Fatalf("explicit position = %d, want 7", l.Position)
	}
	if l := addLesson(t, repo, module.ID, "after explicit", 0); l.Position != 8 {
		t.Fatalf("auto after explicit = %d, want 8", l.Position)
	}
	// Positions are per module.
	other := addModule(t, repo, insertCourse(t, tx), "M2", 0)
	if l := addLesson(t, repo, other.ID, "first", 0); l.Position != 1 {
		t.Fatalf("first lesson of another module = %d, want 1", l.Position)
	}
}

func TestDuplicatePositionIsConflict(t *testing.T) {
	repo, _, courseID := newContentRepo(t)
	ctx := context.Background()

	first := addModule(t, repo, courseID, "first", 1)
	second := addModule(t, repo, courseID, "second", 2)

	var conflict *apperrors.PositionConflictError

	dup := &domain.Module{ID: uuid.New(), CourseID: courseID, Title: "dup", Position: 1}
	if err := repo.CreateModule(ctx, dup); !errors.As(err, &conflict) || conflict.Position != 1 {
		t.Fatalf("create module at taken position: want PositionConflictError(1), got %v", err)
	}

	second.Position = first.Position
	if err := repo.UpdateModule(ctx, second); !errors.As(err, &conflict) {
		t.Fatalf("update module to taken position: want PositionConflictError, got %v", err)
	}

	lesson := addLesson(t, repo, first.ID, "L", 1)
	dupLesson := &domain.Lesson{ID: uuid.New(), ModuleID: first.ID, Title: "dup", Position: 1}
	if err := repo.CreateLesson(ctx, dupLesson); !errors.As(err, &conflict) {
		t.Fatalf("create lesson at taken position: want PositionConflictError, got %v", err)
	}
	other := addLesson(t, repo, first.ID, "L2", 2)
	other.Position = lesson.Position
	if err := repo.UpdateLesson(ctx, other); !errors.As(err, &conflict) {
		t.Fatalf("update lesson to taken position: want PositionConflictError, got %v", err)
	}

	// The failed statements did not poison the surrounding transaction, and
	// nothing was written.
	modules, err := repo.ListModules(ctx, courseID)
	if err != nil || len(modules) != 2 {
		t.Fatalf("modules after conflicts: %d, %v; want 2", len(modules), err)
	}
	if modules[1].Position != 2 {
		t.Fatalf("a failed update changed the module: %+v", modules[1])
	}
}

func TestSamePositionInDifferentParentsIsAllowed(t *testing.T) {
	repo, tx, courseID := newContentRepo(t)

	addModule(t, repo, courseID, "course A", 1)
	addModule(t, repo, insertCourse(t, tx), "course B", 1)

	m1 := addModule(t, repo, courseID, "M", 2)
	m2 := addModule(t, repo, courseID, "M2", 3)
	addLesson(t, repo, m1.ID, "L", 1)
	addLesson(t, repo, m2.ID, "L", 1)
}

func TestDeletingModuleCascadesItsLessons(t *testing.T) {
	repo, tx, courseID := newContentRepo(t)
	ctx := context.Background()

	doomed := addModule(t, repo, courseID, "doomed", 0)
	kept := addModule(t, repo, courseID, "kept", 0)
	addLesson(t, repo, doomed.ID, "L1", 0)
	addLesson(t, repo, doomed.ID, "L2", 0)
	keptLesson := addLesson(t, repo, kept.ID, "L3", 0)

	if err := repo.DeleteModule(ctx, doomed.ID); err != nil {
		t.Fatal(err)
	}

	var lessons int64
	if err := tx.Model(&persistence.LessonDB{}).Where("module_id = ?", doomed.ID).Count(&lessons).Error; err != nil || lessons != 0 {
		t.Fatalf("lessons of the deleted module = %d (err %v), want 0", lessons, err)
	}
	if _, _, err := repo.GetLessonWithCourse(ctx, keptLesson.ID); err != nil {
		t.Fatalf("a lesson of another module was deleted: %v", err)
	}

	var notFound *apperrors.ModuleNotFoundError
	if err := repo.DeleteModule(ctx, doomed.ID); !errors.As(err, &notFound) {
		t.Fatalf("deleting again: want ModuleNotFoundError, got %v", err)
	}
}

func TestGetWithCourse(t *testing.T) {
	repo, tx, courseID := newContentRepo(t)
	ctx := context.Background()
	module := addModule(t, repo, courseID, "M", 0)
	lesson := addLesson(t, repo, module.ID, "L", 0)

	// A second course with its own content, to prove the join picks the right one.
	otherModule := addModule(t, repo, insertCourse(t, tx), "other", 0)
	otherLesson := addLesson(t, repo, otherModule.ID, "other", 0)

	m, course, err := repo.GetModuleWithCourse(ctx, module.ID)
	if err != nil || m.ID != module.ID || course.ID != courseID {
		t.Fatalf("GetModuleWithCourse: module %v course %v err %v; want course %v", m, course, err, courseID)
	}
	l, course, err := repo.GetLessonWithCourse(ctx, lesson.ID)
	if err != nil || l.ID != lesson.ID || course.ID != courseID {
		t.Fatalf("GetLessonWithCourse: lesson %v course %v err %v; want course %v", l, course, err, courseID)
	}
	if _, course, _ := repo.GetLessonWithCourse(ctx, otherLesson.ID); course.ID == courseID {
		t.Fatal("GetLessonWithCourse returned the wrong course")
	}

	var moduleNotFound *apperrors.ModuleNotFoundError
	if _, _, err := repo.GetModuleWithCourse(ctx, uuid.New()); !errors.As(err, &moduleNotFound) {
		t.Fatalf("unknown module: want ModuleNotFoundError, got %v", err)
	}
	var lessonNotFound *apperrors.LessonNotFoundError
	if _, _, err := repo.GetLessonWithCourse(ctx, uuid.New()); !errors.As(err, &lessonNotFound) {
		t.Fatalf("unknown lesson: want LessonNotFoundError, got %v", err)
	}
}

func TestGetCourseContentNestsLessonsInPositionOrder(t *testing.T) {
	repo, tx, courseID := newContentRepo(t)
	ctx := context.Background()

	second := addModule(t, repo, courseID, "second", 2)
	first := addModule(t, repo, courseID, "first", 1)
	empty := addModule(t, repo, courseID, "empty", 3)
	addLesson(t, repo, first.ID, "b", 2)
	addLesson(t, repo, first.ID, "a", 1)
	addLesson(t, repo, second.ID, "only", 1)
	addModule(t, repo, insertCourse(t, tx), "another course", 1)

	content, err := repo.GetCourseContent(ctx, courseID)
	if err != nil {
		t.Fatal(err)
	}
	if len(content) != 3 || content[0].ID != first.ID || content[1].ID != second.ID || content[2].ID != empty.ID {
		t.Fatalf("modules not returned in position order for this course: %+v", content)
	}
	if got := []string{content[0].Lessons[0].Title, content[0].Lessons[1].Title}; got[0] != "a" || got[1] != "b" {
		t.Fatalf("lessons not in position order: %v", got)
	}
	if len(content[1].Lessons) != 1 || content[1].Lessons[0].Title != "only" {
		t.Fatalf("second module's lessons: %+v", content[1].Lessons)
	}
	if content[2].Lessons == nil || len(content[2].Lessons) != 0 {
		t.Fatalf("module without lessons should have an empty, non-nil list, got %#v", content[2].Lessons)
	}

	none, err := repo.GetCourseContent(ctx, insertCourse(t, tx))
	if err != nil || none == nil || len(none) != 0 {
		t.Fatalf("course without modules: %#v, %v; want empty non-nil list", none, err)
	}
}

func TestUpdateAndDeleteLesson(t *testing.T) {
	repo, _, courseID := newContentRepo(t)
	ctx := context.Background()
	module := addModule(t, repo, courseID, "M", 0)
	lesson := addLesson(t, repo, module.ID, "old", 0)

	lesson.Title, lesson.Content, lesson.Position = "new", "body", 9
	if err := repo.UpdateLesson(ctx, lesson); err != nil {
		t.Fatal(err)
	}
	got, _, err := repo.GetLessonWithCourse(ctx, lesson.ID)
	if err != nil || got.Title != "new" || got.Content != "body" || got.Position != 9 {
		t.Fatalf("after update: %+v, %v", got, err)
	}

	if err := repo.DeleteLesson(ctx, lesson.ID); err != nil {
		t.Fatal(err)
	}
	var lessonNotFound *apperrors.LessonNotFoundError
	if err := repo.DeleteLesson(ctx, lesson.ID); !errors.As(err, &lessonNotFound) {
		t.Fatalf("deleting again: want LessonNotFoundError, got %v", err)
	}
	if err := repo.UpdateLesson(ctx, lesson); !errors.As(err, &lessonNotFound) {
		t.Fatalf("updating a deleted lesson: want LessonNotFoundError, got %v", err)
	}
}
