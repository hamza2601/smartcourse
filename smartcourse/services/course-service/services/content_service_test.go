package services

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"

	apperrors "smartcourse/libs/common/errors"
	"smartcourse/services/course-service/domain"
)

// fakeContentStore keeps modules and lessons in memory and looks courses up
// in the (existing) fakeCourseStore, like the real repository's joins do.
type fakeContentStore struct {
	courses *fakeCourseStore
	modules map[uuid.UUID]*domain.Module
	lessons map[uuid.UUID]*domain.Lesson
}

func newFakeContentStore(courses *fakeCourseStore) *fakeContentStore {
	return &fakeContentStore{
		courses: courses,
		modules: map[uuid.UUID]*domain.Module{},
		lessons: map[uuid.UUID]*domain.Lesson{},
	}
}

func (f *fakeContentStore) CreateModule(_ context.Context, m *domain.Module) error {
	if m.Position == 0 {
		for _, other := range f.modules {
			if other.CourseID == m.CourseID && other.Position >= m.Position {
				m.Position = other.Position
			}
		}
		m.Position++
	}
	f.modules[m.ID] = m
	return nil
}

func (f *fakeContentStore) ListModules(_ context.Context, courseID uuid.UUID) ([]domain.Module, error) {
	out := []domain.Module{}
	for _, m := range f.modules {
		if m.CourseID == courseID {
			out = append(out, *m)
		}
	}
	return out, nil
}

func (f *fakeContentStore) GetModuleWithCourse(_ context.Context, id uuid.UUID) (*domain.Module, *domain.Course, error) {
	m, ok := f.modules[id]
	if !ok {
		return nil, nil, &apperrors.ModuleNotFoundError{ModuleID: id}
	}
	course := *f.courses.courses[m.CourseID]
	mod := *m
	return &mod, &course, nil
}

func (f *fakeContentStore) UpdateModule(_ context.Context, m *domain.Module) error {
	f.modules[m.ID] = m
	return nil
}

func (f *fakeContentStore) DeleteModule(_ context.Context, id uuid.UUID) error {
	delete(f.modules, id)
	return nil
}

func (f *fakeContentStore) CreateLesson(_ context.Context, l *domain.Lesson) error {
	if l.Position == 0 {
		l.Position = 1
	}
	f.lessons[l.ID] = l
	return nil
}

func (f *fakeContentStore) ListLessons(_ context.Context, moduleID uuid.UUID) ([]domain.Lesson, error) {
	out := []domain.Lesson{}
	for _, l := range f.lessons {
		if l.ModuleID == moduleID {
			out = append(out, *l)
		}
	}
	return out, nil
}

func (f *fakeContentStore) GetLessonWithCourse(_ context.Context, id uuid.UUID) (*domain.Lesson, *domain.Course, error) {
	l, ok := f.lessons[id]
	if !ok {
		return nil, nil, &apperrors.LessonNotFoundError{LessonID: id}
	}
	course := *f.courses.courses[f.modules[l.ModuleID].CourseID]
	lesson := *l
	return &lesson, &course, nil
}

func (f *fakeContentStore) UpdateLesson(_ context.Context, l *domain.Lesson) error {
	f.lessons[l.ID] = l
	return nil
}

func (f *fakeContentStore) DeleteLesson(_ context.Context, id uuid.UUID) error {
	delete(f.lessons, id)
	return nil
}

func (f *fakeContentStore) GetCourseContent(_ context.Context, courseID uuid.UUID) ([]domain.ModuleWithLessons, error) {
	out := []domain.ModuleWithLessons{}
	for _, m := range f.modules {
		if m.CourseID == courseID {
			out = append(out, domain.ModuleWithLessons{Module: *m, Lessons: []domain.Lesson{}})
		}
	}
	return out, nil
}

// contentFixture is a course owned by `owner` in the given status, holding one
// module with one lesson.
type contentFixture struct {
	svc      *ContentService
	store    *fakeContentStore
	owner    uuid.UUID
	stranger uuid.UUID
	courseID uuid.UUID
	moduleID uuid.UUID
	lessonID uuid.UUID
}

func newContentFixture(status string) *contentFixture {
	f := &contentFixture{owner: uuid.New(), stranger: uuid.New(), courseID: uuid.New(), moduleID: uuid.New(), lessonID: uuid.New()}
	courses := newFakeCourseStore()
	courses.courses[f.courseID] = &domain.Course{ID: f.courseID, InstructorID: f.owner, Status: status}
	f.store = newFakeContentStore(courses)
	f.store.modules[f.moduleID] = &domain.Module{ID: f.moduleID, CourseID: f.courseID, Title: "M1", Position: 1}
	f.store.lessons[f.lessonID] = &domain.Lesson{ID: f.lessonID, ModuleID: f.moduleID, Title: "L1", Position: 1}
	f.svc = NewContentService(courses, f.store)
	return f
}

func TestOwnerCanCreateModuleAndLesson(t *testing.T) {
	f := newContentFixture(domain.CourseStatusDraft)
	ctx := context.Background()

	module, err := f.svc.CreateModule(ctx, f.owner, f.courseID, "Basics", nil)
	if err != nil {
		t.Fatal(err)
	}
	if module.CourseID != f.courseID || module.Title != "Basics" || module.Position != 2 {
		t.Fatalf("unexpected module: %+v", module)
	}

	lesson, err := f.svc.CreateLesson(ctx, f.owner, module.ID, "Intro", "body", intPtr(5))
	if err != nil {
		t.Fatal(err)
	}
	if lesson.ModuleID != module.ID || lesson.Title != "Intro" || lesson.Content != "body" || lesson.Position != 5 {
		t.Fatalf("unexpected lesson: %+v", lesson)
	}
}

func TestOwnerCanUpdateAndDelete(t *testing.T) {
	f := newContentFixture(domain.CourseStatusDraft)
	ctx := context.Background()
	newTitle := "Renamed"

	module, err := f.svc.UpdateModule(ctx, f.owner, f.moduleID, &newTitle, intPtr(3))
	if err != nil || module.Title != "Renamed" || module.Position != 3 {
		t.Fatalf("update module: %+v, %v", module, err)
	}
	// Omitted fields stay as they were.
	lesson, err := f.svc.UpdateLesson(ctx, f.owner, f.lessonID, nil, strPtr("new body"), nil)
	if err != nil || lesson.Title != "L1" || lesson.Content != "new body" || lesson.Position != 1 {
		t.Fatalf("update lesson: %+v, %v", lesson, err)
	}

	if err := f.svc.DeleteLesson(ctx, f.owner, f.lessonID); err != nil {
		t.Fatal(err)
	}
	if err := f.svc.DeleteModule(ctx, f.owner, f.moduleID); err != nil {
		t.Fatal(err)
	}
}

func strPtr(s string) *string { return &s }

func TestNonOwnerCannotWrite(t *testing.T) {
	f := newContentFixture(domain.CourseStatusDraft)
	ctx := context.Background()
	title := "x"

	writes := map[string]func() error{
		"create module": func() error { _, err := f.svc.CreateModule(ctx, f.stranger, f.courseID, "x", nil); return err },
		"update module": func() error { _, err := f.svc.UpdateModule(ctx, f.stranger, f.moduleID, &title, nil); return err },
		"delete module": func() error { return f.svc.DeleteModule(ctx, f.stranger, f.moduleID) },
		"create lesson": func() error { _, err := f.svc.CreateLesson(ctx, f.stranger, f.moduleID, "x", "", nil); return err },
		"update lesson": func() error { _, err := f.svc.UpdateLesson(ctx, f.stranger, f.lessonID, &title, nil, nil); return err },
		"delete lesson": func() error { return f.svc.DeleteLesson(ctx, f.stranger, f.lessonID) },
	}
	for name, write := range writes {
		var notOwned *apperrors.CourseNotOwnedError
		if err := write(); !errors.As(err, &notOwned) {
			t.Errorf("%s: want CourseNotOwnedError, got %v", name, err)
		}
	}
	if len(f.store.modules) != 1 || len(f.store.lessons) != 1 {
		t.Fatal("a rejected write changed the store")
	}
}

func TestPublishedCourseRejectsWrites(t *testing.T) {
	for _, status := range []string{domain.CourseStatusPublished, domain.CourseStatusArchived} {
		f := newContentFixture(status)
		ctx := context.Background()
		title := "x"

		writes := map[string]func() error{
			"create module": func() error { _, err := f.svc.CreateModule(ctx, f.owner, f.courseID, "x", nil); return err },
			"update module": func() error { _, err := f.svc.UpdateModule(ctx, f.owner, f.moduleID, &title, nil); return err },
			"delete module": func() error { return f.svc.DeleteModule(ctx, f.owner, f.moduleID) },
			"create lesson": func() error { _, err := f.svc.CreateLesson(ctx, f.owner, f.moduleID, "x", "", nil); return err },
			"update lesson": func() error { _, err := f.svc.UpdateLesson(ctx, f.owner, f.lessonID, &title, nil, nil); return err },
			"delete lesson": func() error { return f.svc.DeleteLesson(ctx, f.owner, f.lessonID) },
		}
		for name, write := range writes {
			var notEditable *apperrors.CourseNotEditableError
			if err := write(); !errors.As(err, &notEditable) || notEditable.Status != status {
				t.Errorf("%s (%s): want CourseNotEditableError, got %v", name, status, err)
			}
		}
		if len(f.store.modules) != 1 || len(f.store.lessons) != 1 {
			t.Fatalf("%s: a rejected write changed the store", status)
		}
	}
}

func TestReadVisibility(t *testing.T) {
	ctx := context.Background()

	t.Run("owner reads a draft", func(t *testing.T) {
		f := newContentFixture(domain.CourseStatusDraft)
		if modules, err := f.svc.ListModules(ctx, f.owner, f.courseID); err != nil || len(modules) != 1 {
			t.Fatalf("ListModules: %v, %v", modules, err)
		}
		if content, err := f.svc.GetCourseContent(ctx, f.owner, f.courseID); err != nil || len(content) != 1 {
			t.Fatalf("GetCourseContent: %v, %v", content, err)
		}
		if lessons, err := f.svc.ListLessons(ctx, f.owner, f.moduleID); err != nil || len(lessons) != 1 {
			t.Fatalf("ListLessons: %v, %v", lessons, err)
		}
	})

	t.Run("others cannot read a draft: not found", func(t *testing.T) {
		f := newContentFixture(domain.CourseStatusDraft)
		var courseNotFound *apperrors.CourseNotFoundError
		var moduleNotFound *apperrors.ModuleNotFoundError

		if _, err := f.svc.ListModules(ctx, f.stranger, f.courseID); !errors.As(err, &courseNotFound) {
			t.Errorf("ListModules: want CourseNotFoundError, got %v", err)
		}
		if _, err := f.svc.GetCourseContent(ctx, f.stranger, f.courseID); !errors.As(err, &courseNotFound) {
			t.Errorf("GetCourseContent: want CourseNotFoundError, got %v", err)
		}
		if _, err := f.svc.ListLessons(ctx, f.stranger, f.moduleID); !errors.As(err, &moduleNotFound) {
			t.Errorf("ListLessons: want ModuleNotFoundError, got %v", err)
		}
	})

	t.Run("others can read a published course", func(t *testing.T) {
		f := newContentFixture(domain.CourseStatusPublished)
		if modules, err := f.svc.ListModules(ctx, f.stranger, f.courseID); err != nil || len(modules) != 1 {
			t.Fatalf("ListModules: %v, %v", modules, err)
		}
		if content, err := f.svc.GetCourseContent(ctx, f.stranger, f.courseID); err != nil || len(content) != 1 {
			t.Fatalf("GetCourseContent: %v, %v", content, err)
		}
		if lessons, err := f.svc.ListLessons(ctx, f.stranger, f.moduleID); err != nil || len(lessons) != 1 {
			t.Fatalf("ListLessons: %v, %v", lessons, err)
		}
	})

	t.Run("others cannot read an archived course", func(t *testing.T) {
		f := newContentFixture(domain.CourseStatusArchived)
		var courseNotFound *apperrors.CourseNotFoundError
		if _, err := f.svc.ListModules(ctx, f.stranger, f.courseID); !errors.As(err, &courseNotFound) {
			t.Errorf("want CourseNotFoundError, got %v", err)
		}
	})

	t.Run("unknown course and module", func(t *testing.T) {
		f := newContentFixture(domain.CourseStatusPublished)
		var courseNotFound *apperrors.CourseNotFoundError
		var moduleNotFound *apperrors.ModuleNotFoundError
		if _, err := f.svc.ListModules(ctx, f.owner, uuid.New()); !errors.As(err, &courseNotFound) {
			t.Errorf("want CourseNotFoundError, got %v", err)
		}
		if _, err := f.svc.ListLessons(ctx, f.owner, uuid.New()); !errors.As(err, &moduleNotFound) {
			t.Errorf("want ModuleNotFoundError, got %v", err)
		}
	})
}

func TestTitleValidation(t *testing.T) {
	f := newContentFixture(domain.CourseStatusDraft)
	ctx := context.Background()

	long := strings.Repeat("a", 256)
	maxOK := strings.Repeat("a", 255)
	maxOKMultibyte := strings.Repeat("é", 255) // 255 characters, 510 bytes
	tooLongMultibyte := strings.Repeat("é", 256)

	for _, bad := range []string{"", "   ", "\t\n", long, tooLongMultibyte} {
		if _, err := f.svc.CreateModule(ctx, f.owner, f.courseID, bad, nil); err == nil {
			t.Errorf("CreateModule accepted title %q", short(bad))
		}
		if _, err := f.svc.CreateLesson(ctx, f.owner, f.moduleID, bad, "", nil); err == nil {
			t.Errorf("CreateLesson accepted title %q", short(bad))
		}
		if _, err := f.svc.UpdateModule(ctx, f.owner, f.moduleID, &bad, nil); err == nil {
			t.Errorf("UpdateModule accepted title %q", short(bad))
		}
		if _, err := f.svc.UpdateLesson(ctx, f.owner, f.lessonID, &bad, nil, nil); err == nil {
			t.Errorf("UpdateLesson accepted title %q", short(bad))
		}
	}
	for _, good := range []string{maxOK, maxOKMultibyte} {
		if _, err := f.svc.CreateModule(ctx, f.owner, f.courseID, good, nil); err != nil {
			t.Errorf("CreateModule rejected a title of exactly 255 characters: %v", err)
		}
		if _, err := f.svc.CreateLesson(ctx, f.owner, f.moduleID, good, "", nil); err != nil {
			t.Errorf("CreateLesson rejected a title of exactly 255 characters: %v", err)
		}
	}
	if m := f.store.modules[f.moduleID]; m.Title != "M1" {
		t.Fatalf("a rejected update changed the module: %+v", m)
	}
}

func short(s string) string {
	if len(s) > 12 {
		return s[:12] + "..."
	}
	return s
}

func TestPositionValidation(t *testing.T) {
	f := newContentFixture(domain.CourseStatusDraft)
	ctx := context.Background()

	for _, bad := range []int{0, -1} {
		if _, err := f.svc.CreateModule(ctx, f.owner, f.courseID, "x", intPtr(bad)); err == nil {
			t.Errorf("CreateModule accepted position %d", bad)
		}
		if _, err := f.svc.CreateLesson(ctx, f.owner, f.moduleID, "x", "", intPtr(bad)); err == nil {
			t.Errorf("CreateLesson accepted position %d", bad)
		}
		if _, err := f.svc.UpdateModule(ctx, f.owner, f.moduleID, nil, intPtr(bad)); err == nil {
			t.Errorf("UpdateModule accepted position %d", bad)
		}
		if _, err := f.svc.UpdateLesson(ctx, f.owner, f.lessonID, nil, nil, intPtr(bad)); err == nil {
			t.Errorf("UpdateLesson accepted position %d", bad)
		}
	}
	if _, err := f.svc.CreateModule(ctx, f.owner, f.courseID, "x", intPtr(1)); err != nil {
		t.Errorf("position 1 should be accepted: %v", err)
	}
}
