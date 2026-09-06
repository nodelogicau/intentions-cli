package temporal

import (
	"strings"
	"testing"
	"time"
)

func melbourne(t *testing.T) Context {
	t.Helper()
	loc, err := time.LoadLocation("Australia/Melbourne")
	if err != nil {
		t.Fatal(err)
	}
	return Context{Location: loc, Hemisphere: North, Now: time.Date(2026, 9, 4, 9, 0, 0, 0, time.UTC)}
}

func TestDurationParseAndNormalise(t *testing.T) {
	cases := []struct{ in, str, norm string }{
		{"PT90M", "PT90M", "PT1H30M"},
		{"PT60M", "PT60M", "PT1H"},
		{"PT1H", "PT1H", "PT1H"},
		{"P1W", "P1W", "P7D"},
		{"P1D", "P1D", "P1D"},
		{"PT24H", "PT24H", "PT24H"},
		{"P1Y2M3DT4H5M6S", "P1Y2M3DT4H5M6S", "P1Y2M3DT4H5M6S"},
		{"P0D", "P0D", "P0D"},
		{"PT3600S", "PT3600S", "PT1H"},
	}
	for _, c := range cases {
		d, err := ParseDuration(c.in)
		if err != nil {
			t.Fatalf("%s: %v", c.in, err)
		}
		if d.String() != c.str {
			t.Errorf("%s: String = %s, want %s", c.in, d.String(), c.str)
		}
		if d.Normalise().String() != c.norm {
			t.Errorf("%s: Normalise = %s, want %s", c.in, d.Normalise().String(), c.norm)
		}
	}
	for _, bad := range []string{"90m", "P", "PT", "PT1.5H", "P1DT", "1H", "PT1H2"} {
		if _, err := ParseDuration(bad); err == nil {
			t.Errorf("%s: expected error", bad)
		}
	}
	if _, err := ParseDuration("PT1.5H"); err == nil || !strings.Contains(err.Error(), "fractional") {
		t.Errorf("fraction message: %v", err)
	}
}

func TestDurationSpec(t *testing.T) {
	s, err := ParseDurationSpec("PT1H:PT30M:PT2H")
	if err != nil || !s.Ranged() || s.Nominal.String() != "PT1H" || s.Min.String() != "PT30M" || s.Max.String() != "PT2H" {
		t.Fatalf("ranged: %+v %v", s, err)
	}
	if _, err := ParseDurationSpec("PT3H:PT30M:PT2H"); err == nil {
		t.Error("nominal outside range accepted")
	}
	if _, err := ParseDurationSpec("PT1H:PT30M"); err == nil {
		t.Error("two-part spec accepted")
	}
}

func TestGranuleParse(t *testing.T) {
	good := map[string]string{
		"2026": "2026", "2026-09": "2026-09", "2026-W36": "2026-W36", "2026-w36": "2026-W36",
		"2026-09-04": "2026-09-04", "2026-21": "2026-21", "2026-35": "2026-35",
	}
	for in, want := range good {
		g, err := ParseGranule(in)
		if err != nil {
			t.Fatalf("%s: %v", in, err)
		}
		if g.String() != want {
			t.Errorf("%s: %s want %s", in, g, want)
		}
	}
	for _, bad := range []string{"2026-09~", "2026-09?", "2026-13", "2026-20", "2026-W54", "2026-02-30", "sept", "2026-9"} {
		if _, err := ParseGranule(bad); err == nil {
			t.Errorf("%s: expected error", bad)
		}
	}
	_, err := ParseGranule("2026-09~")
	if err == nil || !strings.Contains(err.Error(), "qualifier") {
		t.Errorf("qualifier message: %v", err)
	}
}

func TestCalendarParse(t *testing.T) {
	cases := map[string]string{
		"2026-W36":              "2026-W36",
		"2026-W36/2026-W38":     "2026-W36/2026-W38",
		"../2026-09":            "../2026-09",
		"2026-09/..":            "2026-09/..",
		"2026-09-04/2026-09-04": "2026-09-04",
	}
	for in, want := range cases {
		c, err := ParseCalendar(in)
		if err != nil {
			t.Fatalf("%s: %v", in, err)
		}
		if c.String() != want {
			t.Errorf("%s: %s want %s", in, c, want)
		}
	}
	for _, bad := range []string{"2026-W38/2026-W36", "../..", "2026/2027/2028", ""} {
		if _, err := ParseCalendar(bad); err == nil {
			t.Errorf("%s: expected error", bad)
		}
	}
}

func TestClock(t *testing.T) {
	c, err := ParseClock("09:00/12:00")
	if err != nil || c.String() != "09:00/12:00" || c.CrossesMidnight() {
		t.Fatalf("%+v %v", c, err)
	}
	c, err = ParseClock("22:00/02:00")
	if err != nil || !c.CrossesMidnight() {
		t.Fatalf("midnight: %+v %v", c, err)
	}
	if c, err := ParseClock("09:00:00/12:00:00"); err != nil || c.String() != "09:00/12:00" {
		t.Errorf("seconds dropped: %v %v", c, err)
	}
	for _, bad := range []string{"09:00/09:00", "9:00/12:00", "09:00", "25:00/26:00"} {
		if _, err := ParseClock(bad); err == nil {
			t.Errorf("%s: expected error", bad)
		}
	}
}

func TestRelative(t *testing.T) {
	r, err := ParseRelative("int_A:FINISHTOSTART:P0D:P3D")
	if err != nil || r.Target != "int_A" || r.Relation != FinishToStart || r.Gap == nil || r.Gap.Max.String() != "P3D" {
		t.Fatalf("%+v %v", r, err)
	}
	if _, err := ParseRelative("int_A:DEPENDS-ON"); err == nil || !strings.Contains(err.Error(), "FINISHTOSTART") {
		t.Errorf("relation message: %v", err)
	}
	if _, err := ParseRelative("int_A:FINISHTOSTART:P3D:P1D"); err == nil {
		t.Error("gap ordering accepted")
	}
}

func TestCadence(t *testing.T) {
	if _, err := ParseCadence("FREQ=WEEKLY;BYDAY=TU"); err != nil {
		t.Fatal(err)
	}
	_, err := ParseCadence("FREQ=WEEKLY;BYDAY=TU;BYHOUR=9")
	if err == nil || !strings.Contains(err.Error(), "clock") {
		t.Errorf("BYHOUR: %v", err)
	}
	for _, bad := range []string{"WEEKLY", "FREQ=HOURLY", "FREQ=WEEKLY;RDATE=20260901", ""} {
		if _, err := ParseCadence(bad); err == nil {
			t.Errorf("%s: expected error", bad)
		}
	}
}

func TestPlacement(t *testing.T) {
	s, err := ParsePlacementStart("2026-09-15T10:00:00+10:00")
	if err != nil || s.AllDay || s.Raw != "2026-09-15T10:00:00+10:00" {
		t.Fatalf("%+v %v", s, err)
	}
	d, err := ParsePlacementStart("2026-09-21")
	if err != nil || !d.AllDay {
		t.Fatalf("%+v %v", d, err)
	}
	bad := Placement{Start: d, Duration: DurationSpec{Nominal: Duration{Hours: 4}}}
	if err := bad.Validate(); err == nil {
		t.Error("fractional all-day accepted")
	}
	good := Placement{Start: d, Duration: DurationSpec{Nominal: Duration{Days: 5}}}
	if err := good.Validate(); err != nil {
		t.Error(err)
	}
	ctx := melbourne(t)
	iv := good.Bounds(ctx)
	if iv.Start.Format(time.RFC3339) != "2026-09-21T00:00:00+10:00" || iv.End.Format(time.RFC3339) != "2026-09-26T00:00:00+10:00" {
		t.Errorf("all-day bounds: %s / %s", iv.Start.Format(time.RFC3339), iv.End.Format(time.RFC3339))
	}
}

func fmtIv(iv Interval) string {
	return iv.Start.Format(time.RFC3339) + " / " + iv.End.Format(time.RFC3339)
}

func TestWeekBounds(t *testing.T) {
	ctx := melbourne(t)
	c, _ := ParseCalendar("2026-W36")
	ivs, err := Bounds(Window{Calendar: &c}, ctx)
	if err != nil || len(ivs) != 1 {
		t.Fatalf("%v %v", ivs, err)
	}
	if got := fmtIv(ivs[0]); got != "2026-08-31T00:00:00+10:00 / 2026-09-07T00:00:00+10:00" {
		t.Errorf("monday week: %s", got)
	}
	// Timezone change moves bounds, stored window unchanged.
	london, _ := time.LoadLocation("Europe/London")
	ctx.Location = london
	ivs, _ = Bounds(Window{Calendar: &c}, ctx)
	if got := fmtIv(ivs[0]); got != "2026-08-31T00:00:00+01:00 / 2026-09-07T00:00:00+01:00" {
		t.Errorf("london week: %s", got)
	}
}

func TestSeasonQuarterBounds(t *testing.T) {
	ctx := melbourne(t)
	ctx.Hemisphere = South
	c, _ := ParseCalendar("2026-21")
	ivs, _ := Bounds(Window{Calendar: &c}, ctx)
	if got := fmtIv(ivs[0]); got != "2026-09-01T00:00:00+10:00 / 2026-12-01T00:00:00+11:00" {
		t.Errorf("southern spring: %s", got)
	}
	ctx.Hemisphere = North
	ivs, _ = Bounds(Window{Calendar: &c}, ctx)
	if got := fmtIv(ivs[0]); got != "2026-03-01T00:00:00+11:00 / 2026-06-01T00:00:00+10:00" {
		t.Errorf("northern spring: %s", got)
	}
	q, _ := ParseCalendar("2026-35")
	ivs, _ = Bounds(Window{Calendar: &q}, ctx)
	if got := fmtIv(ivs[0]); got != "2026-07-01T00:00:00+10:00 / 2026-10-01T00:00:00+10:00" {
		t.Errorf("quarter: %s", got)
	}
	w, _ := ParseCalendar("2026-24")
	ivs, _ = Bounds(Window{Calendar: &w}, ctx)
	if got := fmtIv(ivs[0]); got != "2026-12-01T00:00:00+11:00 / 2027-03-01T00:00:00+11:00" {
		t.Errorf("northern winter spans years: %s", got)
	}
	// Explicit codes ignore the context hemisphere.
	ctx.Hemisphere = South
	nw, _ := ParseCalendar("2026-28")
	ivs, _ = Bounds(Window{Calendar: &nw}, ctx)
	if got := fmtIv(ivs[0]); got != "2026-12-01T00:00:00+11:00 / 2027-03-01T00:00:00+11:00" {
		t.Errorf("explicit northern winter in the south: %s", got)
	}
	ctx.Hemisphere = North
	ss, _ := ParseCalendar("2026-29")
	ivs, _ = Bounds(Window{Calendar: &ss}, ctx)
	if got := fmtIv(ivs[0]); got != "2026-09-01T00:00:00+10:00 / 2026-12-01T00:00:00+11:00" {
		t.Errorf("explicit southern spring in the north: %s", got)
	}
	if _, err := ParseGranule("2026-40"); err == nil {
		t.Error("code 40 accepted")
	}
	if g, err := ParseGranule("2026-32"); err != nil || g.String() != "2026-32" {
		t.Errorf("code 32: %v %v", g, err)
	}
}

func TestOpenIntervalBounds(t *testing.T) {
	ctx := melbourne(t)
	c, _ := ParseCalendar("../2026-09")
	ivs, err := Bounds(Window{Calendar: &c}, ctx)
	if err != nil || !ivs[0].OpenStart || ivs[0].OpenEnd {
		t.Fatalf("%+v %v", ivs, err)
	}
	if ivs[0].End.Format(time.RFC3339) != "2026-10-01T00:00:00+10:00" {
		t.Errorf("deadline end: %s", ivs[0].End.Format(time.RFC3339))
	}
	// Horizon clamps the open side.
	h := Interval{Start: time.Date(2026, 9, 1, 0, 0, 0, 0, ctx.Location), End: time.Date(2027, 1, 1, 0, 0, 0, 0, ctx.Location)}
	ctx.Horizon = &h
	ivs, _ = Bounds(Window{Calendar: &c}, ctx)
	if ivs[0].OpenStart || ivs[0].Start.Format(time.RFC3339) != "2026-09-01T00:00:00+10:00" {
		t.Errorf("clamped: %+v", ivs[0])
	}
}

func TestClockBounds(t *testing.T) {
	ctx := melbourne(t)
	c, _ := ParseCalendar("2026-W37")
	k, _ := ParseClock("09:00/12:00")
	ivs, err := Bounds(Window{Calendar: &c, Clock: &k}, ctx)
	if err != nil || len(ivs) != 7 {
		t.Fatalf("%d intervals %v", len(ivs), err)
	}
	if got := fmtIv(ivs[0]); got != "2026-09-07T09:00:00+10:00 / 2026-09-07T12:00:00+10:00" {
		t.Errorf("first morning: %s", got)
	}
	d, _ := ParseCalendar("2026-09-18")
	m, _ := ParseClock("22:00/02:00")
	ivs, _ = Bounds(Window{Calendar: &d, Clock: &m}, ctx)
	if got := fmtIv(ivs[0]); got != "2026-09-18T22:00:00+10:00 / 2026-09-19T02:00:00+10:00" {
		t.Errorf("midnight crossing: %s", got)
	}
	// Whole day by default.
	ivs, _ = Bounds(Window{Calendar: &d}, ctx)
	if got := fmtIv(ivs[0]); got != "2026-09-18T00:00:00+10:00 / 2026-09-19T00:00:00+10:00" {
		t.Errorf("whole day: %s", got)
	}
	// Clock over an open bound without horizon is an error.
	o, _ := ParseCalendar("2026-09/..")
	if _, err := Bounds(Window{Calendar: &o, Clock: &k}, ctx); err == nil {
		t.Error("open clock accepted without horizon")
	}
}

func TestTransitions(t *testing.T) {
	ctx := melbourne(t)
	// Spring forward 2026-10-04: 02:00 -> 03:00.
	gap, _ := ParseCalendar("2026-10-04")
	k, _ := ParseClock("02:30/04:00")
	ivs, _ := Bounds(Window{Calendar: &gap, Clock: &k}, ctx)
	if got := fmtIv(ivs[0]); got != "2026-10-04T03:30:00+11:00 / 2026-10-04T04:00:00+11:00" {
		t.Errorf("gap: %s", got)
	}
	// Fall back 2026-04-05: 03:00 -> 02:00, 02:30 occurs twice; first wins.
	ovl, _ := ParseCalendar("2026-04-05")
	k2, _ := ParseClock("02:30/03:30")
	ivs, _ = Bounds(Window{Calendar: &ovl, Clock: &k2}, ctx)
	if got := ivs[0].Start.Format(time.RFC3339); got != "2026-04-05T02:30:00+11:00" {
		t.Errorf("overlap start: %s", got)
	}
	if !ivs[0].End.After(ivs[0].Start) {
		t.Errorf("overlap end precedes start: %s", fmtIv(ivs[0]))
	}
	// All-day placement on the transition day is one local day of 23 hours.
	s, _ := ParsePlacementStart("2026-10-04")
	p := Placement{Start: s, Duration: DurationSpec{Nominal: Duration{Days: 1}}}
	iv := p.Bounds(ctx)
	if got := fmtIv(iv); got != "2026-10-04T00:00:00+10:00 / 2026-10-05T00:00:00+11:00" {
		t.Errorf("all-day transition: %s", got)
	}
	if iv.End.Sub(iv.Start) != 23*time.Hour {
		t.Errorf("all-day transition length %v", iv.End.Sub(iv.Start))
	}
}

func TestExpand(t *testing.T) {
	ctx := melbourne(t)
	cad, _ := ParseCadence("FREQ=WEEKLY;BYDAY=TU")
	c, _ := ParseCalendar("2026-09/2026-10")
	gs, err := Expand(cad, Window{Calendar: &c}, ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(gs) != 9 || gs[0].String() != "2026-09-01" || gs[8].String() != "2026-10-27" {
		t.Errorf("tuesdays: %v", gs)
	}
	weekly, _ := ParseCadence("FREQ=WEEKLY")
	w, _ := ParseCalendar("2026-W37/2026-W38")
	gs, _ = Expand(weekly, Window{Calendar: &w}, ctx)
	if len(gs) != 2 || gs[0].String() != "2026-09-07" || gs[1].String() != "2026-09-14" {
		t.Errorf("weekly seed: %v", gs)
	}
	// Seed from the anchor: a Wednesday start makes every occurrence a Wednesday.
	wed, _ := ParseCalendar("2026-09-16/2026-12")
	gs, _ = Expand(weekly, Window{Calendar: &wed}, ctx)
	if len(gs) == 0 || gs[0].String() != "2026-09-16" {
		t.Errorf("anchor seed: %v", gs)
	}
	for _, g := range gs {
		if civil(g.Year, time.Month(g.Month), g.Day).Weekday() != time.Wednesday {
			t.Errorf("not a wednesday: %s", g)
		}
	}
	// Seed from the horizon: monthly on the tenth.
	monthlyPlain, _ := ParseCadence("FREQ=MONTHLY")
	openLow, _ := ParseCalendar("../2026-12")
	hz := Interval{Start: time.Date(2026, 9, 10, 0, 0, 0, 0, ctx.Location), OpenEnd: true}
	ctx.Horizon = &hz
	gs, err = Expand(monthlyPlain, Window{Calendar: &openLow}, ctx)
	if err != nil || len(gs) != 4 || gs[0].String() != "2026-09-10" || gs[3].String() != "2026-12-10" {
		t.Errorf("horizon seed monthly: %v %v", gs, err)
	}
	ctx.Horizon = nil
	open, _ := ParseCalendar("../2026-12")
	if _, err := Expand(cad, Window{Calendar: &open}, ctx); err == nil {
		t.Error("open lower bound without horizon accepted")
	}
	h := Interval{Start: time.Date(2026, 9, 1, 0, 0, 0, 0, ctx.Location), End: time.Date(2026, 9, 30, 0, 0, 0, 0, ctx.Location)}
	ctx.Horizon = &h
	gs, err = Expand(cad, Window{Calendar: &open}, ctx)
	if err != nil || len(gs) != 5 || gs[0].String() != "2026-09-01" {
		t.Errorf("horizon seeded: %v %v", gs, err)
	}
	// RFC 5545 examples at date level.
	monthly, _ := ParseCadence("FREQ=MONTHLY;BYDAY=-1FR")
	y, _ := ParseCalendar("2026-09/2026-11")
	ctx.Horizon = nil
	gs, _ = Expand(monthly, Window{Calendar: &y}, ctx)
	if len(gs) != 3 || gs[0].String() != "2026-09-25" || gs[2].String() != "2026-11-27" {
		t.Errorf("last friday: %v", gs)
	}
}

func TestDeixis(t *testing.T) {
	ctx := melbourne(t) // now = 2026-09-04T09:00Z = 19:00 Melbourne, Friday
	cases := map[string]string{
		"today": "2026-09-04", "tomorrow": "2026-09-05", "this-week": "2026-W36", "next-week": "2026-W37",
		"this-month": "2026-09", "next-month": "2026-10", "this-quarter": "2026-35", "next-quarter": "2026-36", "this-year": "2026",
	}
	for term, want := range cases {
		g, err := ResolveDeixis(term, ctx)
		if err != nil || g.String() != want {
			t.Errorf("%s: %s %v want %s", term, g, err, want)
		}
	}
	// Late UTC evening is the next day in Melbourne.
	ctx.Now = time.Date(2026, 9, 4, 15, 0, 0, 0, time.UTC)
	if g, _ := ResolveDeixis("today", ctx); g.String() != "2026-09-05" {
		t.Errorf("zone-aware today: %s", g)
	}
	ctx.Now = time.Date(2026, 12, 15, 0, 0, 0, 0, time.UTC)
	if g, _ := ResolveDeixis("next-quarter", ctx); g.String() != "2027-33" {
		t.Errorf("year roll: %s", g)
	}
	if _, err := ResolveDeixis("yesterday", ctx); err == nil {
		t.Error("unknown term accepted")
	}
}

func TestValidUntil(t *testing.T) {
	ctx := melbourne(t)
	v, err := ParseValidUntil("2026-12")
	if err != nil || v.String() != "2026-12" {
		t.Fatalf("%+v %v", v, err)
	}
	if got := v.Instant(ctx).Format(time.RFC3339); got != "2027-01-01T00:00:00+11:00" {
		t.Errorf("edtf instant: %s", got)
	}
	v, err = ParseValidUntil("2026-12-04T00:00:00Z")
	if err != nil || !v.Instant(ctx).Equal(time.Date(2026, 12, 4, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("datetime: %+v %v", v, err)
	}
	if _, err := ParseValidUntil("2026-09/.."); err == nil {
		t.Error("open end accepted")
	}
}

func TestIntervalArithmetic(t *testing.T) {
	at := func(h int) time.Time { return time.Date(2026, 9, 15, h, 0, 0, 0, time.UTC) }
	a := Interval{Start: at(9), End: at(12)}
	b := Interval{Start: at(11), End: at(14)}
	c := Interval{Start: at(12), End: at(13)}
	if !a.Overlaps(b) || a.Overlaps(c) || !a.Contains(Interval{Start: at(10), End: at(11)}) || a.Contains(b) {
		t.Error("overlap/contains")
	}
	if iv, ok := a.Intersect(b); !ok || iv.Start != at(11) || iv.End != at(12) {
		t.Errorf("intersect: %v %v", iv, ok)
	}
	m := Merge([]Interval{b, a, c})
	if len(m) != 1 || m[0].Start != at(9) || m[0].End != at(14) {
		t.Errorf("merge: %v", m)
	}
	x := IntersectSets([]Interval{a, {Start: at(15), End: at(17)}}, []Interval{{Start: at(10), End: at(16)}})
	if len(x) != 2 || x[0].End != at(12) || x[1].Start != at(15) {
		t.Errorf("intersect sets: %v", x)
	}
	open := Interval{OpenStart: true, End: at(12)}
	if !open.Overlaps(a) || !open.Contains(a) || open.Contains(b) {
		t.Error("open interval")
	}
}

func TestResolutionRangeAndOccasions(t *testing.T) {
	ctx := melbourne(t)
	ctx.Now = time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC)
	h := Duration{Weeks: 4}
	open, _ := ParseCalendar("../2026-12")
	r, reason := ResolutionRange(Window{Calendar: &open}, ctx, h)
	if reason != "" || !r.Start.Equal(ctx.Now) || r.End.Format("2006-01-02") != "2026-10-08" {
		t.Errorf("open deadline: %v %q", r, reason)
	}
	past, _ := ParseCalendar("2026-W30")
	if _, reason := ResolutionRange(Window{Calendar: &past}, ctx, h); reason == "" {
		t.Error("past window accepted")
	}
	wk, _ := ParseCalendar("2026-W38")
	r, _ = ResolutionRange(Window{Calendar: &wk}, ctx, h)
	if r.Start.Format(time.RFC3339) != "2026-09-14T00:00:00+10:00" || r.End.Format(time.RFC3339) != "2026-09-21T00:00:00+10:00" {
		t.Errorf("closed week: %s", r)
	}
	far, _ := ParseCalendar("2027-03")
	if _, reason := ResolutionRange(Window{Calendar: &far}, ctx, h); reason == "" {
		t.Error("window beyond the horizon accepted")
	}
	long, _ := ParseCalendar("2026-09/2027-03")
	r, _ = ResolutionRange(Window{Calendar: &long}, ctx, h)
	if r.End.Format("2006-01-02") != "2026-10-08" {
		t.Errorf("horizon caps a long closed window: %s", r)
	}
	// Half-past window: start clamps to now.
	cur, _ := ParseCalendar("2026-W37")
	r, _ = ResolutionRange(Window{Calendar: &cur}, ctx, h)
	if !r.Start.Equal(ctx.Now) {
		t.Errorf("clamped start: %s", r)
	}
	// No calendar: now for the horizon.
	k, _ := ParseClock("09:00/12:00")
	r, _ = ResolutionRange(Window{Clock: &k}, ctx, h)
	if !r.Start.Equal(ctx.Now) || r.End.Format("2006-01-02") != "2026-10-08" {
		t.Errorf("no calendar: %s", r)
	}
	// Occasions: recurring Tuesday mornings within a two-week range.
	cal, _ := ParseCalendar("2026-09/2026-12")
	cad, _ := ParseCadence("FREQ=WEEKLY;BYDAY=TU")
	rng := Interval{Start: time.Date(2026, 9, 10, 0, 0, 0, 0, ctx.Location), End: time.Date(2026, 9, 24, 0, 0, 0, 0, ctx.Location)}
	occ, err := Occasions(Window{Calendar: &cal, Clock: &k}, &cad, ctx, rng)
	if err != nil || len(occ) != 2 || occ[0].Start.Format(time.RFC3339) != "2026-09-15T09:00:00+10:00" || occ[1].Start.Format(time.RFC3339) != "2026-09-22T09:00:00+10:00" {
		t.Errorf("occasions: %v %v", occ, err)
	}
	// One-off with clock, clamped.
	occ, _ = Occasions(Window{Calendar: &wk, Clock: &k}, nil, ctx, Interval{Start: rng.Start, End: time.Date(2026, 9, 16, 0, 0, 0, 0, ctx.Location)})
	if len(occ) != 2 || occ[0].Start.Day() != 14 {
		t.Errorf("clamped occasions: %v", occ)
	}
	// Midnight-crossing occasion.
	late, _ := ParseClock("22:00/02:00")
	fri, _ := ParseCadence("FREQ=WEEKLY;BYDAY=FR")
	occ, _ = Occasions(Window{Calendar: &cal, Clock: &late}, &fri, ctx, rng)
	if len(occ) != 2 || occ[0].End.Format(time.RFC3339) != "2026-09-12T02:00:00+10:00" {
		t.Errorf("midnight occasion: %v", occ)
	}
}

func TestRelativeBounds(t *testing.T) {
	at := func(d, h int) time.Time { return time.Date(2026, 9, d, h, 0, 0, 0, time.UTC) }
	target := Interval{Start: at(15, 10), End: at(15, 11)}
	r, _ := ParseRelative("int_A:FINISHTOSTART:P0D:P3D")
	c := RelativeBounds(r, target)
	if c.OnEnd || !c.Min.Equal(at(15, 11)) || !c.HasMax || !c.Max.Equal(at(18, 11)) {
		t.Errorf("finishtostart: %+v", c)
	}
	if !c.Admits(Interval{Start: at(16, 9), End: at(16, 10)}) || c.Admits(Interval{Start: at(15, 10), End: at(15, 11)}) || c.Admits(Interval{Start: at(19, 9), End: at(19, 10)}) {
		t.Error("admits")
	}
	r, _ = ParseRelative("int_A:FINISHTOFINISH")
	c = RelativeBounds(r, target)
	if !c.OnEnd || c.HasMax || !c.Min.Equal(at(15, 11)) || !c.Admits(Interval{Start: at(20, 9), End: at(20, 10)}) {
		t.Errorf("finishtofinish: %+v", c)
	}
	r, _ = ParseRelative("int_A:STARTTOSTART:PT1H:PT2H")
	c = RelativeBounds(r, target)
	if c.OnEnd || !c.Min.Equal(at(15, 11)) || !c.Max.Equal(at(15, 12)) {
		t.Errorf("starttostart: %+v", c)
	}
	r, _ = ParseRelative("int_A:STARTTOFINISH:PT30M")
	c = RelativeBounds(r, target)
	if !c.OnEnd || !c.Min.Equal(at(15, 10).Add(30*time.Minute)) {
		t.Errorf("starttofinish: %+v", c)
	}
}
