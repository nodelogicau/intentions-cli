package resolve

import (
	"strings"
	"testing"
	"time"

	"github.com/nodelogicau/intentions-cli/internal/model"
	"github.com/nodelogicau/intentions-cli/internal/store"
	"github.com/nodelogicau/intentions-cli/internal/temporal"
)

const (
	ada  = "https://example.com/people/ada"
	rob  = "https://example.com/people/rob"
	room = "https://example.com/rooms/3"
	home = "https://example.com/places/home"
	off  = "https://example.com/places/office"
)

var now = time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC) // Thursday 10 Sep, 10:00 Melbourne

type fixture struct {
	t  *testing.T
	ws *store.Workspace
	g  *store.Graph
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	cfg := store.NewConfig()
	cfg.Resolver.Timezone = "Australia/Melbourne"
	cfg.Defaults.Subject = ada
	cfg.Defaults.Source.Author = ada
	ws, _, err := store.Init(t.TempDir(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	g, _ := ws.Load()
	return &fixture{t: t, ws: ws, g: g}
}

func (f *fixture) env() Env {
	g, err := f.ws.Load()
	if err != nil {
		f.t.Fatal(err)
	}
	f.g = g
	return NewEnv(f.ws, g, now)
}

func (f *fixture) write(objs ...model.Object) {
	f.t.Helper()
	for _, o := range objs {
		if err := f.ws.WriteObject(o); err != nil {
			f.t.Fatal(err)
		}
	}
}

func dur(s string) *temporal.DurationSpec {
	d, err := temporal.ParseDurationSpec(s)
	if err != nil {
		panic(err)
	}
	return &d
}

func win(cal, clock string) *temporal.Window {
	w := &temporal.Window{}
	if cal != "" {
		c, err := temporal.ParseCalendar(cal)
		if err != nil {
			panic(err)
		}
		w.Calendar = &c
	}
	if clock != "" {
		k, err := temporal.ParseClock(clock)
		if err != nil {
			panic(err)
		}
		w.Clock = &k
	}
	return w
}

func cad(s string) *temporal.Cadence {
	c, err := temporal.ParseCadence(s)
	if err != nil {
		panic(err)
	}
	return &c
}

func intention(title string, d, cal, clock string) *model.Intention {
	o := &model.Intention{ID: model.MintID(model.TypeIntention), Subject: ada, Title: title, Stability: "tentative", Serves: []model.Ref{},
		Source: model.Source{Author: ada}, Timestamp: now, Acknowledgements: []model.Acknowledgement{}}
	if d != "" {
		o.Duration = dur(d)
	}
	if cal != "" || clock != "" {
		o.Window = win(cal, clock)
	}
	return o
}

func availability(subject, d, cal, clock, cadence string) *model.Availability {
	o := &model.Availability{ID: model.MintID(model.TypeAvailability), Subject: subject, Duration: dur(d), Window: win(cal, clock), Scope: "personal",
		Source: model.Source{Author: ada}, Timestamp: now}
	if cadence != "" {
		o.Cadence = cad(cadence)
	}
	return o
}

func placement(start, d string) *temporal.Placement {
	s, err := temporal.ParsePlacementStart(start)
	if err != nil {
		panic(err)
	}
	return &temporal.Placement{Start: s, Duration: *dur(d)}
}

// --- generation ------------------------------------------------------------

func TestGenerate(t *testing.T) {
	f := newFixture(t)
	rec := intention("Weekly 1:1", "PT30M", "2026-09/2026-12", "09:00/12:00")
	rec.Cadence = cad("FREQ=WEEKLY;BYDAY=TU")
	rec.Location = []string{home}
	rec.Activity = "meeting"
	rec.Parties = []string{rob}
	f.write(rec)
	e := f.env()
	r := temporal.Interval{Start: now, End: temporal.AddDuration(now, temporal.Duration{Weeks: 2})}
	created, skipped, err := Generate(e, GenerateOptions{Range: r, Source: model.Source{Author: ada, Harness: "claude"}, Timestamp: now})
	if err != nil || len(created) != 2 || len(skipped) != 0 {
		t.Fatalf("generate: %d created %d skipped %v", len(created), len(skipped), err)
	}
	inst := created[0]
	if inst.Occurrence != "2026-09-15" || inst.Window.Calendar.String() != "2026-09-15" || inst.Window.Clock.String() != "09:00/12:00" {
		t.Errorf("instance window: %s %v", inst.Occurrence, inst.Window)
	}
	if inst.Subject != ada || inst.Duration.String() != "PT30M" || inst.Activity != "meeting" || len(inst.Parties) != 1 || len(inst.Location) != 1 || inst.Stability != "tentative" {
		t.Errorf("instance copied fields: %+v", inst)
	}
	if inst.InstanceOf() != rec.ID || inst.Source.Harness != "claude" {
		t.Errorf("instance serves/source: %v %v", inst.Serves, inst.Source)
	}
	if created[1].Occurrence != "2026-09-22" {
		t.Errorf("second occurrence: %s", created[1].Occurrence)
	}
	// Idempotent.
	e = f.env()
	created, skipped, _ = Generate(e, GenerateOptions{Range: r, Source: model.Source{Author: ada}, Timestamp: now})
	if len(created) != 0 || len(skipped) != 2 || skipped[0].Reason != "instance exists" {
		t.Errorf("second run: %d created %v", len(created), skipped)
	}
	// Retired instance is not regenerated.
	inst.Retired = &model.Retired{Kind: "abandoned", Source: model.Source{Author: ada}, Timestamp: now}
	f.write(inst)
	e = f.env()
	created, skipped, _ = Generate(e, GenerateOptions{Range: r, Source: model.Source{Author: ada}, Timestamp: now})
	if len(created) != 0 || !strings.Contains(skipped[0].Reason, "retired") {
		t.Errorf("retired instance regenerated: %d %v", len(created), skipped)
	}
	// Retired recurring intention generates nothing; occurrences past the anchor are not generated.
	rec.Retired = &model.Retired{Kind: "abandoned", Source: model.Source{Author: ada}, Timestamp: now}
	f.write(rec)
	short := intention("Short run", "PT30M", "2026-09/2026-09-16", "")
	short.Cadence = cad("FREQ=WEEKLY;BYDAY=TU")
	f.write(short)
	e = f.env()
	created, _, _ = Generate(e, GenerateOptions{Range: r, Source: model.Source{Author: ada}, Timestamp: now})
	if len(created) != 1 || created[0].Occurrence != "2026-09-15" || created[0].InstanceOf() != short.ID {
		t.Errorf("anchor end and retired recurring: %v", created)
	}
}

// --- resolution ------------------------------------------------------------

func TestResolveSupplyAndClock(t *testing.T) {
	f := newFixture(t)
	f.write(availability(ada, "PT6H", "2026-W38", "10:00/16:00", ""))
	in := intention("Budget", "PT2H", "2026-W38", "08:00/12:00")
	f.write(in)
	res, err := Resolve(f.env(), in, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.All()) == 0 {
		t.Fatalf("no candidates: %s", res.Reason)
	}
	for _, c := range res.All() {
		s, e := c.Interval.Start.In(f.env().Ctx.Location), c.Interval.End.In(f.env().Ctx.Location)
		if s.Hour() < 10 || e.Hour() > 12 || (e.Hour() == 12 && e.Minute() > 0) {
			t.Errorf("candidate outside 10:00-12:00: %s", c.Start)
		}
	}
	// Seven days, 10:00-12:00, 2h on a 15m grid: one per day.
	if res.Considered != 7 {
		t.Errorf("considered %d", res.Considered)
	}
	// Disjoint clocks.
	evening := intention("Evening", "PT1H", "2026-W38", "18:00/21:00")
	f.write(evening)
	res, _ = Resolve(f.env(), evening, Options{})
	if len(res.All()) != 0 || !strings.Contains(res.Reason, "does not intersect") {
		t.Errorf("disjoint: %d %q", len(res.All()), res.Reason)
	}
	// Limit keeps the full count.
	res, _ = Resolve(f.env(), in, Options{Limit: 3})
	if len(res.Candidates) != 3 || res.Considered != 7 {
		t.Errorf("limit: %d of %d", len(res.Candidates), res.Considered)
	}
}

func TestResolveRangeReasons(t *testing.T) {
	f := newFixture(t)
	f.write(availability(ada, "PT8H", "2026-09/2027-06", "09:00/17:00", "FREQ=DAILY"))
	open := intention("Deadline", "PT1H", "../2026-12", "")
	past := intention("Past", "PT1H", "2026-W30", "")
	far := intention("Far", "PT1H", "2027-03", "")
	f.write(open, past, far)
	e := f.env()
	res, _ := Resolve(e, open, Options{})
	last := res.All()[len(res.All())-1].Interval.End
	if !last.Before(temporal.AddDuration(now, temporal.Duration{Weeks: 4}).Add(time.Minute)) || len(res.All()) == 0 {
		t.Errorf("open deadline range: last %s", last)
	}
	if r, _ := Resolve(e, past, Options{}); !strings.Contains(r.Reason, "before now") {
		t.Errorf("past: %q", r.Reason)
	}
	if r, _ := Resolve(e, far, Options{}); !strings.Contains(r.Reason, "beyond") {
		t.Errorf("far: %q", r.Reason)
	}
}

func TestResolveParties(t *testing.T) {
	f := newFixture(t)
	f.write(availability(ada, "PT8H", "2026-W38", "09:00/17:00", ""))
	f.write(availability(rob, "PT8H", "2026-W38", "13:00/18:00", ""))
	f.write(availability(room, "PT8H", "2026-09-15", "08:00/15:00", ""))
	in := intention("Meet", "PT1H", "2026-W38", "")
	in.Parties = []string{rob, room}
	f.write(in)
	res, _ := Resolve(f.env(), in, Options{})
	if len(res.All()) == 0 {
		t.Fatalf("no candidates: %s", res.Reason)
	}
	for _, c := range res.All() {
		s := c.Interval.Start.In(f.env().Ctx.Location)
		if s.Day() != 15 || s.Hour() < 13 || c.Interval.End.In(f.env().Ctx.Location).Hour() > 15 {
			t.Errorf("outside the three-way intersection: %s", c.Start)
		}
		if len(c.Supply) != 3 {
			t.Errorf("supply ids: %v", c.Supply)
		}
	}
	// Party held elsewhere.
	in2 := intention("Meet Zoe", "PT1H", "2026-W38", "")
	in2.Parties = []string{"https://example.com/people/zoe"}
	f.write(in2)
	res, _ = Resolve(f.env(), in2, Options{})
	if len(res.NoSupply) != 1 || !strings.Contains(res.Reason, "zoe") {
		t.Errorf("party elsewhere: %v %q", res.NoSupply, res.Reason)
	}
}

func TestResolveEligibility(t *testing.T) {
	f := newFixture(t)
	office := availability(ada, "PT8H", "2026-W38", "09:00/17:00", "")
	office.Location = []string{off}
	f.write(office)
	in := intention("At home", "PT1H", "2026-W38", "")
	in.Location = []string{home}
	f.write(in)
	res, _ := Resolve(f.env(), in, Options{})
	if len(res.All()) != 0 || len(res.Excluded) != 1 || !strings.Contains(res.Excluded[0].Reason, "location") {
		t.Errorf("location excludes: %v %v", res.All(), res.Excluded)
	}
	// Retired.
	office.Retired = &model.Retired{Kind: "retracted", Source: model.Source{Author: ada}, Timestamp: now}
	f.write(office)
	anywhere := intention("Anywhere", "PT1H", "2026-W38", "")
	f.write(anywhere)
	res, _ = Resolve(f.env(), anywhere, Options{})
	if len(res.All()) != 0 || !strings.Contains(res.Excluded[0].Reason, "retired") {
		t.Errorf("retired: %v", res.Excluded)
	}
	// Expired recurring: timestamp long ago, no valid_until.
	old := availability(ada, "PT8H", "2026-01/2027-01", "09:00/17:00", "FREQ=DAILY")
	old.Timestamp = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	f.write(old)
	res, _ = Resolve(f.env(), anywhere, Options{})
	if len(res.All()) != 0 {
		t.Errorf("expired supply used")
	}
	found := false
	for _, ex := range res.Excluded {
		if strings.Contains(ex.Reason, "expired") {
			found = true
		}
	}
	if !found {
		t.Errorf("expired not named: %v", res.Excluded)
	}
	// Conditional.
	deep := availability(ada, "PT3H", "2026-W38", "09:00/12:00", "")
	deep.Conditional = []string{"deep-work"}
	f.write(deep)
	meeting := intention("Meeting", "PT1H", "2026-W38", "")
	meeting.Activity = "meeting"
	f.write(meeting)
	if res, _ := Resolve(f.env(), meeting, Options{}); len(res.All()) != 0 {
		t.Error("conditional mismatch supplied")
	}
	anywhere.Activity = "deep-work"
	f.write(anywhere)
	if res, _ := Resolve(f.env(), anywhere, Options{}); len(res.All()) == 0 {
		t.Errorf("conditional match not supplied: %s", res.Reason)
	}
}

func TestResolveCapacityAndGrid(t *testing.T) {
	f := newFixture(t)
	f.write(availability(ada, "PT3H", "2026-09-15", "09:00/17:00", ""))
	three := intention("Three", "PT3H", "2026-09-15", "")
	four := intention("Four", "PT4H", "2026-09-15", "")
	f.write(three, four)
	e := f.env()
	res, _ := Resolve(e, three, Options{})
	if len(res.All()) == 0 || res.All()[0].Interval.Start.In(e.Ctx.Location).Hour() != 9 || res.All()[len(res.All())-1].Interval.End.In(e.Ctx.Location).Hour() != 17 {
		t.Errorf("three-hour anywhere in 09-17: %d candidates", len(res.All()))
	}
	if res, _ := Resolve(e, four, Options{}); len(res.All()) != 0 {
		t.Error("four hours on three-hour capacity")
	}
	// Capacity consumed by a placement.
	two := intention("Two", "PT2H", "2026-09-15", "")
	two.Placement = placement("2026-09-15T09:00:00+10:00", "PT2H")
	f.write(two)
	twoB := intention("Two more", "PT2H", "2026-09-15", "")
	f.write(twoB)
	if res, _ := Resolve(f.env(), twoB, Options{}); len(res.All()) != 0 {
		t.Errorf("capacity not consumed: %d", len(res.All()))
	}
	// Grid alignment.
	f2 := newFixture(t)
	f2.write(availability(ada, "PT2H", "2026-09-15", "10:00/12:00", ""))
	one := intention("One", "PT1H", "2026-09-15", "")
	f2.write(one)
	e2 := f2.env()
	e2.Step = temporal.Duration{Minutes: 30}
	res, _ = Resolve(e2, one, Options{})
	if res.Considered != 3 || res.All()[0].Interval.Start.In(e2.Ctx.Location).Minute() != 0 || res.All()[1].Interval.Start.In(e2.Ctx.Location).Minute() != 30 {
		t.Errorf("grid: %d %v", res.Considered, res.All())
	}
}

func TestResolveRelativeAndPreconditions(t *testing.T) {
	f := newFixture(t)
	f.write(availability(ada, "PT8H", "2026-09/2026-10", "09:00/17:00", "FREQ=DAILY"))
	a := intention("A", "PT1H", "2026-W38", "")
	a.Placement = placement("2026-09-15T10:00:00+10:00", "PT1H")
	b := intention("B", "PT1H", "2026-W38", "")
	r, _ := temporal.ParseRelative(a.ID + ":FINISHTOSTART:P0D:P3D")
	b.Window.Relative = &r
	f.write(a, b)
	e := f.env()
	res, _ := Resolve(e, b, Options{})
	if len(res.All()) == 0 {
		t.Fatalf("relative: %s", res.Reason)
	}
	first := res.All()[0].Interval.Start
	last := res.All()[len(res.All())-1].Interval.Start
	if first.Before(a.Placement.Bounds(e.Ctx).End) || last.After(temporal.AddDuration(a.Placement.Bounds(e.Ctx).End, temporal.Duration{Days: 3})) {
		t.Errorf("relative bounds: %s .. %s", first, last)
	}
	// Blocked.
	a.Placement = nil
	f.write(a)
	res, _ = Resolve(f.env(), b, Options{})
	if res.Blocked != a.ID || len(res.All()) != 0 {
		t.Errorf("blocked: %q %d", res.Blocked, len(res.All()))
	}
	// Preconditions.
	noDur := intention("No duration", "", "2026-W38", "")
	f.write(noDur)
	if _, err := Resolve(f.env(), noDur, Options{}); err == nil || !strings.Contains(err.Error(), "duration") {
		t.Errorf("no duration: %v", err)
	}
	c1, c2 := intention("C1", "PT1H", "2026-W38", ""), intention("C2", "PT1H", "2026-W38", "")
	c1.Serves = []model.Ref{{ID: c2.ID, Role: model.RoleInOrderTo}}
	c2.Serves = []model.Ref{{ID: c1.ID, Role: model.RoleInOrderTo}}
	f.write(c1, c2)
	if _, err := Resolve(f.env(), c1, Options{}); err == nil || !strings.Contains(err.Error(), "cycle") {
		t.Errorf("cycle: %v", err)
	}
	if _, err := Resolve(f.env(), a, Options{}); err != nil {
		t.Errorf("unplaced again: %v", err)
	}
	a.Placement = placement("2026-09-15T10:00:00+10:00", "PT1H")
	f.write(a)
	if _, err := Resolve(f.env(), a, Options{}); err == nil || !strings.Contains(err.Error(), "--replace") {
		t.Errorf("placed: %v", err)
	}
}

func TestRankingAndPreference(t *testing.T) {
	f := newFixture(t)
	f.write(availability(ada, "PT8H", "2026-09-15", "09:00/17:00", ""))
	firm := intention("Firm", "PT1H", "2026-09-15", "")
	firm.Stability = "firm"
	firm.Placement = placement("2026-09-15T09:00:00+10:00", "PT1H")
	tent := intention("Tentative", "PT1H", "2026-09-15", "")
	tent.Placement = placement("2026-09-15T11:00:00+10:00", "PT1H")
	f.write(firm, tent)
	x := intention("X", "PT1H", "2026-09-15", "")
	f.write(x)
	e := f.env()
	res, _ := Resolve(e, x, Options{})
	all := res.All()
	byStart := map[int]Candidate{}
	for _, c := range all {
		byStart[c.Interval.Start.In(e.Ctx.Location).Hour()*100+c.Interval.Start.In(e.Ctx.Location).Minute()] = c
	}
	if byStart[900].Rank != 3 || byStart[1100].Rank != 2 || byStart[1300].Rank != 1 {
		t.Errorf("ranks: 09:00=%d 11:00=%d 13:00=%d", byStart[900].Rank, byStart[1100].Rank, byStart[1300].Rank)
	}
	if all[0].Rank != 1 || all[len(all)-1].Rank != 3 {
		t.Error("ordering by rank")
	}
	if byStart[1100].Displaces[0] != tent.ID {
		t.Errorf("displaces: %v", byStart[1100].Displaces)
	}
	// Preference latest: first rank-1 candidate is the latest start.
	x.Preference = "latest"
	f.write(x)
	res, _ = Resolve(f.env(), x, Options{})
	if res.All()[0].Interval.Start.In(e.Ctx.Location).Hour() != 16 {
		t.Errorf("latest: %s", res.All()[0].Start)
	}
	// Adjacent: abutting the deep-work placement ranks first among rank 1.
	deep := intention("Deep", "PT1H", "2026-09-15", "")
	deep.Activity = "deep-work"
	deep.Placement = placement("2026-09-15T14:00:00+10:00", "PT1H")
	f.write(deep)
	x.Preference = "adjacent"
	x.Activity = "deep-work"
	f.write(x)
	res, _ = Resolve(f.env(), x, Options{})
	h := res.All()[0].Interval.Start.In(e.Ctx.Location).Hour()
	if res.All()[0].Rank != 1 || (h != 13 && h != 15) {
		t.Errorf("adjacent: rank %d at %s", res.All()[0].Rank, res.All()[0].Start)
	}
	// Transparent overlap is free.
	tr := true
	cmt := &model.Commitment{ID: model.MintID(model.TypeCommitment), Parties: []model.Party{{URI: ada, Status: "accepted"}}, Placement: placement("2026-09-15T15:30:00+10:00", "PT1H"),
		Origin: model.Origin{Import: true}, Transparent: &tr, Source: model.Source{Author: ada}, Timestamp: now, Acknowledgements: []model.Acknowledgement{}}
	f.write(cmt)
	res, _ = Resolve(f.env(), x, Options{})
	for _, c := range res.All() {
		if c.Interval.Start.In(e.Ctx.Location).Hour() == 15 && c.Interval.Start.In(e.Ctx.Location).Minute() == 30 && c.Rank != 1 {
			t.Errorf("transparent overlap raised rank to %d", c.Rank)
		}
	}
}

// --- selection --------------------------------------------------------------

func TestSelect(t *testing.T) {
	f := newFixture(t)
	av := availability(ada, "PT8H", "2026-09-15", "09:00/17:00", "")
	av.Location = []string{off}
	f.write(av)
	f.write(availability(rob, "PT8H", "2026-09-15", "09:00/17:00", ""))
	tent := intention("Tentative", "PT1H", "2026-09-15", "")
	tent.Placement = placement("2026-09-15T09:00:00+10:00", "PT1H")
	f.write(tent)
	in := intention("Meet Rob", "PT1H", "2026-09-15", "")
	in.Parties = []string{rob}
	in.Location = []string{home, off}
	f.write(in)
	e := f.env()
	sel, res, err := Select(e, in, SelectOptions{Candidate: 1, Source: model.Source{Author: ada}, Timestamp: now})
	if err != nil {
		t.Fatal(err)
	}
	rec := sel.Resolution
	if rec.Selector != "person" || rec.Intention != in.ID || *rec.CandidatesConsidered != res.Considered || rec.Source.Author != ada {
		t.Errorf("record: %+v", rec)
	}
	if sel.Intention.Placement == nil || sel.Intention.Placement.Location != off {
		t.Errorf("placement/location: %+v", sel.Intention.Placement)
	}
	if sel.Commitment == nil || len(sel.Commitment.Parties) != 2 || sel.Commitment.Parties[0].Status != "tentative" || sel.Commitment.Origin.Resolution != rec.ID || sel.Commitment.Intention != in.ID || sel.Commitment.Transparent != nil {
		t.Errorf("commitment: %+v", sel.Commitment)
	}
	// The first candidate is the earliest rank-1: 10:00, after the tentative one; nothing displaced.
	if len(rec.Displaced) != 0 || sel.Candidate.Rank != 1 {
		t.Errorf("first candidate should displace nothing: %v rank %d", rec.Displaced, sel.Candidate.Rank)
	}
	// Supply recorded after timestamp, outside the projection.
	data, _ := model.Encode(rec)
	if !strings.Contains(string(data), "timestamp: 2026-09-10T00:00:00Z\nsupply:\n  - avl_") {
		t.Errorf("supply not written after timestamp:\n%s", data)
	}
	// Placed refuses without replace; replace writes a second record.
	e = f.env()
	in2, _ := e.G.Get(in.ID)
	if _, _, err := Select(e, in2.(*model.Intention), SelectOptions{Candidate: 2, Source: model.Source{Author: ada}, Timestamp: now}); err == nil {
		t.Error("placed selected without replace")
	}
	sel2, _, err := Select(e, in2.(*model.Intention), SelectOptions{Candidate: 2, Source: model.Source{Author: ada}, Timestamp: now, Replace: true})
	if err != nil || sel2.Replaced == nil || sel2.Resolution.ID == rec.ID {
		t.Errorf("replace: %v %+v", err, sel2.Replaced)
	}
	if n := len(f.env().G.Resolutions()); n != 2 {
		t.Errorf("records: %d", n)
	}
}

func TestSelectPolicyAndDisplacement(t *testing.T) {
	f := newFixture(t)
	f.write(availability(ada, "PT8H", "2026-09-15", "09:00/17:00", ""))
	d30 := temporal.Duration{Minutes: 30}
	policy := intention("Small things", "", "", "")
	policy.AutoSelect = &model.Policy{MaxDuration: &d30}
	quick := intention("Quick", "PT15M", "2026-09-15", "")
	long := intention("Long", "PT2H", "2026-09-15", "")
	f.write(policy, quick, long)
	harness := model.Source{Author: ada, Harness: "claude"}
	e := f.env()
	if _, _, err := Select(e, quick, SelectOptions{Policy: policy.ID, Source: model.Source{Author: ada}, Timestamp: now}); err == nil {
		t.Error("person with --policy accepted")
	}
	sel, _, err := Select(e, quick, SelectOptions{Policy: policy.ID, Source: harness, Timestamp: now})
	if err != nil || sel.Resolution.Selector != policy.ID || sel.Resolution.Source.Harness != "claude" {
		t.Errorf("policy select: %v %+v", err, sel.Resolution)
	}
	e = f.env()
	if _, _, err := Select(e, long, SelectOptions{Policy: policy.ID, Source: harness, Timestamp: now}); err == nil || !strings.Contains(err.Error(), "max_duration") {
		t.Errorf("policy does not cover: %v", err)
	}
	// Only displacing candidates: a firm placement fills the whole occasion.
	f2 := newFixture(t)
	f2.write(availability(ada, "PT8H", "2026-09-15", "09:00/11:00", ""))
	firm := intention("Firm", "PT2H", "2026-09-15", "")
	firm.Stability = "firm"
	firm.Placement = placement("2026-09-15T09:00:00+10:00", "PT2H")
	f2.write(firm)
	p2 := intention("Policy", "", "", "")
	p2.AutoSelect = &model.Policy{MaxDuration: &d30}
	q2 := intention("Quick", "PT15M", "2026-09-15", "")
	f2.write(p2, q2)
	// Capacity 8h leaves room, so the 15-minute candidate overlaps the firm placement at rank 3.
	if _, _, err := Select(f2.env(), q2, SelectOptions{Policy: p2.ID, Source: harness, Timestamp: now}); err == nil || !strings.Contains(err.Error(), "displaces") {
		t.Errorf("only displacing: %v", err)
	}
	// A person may select it; the firm one is listed as displaced and untouched.
	sel, _, err = Select(f2.env(), q2, SelectOptions{Candidate: 1, Source: model.Source{Author: ada}, Timestamp: now})
	if err != nil || sel.Candidate.Rank != 3 || len(sel.Resolution.Displaced) != 1 || sel.Resolution.Displaced[0] != firm.ID {
		t.Errorf("displacing selection: %v %+v", err, sel.Resolution)
	}
	after, _ := f2.env().G.Get(firm.ID)
	if fi := after.(*model.Intention); fi.Stability != "firm" || fi.Placement.Start.Raw != firm.Placement.Start.Raw || fi.Retired != nil {
		t.Error("displaced object was changed")
	}
}
