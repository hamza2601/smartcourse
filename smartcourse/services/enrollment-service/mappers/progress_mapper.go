package mappers

import (
	"smartcourse/services/enrollment-service/domain"
	"smartcourse/services/enrollment-service/persistence"
)

func LessonProgressDBToDomain(pdb *persistence.LessonProgressDB) *domain.LessonProgress {
	return &domain.LessonProgress{
		ID:           pdb.ID,
		EnrollmentID: pdb.EnrollmentID,
		LessonID:     pdb.LessonID,
		Status:       pdb.Status,
		CompletedAt:  pdb.CompletedAt,
		CreatedAt:    pdb.CreatedAt,
		UpdatedAt:    pdb.UpdatedAt,
	}
}

func LessonProgressToPersistence(p *domain.LessonProgress) *persistence.LessonProgressDB {
	return &persistence.LessonProgressDB{
		ID:           p.ID,
		EnrollmentID: p.EnrollmentID,
		LessonID:     p.LessonID,
		Status:       p.Status,
		CompletedAt:  p.CompletedAt,
		CreatedAt:    p.CreatedAt,
		UpdatedAt:    p.UpdatedAt,
	}
}

func CourseSeatDBToDomain(sdb *persistence.CourseSeatDB) *domain.CourseSeat {
	return &domain.CourseSeat{
		CourseID: sdb.CourseID,
		Capacity: sdb.Capacity,
		Taken:    sdb.Taken,
	}
}

func CourseSeatToPersistence(s *domain.CourseSeat) *persistence.CourseSeatDB {
	return &persistence.CourseSeatDB{
		CourseID: s.CourseID,
		Capacity: s.Capacity,
		Taken:    s.Taken,
	}
}
