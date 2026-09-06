// Package temporal is the pure temporal engine of the Intentions Format:
// durations, the admitted EDTF calendar subset, clock anchors, cadence, and
// the computation of a window's clock-time bounds in a resolver context. It
// does no I/O and knows nothing about objects.
package temporal

import (
	"fmt"
	"strconv"
	"strings"
)

// Duration is an ISO 8601 duration with integer components. Fractions are not
// admitted. Weeks may be combined with other components on read (ISO forbids
// it, but a reader should not be the one to reject it); Normalise folds them.
type Duration struct {
	Years, Months, Weeks, Days int
	Hours, Minutes, Seconds    int
}

// ParseDuration parses an ISO 8601 duration such as PT90M, P2D, P1W, P1Y2M3DT4H.
func ParseDuration(s string) (Duration, error) {
	var d Duration
	orig := s
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "P") {
		return d, fmt.Errorf("invalid duration %q: must be an ISO 8601 duration such as PT90M", orig)
	}
	s = s[1:]
	if s == "" {
		return d, fmt.Errorf("invalid duration %q: no components", orig)
	}
	inTime := false
	num := ""
	seen := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= '0' && c <= '9':
			num += string(c)
		case c == 'T':
			if inTime || num != "" {
				return d, fmt.Errorf("invalid duration %q", orig)
			}
			inTime = true
		case c == '.' || c == ',':
			return d, fmt.Errorf("invalid duration %q: fractional components are not admitted", orig)
		default:
			if num == "" {
				return d, fmt.Errorf("invalid duration %q: designator %c without a number", orig, c)
			}
			n, err := strconv.Atoi(num)
			if err != nil {
				return d, fmt.Errorf("invalid duration %q", orig)
			}
			num = ""
			seen++
			switch {
			case !inTime && c == 'Y':
				d.Years = n
			case !inTime && c == 'M':
				d.Months = n
			case !inTime && c == 'W':
				d.Weeks = n
			case !inTime && c == 'D':
				d.Days = n
			case inTime && c == 'H':
				d.Hours = n
			case inTime && c == 'M':
				d.Minutes = n
			case inTime && c == 'S':
				d.Seconds = n
			default:
				return d, fmt.Errorf("invalid duration %q: unexpected designator %c", orig, c)
			}
		}
	}
	if num != "" {
		return d, fmt.Errorf("invalid duration %q: trailing number without a designator", orig)
	}
	if seen == 0 {
		return d, fmt.Errorf("invalid duration %q: no components", orig)
	}
	if inTime && !d.HasTimePart() && !strings.HasSuffix(s, "0S") && !strings.HasSuffix(s, "0M") && !strings.HasSuffix(s, "0H") {
		return d, fmt.Errorf("invalid duration %q: T with no time component", orig)
	}
	return d, nil
}

// String renders the duration in ISO 8601 form, omitting zero components.
// A zero duration renders as P0D, as the spec's own example writes it.
func (d Duration) String() string {
	var b strings.Builder
	b.WriteString("P")
	if d.Years != 0 {
		fmt.Fprintf(&b, "%dY", d.Years)
	}
	if d.Months != 0 {
		fmt.Fprintf(&b, "%dM", d.Months)
	}
	if d.Weeks != 0 {
		fmt.Fprintf(&b, "%dW", d.Weeks)
	}
	if d.Days != 0 {
		fmt.Fprintf(&b, "%dD", d.Days)
	}
	if d.Hours != 0 || d.Minutes != 0 || d.Seconds != 0 {
		b.WriteString("T")
		if d.Hours != 0 {
			fmt.Fprintf(&b, "%dH", d.Hours)
		}
		if d.Minutes != 0 {
			fmt.Fprintf(&b, "%dM", d.Minutes)
		}
		if d.Seconds != 0 {
			fmt.Fprintf(&b, "%dS", d.Seconds)
		}
	}
	if b.Len() == 1 {
		return "P0D"
	}
	return b.String()
}

// Normalise returns the projection form: zero components dropped, weeks folded
// into days, and the time part re-expressed from total seconds as hours,
// minutes and seconds. Date components are never converted into time ones,
// so P1D and PT24H remain distinct.
func (d Duration) Normalise() Duration {
	n := Duration{Years: d.Years, Months: d.Months, Days: d.Days + 7*d.Weeks}
	secs := d.Hours*3600 + d.Minutes*60 + d.Seconds
	n.Hours = secs / 3600
	n.Minutes = (secs % 3600) / 60
	n.Seconds = secs % 60
	return n
}

// IsZero reports whether every component is zero.
func (d Duration) IsZero() bool {
	return d == Duration{}
}

// HasTimePart reports whether any of hours, minutes or seconds is set.
func (d Duration) HasTimePart() bool {
	return d.Hours != 0 || d.Minutes != 0 || d.Seconds != 0
}

// ApproxSeconds is a nominal length for comparison only: a year is 365 days,
// a month 30, a week 7, a day 24 hours. It is never used to place anything.
func (d Duration) ApproxSeconds() int64 {
	days := int64(d.Years)*365 + int64(d.Months)*30 + int64(d.Weeks)*7 + int64(d.Days)
	return days*86400 + int64(d.Hours)*3600 + int64(d.Minutes)*60 + int64(d.Seconds)
}

// DurationSpec is a DURATION value: a nominal duration, optionally ranged.
type DurationSpec struct {
	Nominal Duration
	Min     *Duration
	Max     *Duration
}

// Ranged reports whether the spec carries a range.
func (s DurationSpec) Ranged() bool { return s.Min != nil || s.Max != nil }

// Validate checks that a ranged nominal lies within [min, max].
func (s DurationSpec) Validate() error {
	if s.Min != nil && s.Nominal.ApproxSeconds() < s.Min.ApproxSeconds() {
		return fmt.Errorf("duration nominal %s is shorter than min %s", s.Nominal, *s.Min)
	}
	if s.Max != nil && s.Nominal.ApproxSeconds() > s.Max.ApproxSeconds() {
		return fmt.Errorf("duration nominal %s is longer than max %s", s.Nominal, *s.Max)
	}
	if s.Min != nil && s.Max != nil && s.Min.ApproxSeconds() > s.Max.ApproxSeconds() {
		return fmt.Errorf("duration min %s is longer than max %s", *s.Min, *s.Max)
	}
	return nil
}

// Normalise returns the projection form of every component.
func (s DurationSpec) Normalise() DurationSpec {
	n := DurationSpec{Nominal: s.Nominal.Normalise()}
	if s.Min != nil {
		m := s.Min.Normalise()
		n.Min = &m
	}
	if s.Max != nil {
		m := s.Max.Normalise()
		n.Max = &m
	}
	return n
}

// ParseDurationSpec parses the command-line form: a plain duration, or
// nominal:min:max.
func ParseDurationSpec(s string) (DurationSpec, error) {
	parts := strings.Split(strings.TrimSpace(s), ":")
	var spec DurationSpec
	var err error
	switch len(parts) {
	case 1:
		spec.Nominal, err = ParseDuration(parts[0])
		if err != nil {
			return spec, err
		}
	case 3:
		spec.Nominal, err = ParseDuration(parts[0])
		if err != nil {
			return spec, err
		}
		mn, err := ParseDuration(parts[1])
		if err != nil {
			return spec, err
		}
		mx, err := ParseDuration(parts[2])
		if err != nil {
			return spec, err
		}
		spec.Min, spec.Max = &mn, &mx
	default:
		return spec, fmt.Errorf("invalid duration %q: use PT90M or nominal:min:max such as PT1H:PT30M:PT2H", s)
	}
	return spec, spec.Validate()
}

// String renders a plain duration as ISO 8601 and a ranged one as
// nominal:min:max.
func (s DurationSpec) String() string {
	if !s.Ranged() {
		return s.Nominal.String()
	}
	mn, mx := "", ""
	if s.Min != nil {
		mn = s.Min.String()
	}
	if s.Max != nil {
		mx = s.Max.String()
	}
	return s.Nominal.String() + ":" + mn + ":" + mx
}
