package temporal

import (
	"fmt"
	"strings"
	"time"
)

// Placement is the collapse to clock time: a concrete start, a duration, and
// optionally one location URI.
type Placement struct {
	Start    PlacementStart
	Duration DurationSpec
	Location string
}

// PlacementStart is an RFC 3339 datetime with offset, or a calendar day for
// an all-day placement. Raw keeps the datetime exactly as written so the
// offset survives a rewrite.
type PlacementStart struct {
	Raw    string
	Time   time.Time // zero for all-day
	Day    Granule   // set for all-day
	AllDay bool
}

// ParsePlacementStart accepts either form.
func ParsePlacementStart(s string) (PlacementStart, error) {
	s = strings.TrimSpace(s)
	if reDay.MatchString(s) {
		g, err := ParseGranule(s)
		if err != nil {
			return PlacementStart{}, err
		}
		return PlacementStart{Raw: g.String(), Day: g, AllDay: true}, nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return PlacementStart{}, fmt.Errorf("invalid placement start %q: must be an RFC 3339 datetime with offset or a calendar day", s)
	}
	return PlacementStart{Raw: s, Time: t}, nil
}

// String returns the start as written.
func (p PlacementStart) String() string { return p.Raw }

// Validate applies the all-day rule.
func (p Placement) Validate() error {
	if p.Start.AllDay {
		d := p.Duration.Nominal
		if d.HasTimePart() {
			return fmt.Errorf("all-day placement starting %s has a fractional duration %s: it must be whole days", p.Start.Raw, d)
		}
		if p.Duration.Min != nil && p.Duration.Min.HasTimePart() || p.Duration.Max != nil && p.Duration.Max.HasTimePart() {
			return fmt.Errorf("all-day placement starting %s has a fractional duration bound", p.Start.Raw)
		}
	}
	return p.Duration.Validate()
}

// Bounds computes the placement's absolute interval. An all-day placement
// spans its local days in the context timezone.
func (p Placement) Bounds(ctx Context) Interval {
	d := p.Duration.Nominal
	if p.Start.AllDay {
		start := ctx.Midnight(p.Start.Day.civilStart(ctx.Hemisphere))
		days := d.Years*365 + d.Months*30 + d.Weeks*7 + d.Days
		if d.Years != 0 || d.Months != 0 {
			endCivil := p.Start.Day.civilStart(ctx.Hemisphere).AddDate(d.Years, d.Months, d.Weeks*7+d.Days)
			return Interval{Start: start, End: ctx.Midnight(endCivil)}
		}
		end := ctx.Midnight(p.Start.Day.civilStart(ctx.Hemisphere).AddDate(0, 0, days))
		return Interval{Start: start, End: end}
	}
	start := p.Start.Time
	end := start.AddDate(d.Years, d.Months, d.Weeks*7+d.Days).
		Add(time.Duration(d.Hours)*time.Hour + time.Duration(d.Minutes)*time.Minute + time.Duration(d.Seconds)*time.Second)
	return Interval{Start: start, End: end}
}
