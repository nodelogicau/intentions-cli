package temporal

import (
	"fmt"
	"sort"
	"time"
)

// Interval arithmetic over closed intervals. Open sides are resolved by
// ResolutionRange before any of this runs; these helpers treat an open side
// as unbounded only where it says so.

// Overlaps reports whether the two half-open intervals share any instant.
func (iv Interval) Overlaps(o Interval) bool {
	if iv.Empty() || o.Empty() {
		return false
	}
	startsBefore := iv.OpenStart || o.OpenEnd || iv.Start.Before(o.End)
	endsAfter := iv.OpenEnd || o.OpenStart || iv.End.After(o.Start)
	return startsBefore && endsAfter
}

// Contains reports whether o lies wholly within iv.
func (iv Interval) Contains(o Interval) bool {
	if iv.Empty() || o.Empty() || o.OpenStart || o.OpenEnd {
		return false
	}
	return (iv.OpenStart || !iv.Start.After(o.Start)) && (iv.OpenEnd || !iv.End.Before(o.End))
}

// Intersect returns the common part of two intervals.
func (iv Interval) Intersect(o Interval) (Interval, bool) {
	if !iv.Overlaps(o) {
		return Interval{}, false
	}
	out := Interval{OpenStart: iv.OpenStart && o.OpenStart, OpenEnd: iv.OpenEnd && o.OpenEnd}
	switch {
	case iv.OpenStart:
		out.Start = o.Start
	case o.OpenStart:
		out.Start = iv.Start
	default:
		out.Start = later(iv.Start, o.Start)
	}
	switch {
	case iv.OpenEnd:
		out.End = o.End
	case o.OpenEnd:
		out.End = iv.End
	default:
		out.End = earlier(iv.End, o.End)
	}
	return out, !out.Empty()
}

// Empty reports whether the interval holds no instant.
func (iv Interval) Empty() bool {
	if iv.OpenStart || iv.OpenEnd {
		return false
	}
	return !iv.End.After(iv.Start)
}

// Length is the interval's duration; zero when a side is open.
func (iv Interval) Length() time.Duration {
	if iv.OpenStart || iv.OpenEnd {
		return 0
	}
	return iv.End.Sub(iv.Start)
}

// String renders the interval for messages.
func (iv Interval) String() string {
	s, e := "..", ".."
	if !iv.OpenStart {
		s = iv.Start.Format(time.RFC3339)
	}
	if !iv.OpenEnd {
		e = iv.End.Format(time.RFC3339)
	}
	return s + "/" + e
}

func later(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}

func earlier(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}

// Merge sorts closed intervals by start and coalesces those that touch or
// overlap.
func Merge(ivs []Interval) []Interval {
	var in []Interval
	for _, iv := range ivs {
		if !iv.Empty() && !iv.OpenStart && !iv.OpenEnd {
			in = append(in, iv)
		}
	}
	sort.Slice(in, func(i, j int) bool { return in[i].Start.Before(in[j].Start) })
	var out []Interval
	for _, iv := range in {
		if n := len(out); n > 0 && !out[n-1].End.Before(iv.Start) {
			if iv.End.After(out[n-1].End) {
				out[n-1].End = iv.End
			}
			continue
		}
		out = append(out, iv)
	}
	return out
}

// IntersectSets returns the intervals common to both sets.
func IntersectSets(a, b []Interval) []Interval {
	var out []Interval
	for _, x := range Merge(a) {
		for _, y := range Merge(b) {
			if iv, ok := x.Intersect(y); ok {
				out = append(out, iv)
			}
		}
	}
	return Merge(out)
}

// Clamp restricts every interval to r.
func Clamp(ivs []Interval, r Interval) []Interval {
	var out []Interval
	for _, iv := range ivs {
		if c, ok := iv.Intersect(r); ok {
			out = append(out, c)
		}
	}
	return out
}

// ResolutionRange is the closed range an intention is resolved within. It
// starts at the later of the window's start and now, and ends at the earlier
// of the window's end and now plus horizon; an open side contributes nothing,
// and a window with no calendar anchor runs from now for the horizon. reason
// is set when the range is empty: the window has passed, or it starts beyond
// the horizon.
func ResolutionRange(w Window, ctx Context, horizon Duration) (Interval, string) {
	now := ctx.Now
	if now.IsZero() {
		now = time.Now()
	}
	limit := AddDuration(now, horizon)
	r := Interval{Start: now, End: limit}
	if w.Calendar != nil {
		ivs, err := Bounds(Window{Calendar: w.Calendar}, ctx)
		if err == nil && len(ivs) == 1 {
			b := ivs[0]
			if !b.OpenEnd && !b.End.After(now) {
				return Interval{}, fmt.Sprintf("the window %s ended at %s, before now", w.Calendar, b.End.Format(time.RFC3339))
			}
			if !b.OpenStart {
				r.Start = later(b.Start, now)
				if !b.Start.Before(limit) {
					return Interval{}, fmt.Sprintf("the window %s starts at %s, beyond the resolution horizon %s", w.Calendar, b.Start.Format(time.RFC3339), limit.Format(time.RFC3339))
				}
			}
			if !b.OpenEnd {
				r.End = earlier(b.End, limit)
			}
		}
	}
	if !r.End.After(r.Start) {
		return Interval{}, "the window admits no time after now"
	}
	return r, ""
}

// Occasions returns the intervals a window admits within r: one per cadence
// day when recurring, each clipped to the window's clock, else the plain
// bounds of the window with the clock applied per day. Everything is clamped
// to r.
func Occasions(w Window, cadence *Cadence, ctx Context, r Interval) ([]Interval, error) {
	ctx.Horizon = &r
	if cadence == nil {
		ivs, err := Bounds(w, ctx)
		if err != nil {
			return nil, err
		}
		return Clamp(ivs, r), nil
	}
	days, err := Expand(*cadence, w, ctx)
	if err != nil {
		return nil, err
	}
	var out []Interval
	for _, d := range days {
		day := d
		cal := Calendar{Start: &day, End: &day}
		ivs, err := Bounds(Window{Calendar: &cal, Clock: w.Clock}, ctx)
		if err != nil {
			return nil, err
		}
		out = append(out, Clamp(ivs, r)...)
	}
	return Merge(out), nil
}

// Constraint is what a relational anchor imposes on a candidate: a bound on
// its start or on its end.
type Constraint struct {
	OnEnd  bool // constrain the end instead of the start
	Min    time.Time
	Max    time.Time
	HasMax bool
}

// RelativeBounds turns a relational anchor and its target's placement
// interval into a constraint. An absent gap means a minimum of zero and no
// maximum.
func RelativeBounds(rel Relative, target Interval) Constraint {
	var mn, mx Duration
	hasMax := false
	if rel.Gap != nil {
		if rel.Gap.Min != nil {
			mn = *rel.Gap.Min
		}
		if rel.Gap.Max != nil {
			mx, hasMax = *rel.Gap.Max, true
		}
	}
	var base time.Time
	c := Constraint{HasMax: hasMax}
	switch rel.Relation {
	case FinishToStart:
		base = target.End
	case StartToStart:
		base = target.Start
	case FinishToFinish:
		base, c.OnEnd = target.End, true
	case StartToFinish:
		base, c.OnEnd = target.Start, true
	}
	c.Min = AddDuration(base, mn)
	if hasMax {
		c.Max = AddDuration(base, mx)
	}
	return c
}

// Admits reports whether a candidate interval satisfies the constraint.
func (c Constraint) Admits(cand Interval) bool {
	t := cand.Start
	if c.OnEnd {
		t = cand.End
	}
	if t.Before(c.Min) {
		return false
	}
	if c.HasMax && t.After(c.Max) {
		return false
	}
	return true
}
