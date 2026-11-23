// Package framecheck compares what the app draws with the frames in
// design.figcraft, cell by cell.
package framecheck

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"

	"github.com/charmbracelet/x/ansi"
)

const (
	cellW = 8.4
	cellH = 18.0
	eps   = 0.01
)

// Cell is what one terminal cell should show, with colours as Dusk token
// names. FG is "" on blank cells.
type Cell struct {
	Ch   rune
	FG   string
	BG   string
	Bold bool
}

// Grid is a W×H block of cells, row-major.
type Grid struct {
	W, H  int
	Cells []Cell
}

func newGrid(w, h int, bg string) *Grid {
	g := &Grid{W: w, H: h, Cells: make([]Cell, w*h)}
	for i := range g.Cells {
		g.Cells[i] = Cell{Ch: ' ', BG: bg}
	}
	return g
}

// At returns the cell at col x, row y.
func (g *Grid) At(x, y int) Cell {
	if x < 0 || y < 0 || x >= g.W || y >= g.H {
		return Cell{}
	}
	return g.Cells[y*g.W+x]
}

func (g *Grid) set(x, y int, c Cell) {
	if x < 0 || y < 0 || x >= g.W || y >= g.H {
		return
	}
	g.Cells[y*g.W+x] = c
}

// Rows returns each row's glyphs as text.
func (g *Grid) Rows() []string {
	rows := make([]string, g.H)
	for y := range rows {
		var b strings.Builder
		for x := 0; x < g.W; x++ {
			b.WriteRune(g.At(x, y).Ch)
		}
		rows[y] = b.String()
	}
	return rows
}

type paint struct {
	Type    string                       `json:"type"`
	Visible *bool                        `json:"visible"`
	Color   struct{ R, G, B, A float64 } `json:"color"`
}

func (p paint) shown() bool { return p.Type == "solid" && (p.Visible == nil || *p.Visible) }

type node struct {
	ID         int               `json:"id"`
	Name       string            `json:"name"`
	Type       string            `json:"type"`
	Parent     int               `json:"parent"`
	Transform  []float64         `json:"transform"`
	Width      float64           `json:"width"`
	Height     float64           `json:"height"`
	Visible    *bool             `json:"visible"`
	Clip       bool              `json:"clip"`
	Fills      []paint           `json:"fills"`
	Strokes    []paint           `json:"strokes"`
	Bound      map[string]string `json:"bound"`
	Characters string            `json:"characters"`
	Style      struct {
		StyleID string `json:"style_id"`
	} `json:"style"`
	Children []int             `json:"children"`
	VarModes map[string]string `json:"var_modes"`
}

func (n *node) shown() bool { return n.Visible == nil || *n.Visible }

func (n *node) offset() (float64, float64) {
	if len(n.Transform) == 6 {
		return n.Transform[4], n.Transform[5]
	}
	return 0, 0
}

func (n *node) fill() (paint, bool) {
	for _, f := range n.Fills {
		if f.shown() {
			return f, true
		}
	}
	return paint{}, false
}

func (n *node) stroke() (paint, bool) {
	for _, s := range n.Strokes {
		if s.shown() {
			return s, true
		}
	}
	return paint{}, false
}

// Design is a loaded design.figcraft.
type Design struct {
	nodes     map[int]*node
	pages     []int
	varNames  map[string]string // variable id → token name
	modeNames map[string]string // mode id → mode name
	styles    map[string]string // style id → style name
	defMode   string
}

type document struct {
	Pages     []int                       `json:"pages"`
	Nodes     map[string]json.RawMessage  `json:"nodes"`
	Styles    []struct{ ID, Name string } `json:"styles"`
	Variables struct {
		Collections []struct {
			Name  string
			Modes []struct{ ID, Name string }
		}
		Variables []struct{ ID, Name string }
	} `json:"variables"`
}

// Load reads design.figcraft (a zip holding document.json).
func Load(path string) (*Design, error) {
	zr, err := zip.OpenReader(path)
	if err != nil {
		return nil, err
	}
	defer zr.Close()
	for _, f := range zr.File {
		if f.Name != "document.json" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		defer rc.Close()
		data, err := io.ReadAll(rc)
		if err != nil {
			return nil, err
		}
		return parseDocument(data)
	}
	return nil, fmt.Errorf("%s: no document.json", path)
}

func parseDocument(data []byte) (*Design, error) {
	var doc document
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	d := &Design{
		nodes:     make(map[int]*node, len(doc.Nodes)),
		pages:     doc.Pages,
		varNames:  map[string]string{},
		modeNames: map[string]string{},
		styles:    map[string]string{},
	}
	for key, raw := range doc.Nodes {
		var n node
		if err := json.Unmarshal(raw, &n); err != nil {
			return nil, fmt.Errorf("node %s: %w", key, err)
		}
		id, err := strconv.Atoi(key)
		if err != nil {
			return nil, fmt.Errorf("node key %q: %w", key, err)
		}
		d.nodes[id] = &n
	}
	for _, s := range doc.Styles {
		d.styles[s.ID] = s.Name
	}
	for _, c := range doc.Variables.Collections {
		if c.Name != "Dusk" {
			continue
		}
		for i, m := range c.Modes {
			d.modeNames[m.ID] = m.Name
			if i == 0 {
				d.defMode = m.Name
			}
		}
	}
	for _, v := range doc.Variables.Variables {
		d.varNames[v.ID] = v.Name
	}
	return d, nil
}

// ScreenFrames lists every frame named "<ticket>/<state>" on pages 01–18,
// except 02/breakpoint-rule, in page order.
func (d *Design) ScreenFrames() []string {
	var out []string
	for _, pid := range d.pages {
		page := d.nodes[pid]
		if page == nil || len(page.Name) < 3 {
			continue
		}
		num, err := strconv.Atoi(page.Name[:2])
		if err != nil || num < 1 || num > 18 {
			continue
		}
		for _, cid := range page.Children {
			c := d.nodes[cid]
			if c == nil || c.Type != "frame" || !strings.HasPrefix(c.Name, page.Name[:2]+"/") {
				continue
			}
			if c.Name == "02/breakpoint-rule" {
				continue
			}
			out = append(out, c.Name)
		}
	}
	return out
}

// Find resolves a slash-separated path of node names from a page child, e.g.
// "Components/App header/Sync=live". A segment may end in "#n" to pick the
// n-th (0-based) child with that name.
func (d *Design) Find(path string) (int, error) {
	segs := strings.Split(path, "/")
	var cands []int
	for _, pid := range d.pages {
		cands = append(cands, d.nodes[pid].Children...)
	}
	// Screen frames are named "01/default", so try the longest page-level match first.
	for take := len(segs); take >= 1; take-- {
		name, idx := splitIndex(strings.Join(segs[:take], "/"))
		id, ok := d.pick(cands, name, idx)
		if !ok {
			continue
		}
		cur := id
		found := true
		for _, seg := range segs[take:] {
			name, idx := splitIndex(seg)
			next, ok := d.pick(d.nodes[cur].Children, name, idx)
			if !ok {
				found = false
				break
			}
			cur = next
		}
		if found {
			return cur, nil
		}
	}
	return 0, fmt.Errorf("no node at %q", path)
}

func splitIndex(seg string) (string, int) {
	if i := strings.LastIndex(seg, "#"); i > 0 {
		if n, err := strconv.Atoi(seg[i+1:]); err == nil {
			return seg[:i], n
		}
	}
	return seg, 0
}

func (d *Design) pick(ids []int, name string, idx int) (int, bool) {
	seen := 0
	for _, id := range ids {
		if n := d.nodes[id]; n != nil && n.Name == name {
			if seen == idx {
				return id, true
			}
			seen++
		}
	}
	return 0, false
}

// Expected is a frame or component turned into cells.
type Expected struct {
	Grid *Grid
	Mode string
	// Problems lists nodes the cell rules could not place (off the grid,
	// unbound colours, unknown text styles).
	Problems []string
}

// Expect turns the node at path into the cells it asks for.
func (d *Design) Expect(path string) (*Expected, error) {
	id, err := d.Find(path)
	if err != nil {
		return nil, err
	}
	root := d.nodes[id]
	w, okW := whole(root.Width / cellW)
	h, okH := whole(root.Height / cellH)
	e := &Expected{Mode: d.modeOf(id)}
	if !okW || !okH {
		e.Problems = append(e.Problems, fmt.Sprintf("%s is %.2f×%.2f cells, not whole", path, root.Width/cellW, root.Height/cellH))
	}
	e.Grid = newGrid(w, h, d.behind(id, e))
	r := &renderer{d: d, e: e}
	r.paint(id, 0, 0, true, cellArea{right: w, bottom: h})
	return e, nil
}

// modeOf returns the Dusk mode in effect at a node.
func (d *Design) modeOf(id int) string {
	for cur := id; cur != 0; {
		n := d.nodes[cur]
		if n == nil {
			break
		}
		for _, m := range n.VarModes {
			if name, ok := d.modeNames[m]; ok {
				return name
			}
		}
		cur = n.Parent
	}
	return d.defMode
}

// behind returns the token of the nearest ancestor fill.
func (d *Design) behind(id int, e *Expected) string {
	for cur := d.nodes[id].Parent; cur != 0; {
		n := d.nodes[cur]
		if n == nil || n.Type == "page" {
			break
		}
		if f, ok := n.fill(); ok {
			return d.token(n, "fill", f, e)
		}
		cur = n.Parent
	}
	return ""
}

func (d *Design) token(n *node, kind string, p paint, e *Expected) string {
	if v, ok := n.Bound[kind]; ok {
		if name, ok := d.varNames[v]; ok {
			return name
		}
	}
	raw := fmt.Sprintf("raw:#%02X%02X%02X", int(math.Round(p.Color.R*255)), int(math.Round(p.Color.G*255)), int(math.Round(p.Color.B*255)))
	e.Problems = append(e.Problems, fmt.Sprintf("node %d %q: %s not bound to a Dusk variable (%s)", n.ID, n.Name, kind, raw))
	return raw
}

func whole(v float64) (int, bool) {
	r := math.Round(v)
	return int(r), math.Abs(v-r) < eps
}

type renderer struct {
	d *Design
	e *Expected
}

type cellArea struct{ left, top, right, bottom int }

func (r *renderer) problem(n *node, format string, args ...any) {
	r.e.Problems = append(r.e.Problems, fmt.Sprintf("node %d %q: ", n.ID, n.Name)+fmt.Sprintf(format, args...))
}

// paint draws node id whose parent sits at (px, py) pixels from the root.
func (r *renderer) paint(id int, px, py float64, isRoot bool, clip cellArea) {
	n := r.d.nodes[id]
	if n == nil || !n.shown() {
		return
	}
	x, y := px, py
	if !isRoot {
		ox, oy := n.offset()
		x, y = px+ox, py+oy
	}
	cx, cy := x/cellW, y/cellH
	cw, ch := n.Width/cellW, n.Height/cellH

	switch n.Type {
	case "text":
		r.text(n, cx, cy, cw, clip)
	default:
		if f, ok := n.fill(); ok && (!isRoot || (n.Type != "frame" && r.panel(n, f))) {
			r.fill(n, f, cx, cy, cw, ch)
		} else if ok && isRoot {
			// The root's own fill is the background of every cell.
			tok := r.d.token(n, "fill", f, r.e)
			if tok != "scrim" {
				for i := range r.e.Grid.Cells {
					r.e.Grid.Cells[i] = Cell{Ch: ' ', BG: tok}
				}
			}
		}
		if s, ok := n.stroke(); ok && n.Type != "componentSet" {
			r.box(n, s, cx, cy, cw, ch)
		}
	}
	if n.Clip {
		left, okL := whole(cx)
		top, okT := whole(cy)
		right, okR := whole(cx + cw)
		bottom, okB := whole(cy + ch)
		if okL && okT && okR && okB {
			clip = cellArea{max(clip.left, left), max(clip.top, top), min(clip.right, right), min(clip.bottom, bottom)}
		}
	}
	for _, c := range n.Children {
		r.paint(c, x, y, false, clip)
	}
}

func (r *renderer) text(n *node, cx, cy, cw float64, clip cellArea) {
	x, okX := whole(cx)
	y, okY := whole(cy)
	if !okX || !okY {
		r.problem(n, "text at %.2f,%.2f is off the cell grid", cx, cy)
		return
	}
	bold := false
	switch r.d.styles[n.Style.StyleID] {
	case "Dusk/Title":
		bold = true
	case "Dusk/Body", "Dusk/Hint":
	default:
		r.problem(n, "text has no Dusk text style")
	}
	fg := ""
	if f, ok := n.fill(); ok {
		fg = r.d.token(n, "fill", f, r.e)
	}
	col := x
	text := n.Characters
	if w, ok := whole(cw); ok && w > 0 {
		text = ansi.Wrap(text, w, "")
	}
	for _, ch := range text {
		if ch == '\n' {
			y++
			col = x
			continue
		}
		if col < clip.left || col >= clip.right || y < clip.top || y >= clip.bottom {
			col++
			continue
		}
		bg := r.e.Grid.At(col, y).BG
		c := Cell{Ch: ch, FG: fg, BG: bg, Bold: bold}
		if ch == ' ' {
			c = Cell{Ch: ' ', BG: bg}
		}
		r.e.Grid.set(col, y, c)
		col++
	}
}

// fill applies a filled frame or rectangle, including the shape → glyph rules.
// panel is a frame filled with a surface colour (Dialog, Menu); cards draw
// their surface as a rectangle, so only panels are frames.
func (r *renderer) panel(n *node, f paint) bool {
	if n.Type == "rectangle" {
		return false
	}
	tok := r.d.token(n, "fill", f, r.e)
	return tok == "surface" || tok == "surface-selected"
}

func (r *renderer) fill(n *node, f paint, cx, cy, cw, ch float64) {
	tok := r.d.token(n, "fill", f, r.e)
	x, okX := whole(cx)
	y, okY := whole(cy)
	w, okW := whole(cw)
	h, okH := whole(ch)

	// A panel (Dialog, Menu): the fill stops half a cell inside its
	// outer ring, whose half blocks in the fill colour draw the edge.
	if r.panel(n, f) && okX && okY && okW && okH && w >= 2 && h >= 2 {
		for row := y; row < y+h; row++ {
			for col := x; col < x+w; col++ {
				var ch rune
				switch {
				case row == y && col == x:
					ch = '▗'
				case row == y && col == x+w-1:
					ch = '▖'
				case row == y+h-1 && col == x:
					ch = '▝'
				case row == y+h-1 && col == x+w-1:
					ch = '▘'
				case row == y:
					ch = '▄'
				case row == y+h-1:
					ch = '▀'
				case col == x:
					ch = '▐'
				case col == x+w-1:
					ch = '▌'
				default:
					r.e.Grid.set(col, row, Cell{Ch: ' ', BG: tok})
					continue
				}
				r.e.Grid.set(col, row, Cell{Ch: ch, FG: tok, BG: r.e.Grid.At(col, row).BG})
			}
		}
		return
	}
	// The FigCraft overlay stands for a repaint, not a terminal fill.
	if tok == "scrim" && okX && okY && okW && okH {
		bg := r.d.behind(n.ID, r.e)
		for row := y; row < y+h; row++ {
			for col := x; col < x+w; col++ {
				c := r.e.Grid.At(col, row)
				isFill := func(t string) bool { return strings.HasPrefix(t, "surface") }
				if isFill(c.BG) {
					c.BG = "surface"
				} else {
					c.BG = bg
				}
				if strings.ContainsRune("▄▀▗▖▝▘▐▌", c.Ch) {
					// Fill edges follow their fill; bars fade like text.
					if isFill(c.FG) {
						c.FG = "surface"
						r.e.Grid.set(col, row, c)
						continue
					}
				}
				if c.Ch != ' ' && c.Ch != 0 {
					c.FG = "text-faint"
				}
				r.e.Grid.set(col, row, c)
			}
		}
		return
	}
	// Card bar in the right half of its own column: ▐ on every row, ▗ and ▝
	// in a regular card's half-row insets.
	if math.Abs(cw-0.5) < eps && math.Abs(cx-math.Floor(cx)-0.5) < eps && okH {
		bx := int(math.Floor(cx))
		top, bottom := int(math.Floor(cy)), int(math.Floor(cy))+h-1
		half := math.Abs(cy-math.Floor(cy)-0.5) < eps
		if half {
			bottom = top + h
		}
		for row := top; row <= bottom; row++ {
			ch := '▐'
			if half && row == top {
				ch = '▗'
			} else if half && row == bottom {
				ch = '▝'
			}
			r.e.Grid.set(bx, row, Cell{Ch: ch, FG: tok, BG: r.e.Grid.At(bx, row).BG})
		}
		return
	}
	// Accent bar spanning a regular card's half-row insets: ▖, ▌…, ▘.
	if okX && okH && math.Abs(cw-0.5) < eps && math.Abs(cy-math.Floor(cy)-0.5) < eps && h >= 2 {
		top := int(math.Floor(cy))
		bottom := top + h
		for row := top; row <= bottom; row++ {
			ch := '▌'
			switch row {
			case top:
				ch = '▖'
			case bottom:
				ch = '▘'
			}
			r.e.Grid.set(x, row, Cell{Ch: ch, FG: tok, BG: r.e.Grid.At(x, row).BG})
		}
		return
	}
	// Accent bar: half a cell wide at col 0 → ▌ on every row it spans.
	if okX && okY && okH && math.Abs(cw-0.5) < eps {
		for row := y; row < y+h; row++ {
			bg := r.e.Grid.At(x, row).BG
			r.e.Grid.set(x, row, Cell{Ch: '▌', FG: tok, BG: bg})
		}
		return
	}
	// Regular card surface: inset half a row top and bottom.
	if okX && okW && okH && math.Abs(cy-math.Floor(cy)-0.5) < eps && w >= 2 {
		top := int(math.Floor(cy))
		bottom := top + h
		for col := x; col < x+w; col++ {
			tg, bgl := '▄', '▀'
			r.e.Grid.set(col, top, Cell{Ch: tg, FG: tok, BG: r.e.Grid.At(col, top).BG})
			r.e.Grid.set(col, bottom, Cell{Ch: bgl, FG: tok, BG: r.e.Grid.At(col, bottom).BG})
		}
		for row := top + 1; row < bottom; row++ {
			for col := x; col < x+w; col++ {
				r.e.Grid.set(col, row, Cell{Ch: ' ', BG: tok})
			}
		}
		return
	}
	if !okX || !okY || !okW || !okH {
		r.problem(n, "fill at %.2f,%.2f size %.2f×%.2f is off the cell grid", cx, cy, cw, ch)
		return
	}
	if n.Type == "rectangle" && w == 1 && h == 1 && tok == "text" && strings.HasSuffix(n.Name, "caret") {
		c := r.e.Grid.At(x, y)
		if c.Ch != ' ' && c.Ch != 0 {
			c.FG = c.BG
		}
		c.BG = tok
		r.e.Grid.set(x, y, c)
		return
	}
	for row := y; row < y+h; row++ {
		for col := x; col < x+w; col++ {
			r.e.Grid.set(col, row, Cell{Ch: ' ', BG: tok})
		}
	}
}

// box draws a frame stroke as a rounded box on the outer ring of cells.
func (r *renderer) box(n *node, s paint, cx, cy, cw, ch float64) {
	x, okX := whole(cx)
	y, okY := whole(cy)
	w, okW := whole(cw)
	h, okH := whole(ch)
	if !okX || !okY || !okW || !okH || w < 2 || h < 2 {
		r.problem(n, "stroke box at %.2f,%.2f size %.2f×%.2f is off the cell grid", cx, cy, cw, ch)
		return
	}
	tok := r.d.token(n, "stroke", s, r.e)
	put := func(col, row int, g rune) {
		r.e.Grid.set(col, row, Cell{Ch: g, FG: tok, BG: r.e.Grid.At(col, row).BG})
	}
	for col := x + 1; col < x+w-1; col++ {
		put(col, y, '─')
		put(col, y+h-1, '─')
	}
	for row := y + 1; row < y+h-1; row++ {
		put(x, row, '│')
		put(x+w-1, row, '│')
	}
	put(x, y, '╭')
	put(x+w-1, y, '╮')
	put(x, y+h-1, '╰')
	put(x+w-1, y+h-1, '╯')
}
