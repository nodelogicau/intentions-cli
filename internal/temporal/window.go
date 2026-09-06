package temporal

import (
	"fmt"
	"strings"
	"time"
)

// Window is the bounds within which an intention should happen or an
// availability holds, stored as the person expressed it: one or more anchors.
type Window struct {
	Calendar *Calendar
	Relative *Relative
	Clock    *Clock
}

// IsZero reports whether no anchor is present.
func (w Window) IsZero() bool { return w.Calendar == nil && w.Relative == nil && w.Clock == nil }

// Validate applies the one-anchor rule and each anchor's own checks.
func (w Window) Validate() error {
	if w.IsZero() {
		return fmt.Errorf("a window needs at least one of calendar, relative, clock")
	}
	if w.Relative != nil {
		if err := w.Relative.Validate(); err != nil {
			return err
		}
	}
	return nil
}

// Bounds computes the absolute intervals a window admits in the context. The
// relational anchor contributes nothing here: it needs its target's placement,
// which a resolver supplies. With a clock anchor the result is one interval
// per admitted local day; otherwise it is one interval spanning the calendar
// anchor. Open sides are clamped to ctx.Horizon when present, else marked.
func Bounds(w Window, ctx Context) ([]Interval, error) {
	var start, end time.Time
	openStart, openEnd := true, true
	if w.Calendar != nil {
		s, e := w.Calendar.civilBounds(ctx.Hemisphere, ctx.WeekStart)
		if !s.IsZero() {
			start, openStart = s, false
		}
		if !e.IsZero() {
			end, openEnd = e, false
		}
	}
	if ctx.Horizon != nil {
		if openStart && !ctx.Horizon.OpenStart {
			l := ctx.Horizon.Start.In(ctx.loc())
			start, openStart = civil(l.Year(), l.Month(), l.Day()), false
		}
		if openEnd && !ctx.Horizon.OpenEnd {
			l := ctx.Horizon.End.In(ctx.loc())
			end = civil(l.Year(), l.Month(), l.Day())
			if !ctx.Midnight(end).Equal(ctx.Horizon.End) {
				end = end.AddDate(0, 0, 1)
			}
			openEnd = false
		}
	}
	if w.Clock == nil {
		iv := Interval{OpenStart: openStart, OpenEnd: openEnd}
		if !openStart {
			iv.Start = ctx.Midnight(start)
		}
		if !openEnd {
			iv.End = ctx.Midnight(end)
		}
		return []Interval{iv}, nil
	}
	if openStart || openEnd {
		return nil, fmt.Errorf("a clock anchor over an open calendar bound needs a horizon to enumerate days")
	}
	var out []Interval
	for d := start; d.Before(end); d = d.AddDate(0, 0, 1) {
		s := ctx.LocalTime(d, w.Clock.Start)
		endDay := d
		if w.Clock.CrossesMidnight() {
			endDay = d.AddDate(0, 0, 1)
		}
		e := ctx.LocalTime(endDay, w.Clock.End)
		out = append(out, Interval{Start: s, End: e})
	}
	return out, nil
}

// String renders the anchors for comparison and display.
func (w Window) String() string {
	var parts []string
	if w.Calendar != nil {
		parts = append(parts, "calendar="+w.Calendar.String())
	}
	if w.Relative != nil {
		s := "relative=" + w.Relative.Target + ":" + string(w.Relative.Relation)
		if w.Relative.Gap != nil {
			if w.Relative.Gap.Min != nil {
				s += ":" + w.Relative.Gap.Min.String()
			} else {
				s += ":"
			}
			if w.Relative.Gap.Max != nil {
				s += ":" + w.Relative.Gap.Max.String()
			}
		}
		parts = append(parts, s)
	}
	if w.Clock != nil {
		parts = append(parts, "clock="+w.Clock.String())
	}
	return strings.Join(parts, " ")
}
