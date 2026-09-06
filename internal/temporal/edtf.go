package temporal

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// GranuleKind is the precision of an EDTF granule.
type GranuleKind int

// Granule kinds in the admitted subset.
const (
	KindYear GranuleKind = iota + 1
	KindMonth
	KindWeek
	KindDay
	KindSeason  // EDTF 21..24
	KindQuarter // EDTF 33..36
)

func (k GranuleKind) String() string {
	switch k {
	case KindYear:
		return "year"
	case KindMonth:
		return "month"
	case KindWeek:
		return "week"
	case KindDay:
		return "day"
	case KindSeason:
		return "season"
	case KindQuarter:
		return "quarter"
	}
	return "unknown"
}

// Granule is one EDTF expression from the admitted subset: a year, month,
// ISO week, day, season code or quarter code.
type Granule struct {
	Kind  GranuleKind
	Year  int
	Month int // month, or season/quarter code (21..24, 33..36)
	Week  int
	Day   int
}

var (
	reYear  = regexp.MustCompile(`^(\d{4})$`)
	reMonth = regexp.MustCompile(`^(\d{4})-(\d{2})$`)
	reWeek  = regexp.MustCompile(`^(\d{4})-[Ww](\d{2})$`)
	reDay   = regexp.MustCompile(`^(\d{4})-(\d{2})-(\d{2})$`)
)

// ParseGranule parses a single admitted EDTF expression.
func ParseGranule(s string) (Granule, error) {
	s = strings.TrimSpace(s)
	if i := strings.IndexAny(s, "?~%"); i >= 0 {
		return Granule{}, fmt.Errorf("invalid calendar expression %q: EDTF qualifier %q is not admitted; a qualifier with no resolution semantics is decoration", s, string(s[i]))
	}
	switch {
	case reYear.MatchString(s):
		y, _ := strconv.Atoi(s)
		return Granule{Kind: KindYear, Year: y}, nil
	case reWeek.MatchString(s):
		m := reWeek.FindStringSubmatch(s)
		y, _ := strconv.Atoi(m[1])
		w, _ := strconv.Atoi(m[2])
		if w < 1 || w > isoWeeksInYear(y) {
			return Granule{}, fmt.Errorf("invalid calendar expression %q: week %d does not exist in %d", s, w, y)
		}
		return Granule{Kind: KindWeek, Year: y, Week: w}, nil
	case reMonth.MatchString(s):
		m := reMonth.FindStringSubmatch(s)
		y, _ := strconv.Atoi(m[1])
		mo, _ := strconv.Atoi(m[2])
		switch {
		case mo >= 1 && mo <= 12:
			return Granule{Kind: KindMonth, Year: y, Month: mo}, nil
		case mo >= 21 && mo <= 32:
			return Granule{Kind: KindSeason, Year: y, Month: mo}, nil
		case mo >= 33 && mo <= 36:
			return Granule{Kind: KindQuarter, Year: y, Month: mo}, nil
		}
		return Granule{}, fmt.Errorf("invalid calendar expression %q: %02d is not a month, a season code (21-24 neutral, 25-28 Northern, 29-32 Southern) or a quarter code (33-36)", s, mo)
	case reDay.MatchString(s):
		m := reDay.FindStringSubmatch(s)
		y, _ := strconv.Atoi(m[1])
		mo, _ := strconv.Atoi(m[2])
		d, _ := strconv.Atoi(m[3])
		if mo < 1 || mo > 12 || d < 1 || d > daysIn(y, time.Month(mo)) {
			return Granule{}, fmt.Errorf("invalid calendar expression %q: no such day", s)
		}
		return Granule{Kind: KindDay, Year: y, Month: mo, Day: d}, nil
	}
	return Granule{}, fmt.Errorf("invalid calendar expression %q: admitted forms are 2026, 2026-09, 2026-W36, 2026-09-04, a season 2026-21..24, a quarter 2026-33..36, an interval A/B, an open interval ../B or A/.. (two dots)", s)
}

// String renders the granule in shortest admitted form.
func (g Granule) String() string {
	switch g.Kind {
	case KindYear:
		return fmt.Sprintf("%04d", g.Year)
	case KindMonth, KindSeason, KindQuarter:
		return fmt.Sprintf("%04d-%02d", g.Year, g.Month)
	case KindWeek:
		return fmt.Sprintf("%04d-W%02d", g.Year, g.Week)
	case KindDay:
		return fmt.Sprintf("%04d-%02d-%02d", g.Year, g.Month, g.Day)
	}
	return ""
}

// Day returns a day granule.
func Day(t time.Time) Granule {
	return Granule{Kind: KindDay, Year: t.Year(), Month: int(t.Month()), Day: t.Day()}
}

// civilStart returns the first civil day of the granule; civilEnd the day
// after its last. Civil days are time.Time values at UTC midnight used purely
// as dates; they are turned into instants by Context.
func (g Granule) civilStart(hemi Hemisphere) time.Time {
	switch g.Kind {
	case KindYear:
		return civil(g.Year, 1, 1)
	case KindMonth:
		return civil(g.Year, time.Month(g.Month), 1)
	case KindDay:
		return civil(g.Year, time.Month(g.Month), g.Day)
	case KindWeek:
		return isoWeekMonday(g.Year, g.Week)
	case KindQuarter:
		return civil(g.Year, time.Month((g.Month-33)*3+1), 1)
	case KindSeason:
		y, m := seasonStart(g.Year, g.Month, hemi)
		return civil(y, m, 1)
	}
	return time.Time{}
}

func (g Granule) civilEnd(hemi Hemisphere) time.Time {
	switch g.Kind {
	case KindYear:
		return civil(g.Year+1, 1, 1)
	case KindMonth:
		return civil(g.Year, time.Month(g.Month), 1).AddDate(0, 1, 0)
	case KindDay:
		return civil(g.Year, time.Month(g.Month), g.Day).AddDate(0, 0, 1)
	case KindWeek:
		return isoWeekMonday(g.Year, g.Week).AddDate(0, 0, 7)
	case KindQuarter:
		return civil(g.Year, time.Month((g.Month-33)*3+1), 1).AddDate(0, 3, 0)
	case KindSeason:
		y, m := seasonStart(g.Year, g.Month, hemi)
		return civil(y, m, 1).AddDate(0, 3, 0)
	}
	return time.Time{}
}

// seasonStart maps an EDTF season code to the first month of the season.
// Neutral codes 21-24 resolve through the context hemisphere; 25-28 are
// Northern and 29-32 Southern regardless of it. In the north spring is March;
// in the south it is September. A season that begins in December starts in
// the granule's year and runs into the next.
func seasonStart(year, code int, hemi Hemisphere) (int, time.Month) {
	season := code
	switch {
	case code >= 25 && code <= 28:
		hemi, season = North, code-4
	case code >= 29 && code <= 32:
		hemi, season = South, code-8
	}
	var m time.Month
	if hemi == South {
		switch season {
		case 21:
			m = time.September
		case 22:
			m = time.December
		case 23:
			m = time.March
		case 24:
			m = time.June
		}
	} else {
		switch season {
		case 21:
			m = time.March
		case 22:
			m = time.June
		case 23:
			m = time.September
		case 24:
			m = time.December
		}
	}
	return year, m
}

// Calendar is a WINDOW's calendar anchor: a single granule or an interval,
// either side of which may be open.
type Calendar struct {
	Start    *Granule // nil when the lower bound is open ("../B")
	End      *Granule // nil when the upper bound is open ("A/..")
	Interval bool     // written with "/"
}

// ParseCalendar parses a calendar anchor from the admitted subset.
func ParseCalendar(s string) (Calendar, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return Calendar{}, fmt.Errorf("empty calendar expression")
	}
	if !strings.Contains(s, "/") {
		g, err := ParseGranule(s)
		if err != nil {
			return Calendar{}, err
		}
		return Calendar{Start: &g, End: &g}, nil
	}
	parts := strings.SplitN(s, "/", 2)
	if strings.Contains(parts[1], "/") {
		return Calendar{}, fmt.Errorf("invalid calendar expression %q: an interval has exactly two sides", s)
	}
	c := Calendar{Interval: true}
	if parts[0] != ".." {
		g, err := ParseGranule(parts[0])
		if err != nil {
			return Calendar{}, err
		}
		c.Start = &g
	}
	if parts[1] != ".." {
		g, err := ParseGranule(parts[1])
		if err != nil {
			return Calendar{}, err
		}
		c.End = &g
	}
	if c.Start == nil && c.End == nil {
		return Calendar{}, fmt.Errorf("invalid calendar expression %q: both sides open", s)
	}
	if c.Start != nil && c.End != nil && c.Start.civilStart(North).After(c.End.civilStart(North)) {
		return Calendar{}, fmt.Errorf("invalid calendar expression %q: %s is after %s", s, c.Start, c.End)
	}
	return c, nil
}

// Normalise returns the shortest admitted form: an interval whose sides are
// the same granule collapses to that granule.
func (c Calendar) Normalise() Calendar {
	if c.Interval && c.Start != nil && c.End != nil && *c.Start == *c.End {
		return Calendar{Start: c.Start, End: c.End}
	}
	return c
}

// String renders the anchor in normalised form.
func (c Calendar) String() string {
	c = c.Normalise()
	if !c.Interval {
		if c.Start != nil {
			return c.Start.String()
		}
		return ""
	}
	l, r := "..", ".."
	if c.Start != nil {
		l = c.Start.String()
	}
	if c.End != nil {
		r = c.End.String()
	}
	return l + "/" + r
}

// OpenStart and OpenEnd report open sides.
func (c Calendar) OpenStart() bool { return c.Start == nil }
func (c Calendar) OpenEnd() bool   { return c.End == nil }

// civilBounds returns the first civil day and the day after the last, with
// zero values on open sides. Weeks are ISO weeks in every context.
func (c Calendar) civilBounds(hemi Hemisphere) (start, end time.Time) {
	if c.Start != nil {
		start = c.Start.civilStart(hemi)
	}
	if c.End != nil {
		end = c.End.civilEnd(hemi)
	}
	return start, end
}

// --- civil date helpers ---------------------------------------------------

func civil(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func daysIn(y int, m time.Month) int {
	return civil(y, m, 1).AddDate(0, 1, -1).Day()
}

// isoWeekMonday returns the Monday of ISO week w of ISO year y.
func isoWeekMonday(y, w int) time.Time {
	jan4 := civil(y, 1, 4)
	wd := int(jan4.Weekday())
	if wd == 0 {
		wd = 7
	}
	week1Monday := jan4.AddDate(0, 0, 1-wd)
	return week1Monday.AddDate(0, 0, (w-1)*7)
}

func isoWeeksInYear(y int) int {
	_, w := civil(y, 12, 28).ISOWeek()
	return w
}
