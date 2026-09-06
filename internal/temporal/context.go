package temporal

import (
	"fmt"
	"strings"
	"time"
)

// Hemisphere selects the month mapping for EDTF season codes.
type Hemisphere string

// Hemispheres.
const (
	North Hemisphere = "north"
	South Hemisphere = "south"
)

// ParseHemisphere validates a hemisphere name; empty means north.
func ParseHemisphere(s string) (Hemisphere, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "north":
		return North, nil
	case "south":
		return South, nil
	}
	return "", fmt.Errorf("invalid hemisphere %q: must be north or south", s)
}

// ParseWeekday parses a lowercase weekday name such as monday.
func ParseWeekday(s string) (time.Weekday, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "monday":
		return time.Monday, nil
	case "tuesday":
		return time.Tuesday, nil
	case "wednesday":
		return time.Wednesday, nil
	case "thursday":
		return time.Thursday, nil
	case "friday":
		return time.Friday, nil
	case "saturday":
		return time.Saturday, nil
	case "sunday":
		return time.Sunday, nil
	}
	return 0, fmt.Errorf("invalid week start %q: use a weekday name such as monday", s)
}

// WeekdayName renders a weekday as the configuration expects it.
func WeekdayName(d time.Weekday) string { return strings.ToLower(d.String()) }

// Interval is a half-open span of absolute instants. An open side carries a
// zero time and the matching flag; callers clamp to their own horizon.
type Interval struct {
	Start, End         time.Time
	OpenStart, OpenEnd bool
}

// Context is the resolver context: everything needed to turn a stored
// expression into instants.
type Context struct {
	Location   *time.Location
	WeekStart  time.Weekday
	Hemisphere Hemisphere
	Now        time.Time
	// Horizon, when set, clamps open interval sides and seeds cadence
	// expansion over an open lower bound.
	Horizon *Interval
}

// DefaultContext is UTC, Monday, north, now.
func DefaultContext() Context {
	return Context{Location: time.UTC, WeekStart: time.Monday, Hemisphere: North, Now: time.Now()}
}

func (c Context) loc() *time.Location {
	if c.Location == nil {
		return time.UTC
	}
	return c.Location
}

// Midnight returns the instant at which the given civil day begins in the
// context timezone. Where midnight does not exist (a transition at 00:00) the
// RFC 5545 rule applies through LocalTime.
func (c Context) Midnight(day time.Time) time.Time {
	return c.LocalTime(day, TimeOfDay{})
}

// LocalTime resolves a wall-clock time on a civil day to an instant per
// RFC 5545 §3.3.5: a time inside a forward-transition gap is interpreted with
// the UTC offset in force before the transition; an ambiguous time takes its
// first occurrence.
func (c Context) LocalTime(day time.Time, tod TimeOfDay) time.Time {
	loc := c.loc()
	naive := time.Date(day.Year(), day.Month(), day.Day(), tod.Hour, tod.Minute, tod.Second, 0, time.UTC)
	// Offsets in force well before and well after the wall time: at most one
	// transition lies within the window, so these are the only candidates.
	before := naive.Add(-36 * time.Hour)
	after := naive.Add(36 * time.Hour)
	_, offBefore := before.In(loc).Zone()
	_, offAfter := after.In(loc).Zone()
	offsets := []int{offBefore}
	if offAfter != offBefore {
		offsets = append(offsets, offAfter)
	}
	var matches []time.Time
	for _, off := range offsets {
		cand := naive.Add(-time.Duration(off) * time.Second)
		l := cand.In(loc)
		if l.Year() == naive.Year() && l.Month() == naive.Month() && l.Day() == naive.Day() &&
			l.Hour() == naive.Hour() && l.Minute() == naive.Minute() && l.Second() == naive.Second() {
			matches = append(matches, cand)
		}
	}
	switch len(matches) {
	case 0:
		// Gap: the offset before the transition.
		return naive.Add(-time.Duration(offBefore) * time.Second).In(loc)
	case 1:
		return matches[0].In(loc)
	}
	first := matches[0]
	for _, m := range matches[1:] {
		if m.Before(first) {
			first = m
		}
	}
	return first.In(loc)
}

// today returns the civil day of Now in the context timezone.
func (c Context) today() time.Time {
	n := c.Now
	if n.IsZero() {
		n = time.Now()
	}
	l := n.In(c.loc())
	return civil(l.Year(), l.Month(), l.Day())
}
