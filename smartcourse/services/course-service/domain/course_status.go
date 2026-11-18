package domain

// allowedStatusTransitions lists every permitted status change. A status that
// is missing as a key (archived, or any unknown value) has no way out.
var allowedStatusTransitions = map[string]map[string]bool{
	CourseStatusDraft:      {CourseStatusPublishing: true, CourseStatusArchived: true},
	CourseStatusReady:      {CourseStatusPublishing: true, CourseStatusArchived: true},
	CourseStatusPublishing: {CourseStatusReady: true, CourseStatusDraft: true},
}

// CanTransition reports whether a course may move from one status to another.
// Same-status moves and unknown values are never allowed.
func CanTransition(from, to string) bool {
	return allowedStatusTransitions[from][to]
}
