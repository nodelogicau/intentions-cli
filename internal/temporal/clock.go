package temporal

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// TimeOfDay is a wall-clock time without a date or zone.
type TimeOfDay struct {
	Hour, Minute, Second int
}

func (t TimeOfDay) seconds() int { return t.Hour*3600 + t.Minute*60 + t.Second }

// String renders HH:MM, with :SS when seconds are set.
func (t TimeOfDay) String() string {
	if t.Second != 0 {
		return fmt.Sprintf("%02d:%02d:%02d", t.Hour, t.Minute, t.Second)
	}
	return fmt.Sprintf("%02d:%02d", t.Hour, t.Minute)
}

var reTOD = regexp.MustCompile(`^(\d{2}):(\d{2})(?::(\d{2}))?$`)

func parseTOD(s string) (TimeOfDay, error) {
	m := reTOD.FindStringSubmatch(s)
	if m == nil {
		return TimeOfDay{}, fmt.Errorf("invalid time of day %q: use HH:MM", s)
	}
	h, _ := strconv.Atoi(m[1])
	mi, _ := strconv.Atoi(m[2])
	sec := 0
	if m[3] != "" {
		sec, _ = strconv.Atoi(m[3])
	}
	if h > 24 || mi > 59 || sec > 59 || (h == 24 && (mi != 0 || sec != 0)) {
		return TimeOfDay{}, fmt.Errorf("invalid time of day %q", s)
	}
	return TimeOfDay{Hour: h, Minute: mi, Second: sec}, nil
}

// Clock is a WINDOW's clock anchor: a time-of-day interval, start inclusive
// and end exclusive, which may cross midnight.
type Clock struct {
	Start, End TimeOfDay
}

// ParseClock parses HH:MM/HH:MM with optional seconds.
func ParseClock(s string) (Clock, error) {
	s = strings.TrimSpace(s)
	parts := strings.Split(s, "/")
	if len(parts) != 2 {
		return Clock{}, fmt.Errorf("invalid clock interval %q: use HH:MM/HH:MM", s)
	}
	st, err := parseTOD(parts[0])
	if err != nil {
		return Clock{}, err
	}
	en, err := parseTOD(parts[1])
	if err != nil {
		return Clock{}, err
	}
	if st == en {
		return Clock{}, fmt.Errorf("invalid clock interval %q: start and end are equal", s)
	}
	if st.Hour == 24 {
		return Clock{}, fmt.Errorf("invalid clock interval %q: a start of 24:00 is the next day's 00:00", s)
	}
	return Clock{Start: st, End: en}, nil
}

// CrossesMidnight reports whether the end falls on the following local day.
func (c Clock) CrossesMidnight() bool { return c.End.seconds() <= c.Start.seconds() }

// String renders the interval in normalised form.
func (c Clock) String() string { return c.Start.String() + "/" + c.End.String() }
