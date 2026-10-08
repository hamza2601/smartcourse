package mappers

import (
	"smartcourse/internal/domain"
	"smartcourse/internal/persistence"
)

func EnrollmentDBToDomain(edb *persistence.EnrollmentDB) *domain.Enrollment {
	return &domain.Enrollment{
		ID:        edb.ID,
		StudentID: edb.StudentID,
		CourseID:  edb.CourseID,
		Status:    edb.Status,
		CreatedAt: edb.CreatedAt,
		UpdatedAt: edb.UpdatedAt,
	}
}

func EnrollmentToPersistence(e *domain.Enrollment) *persistence.EnrollmentDB {
	return &persistence.EnrollmentDB{
		ID:        e.ID,
		StudentID: e.StudentID,
		CourseID:  e.CourseID,
		Status:    e.Status,
		CreatedAt: e.CreatedAt,
		UpdatedAt: e.UpdatedAt,
	}
}
