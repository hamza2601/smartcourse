package repositories

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	apperrors "smartcourse/libs/common/errors"
	"smartcourse/services/course-service/domain"
	"smartcourse/services/course-service/mappers"
	"smartcourse/services/course-service/persistence"
)

type ContentRepository struct {
	DB *gorm.DB
}

func NewContentRepository(db *gorm.DB) *ContentRepository {
	return &ContentRepository{DB: db}
}

// CreateModule inserts a module. A zero Position means "omitted": it is
// assigned MAX(position)+1 within the course in the same transaction.
func (r *ContentRepository) CreateModule(ctx context.Context, module *domain.Module) error {
	moduleDB := mappers.ModuleToPersistence(module)
	if moduleDB.ID == uuid.Nil {
		moduleDB.ID = uuid.New()
	}

	err := r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if moduleDB.Position == 0 {
			if err := tx.Model(&persistence.ModuleDB{}).
				Where("course_id = ?", moduleDB.CourseID).
				Select("COALESCE(MAX(position), 0) + 1").
				Scan(&moduleDB.Position).Error; err != nil {
				return err
			}
		}
		return tx.Create(moduleDB).Error
	})
	if err != nil {
		if isDuplicateKey(err) {
			return &apperrors.PositionConflictError{Position: moduleDB.Position}
		}
		return &apperrors.DatabaseError{Operation: "create module", Err: err}
	}
	*module = *mappers.ModuleDBToDomain(moduleDB)
	return nil
}

func (r *ContentRepository) ListModules(ctx context.Context, courseID uuid.UUID) ([]domain.Module, error) {
	var modulesDB []persistence.ModuleDB
	if err := r.DB.WithContext(ctx).Where("course_id = ?", courseID).Order("position").Find(&modulesDB).Error; err != nil {
		return nil, &apperrors.DatabaseError{Operation: "list modules", Err: err}
	}
	modules := make([]domain.Module, len(modulesDB))
	for i := range modulesDB {
		modules[i] = *mappers.ModuleDBToDomain(&modulesDB[i])
	}
	return modules, nil
}

// GetModuleWithCourse returns the module and the course it belongs to.
func (r *ContentRepository) GetModuleWithCourse(ctx context.Context, moduleID uuid.UUID) (*domain.Module, *domain.Course, error) {
	db := r.DB.WithContext(ctx)

	var moduleDB persistence.ModuleDB
	if err := db.Where("id = ?", moduleID).First(&moduleDB).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, &apperrors.ModuleNotFoundError{ModuleID: moduleID}
		}
		return nil, nil, &apperrors.DatabaseError{Operation: "get module", Err: err}
	}

	var courseDB persistence.CourseDB
	if err := db.Where("id = ?", moduleDB.CourseID).First(&courseDB).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Deleted between the two reads; its modules went with it.
			return nil, nil, &apperrors.ModuleNotFoundError{ModuleID: moduleID}
		}
		return nil, nil, &apperrors.DatabaseError{Operation: "get course of module", Err: err}
	}
	return mappers.ModuleDBToDomain(&moduleDB), mappers.CourseDBToDomain(&courseDB), nil
}

func (r *ContentRepository) UpdateModule(ctx context.Context, module *domain.Module) error {
	var moduleDB persistence.ModuleDB
	err := r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&persistence.ModuleDB{}).
			Where("id = ?", module.ID).
			Updates(map[string]any{"title": module.Title, "position": module.Position})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return tx.Where("id = ?", module.ID).First(&moduleDB).Error
	})
	switch {
	case err == nil:
	case errors.Is(err, gorm.ErrRecordNotFound):
		return &apperrors.ModuleNotFoundError{ModuleID: module.ID}
	case isDuplicateKey(err):
		return &apperrors.PositionConflictError{Position: module.Position}
	default:
		return &apperrors.DatabaseError{Operation: "update module", Err: err}
	}
	*module = *mappers.ModuleDBToDomain(&moduleDB)
	return nil
}

// DeleteModule removes the module; its lessons go with it (ON DELETE CASCADE).
func (r *ContentRepository) DeleteModule(ctx context.Context, moduleID uuid.UUID) error {
	result := r.DB.WithContext(ctx).Where("id = ?", moduleID).Delete(&persistence.ModuleDB{})
	if result.Error != nil {
		return &apperrors.DatabaseError{Operation: "delete module", Err: result.Error}
	}
	if result.RowsAffected == 0 {
		return &apperrors.ModuleNotFoundError{ModuleID: moduleID}
	}
	return nil
}

// CreateLesson inserts a lesson. A zero Position means "omitted": it is
// assigned MAX(position)+1 within the module in the same transaction.
func (r *ContentRepository) CreateLesson(ctx context.Context, lesson *domain.Lesson) error {
	lessonDB := mappers.LessonToPersistence(lesson)
	if lessonDB.ID == uuid.Nil {
		lessonDB.ID = uuid.New()
	}

	err := r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if lessonDB.Position == 0 {
			if err := tx.Model(&persistence.LessonDB{}).
				Where("module_id = ?", lessonDB.ModuleID).
				Select("COALESCE(MAX(position), 0) + 1").
				Scan(&lessonDB.Position).Error; err != nil {
				return err
			}
		}
		return tx.Create(lessonDB).Error
	})
	if err != nil {
		if isDuplicateKey(err) {
			return &apperrors.PositionConflictError{Position: lessonDB.Position}
		}
		return &apperrors.DatabaseError{Operation: "create lesson", Err: err}
	}
	*lesson = *mappers.LessonDBToDomain(lessonDB)
	return nil
}

func (r *ContentRepository) ListLessons(ctx context.Context, moduleID uuid.UUID) ([]domain.Lesson, error) {
	var lessonsDB []persistence.LessonDB
	if err := r.DB.WithContext(ctx).Where("module_id = ?", moduleID).Order("position").Find(&lessonsDB).Error; err != nil {
		return nil, &apperrors.DatabaseError{Operation: "list lessons", Err: err}
	}
	lessons := make([]domain.Lesson, len(lessonsDB))
	for i := range lessonsDB {
		lessons[i] = *mappers.LessonDBToDomain(&lessonsDB[i])
	}
	return lessons, nil
}

// GetLessonWithCourse returns the lesson and its course, found by joining
// lessons -> modules -> courses.
func (r *ContentRepository) GetLessonWithCourse(ctx context.Context, lessonID uuid.UUID) (*domain.Lesson, *domain.Course, error) {
	db := r.DB.WithContext(ctx)

	var lessonDB persistence.LessonDB
	if err := db.Where("id = ?", lessonID).First(&lessonDB).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, &apperrors.LessonNotFoundError{LessonID: lessonID}
		}
		return nil, nil, &apperrors.DatabaseError{Operation: "get lesson", Err: err}
	}

	var courseDB persistence.CourseDB
	if err := db.
		Joins("JOIN course_service.modules m ON m.course_id = course_service.courses.id").
		Joins("JOIN course_service.lessons l ON l.module_id = m.id").
		Where("l.id = ?", lessonID).
		First(&courseDB).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Deleted between the two reads; its lessons went with it.
			return nil, nil, &apperrors.LessonNotFoundError{LessonID: lessonID}
		}
		return nil, nil, &apperrors.DatabaseError{Operation: "get course of lesson", Err: err}
	}
	return mappers.LessonDBToDomain(&lessonDB), mappers.CourseDBToDomain(&courseDB), nil
}

func (r *ContentRepository) UpdateLesson(ctx context.Context, lesson *domain.Lesson) error {
	var lessonDB persistence.LessonDB
	err := r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&persistence.LessonDB{}).
			Where("id = ?", lesson.ID).
			Updates(map[string]any{"title": lesson.Title, "content": lesson.Content, "position": lesson.Position})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return tx.Where("id = ?", lesson.ID).First(&lessonDB).Error
	})
	switch {
	case err == nil:
	case errors.Is(err, gorm.ErrRecordNotFound):
		return &apperrors.LessonNotFoundError{LessonID: lesson.ID}
	case isDuplicateKey(err):
		return &apperrors.PositionConflictError{Position: lesson.Position}
	default:
		return &apperrors.DatabaseError{Operation: "update lesson", Err: err}
	}
	*lesson = *mappers.LessonDBToDomain(&lessonDB)
	return nil
}

func (r *ContentRepository) DeleteLesson(ctx context.Context, lessonID uuid.UUID) error {
	result := r.DB.WithContext(ctx).Where("id = ?", lessonID).Delete(&persistence.LessonDB{})
	if result.Error != nil {
		return &apperrors.DatabaseError{Operation: "delete lesson", Err: result.Error}
	}
	if result.RowsAffected == 0 {
		return &apperrors.LessonNotFoundError{LessonID: lessonID}
	}
	return nil
}

// GetCourseContent returns the course's modules, each with its lessons, both
// ordered by position. Modules without lessons get an empty (non-nil) list.
func (r *ContentRepository) GetCourseContent(ctx context.Context, courseID uuid.UUID) ([]domain.ModuleWithLessons, error) {
	db := r.DB.WithContext(ctx)

	var modulesDB []persistence.ModuleDB
	if err := db.Where("course_id = ?", courseID).Order("position").Find(&modulesDB).Error; err != nil {
		return nil, &apperrors.DatabaseError{Operation: "get course content", Err: err}
	}

	content := make([]domain.ModuleWithLessons, len(modulesDB))
	indexByModule := make(map[uuid.UUID]int, len(modulesDB))
	moduleIDs := make([]uuid.UUID, len(modulesDB))
	for i := range modulesDB {
		content[i] = domain.ModuleWithLessons{Module: *mappers.ModuleDBToDomain(&modulesDB[i]), Lessons: []domain.Lesson{}}
		indexByModule[modulesDB[i].ID] = i
		moduleIDs[i] = modulesDB[i].ID
	}
	if len(moduleIDs) == 0 {
		return content, nil
	}

	var lessonsDB []persistence.LessonDB
	if err := db.Where("module_id IN ?", moduleIDs).Order("position").Find(&lessonsDB).Error; err != nil {
		return nil, &apperrors.DatabaseError{Operation: "get course content lessons", Err: err}
	}
	for i := range lessonsDB {
		m := indexByModule[lessonsDB[i].ModuleID]
		content[m].Lessons = append(content[m].Lessons, *mappers.LessonDBToDomain(&lessonsDB[i]))
	}
	return content, nil
}
