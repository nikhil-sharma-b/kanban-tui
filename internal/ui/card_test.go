package ui

import "testing"

func TestTruncateCell(t *testing.T) {
	cases := []struct {
		in    string
		width int
		want  string
	}{
		{"Archive search by date", 23, "Archive search by date"},
		{"Import from Trello JSON", 23, "Import from Trello JSON"},
		{"Export board as markdown", 23, "Export board as markdo…"},
		{"Undo for move and delete", 23, "Undo for move and dele…"},
		// The cut lands after a space: trailing spaces are dropped before the ….
		{"Undo the last move or delete with u", 23, "Undo the last move or…"},
		{"Lists → columns, cards and checklists", 23, "Lists → columns, cards…"},
		// 80×24 card text area.
		{"Undo the last move or delete with u, from a toast", 34, "Undo the last move or delete with…"},
		{"", 23, ""},
	}
	for _, c := range cases {
		if got := truncateCell(c.in, c.width); got != c.want {
			t.Errorf("truncateCell(%q, %d) = %q, want %q", c.in, c.width, got, c.want)
		}
	}
}
