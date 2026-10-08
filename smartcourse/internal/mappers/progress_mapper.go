package mappers

import (
	"smartcourse/internal/domain"
	"smartcourse/internal/persistence"
)

func ProgressDBToDomain(pdb *persistence.ProgressDB) *domain.Progress {
	return &domain.Progress{
		StudentID:        pdb.StudentID,
		CourseID:         pdb.CourseID,
		CompletedLessons: pdb.CompletedLessons,
		LastAccessed:     pdb.LastAccessed,
	}
}

func ProgressToPersistence(p *domain.Progress) *persistence.ProgressDB {
	return &persistence.ProgressDB{
		StudentID:        p.StudentID,
		CourseID:         p.CourseID,
		CompletedLessons: p.CompletedLessons,
		LastAccessed:     p.LastAccessed,
	}
}
