package ui

import (
	"fmt"
	"strings"
	"unicode"
)

// vimBuffer abstracts a focused text input as plain text with an absolute
// rune-offset cursor, so motions and operators are pure text operations.
type vimBuffer interface {
	Text() string
	Cursor() int
	SetText(text string, cursor int)
	SetCursor(pos int)
	SingleLine() bool
}

// vimEngine implements composable operator × motion editing for normal mode.
// Operators (d, y, c) are written once and work with every motion; adding a
// motion to motionTarget automatically enables it for all operators.
type vimEngine struct {
	op           rune // pending operator: 'd', 'y' or 'c'
	prefix       rune // pending multi-key sequence: 'g', 'f', 't', 'F', 'T', 'r', 'i', 'a'
	count        int  // count prefix being accumulated (0 = none)
	register     string
	registerLine bool   // register holds whole line(s)
	status       string // transient feedback, e.g. yanked "foo"
}

type vimResult struct {
	handled     bool
	enterInsert bool
}

func (v *vimEngine) reset() {
	v.op, v.prefix, v.count = 0, 0, 0
}

func (v *vimEngine) pending() bool {
	return v.op != 0 || v.prefix != 0 || v.count != 0
}

// HandleKey processes one normal-mode key against the focused buffer.
func (v *vimEngine) HandleKey(buf vimBuffer, key string) vimResult {
	v.status = ""
	runes := []rune(key)
	if len(runes) != 1 {
		v.reset()
		return vimResult{}
	}
	ch := runes[0]
	text := []rune(buf.Text())
	pos := clampInt(buf.Cursor(), 0, len(text))

	// Multi-key sequences waiting for their final character.
	if v.prefix != 0 {
		prefix := v.prefix
		v.prefix = 0
		return v.finishPrefix(buf, text, pos, prefix, ch)
	}

	// Count prefix ('0' alone is a motion, not a count).
	if unicode.IsDigit(ch) && !(ch == '0' && v.count == 0) {
		v.count = v.count*10 + int(ch-'0')
		return vimResult{handled: true}
	}
	count0 := v.count
	count := max(v.count, 1)
	v.count = 0

	switch ch {
	case 'd', 'y', 'c':
		if v.op == ch { // dd / yy / cc – operate on whole line(s)
			op := v.op
			v.op = 0
			start, end := lineSpan(text, pos, count)
			return v.opRange(buf, text, start, end, true, op)
		}
		if v.op != 0 {
			v.op = 0
			return vimResult{handled: true}
		}
		v.op = ch
		v.count = count0 // "2dd" == "d2d": keep count for the motion
		return vimResult{handled: true}

	case 'g', 'f', 't', 'F', 'T':
		v.prefix = ch
		return vimResult{handled: true}
	case 'r':
		if v.op == 0 {
			v.prefix = ch
			return vimResult{handled: true}
		}
		v.op = 0
		return vimResult{handled: true}

	// Insert-mode entry.
	case 'i', 'a':
		if v.op != 0 { // text object: diw, ci(, ya" ...
			v.prefix = ch
			return vimResult{handled: true}
		}
		if ch == 'a' {
			buf.SetCursor(min(pos+1, lineEnd(text, pos)))
		}
		return vimResult{handled: true, enterInsert: true}
	case 'A':
		buf.SetCursor(lineEnd(text, pos))
		return vimResult{handled: true, enterInsert: true}
	case 'I':
		buf.SetCursor(firstNonBlank(text, pos))
		return vimResult{handled: true, enterInsert: true}
	case 'o':
		if buf.SingleLine() {
			return vimResult{handled: true}
		}
		le := lineEnd(text, pos)
		buf.SetText(string(text[:le])+"\n"+string(text[le:]), le+1)
		return vimResult{handled: true, enterInsert: true}
	case 'O':
		if buf.SingleLine() {
			return vimResult{handled: true}
		}
		ls := lineStart(text, pos)
		buf.SetText(string(text[:ls])+"\n"+string(text[ls:]), ls)
		return vimResult{handled: true, enterInsert: true}

	// Shorthand operators.
	case 'x':
		return v.opRange(buf, text, pos, min(pos+count, lineEnd(text, pos)), false, 'd')
	case 'X':
		return v.opRange(buf, text, max(pos-count, lineStart(text, pos)), pos, false, 'd')
	case 's':
		return v.opRange(buf, text, pos, min(pos+count, lineEnd(text, pos)), false, 'c')
	case 'D':
		return v.opRange(buf, text, pos, lineEnd(text, pos), false, 'd')
	case 'C':
		return v.opRange(buf, text, pos, lineEnd(text, pos), false, 'c')
	case 'S':
		start, end := lineSpan(text, pos, count)
		return v.opRange(buf, text, start, end, true, 'c')
	case 'Y':
		start, end := lineSpan(text, pos, count)
		return v.opRange(buf, text, start, end, true, 'y')

	case 'p':
		v.paste(buf, text, pos, true)
		return vimResult{handled: true}
	case 'P':
		v.paste(buf, text, pos, false)
		return vimResult{handled: true}
	}

	if mr, ok := motionTarget(ch, text, pos, count); ok {
		return v.applyMotion(buf, text, pos, mr)
	}
	v.op = 0
	return vimResult{}
}

// motionResult describes where a motion lands and how operators treat it.
type motionResult struct {
	pos       int
	inclusive bool // operator range includes the character at pos
	linewise  bool // operator range expands to whole lines
}

func motionTarget(ch rune, text []rune, pos, count int) (motionResult, bool) {
	switch ch {
	case 'h':
		p := pos
		for i := 0; i < count && p > lineStart(text, p); i++ {
			p--
		}
		return motionResult{pos: p}, true
	case 'l', ' ':
		p := pos
		for i := 0; i < count && p < lineEnd(text, p); i++ {
			p++
		}
		return motionResult{pos: p}, true
	case '0':
		return motionResult{pos: lineStart(text, pos)}, true
	case '^':
		return motionResult{pos: firstNonBlank(text, pos)}, true
	case '$':
		return motionResult{pos: lineEnd(text, pos)}, true
	case 'w', 'W':
		p := pos
		for i := 0; i < count; i++ {
			p = wordForward(text, p)
		}
		return motionResult{pos: p}, true
	case 'b', 'B':
		p := pos
		for i := 0; i < count; i++ {
			p = wordBack(text, p)
		}
		return motionResult{pos: p}, true
	case 'e', 'E':
		p := pos
		for i := 0; i < count; i++ {
			p = wordEnd(text, p)
		}
		return motionResult{pos: p, inclusive: true}, true
	case 'G':
		return motionResult{pos: len(text), linewise: true}, true
	}
	return motionResult{}, false
}

func (v *vimEngine) applyMotion(buf vimBuffer, text []rune, pos int, mr motionResult) vimResult {
	if v.op == 0 {
		p := clampInt(mr.pos, 0, len(text))
		if mr.linewise { // gg/G land on the first non-blank of the target line
			p = firstNonBlank(text, p)
		}
		buf.SetCursor(p)
		return vimResult{handled: true}
	}
	op := v.op
	v.op = 0
	start, end := pos, clampInt(mr.pos, 0, len(text))
	if start > end {
		start, end = end, start
	}
	if mr.inclusive && end < len(text) {
		end++
	}
	return v.opRange(buf, text, start, end, mr.linewise, op)
}

// opRange applies operator op over text[start:end).
func (v *vimEngine) opRange(buf vimBuffer, text []rune, start, end int, linewise bool, op rune) vimResult {
	if linewise {
		start = lineStart(text, start)
		end = lineEnd(text, end)
	}
	start = clampInt(start, 0, len(text))
	end = clampInt(end, start, len(text))
	seg := string(text[start:end])
	v.register, v.registerLine = seg, linewise

	switch op {
	case 'y':
		v.status = vimPreview("yanked", seg)
		buf.SetCursor(start)
		return vimResult{handled: true}
	case 'd', 'c':
		delStart, delEnd := start, end
		// Linewise delete removes the line itself, not just its content;
		// linewise change (cc/S) keeps the empty line for insert mode.
		if linewise && op == 'd' {
			if delEnd < len(text) {
				delEnd++ // trailing newline
			} else if delStart > 0 {
				delStart-- // last line: eat preceding newline
			}
		}
		buf.SetText(string(text[:delStart])+string(text[delEnd:]), delStart)
		v.status = vimPreview("deleted", seg)
		return vimResult{handled: true, enterInsert: op == 'c'}
	}
	return vimResult{handled: true}
}

// finishPrefix completes a multi-key sequence (g_, f_, t_, r_, text objects).
func (v *vimEngine) finishPrefix(buf vimBuffer, text []rune, pos int, prefix, ch rune) vimResult {
	count := max(v.count, 1)
	v.count = 0

	switch prefix {
	case 'g':
		switch ch {
		case 'g': // gg – first line
			return v.applyMotion(buf, text, pos, motionResult{pos: 0, linewise: true})
		case 'e': // ge – end of previous word
			p := pos
			for i := 0; i < count; i++ {
				p = prevWordEnd(text, p)
			}
			return v.applyMotion(buf, text, pos, motionResult{pos: p, inclusive: true})
		}
		v.op = 0
		return vimResult{handled: true}

	case 'f', 't':
		idx := pos
		for i := 0; i < count; i++ {
			next := indexOnLine(text, idx+1, ch)
			if next < 0 {
				v.op = 0
				return vimResult{handled: true}
			}
			idx = next
		}
		if prefix == 'f' {
			return v.applyMotion(buf, text, pos, motionResult{pos: idx, inclusive: true})
		}
		return v.applyMotion(buf, text, pos, motionResult{pos: idx}) // t – stop before char

	case 'F', 'T':
		idx := pos
		for i := 0; i < count; i++ {
			prev := lastIndexOnLine(text, idx-1, ch)
			if prev < 0 {
				v.op = 0
				return vimResult{handled: true}
			}
			idx = prev
		}
		if prefix == 'F' {
			return v.applyMotion(buf, text, pos, motionResult{pos: idx})
		}
		return v.applyMotion(buf, text, pos, motionResult{pos: min(idx+1, len(text))})

	case 'r': // r{char} – replace character under cursor
		if pos < lineEnd(text, pos) {
			out := make([]rune, len(text))
			copy(out, text)
			out[pos] = ch
			buf.SetText(string(out), pos)
		}
		return vimResult{handled: true}

	case 'i', 'a': // text objects, only valid with a pending operator
		op := v.op
		v.op = 0
		if op == 0 {
			return vimResult{handled: true}
		}
		start, end, ok := textObject(text, pos, ch, prefix == 'a')
		if !ok {
			return vimResult{handled: true}
		}
		return v.opRange(buf, text, start, end, false, op)
	}
	return vimResult{handled: true}
}

func (v *vimEngine) paste(buf vimBuffer, text []rune, pos int, after bool) {
	if v.register == "" {
		v.status = "nothing to paste"
		return
	}
	reg := v.register
	switch {
	case v.registerLine && buf.SingleLine():
		reg = strings.ReplaceAll(reg, "\n", " ")
		buf.SetText(string(text[:pos])+reg+string(text[pos:]), pos+len([]rune(reg)))
	case v.registerLine:
		if after {
			at := lineEnd(text, pos)
			buf.SetText(string(text[:at])+"\n"+reg+string(text[at:]), at+1)
		} else {
			at := lineStart(text, pos)
			buf.SetText(string(text[:at])+reg+"\n"+string(text[at:]), at)
		}
	default:
		buf.SetText(string(text[:pos])+reg+string(text[pos:]), pos+len([]rune(reg)))
	}
	v.status = vimPreview("pasted", v.register)
}

// ---- text helpers (all operate on rune slices with absolute offsets) ----

func clampInt(n, lo, hi int) int {
	return max(lo, min(n, hi))
}

func lineStart(text []rune, pos int) int {
	p := min(pos, len(text))
	for p > 0 && text[p-1] != '\n' {
		p--
	}
	return p
}

func lineEnd(text []rune, pos int) int {
	p := min(pos, len(text))
	for p < len(text) && text[p] != '\n' {
		p++
	}
	return p
}

func firstNonBlank(text []rune, pos int) int {
	p := lineStart(text, pos)
	end := lineEnd(text, pos)
	for p < end && unicode.IsSpace(text[p]) {
		p++
	}
	return p
}

// lineSpan returns the content bounds of count lines starting at pos's line.
func lineSpan(text []rune, pos, count int) (int, int) {
	start := lineStart(text, pos)
	end := lineEnd(text, pos)
	for i := 1; i < count && end < len(text); i++ {
		end = lineEnd(text, end+1)
	}
	return start, end
}

// indexOnLine finds ch at or after from, without crossing a newline.
func indexOnLine(text []rune, from int, ch rune) int {
	for i := max(from, 0); i < len(text) && text[i] != '\n'; i++ {
		if text[i] == ch {
			return i
		}
	}
	return -1
}

// lastIndexOnLine finds ch at or before from, without crossing a newline.
func lastIndexOnLine(text []rune, from int, ch rune) int {
	for i := min(from, len(text)-1); i >= 0 && text[i] != '\n'; i-- {
		if text[i] == ch {
			return i
		}
	}
	return -1
}

// Word motions use WORD (whitespace-delimited) semantics and cross newlines.

func wordForward(text []rune, pos int) int {
	p := pos
	for p < len(text) && !unicode.IsSpace(text[p]) {
		p++
	}
	for p < len(text) && unicode.IsSpace(text[p]) {
		p++
	}
	return p
}

func wordBack(text []rune, pos int) int {
	p := pos
	for p > 0 && unicode.IsSpace(text[p-1]) {
		p--
	}
	for p > 0 && !unicode.IsSpace(text[p-1]) {
		p--
	}
	return p
}

// wordEnd returns the offset of the last character of the next word end.
func wordEnd(text []rune, pos int) int {
	p := pos + 1
	for p < len(text) && unicode.IsSpace(text[p]) {
		p++
	}
	for p < len(text)-1 && !unicode.IsSpace(text[p+1]) {
		p++
	}
	return min(p, max(len(text)-1, 0))
}

// prevWordEnd returns the offset just past the last char of the previous word.
func prevWordEnd(text []rune, pos int) int {
	p := pos
	for p > 0 && !unicode.IsSpace(text[p-1]) {
		p--
	}
	for p > 0 && unicode.IsSpace(text[p-1]) {
		p--
	}
	return max(p-1, 0)
}

// textObject returns the range for iw/aw and delimiter pairs (quotes, brackets).
func textObject(text []rune, pos int, obj rune, around bool) (int, int, bool) {
	switch obj {
	case 'w', 'W':
		return wordObject(text, pos, around)
	case '"', '\'', '`':
		return quoteObject(text, pos, obj, around)
	case '(', ')', 'b':
		return pairObject(text, pos, '(', ')', around)
	case '{', '}', 'B':
		return pairObject(text, pos, '{', '}', around)
	case '[', ']':
		return pairObject(text, pos, '[', ']', around)
	case '<', '>':
		return pairObject(text, pos, '<', '>', around)
	}
	return 0, 0, false
}

func wordObject(text []rune, pos int, around bool) (int, int, bool) {
	if len(text) == 0 {
		return 0, 0, false
	}
	p := min(pos, len(text)-1)
	onSpace := unicode.IsSpace(text[p])
	match := func(r rune) bool { return unicode.IsSpace(r) == onSpace && r != '\n' }
	start, end := p, p+1
	for start > 0 && match(text[start-1]) {
		start--
	}
	for end < len(text) && match(text[end]) {
		end++
	}
	if around && !onSpace {
		grew := false
		for end < len(text) && text[end] != '\n' && unicode.IsSpace(text[end]) {
			end++
			grew = true
		}
		if !grew {
			for start > 0 && text[start-1] != '\n' && unicode.IsSpace(text[start-1]) {
				start--
			}
		}
	}
	return start, end, true
}

func quoteObject(text []rune, pos int, q rune, around bool) (int, int, bool) {
	open := lastIndexOnLine(text, pos, q)
	if open < 0 {
		open = indexOnLine(text, pos, q)
	}
	if open < 0 {
		return 0, 0, false
	}
	close := indexOnLine(text, open+1, q)
	if close < 0 {
		return 0, 0, false
	}
	if around {
		return open, close + 1, true
	}
	return open + 1, close, true
}

func pairObject(text []rune, pos int, open, close rune, around bool) (int, int, bool) {
	depth := 0
	start := -1
	for i := min(pos, len(text)-1); i >= 0; i-- {
		switch text[i] {
		case close:
			if i != pos {
				depth++
			}
		case open:
			if depth == 0 {
				start = i
			} else {
				depth--
			}
		}
		if start >= 0 {
			break
		}
	}
	if start < 0 {
		return 0, 0, false
	}
	depth = 0
	for i := start + 1; i < len(text); i++ {
		switch text[i] {
		case open:
			depth++
		case close:
			if depth == 0 {
				if around {
					return start, i + 1, true
				}
				return start + 1, i, true
			}
			depth--
		}
	}
	return 0, 0, false
}

// ---- buffer adapters over bubbles inputs ----

type titleBuffer struct{ m *model }

func (b titleBuffer) Text() string     { return b.m.titleInput.Value() }
func (b titleBuffer) Cursor() int      { return b.m.titleInput.Position() }
func (b titleBuffer) SingleLine() bool { return true }
func (b titleBuffer) SetCursor(pos int) {
	b.m.titleInput.SetCursor(pos)
}
func (b titleBuffer) SetText(text string, cursor int) {
	b.m.titleInput.SetValue(text)
	b.m.titleInput.SetCursor(cursor)
}

type descBuffer struct{ m *model }

func (b descBuffer) Text() string     { return b.m.descInput.Value() }
func (b descBuffer) SingleLine() bool { return false }

func (b descBuffer) Cursor() int {
	lines := strings.Split(b.m.descInput.Value(), "\n")
	row := b.m.descInput.Line()
	pos := 0
	for i := 0; i < row && i < len(lines); i++ {
		pos += len([]rune(lines[i])) + 1
	}
	if row < len(lines) {
		li := b.m.descInput.LineInfo()
		pos += min(li.StartColumn+li.CharOffset, len([]rune(lines[row])))
	}
	return pos
}

func (b descBuffer) SetText(text string, cursor int) {
	b.m.descInput.SetValue(text)
	b.SetCursor(cursor)
}

func (b descBuffer) SetCursor(pos int) {
	text := []rune(b.m.descInput.Value())
	pos = clampInt(pos, 0, len(text))
	row, col := 0, 0
	for i := 0; i < pos; i++ {
		if text[i] == '\n' {
			row++
			col = 0
		} else {
			col++
		}
	}
	for b.m.descInput.Line() > row {
		before := b.m.descInput.Line()
		b.m.descInput.CursorUp()
		if b.m.descInput.Line() == before {
			break
		}
	}
	for b.m.descInput.Line() < row {
		before := b.m.descInput.Line()
		b.m.descInput.CursorDown()
		if b.m.descInput.Line() == before {
			break
		}
	}
	b.m.descInput.SetCursor(col)
}

// vimPreview builds a short status message like: yanked "some text…"
func vimPreview(verb, text string) string {
	preview := strings.ReplaceAll(text, "\n", "⏎")
	r := []rune(preview)
	if len(r) > 30 {
		preview = string(r[:30]) + "…"
	}
	return fmt.Sprintf("%s %q", verb, preview)
}
