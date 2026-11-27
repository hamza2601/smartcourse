package repositories

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	apperrors "smartcourse/libs/common/errors"
	"smartcourse/libs/common/outbox"
	"smartcourse/services/course-service/domain"
	"smartcourse/services/course-service/mappers"
	"smartcourse/services/course-service/persistence"
)

type CourseRepository struct {
	DB *gorm.DB
}

func NewCourseRepository(db *gorm.DB) *CourseRepository {
	return &CourseRepository{DB: db}
}

// CreateCourse inserts the course and its CourseCreated outbox event in one
// transaction: either both are stored or neither is.
func (cr *CourseRepository) CreateCourse(ctx context.Context, course *domain.Course, event outbox.Event) error {
	courseDB := mappers.CourseToPersistence(course)
	if courseDB.ID == uuid.Nil {
		courseDB.ID = uuid.New()
	}

	err := cr.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(courseDB).Error; err != nil {
			return err
		}
		return outbox.Insert(tx, persistence.OutboxTable, event)
	})
	if err != nil {
		return &apperrors.DatabaseError{Operation: "create course", Err: err}
	}
	*course = *mappers.CourseDBToDomain(courseDB)
	return nil
}

func (cr *CourseRepository) GetCourseByID(ctx context.Context, id uuid.UUID) (*domain.Course, error) {
	var courseDB persistence.CourseDB
	if err := cr.DB.WithContext(ctx).Where("id = ?", id).First(&courseDB).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, &apperrors.CourseNotFoundError{CourseID: id}
		}
		return nil, &apperrors.DatabaseError{Operation: "get course by id", Err: err}
	}
	return mappers.CourseDBToDomain(&courseDB), nil
}

func (cr *CourseRepository) GetCoursesByInstructor(ctx context.Context, instructorID uuid.UUID) ([]domain.Course, error) {
	var coursesDB []persistence.CourseDB
	if err := cr.DB.WithContext(ctx).Where("instructor_id = ?", instructorID).Find(&coursesDB).Error; err != nil {
		return nil, &apperrors.DatabaseError{Operation: "get courses by instructor", Err: err}
	}
	return coursesToDomain(coursesDB), nil
}

func (cr *CourseRepository) ListAllCourses(ctx context.Context) ([]domain.Course, error) {
	var coursesDB []persistence.CourseDB
	if err := cr.DB.WithContext(ctx).Find(&coursesDB).Error; err != nil {
		return nil, &apperrors.DatabaseError{Operation: "list courses", Err: err}
	}
	return coursesToDomain(coursesDB), nil
}

// UpdateCourseStatus is a conditional update so concurrent transitions cannot
// both succeed. The outbox event is written in the same transaction, and only
// if the status actually changed; it reports false when the course was not in
// status `from`.
func (cr *CourseRepository) UpdateCourseStatus(ctx context.Context, id uuid.UUID, from, to string, event outbox.Event) (bool, error) {
	var updated bool
	err := cr.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&persistence.CourseDB{}).
			Where("id = ? AND status = ?", id, from).
			Update("status", to)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return nil
		}
		if err := outbox.Insert(tx, persistence.OutboxTable, event); err != nil {
			return err
		}
		updated = true
		return nil
	})
	if err != nil {
		return false, &apperrors.DatabaseError{Operation: "update course status", Err: err}
	}
	return updated, nil
}

// UpdateCourse changes the non-nil fields in a guarded UPDATE that only matches
// a draft or ready course, and returns the updated course. If nothing matched
// it re-reads to tell a missing course from one that is not editable.
func (cr *CourseRepository) UpdateCourse(ctx context.Context, id uuid.UUID, name, description *string, enrollmentLimit *int) (*domain.Course, error) {
	updates := map[string]any{"updated_at": time.Now()}
	if name != nil {
		updates["name"] = *name
	}
	if description != nil {
		updates["description"] = *description
	}
	if enrollmentLimit != nil {
		updates["enrollment_limit"] = *enrollmentLimit
	}

	result := cr.DB.WithContext(ctx).Model(&persistence.CourseDB{}).
		Where("id = ? AND status IN ?", id, []string{domain.CourseStatusDraft, domain.CourseStatusReady}).
		Updates(updates)
	if result.Error != nil {
		return nil, &apperrors.DatabaseError{Operation: "update course", Err: result.Error}
	}

	course, err := cr.GetCourseByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if result.RowsAffected == 0 {
		return nil, &apperrors.CourseNotEditableError{CourseID: id, Status: course.Status}
	}
	return course, nil
}

// TransitionStatus is a conditional status update that writes no outbox event.
// It reports false when the course was not in status `from`.
func (cr *CourseRepository) TransitionStatus(ctx context.Context, id uuid.UUID, from, to string) (bool, error) {
	result := cr.DB.WithContext(ctx).Model(&persistence.CourseDB{}).
		Where("id = ? AND status = ?", id, from).
		Update("status", to)
	if result.Error != nil {
		return false, &apperrors.DatabaseError{Operation: "transition course status", Err: result.Error}
	}
	return result.RowsAffected > 0, nil
}

func coursesToDomain(coursesDB []persistence.CourseDB) []domain.Course {
	courses := make([]domain.Course, len(coursesDB))
	for i := range coursesDB {
		courses[i] = *mappers.CourseDBToDomain(&coursesDB[i])
	}
	return courses
}
