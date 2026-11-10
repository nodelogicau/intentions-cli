package temporal

import (
	"fmt"
	"regexp"
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

// PlacementStart is an RFC 3339 datetime with offset, optionally followed by
// one RFC 9557 time-zone suffix naming an IANA zone, or a calendar day for
// an all-day placement. Raw keeps the start exactly as written, suffix and
// all, so the offset and the zone survive a rewrite.
type PlacementStart struct {
	Raw    string
	Time   time.Time // zero for all-day
	Zone   string    // the IANA name in the suffix, if any
	Day    Granule   // set for all-day
	AllDay bool
}

var (
	reZoneName   = regexp.MustCompile(`^[A-Za-z0-9._+-]+(/[A-Za-z0-9._+-]+)*$`)
	reZoneOffset = regexp.MustCompile(`^([+-]\d{2}(:?\d{2})?|[Zz])$`)
)

// splitSuffix separates a trailing sequence of bracketed groups from the
// datetime before it.
func splitSuffix(s string) (string, []string) {
	var groups []string
	for strings.HasSuffix(s, "]") {
		i := strings.LastIndex(s, "[")
		if i < 0 {
			break
		}
		groups = append([]string{s[i+1 : len(s)-1]}, groups...)
		s = s[:i]
	}
	return s, groups
}

// ParsePlacementStart accepts either form.
func ParsePlacementStart(s string) (PlacementStart, error) {
	s = strings.TrimSpace(s)
	base, groups := splitSuffix(s)
	if reDay.MatchString(base) {
		if len(groups) > 0 {
			return PlacementStart{}, fmt.Errorf("invalid placement start %q: a calendar day carries no time-zone suffix", s)
		}
		g, err := ParseGranule(base)
		if err != nil {
			return PlacementStart{}, err
		}
		return PlacementStart{Raw: g.String(), Day: g, AllDay: true}, nil
	}
	zone, err := zoneOf(s, groups)
	if err != nil {
		return PlacementStart{}, err
	}
	t, err := time.Parse(time.RFC3339, base)
	if err != nil {
		return PlacementStart{}, fmt.Errorf("invalid placement start %q: must be an RFC 3339 datetime with offset, optionally followed by one [Zone] suffix, or a calendar day", s)
	}
	return PlacementStart{Raw: s, Time: t, Zone: zone}, nil
}

// zoneOf checks the suffix groups: at most one, naming an IANA zone, with no
// critical flag, no tag and no offset.
func zoneOf(s string, groups []string) (string, error) {
	switch {
	case len(groups) == 0:
		return "", nil
	case len(groups) > 1:
		return "", fmt.Errorf("invalid placement start %q: only one time-zone suffix is admitted, found %d bracketed groups", s, len(groups))
	}
	g := groups[0]
	switch {
	case strings.HasPrefix(g, "!"):
		return "", fmt.Errorf("invalid placement start %q: the critical flag [!...] is not admitted", s)
	case strings.Contains(g, "="):
		return "", fmt.Errorf("invalid placement start %q: suffix tags such as [%s] are not admitted; only an IANA zone name", s, g)
	case reZoneOffset.MatchString(g):
		return "", fmt.Errorf("invalid placement start %q: an offset suffix [%s] is not admitted; the suffix names an IANA zone", s, g)
	case !reZoneName.MatchString(g):
		return "", fmt.Errorf("invalid placement start %q: [%s] is not an IANA time-zone name", s, g)
	}
	return g, nil
}

// HasZoneSuffix reports whether a datetime string ends in a bracketed suffix.
func HasZoneSuffix(s string) bool {
	_, groups := splitSuffix(strings.TrimSpace(s))
	return len(groups) > 0
}

// WithZone renders t in loc as an RFC 3339 placement start, suffixed with
// loc's name. A location with no IANA name (empty or Local) gives no
// suffix, and nor does UTC, whose start is already written with Z.
func WithZone(t time.Time, loc *time.Location) PlacementStart {
	s := t.In(loc).Format(time.RFC3339)
	if name := loc.String(); name != "" && name != "Local" && !IsUTCZone(name) {
		s += "[" + name + "]"
	}
	p, _ := ParsePlacementStart(s)
	return p
}

// LoadZone loads an IANA zone by name from this binary's tz database.
// "Local" names no zone and is refused.
func LoadZone(name string) (*time.Location, error) {
	if name == "" || name == "Local" {
		return nil, fmt.Errorf("%q is not an IANA time-zone name", name)
	}
	return time.LoadLocation(name)
}

// IsUTCZone reports whether a zone name means UTC, which no suffix records.
func IsUTCZone(name string) bool {
	switch name {
	case "UTC", "Etc/UTC", "Z", "z":
		return true
	}
	return false
}

// UTCStart reports whether the start is written with the Z designator,
// which RFC 9557 reads as an instant with no local offset.
func (p PlacementStart) UTCStart() bool {
	base, _ := splitSuffix(p.Raw)
	return strings.HasSuffix(base, "Z") || strings.HasSuffix(base, "z")
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

// SamePlacement reports whether two placements occupy the same time at the
// same place: the same start instant (or calendar day), the same duration,
// and the same location. The offset a start was written with is not part of
// the comparison.
func SamePlacement(a, b Placement) bool {
	if a.Start.AllDay != b.Start.AllDay {
		return false
	}
	if a.Start.AllDay {
		if a.Start.Raw != b.Start.Raw {
			return false
		}
	} else if !a.Start.Time.Equal(b.Start.Time) {
		return false
	}
	return a.Duration.String() == b.Duration.String() && a.Location == b.Location
}
