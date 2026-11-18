package domain

import "testing"

func TestCanTransition(t *testing.T) {
	// The only allowed moves, written out here independently of the map in
	// course_status.go so a mistake there cannot hide itself.
	allowed := map[[2]string]bool{
		{"draft", "publishing"}: true,
		{"ready", "publishing"}: true,
		{"publishing", "ready"}: true,
		{"publishing", "draft"}: true,
		{"draft", "archived"}:   true,
		{"ready", "archived"}:   true,
	}
	if len(allowed) != 6 {
		t.Fatalf("test setup: want 6 allowed moves, have %d", len(allowed))
	}

	// The four real statuses plus one unknown value ("published" is the
	// retired name of "ready").
	statuses := []string{"draft", "publishing", "ready", "archived", "published"}

	for _, from := range statuses {
		for _, to := range statuses {
			want := allowed[[2]string{from, to}]
			if got := CanTransition(from, to); got != want {
				t.Errorf("CanTransition(%q, %q) = %v, want %v", from, to, got, want)
			}
		}
	}
}
