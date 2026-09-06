package temporal

import (
	"fmt"
	"strings"
	"time"

	"github.com/teambition/rrule-go"
)

// Expand returns the calendar days a cadence generates within the window's
// calendar bounds, as day granules in order. Expansion is seeded from the
// first local day of the calendar anchor's lower bound, or from the start of
// ctx.Horizon when that bound is open. FREQ=WEEKLY without BYDAY takes the
// seed's weekday. WKST defaults to MO unless the rule states otherwise. The
// upper bound is the calendar anchor's end, clamped to the horizon when one
// is set.
func Expand(c Cadence, w Window, ctx Context) ([]Granule, error) {
	var start, end time.Time
	if w.Calendar != nil {
		start, end = w.Calendar.civilBounds(ctx.Hemisphere)
	}
	if ctx.Horizon != nil {
		if !ctx.Horizon.OpenStart {
			l := ctx.Horizon.Start.In(ctx.loc())
			h := civil(l.Year(), l.Month(), l.Day())
			if start.IsZero() || h.After(start) {
				start = h
			}
		}
		if !ctx.Horizon.OpenEnd {
			l := ctx.Horizon.End.In(ctx.loc())
			h := civil(l.Year(), l.Month(), l.Day())
			if !ctx.Midnight(h).Equal(ctx.Horizon.End) {
				h = h.AddDate(0, 0, 1)
			}
			if end.IsZero() || h.Before(end) {
				end = h
			}
		}
	}
	if start.IsZero() {
		return nil, fmt.Errorf("cadence expansion needs a calendar anchor with a lower bound or a horizon to seed from")
	}
	if end.IsZero() {
		return nil, fmt.Errorf("cadence expansion needs an upper bound or a horizon")
	}
	if !start.Before(end) {
		return nil, nil
	}
	opt := c.option
	opt.Dtstart = start
	if !strings.Contains(strings.ToUpper(c.Raw), "WKST=") {
		opt.Wkst = rrule.MO
	}
	r, err := rrule.NewRRule(opt)
	if err != nil {
		return nil, fmt.Errorf("invalid cadence %q: %v", c.Raw, err)
	}
	// Between is inclusive of its bounds; the window's end is exclusive.
	times := r.Between(start, end.Add(-time.Nanosecond), true)
	out := make([]Granule, 0, len(times))
	for _, t := range times {
		out = append(out, Day(t))
	}
	return out, nil
}
