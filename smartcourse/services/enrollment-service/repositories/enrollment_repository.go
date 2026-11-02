package repositories

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	apperrors "smartcourse/libs/common/errors"
	"smartcourse/libs/common/outbox"
	"smartcourse/services/enrollment-service/domain"
	"smartcourse/services/enrollment-service/mappers"
	"smartcourse/services/enrollment-service/persistence"
)

type EnrollmentRepository struct {
	DB *gorm.DB
}

func NewEnrollmentRepository(db *gorm.DB) *EnrollmentRepository {
	return &EnrollmentRepository{DB: db}
}

var seatsTable = persistence.CourseSeatDB{}.TableName()

// Enroll runs the whole enrollment in one transaction (seat upsert, duplicate
// check, seat reservation, enrollment insert, EnrollmentCreated outbox row), so
// a failure at any step leaves nothing behind: no reserved seat, no event.
func (er *EnrollmentRepository) Enroll(ctx context.Context, enrollment *domain.Enrollment, capacity *int, event outbox.Event) error {
	enrollmentDB := mappers.EnrollmentToPersistence(enrollment)

	err := er.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := upsertSeats(tx, enrollment.CourseID, capacity); err != nil {
			return err
		}

		var active int64
		if err := tx.Model(&persistence.EnrollmentDB{}).
			Where("student_id = ? AND course_id = ? AND status = ?", enrollment.StudentID, enrollment.CourseID, domain.EnrollmentStatusActive).
			Count(&active).Error; err != nil {
			return &apperrors.DatabaseError{Operation: "check existing enrollment", Err: err}
		}
		if active > 0 {
			return &apperrors.EnrollmentAlreadyExistsError{StudentID: enrollment.StudentID, CourseID: enrollment.CourseID}
		}

		if err := reserveSeat(tx, enrollment.CourseID); err != nil {
			return err
		}

		if err := tx.Create(enrollmentDB).Error; err != nil {
			// The partial unique index is the backstop for a concurrent enroll.
			if isDuplicateKey(err) {
				return &apperrors.EnrollmentAlreadyExistsError{StudentID: enrollment.StudentID, CourseID: enrollment.CourseID}
			}
			return &apperrors.DatabaseError{Operation: "create enrollment", Err: err}
		}

		if err := outbox.Insert(tx, persistence.OutboxTable, event); err != nil {
			return &apperrors.DatabaseError{Operation: "write enrollment outbox event", Err: err}
		}
		return nil
	})
	if err != nil {
		return asAppError(err, "enroll")
	}

	*enrollment = *mappers.EnrollmentDBToDomain(enrollmentDB)
	return nil
}

// upsertSeats creates the course's seat row, or refreshes its capacity from
// the course's current enrollment limit. A limit lowered below the seats
// already taken caps capacity at `taken` (the course is then simply full)
// instead of violating CHECK (taken <= capacity).
func upsertSeats(tx *gorm.DB, courseID uuid.UUID, capacity *int) error {
	err := tx.Exec(`
		INSERT INTO `+seatsTable+` AS s (course_id, capacity, taken)
		VALUES (?, ?, 0)
		ON CONFLICT (course_id) DO UPDATE
		SET capacity = CASE WHEN EXCLUDED.capacity IS NULL THEN NULL
		                    ELSE GREATEST(EXCLUDED.capacity, s.taken) END`,
		courseID, capacity).Error
	if err != nil {
		return &apperrors.DatabaseError{Operation: "upsert course seats", Err: err}
	}
	return nil
}

// reserveSeat takes one seat atomically; zero rows updated means the course is
// full. A NULL capacity means unlimited.
func reserveSeat(tx *gorm.DB, courseID uuid.UUID) error {
	result := tx.Exec(`
		UPDATE `+seatsTable+`
		SET taken = taken + 1
		WHERE course_id = ? AND (capacity IS NULL OR taken < capacity)`,
		courseID)
	if result.Error != nil {
		return &apperrors.DatabaseError{Operation: "reserve seat", Err: result.Error}
	}
	if result.RowsAffected == 0 {
		return &apperrors.CourseFullError{CourseID: courseID}
	}
	return nil
}

func (er *EnrollmentRepository) GetEnrollmentByID(ctx context.Context, id uuid.UUID) (*domain.Enrollment, error) {
	var enrollmentDB persistence.EnrollmentDB
	if err := er.DB.WithContext(ctx).Where("id = ?", id).First(&enrollmentDB).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, &apperrors.EnrollmentNotFoundError{EnrollmentID: id}
		}
		return nil, &apperrors.DatabaseError{Operation: "get enrollment by id", Err: err}
	}
	return mappers.EnrollmentDBToDomain(&enrollmentDB), nil
}

func (er *EnrollmentRepository) GetCourseSeat(ctx context.Context, courseID uuid.UUID) (*domain.CourseSeat, error) {
	var seatDB persistence.CourseSeatDB
	if err := er.DB.WithContext(ctx).Where("course_id = ?", courseID).First(&seatDB).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, &apperrors.DatabaseError{Operation: "get course seats", Err: err}
	}
	return mappers.CourseSeatDBToDomain(&seatDB), nil
}

// asAppError keeps the repository's typed errors and wraps anything else
// (e.g. a failed COMMIT) as a DatabaseError.
func asAppError(err error, op string) error {
	var (
		exists *apperrors.EnrollmentAlreadyExistsError
		full   *apperrors.CourseFullError
		dbErr  *apperrors.DatabaseError
	)
	if errors.As(err, &exists) || errors.As(err, &full) || errors.As(err, &dbErr) {
		return err
	}
	return &apperrors.DatabaseError{Operation: op, Err: err}
}
