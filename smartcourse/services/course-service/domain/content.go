package domain

type ModuleWithLessons struct {
	Module
	Lessons []Lesson `json:"lessons"`
}
