package mappers

import (
	"smartcourse/services/course-service/domain"
	"smartcourse/services/course-service/persistence"
)

func ModuleDBToDomain(mdb *persistence.ModuleDB) *domain.Module {
	return &domain.Module{
		ID:        mdb.ID,
		CourseID:  mdb.CourseID,
		Title:     mdb.Title,
		Position:  mdb.Position,
		CreatedAt: mdb.CreatedAt,
		UpdatedAt: mdb.UpdatedAt,
	}
}

func ModuleToPersistence(m *domain.Module) *persistence.ModuleDB {
	return &persistence.ModuleDB{
		ID:        m.ID,
		CourseID:  m.CourseID,
		Title:     m.Title,
		Position:  m.Position,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}

func LessonDBToDomain(ldb *persistence.LessonDB) *domain.Lesson {
	return &domain.Lesson{
		ID:        ldb.ID,
		ModuleID:  ldb.ModuleID,
		Title:     ldb.Title,
		Content:   ldb.Content,
		Position:  ldb.Position,
		CreatedAt: ldb.CreatedAt,
		UpdatedAt: ldb.UpdatedAt,
	}
}

func LessonToPersistence(l *domain.Lesson) *persistence.LessonDB {
	return &persistence.LessonDB{
		ID:        l.ID,
		ModuleID:  l.ModuleID,
		Title:     l.Title,
		Content:   l.Content,
		Position:  l.Position,
		CreatedAt: l.CreatedAt,
		UpdatedAt: l.UpdatedAt,
	}
}
