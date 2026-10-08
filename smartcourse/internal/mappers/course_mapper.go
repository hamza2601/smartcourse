package mappers

import (
	"smartcourse/internal/domain"
	"smartcourse/internal/persistence"
)

func CourseDBToDomain(cdb *persistence.CourseDB) *domain.Course {
	return &domain.Course{
		ID:           cdb.ID,
		Name:         cdb.Name,
		InstructorID: cdb.InstructorID,
		Status:       cdb.Status,
		CreatedAt:    cdb.CreatedAt,
		UpdatedAt:    cdb.UpdatedAt,
	}
}

func CourseToPersistence(c *domain.Course) *persistence.CourseDB {
	return &persistence.CourseDB{
		ID:           c.ID,
		Name:         c.Name,
		InstructorID: c.InstructorID,
		Status:       c.Status,
		CreatedAt:    c.CreatedAt,
		UpdatedAt:    c.UpdatedAt,
	}
}
