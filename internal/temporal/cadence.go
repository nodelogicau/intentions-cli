package temporal

import (
	"fmt"
	"strings"

	"github.com/teambition/rrule-go"
)

// Cadence is an RFC 5545 RRULE restricted to its date-level parts. It
// expresses a generating pattern only.
type Cadence struct {
	Raw    string
	option rrule.ROption
}

// ParseCadence parses an RRULE, refusing sub-day parts with a message naming
// the clock anchor as the place for time of day.
func ParseCadence(s string) (Cadence, error) {
	raw := strings.TrimSpace(s)
	if raw == "" {
		return Cadence{}, fmt.Errorf("empty cadence")
	}
	up := strings.ToUpper(raw)
	for _, part := range strings.Split(up, ";") {
		key := strings.SplitN(part, "=", 2)[0]
		switch key {
		case "BYHOUR", "BYMINUTE", "BYSECOND":
			return Cadence{}, fmt.Errorf("invalid cadence %q: %s is not admitted; time of day belongs to the window's clock anchor", raw, key)
		case "DTSTART", "RDATE", "EXDATE", "RECURRENCE-ID":
			return Cadence{}, fmt.Errorf("invalid cadence %q: %s is not an RRULE part; a cadence is a generating pattern only", raw, key)
		}
	}
	opt, err := rrule.StrToROption(strings.TrimPrefix(up, "RRULE:"))
	if err != nil {
		return Cadence{}, fmt.Errorf("invalid cadence %q: %v", raw, err)
	}
	if !strings.HasPrefix(up, "FREQ=") && !strings.Contains(up, ";FREQ=") {
		return Cadence{}, fmt.Errorf("invalid cadence %q: FREQ is required", raw)
	}
	if opt.Freq > rrule.DAILY {
		return Cadence{}, fmt.Errorf("invalid cadence %q: sub-day frequencies are not admitted", raw)
	}
	return Cadence{Raw: raw, option: *opt}, nil
}

// String returns the cadence as written.
func (c Cadence) String() string { return c.Raw }
