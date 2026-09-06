package temporal

import (
	"fmt"
	"strings"
	"time"
)

// DeicticTerms lists the expressions a writer resolves at write time.
var DeicticTerms = []string{"today", "tomorrow", "this-week", "next-week", "this-month", "next-month", "this-quarter", "next-quarter", "this-year"}

// IsDeictic reports whether s is a deictic term.
func IsDeictic(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	for _, t := range DeicticTerms {
		if t == s {
			return true
		}
	}
	return false
}

// ResolveDeixis maps a deictic term to the granule it denotes at ctx.Now in
// the context timezone. The granule, never the term, is what gets stored.
func ResolveDeixis(term string, ctx Context) (Granule, error) {
	today := ctx.today()
	switch strings.ToLower(strings.TrimSpace(term)) {
	case "today":
		return Day(today), nil
	case "tomorrow":
		return Day(today.AddDate(0, 0, 1)), nil
	case "this-week":
		y, w := today.ISOWeek()
		return Granule{Kind: KindWeek, Year: y, Week: w}, nil
	case "next-week":
		y, w := today.AddDate(0, 0, 7).ISOWeek()
		return Granule{Kind: KindWeek, Year: y, Week: w}, nil
	case "this-month":
		return Granule{Kind: KindMonth, Year: today.Year(), Month: int(today.Month())}, nil
	case "next-month":
		n := civil(today.Year(), today.Month(), 1).AddDate(0, 1, 0)
		return Granule{Kind: KindMonth, Year: n.Year(), Month: int(n.Month())}, nil
	case "this-quarter":
		return Granule{Kind: KindQuarter, Year: today.Year(), Month: 33 + (int(today.Month())-1)/3}, nil
	case "next-quarter":
		q := (int(today.Month())-1)/3 + 1
		y := today.Year()
		if q == 4 {
			q, y = 0, y+1
		}
		return Granule{Kind: KindQuarter, Year: y, Month: 33 + q}, nil
	case "this-year":
		return Granule{Kind: KindYear, Year: today.Year()}, nil
	}
	return Granule{}, fmt.Errorf("unknown deictic term %q: admitted terms are %s", term, strings.Join(DeicticTerms, ", "))
}

// ParseCalendarOrDeixis accepts an EDTF expression or a deictic term.
func ParseCalendarOrDeixis(s string, ctx Context) (Calendar, error) {
	if IsDeictic(s) {
		g, err := ResolveDeixis(s, ctx)
		if err != nil {
			return Calendar{}, err
		}
		return Calendar{Start: &g, End: &g}, nil
	}
	return ParseCalendar(s)
}

// ParseTimestamp accepts RFC 3339 with or without fractional seconds or an
// offset and returns UTC at seconds precision.
func ParseTimestamp(s string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(s))
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid timestamp %q: must be RFC 3339", s)
	}
	return t.UTC().Truncate(time.Second), nil
}

// TimeLayout is the on-disk timestamp form: RFC 3339, UTC, seconds.
const TimeLayout = "2006-01-02T15:04:05Z"

// FormatTimestamp renders t in the on-disk form.
func FormatTimestamp(t time.Time) string { return t.UTC().Format(TimeLayout) }

// ValidUntil is an availability's validity horizon as stored: an EDTF
// expression or a datetime.
type ValidUntil struct {
	Raw      string
	Calendar *Calendar
	Time     time.Time
}

// ParseValidUntil accepts either form.
func ParseValidUntil(s string) (ValidUntil, error) {
	s = strings.TrimSpace(s)
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return ValidUntil{Raw: s, Time: t}, nil
	}
	c, err := ParseCalendar(s)
	if err != nil {
		return ValidUntil{}, fmt.Errorf("invalid valid_until %q: must be an admitted EDTF expression or an RFC 3339 datetime", s)
	}
	if c.OpenEnd() {
		return ValidUntil{}, fmt.Errorf("invalid valid_until %q: an open upper bound is no horizon", s)
	}
	return ValidUntil{Raw: c.String(), Calendar: &c}, nil
}

// String renders the value as stored.
func (v ValidUntil) String() string { return v.Raw }

// IsZero reports whether no value is set.
func (v ValidUntil) IsZero() bool { return v.Raw == "" }

// Instant returns the instant at which validity ends in the context: the
// datetime itself, or the end of the calendar expression's bounds.
func (v ValidUntil) Instant(ctx Context) time.Time {
	if v.Calendar == nil {
		return v.Time
	}
	_, e := v.Calendar.civilBounds(ctx.Hemisphere)
	return ctx.Midnight(e)
}

// AddDuration advances t by a duration using calendar arithmetic for the date
// part and exact arithmetic for the time part.
func AddDuration(t time.Time, d Duration) time.Time {
	return t.AddDate(d.Years, d.Months, d.Weeks*7+d.Days).
		Add(time.Duration(d.Hours)*time.Hour + time.Duration(d.Minutes)*time.Minute + time.Duration(d.Seconds)*time.Second)
}
