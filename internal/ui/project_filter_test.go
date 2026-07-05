package ui

import "testing"

func TestFuzzyMatch(t *testing.T) {
	cases := []struct {
		query, target string
		want          bool
	}{
		{"", "anything", true},
		{"ktu", "kanban-tui", true},
		{"KTU", "kanban-tui", true},
		{"kanban", "kanban-tui", true},
		{"tuk", "kanban-tui", false},
		{"xyz", "kanban-tui", false},
		{"wrk", "Work Projects", true},
	}
	for _, c := range cases {
		if got := fuzzyMatch(c.query, c.target); got != c.want {
			t.Errorf("fuzzyMatch(%q, %q) = %v, want %v", c.query, c.target, got, c.want)
		}
	}
}
