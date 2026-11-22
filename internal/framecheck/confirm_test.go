package framecheck

import "testing"

// The clear-daily frames draw the home board behind the dialog under a daily
// header, which the daily board never shows, so only the dialog is enforced.
func TestClearDailyDialog(t *testing.T) {
	for _, name := range []string{"09/clear-daily", "09/clear-daily-matched"} {
		want, err := design.Expect(name)
		if err != nil {
			t.Fatal(err)
		}
		scn := Scenarios[name]
		got := ReadScreen(scn.Run(), want.Grid.W, want.Grid.H, scn.Palette)
		// The dialog: rows 13–24, cols 31–90 (1-based).
		for y := 12; y < 24; y++ {
			for x := 30; x < 90; x++ {
				if w, g := want.Grid.At(x, y), got.At(x, y); w != g {
					t.Errorf("%s: %s", name, Diff{Row: y + 1, Col: x + 1, Want: w, Got: g})
				}
			}
		}
	}
}
