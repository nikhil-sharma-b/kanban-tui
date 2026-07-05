package ui

import "testing"

type fakeBuf struct {
	text   []rune
	pos    int
	single bool
}

func (b *fakeBuf) Text() string     { return string(b.text) }
func (b *fakeBuf) Cursor() int      { return b.pos }
func (b *fakeBuf) SingleLine() bool { return b.single }
func (b *fakeBuf) SetCursor(pos int) {
	b.pos = clampInt(pos, 0, len(b.text))
}
func (b *fakeBuf) SetText(text string, cursor int) {
	b.text = []rune(text)
	b.SetCursor(cursor)
}

// run feeds keys one rune at a time and returns final text/cursor.
func run(t *testing.T, text string, pos int, keys string) (*vimEngine, *fakeBuf) {
	t.Helper()
	v := &vimEngine{}
	buf := &fakeBuf{text: []rune(text), pos: pos}
	for _, k := range keys {
		v.HandleKey(buf, string(k))
	}
	return v, buf
}

func TestVimMotions(t *testing.T) {
	tests := []struct {
		name    string
		text    string
		pos     int
		keys    string
		wantPos int
	}{
		{"w", "foo bar baz", 0, "w", 4},
		{"2w", "foo bar baz", 0, "2w", 8},
		{"b", "foo bar baz", 8, "b", 4},
		{"e", "foo bar", 0, "e", 2},
		{"0", "foo bar", 5, "0", 0},
		{"dollar", "foo bar", 0, "$", 7},
		{"caret", "  foo", 4, "^", 2},
		{"f", "foo bar", 0, "fb", 4},
		{"F", "foo bar", 6, "Fo", 2},
		{"t", "foo bar", 0, "tb", 4},
		{"gg", "one\ntwo\nthree", 9, "gg", 0},
		{"G", "one\ntwo", 0, "G", 4},
		{"h_stops_at_line_start", "ab\ncd", 3, "h", 3},
		{"l_stops_at_line_end", "ab\ncd", 2, "l", 2},
		{"w_crosses_newline", "foo\nbar", 0, "w", 4},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, buf := run(t, tt.text, tt.pos, tt.keys)
			if buf.pos != tt.wantPos {
				t.Errorf("pos = %d, want %d", buf.pos, tt.wantPos)
			}
		})
	}
}

func TestVimOperators(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		pos      int
		keys     string
		wantText string
		wantReg  string
	}{
		{"dw", "foo bar baz", 0, "dw", "bar baz", "foo "},
		{"d2w", "foo bar baz", 0, "d2w", "baz", "foo bar "},
		{"db", "foo bar", 4, "db", "bar", "foo "},
		{"de", "foo bar", 0, "de", " bar", "foo"},
		{"d$", "foo bar", 3, "d$", "foo", " bar"},
		{"d0", "foo bar", 4, "d0", "bar", "foo "},
		{"dd_middle", "one\ntwo\nthree", 5, "dd", "one\nthree", "two"},
		{"dd_last", "one\ntwo", 5, "dd", "one", "two"},
		{"2dd", "a\nb\nc", 0, "2dd", "c", "a\nb"},
		{"df", "foo bar", 0, "dfb", "ar", "foo b"},
		{"dt", "foo bar", 0, "dtb", "bar", "foo "},
		{"dgg", "one\ntwo\nthree", 5, "dgg", "three", "one\ntwo"},
		{"dG", "one\ntwo\nthree", 5, "dG", "one", "two\nthree"},
		{"x", "abc", 1, "x", "ac", "b"},
		{"3x", "abcdef", 0, "3x", "def", "abc"},
		{"X", "abc", 2, "X", "ac", "b"},
		{"D", "foo bar", 3, "D", "foo", " bar"},
		{"diw", "foo bar baz", 5, "diw", "foo  baz", "bar"},
		{"daw", "foo bar baz", 5, "daw", "foo baz", "bar "},
		{"di_quote", `say "hello" now`, 6, `di"`, `say "" now`, "hello"},
		{"da_quote", `say "hello" now`, 6, `da"`, `say  now`, `"hello"`},
		{"di_paren", "f(a, b) x", 3, "di(", "f() x", "a, b"},
		{"da_paren", "f(a, b) x", 3, "da(", "f x", "(a, b)"},
		{"di_nested", "a(b(c)d)e", 4, "di(", "a(b()d)e", "c"},
		{"r", "abc", 1, "rx", "axc", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, buf := run(t, tt.text, tt.pos, tt.keys)
			if got := buf.Text(); got != tt.wantText {
				t.Errorf("text = %q, want %q", got, tt.wantText)
			}
			if tt.wantReg != "" && v.register != tt.wantReg {
				t.Errorf("register = %q, want %q", v.register, tt.wantReg)
			}
		})
	}
}

func TestVimYankPaste(t *testing.T) {
	// yy then p pastes line below.
	v, buf := run(t, "one\ntwo", 0, "yy")
	if v.register != "one" || !v.registerLine {
		t.Fatalf("register = %q linewise=%v", v.register, v.registerLine)
	}
	v.HandleKey(buf, "p")
	if got := buf.Text(); got != "one\none\ntwo" {
		t.Errorf("after p: %q", got)
	}

	// yw then P pastes charwise at cursor.
	v, buf = run(t, "foo bar", 0, "yw")
	if v.register != "foo " {
		t.Fatalf("register = %q", v.register)
	}
	buf.SetCursor(4)
	v.HandleKey(buf, "P")
	if got := buf.Text(); got != "foo foo bar" {
		t.Errorf("after P: %q", got)
	}

	// linewise paste into single-line buffer joins with spaces.
	v = &vimEngine{register: "a\nb", registerLine: true}
	sb := &fakeBuf{text: []rune("xy"), pos: 1, single: true}
	v.HandleKey(sb, "p")
	if got := sb.Text(); got != "xa by" {
		t.Errorf("single-line paste: %q", got)
	}

	// empty register.
	v = &vimEngine{}
	v.HandleKey(&fakeBuf{text: []rune("x")}, "p")
	if v.status != "nothing to paste" {
		t.Errorf("status = %q", v.status)
	}
}

func TestVimChange(t *testing.T) {
	v := &vimEngine{}
	buf := &fakeBuf{text: []rune("foo bar")}
	res1 := v.HandleKey(buf, "c")
	res2 := v.HandleKey(buf, "w")
	if res1.enterInsert || !res2.enterInsert {
		t.Errorf("cw insert flags: %v %v", res1.enterInsert, res2.enterInsert)
	}
	if got := buf.Text(); got != "bar" {
		t.Errorf("text = %q", got)
	}

	// cc keeps the empty line.
	v, buf = run(t, "one\ntwo\nthree", 5, "cc")
	if got := buf.Text(); got != "one\n\nthree" {
		t.Errorf("cc text = %q", got)
	}
}

func TestVimCountAndEsc(t *testing.T) {
	// esc-equivalent: reset clears pending operator.
	v := &vimEngine{}
	buf := &fakeBuf{text: []rune("foo bar")}
	v.HandleKey(buf, "d")
	if !v.pending() {
		t.Fatal("expected pending op")
	}
	v.reset()
	v.HandleKey(buf, "w")
	if got := buf.Text(); got != "foo bar" {
		t.Errorf("text changed after reset: %q", got)
	}
	if buf.pos != 4 {
		t.Errorf("w should move cursor, pos = %d", buf.pos)
	}
}
