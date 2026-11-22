package framecheck

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// Status says whether a frame must match.
type Status string

const (
	Enforced Status = "enforced"
	Pending  Status = "pending"
)

// Entry is one ledger line.
type Entry struct {
	Frame  string
	Status Status
}

// ParseLedger reads "<status> <frame>" lines; blank lines and # comments are skipped.
func ParseLedger(r io.Reader) ([]Entry, error) {
	var out []Entry
	sc := bufio.NewScanner(r)
	line := 0
	for sc.Scan() {
		line++
		text := strings.TrimSpace(sc.Text())
		if text == "" || strings.HasPrefix(text, "#") {
			continue
		}
		fields := strings.Fields(text)
		if len(fields) != 2 {
			return nil, fmt.Errorf("ledger line %d: want \"<status> <frame>\", got %q", line, text)
		}
		st := Status(fields[0])
		if st != Enforced && st != Pending {
			return nil, fmt.Errorf("ledger line %d: unknown status %q", line, fields[0])
		}
		out = append(out, Entry{Frame: fields[1], Status: st})
	}
	return out, sc.Err()
}
