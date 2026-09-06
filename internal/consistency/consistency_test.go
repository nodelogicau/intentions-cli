package consistency

import (
	"testing"
	"time"

	"github.com/nodelogicau/intentions-cli/internal/model"
	"github.com/nodelogicau/intentions-cli/internal/projection"
	"github.com/nodelogicau/intentions-cli/internal/resolve"
	"github.com/nodelogicau/intentions-cli/internal/store"
	"github.com/nodelogicau/intentions-cli/internal/temporal"
)

const ada = "https://example.com/people/ada"

var now = time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC)

func newWS(t *testing.T) *store.Workspace {
	t.Helper()
	cfg := store.NewConfig()
	cfg.Resolver.Timezone = "Australia/Melbourne"
	cfg.Defaults.Subject = ada
	cfg.Defaults.Source.Author = ada
	ws, _, err := store.Init(t.TempDir(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	return ws
}

func env(t *testing.T, ws *store.Workspace) resolve.Env {
	t.Helper()
	g, err := ws.Load()
	if err != nil {
		t.Fatal(err)
	}
	return resolve.NewEnv(ws, g, now)
}

func write(t *testing.T, ws *store.Workspace, objs ...model.Object) {
	t.Helper()
	for _, o := range objs {
		if err := ws.WriteObject(o); err != nil {
			t.Fatal(err)
		}
	}
}

func dur(s string) *temporal.DurationSpec { d, _ := temporal.ParseDurationSpec(s); return &d }

func win(cal, clock string) *temporal.Window {
	w := &temporal.Window{}
	if cal != "" {
		c, _ := temporal.ParseCalendar(cal)
		w.Calendar = &c
	}
	if clock != "" {
		k, _ := temporal.ParseClock(clock)
		w.Clock = &k
	}
	return w
}

func placement(start, d string) *temporal.Placement {
	s, _ := temporal.ParsePlacementStart(start)
	return &temporal.Placement{Start: s, Duration: *dur(d)}
}

func intention(title, d, cal string, at string) *model.Intention {
	o := &model.Intention{ID: model.MintID(model.TypeIntention), Subject: ada, Title: title, Stability: "tentative", Serves: []model.Ref{},
		Source: model.Source{Author: ada}, Timestamp: now, Acknowledgements: []model.Acknowledgement{}}
	if d != "" {
		o.Duration = dur(d)
	}
	if cal != "" {
		o.Window = win(cal, "")
	}
	if at != "" {
		o.Placement = placement(at, d)
	}
	return o
}

func availability(d, cal, clock string) *model.Availability {
	return &model.Availability{ID: model.MintID(model.TypeAvailability), Subject: ada, Duration: dur(d), Window: win(cal, clock), Scope: "personal",
		Source: model.Source{Author: ada}, Timestamp: now}
}

func commitment(at, d string, transparent bool) *model.Commitment {
	c := &model.Commitment{ID: model.MintID(model.TypeCommitment), Parties: []model.Party{{URI: ada, Status: "accepted"}}, Placement: placement(at, d),
		Origin: model.Origin{Import: true}, Source: model.Source{Author: ada}, Timestamp: now, Acknowledgements: []model.Acknowledgement{}}
	if transparent {
		tr := true
		c.Transparent = &tr
	}
	return c
}

func kinds(flags []Flag, subject string) map[string]Flag {
	out := map[string]Flag{}
	for _, f := range flags {
		if f.Subject == subject {
			out[f.Kind] = f
		}
	}
	return out
}

func TestClashesAndGrounds(t *testing.T) {
	ws := newWS(t)
	write(t, ws, availability("PT8H", "2026-09/2026-10", "09:00/17:00"))
	a := intention("A", "PT1H", "2026-09-15", "2026-09-15T10:00:00+10:00")
	c := commitment("2026-09-15T10:30:00+10:00", "PT1H", false)
	write(t, ws, a, c)
	flags, err := Check(env(t, ws), nil)
	if err != nil {
		t.Fatal(err)
	}
	fa, fc := kinds(flags, a.ID), kinds(flags, c.ID)
	if fa[WindowClash].Counterpart != c.ID || fc[WindowClash].Counterpart != a.ID {
		t.Errorf("clash on both: %+v", flags)
	}
	cv, _ := projection.Version(c)
	if fa[WindowClash].CounterpartVersion != cv {
		t.Errorf("counterpart version: %s want %s", fa[WindowClash].CounterpartVersion, cv)
	}
	// No supply at all: window-clash without counterpart.
	nowhere := intention("Nowhere", "PT1H", "2026-09-20", "2026-09-20T20:00:00+10:00")
	write(t, ws, nowhere)
	flags, _ = Check(env(t, ws), []string{nowhere.ID})
	if f := kinds(flags, nowhere.ID)[WindowClash]; f.Counterpart != "" || f.Kind == "" {
		t.Errorf("no supply: %+v", flags)
	}
	// Condition mismatch: the only containing availability is conditional on deep-work.
	ws2 := newWS(t)
	deep := availability("PT3H", "2026-09-15", "09:00/12:00")
	deep.Conditional = []string{"deep-work"}
	meeting := intention("Meeting", "PT1H", "2026-09-15", "2026-09-15T09:00:00+10:00")
	meeting.Activity = "meeting"
	write(t, ws2, deep, meeting)
	flags, _ = Check(env(t, ws2), nil)
	if f := kinds(flags, meeting.ID)[ConditionMismatch]; f.Counterpart != deep.ID {
		t.Errorf("condition mismatch: %+v", flags)
	}
	// Expired ground: retire it.
	deep.Retired = &model.Retired{Kind: "retracted", Source: model.Source{Author: ada}, Timestamp: now}
	write(t, ws2, deep)
	flags, _ = Check(env(t, ws2), nil)
	if f := kinds(flags, meeting.ID)[ExpiredGround]; f.Counterpart != deep.ID {
		t.Errorf("expired ground: %+v", flags)
	}
}

func TestTransparentAndImport(t *testing.T) {
	ws := newWS(t)
	write(t, ws, availability("PT8H", "2026-09/2026-10", "09:00/17:00"))
	firm := intention("Firm", "PT1H", "2026-09-23", "2026-09-23T10:00:00+10:00")
	firm.Stability = "firm"
	conf := commitment("2026-09-21", "P5D", true)
	write(t, ws, firm, conf)
	flags, _ := Check(env(t, ws), nil)
	for _, f := range flags {
		if f.Kind == WindowClash && (f.Subject == conf.ID || f.Counterpart == conf.ID) {
			t.Errorf("transparent commitment clashed: %+v", f)
		}
		if f.Subject == conf.ID {
			t.Errorf("transparent commitment flagged: %+v", f)
		}
	}
	opaque := commitment("2026-09-23T10:30:00+10:00", "PT1H", false)
	write(t, ws, opaque)
	flags, _ = Check(env(t, ws), nil)
	if kinds(flags, opaque.ID)[WindowClash].Counterpart != firm.ID || kinds(flags, firm.ID)[WindowClash].Counterpart != opaque.ID {
		t.Errorf("opaque import did not clash: %+v", flags)
	}
}

func TestSuppressionAndLapse(t *testing.T) {
	ws := newWS(t)
	write(t, ws, availability("PT8H", "2026-09/2026-10", "09:00/17:00"))
	a := intention("A", "PT1H", "2026-09-15", "2026-09-15T10:00:00+10:00")
	b := intention("B", "PT1H", "2026-09-15", "2026-09-15T10:30:00+10:00")
	write(t, ws, a, b)
	bv, _ := projection.Version(b)
	a.Acknowledgements = []model.Acknowledgement{{Kind: WindowClash, Counterpart: b.ID, CounterpartVersion: bv, Reason: "fine", Source: model.Source{Author: ada}, Timestamp: now}}
	write(t, ws, a)
	flags, _ := Check(env(t, ws), nil)
	if _, has := kinds(flags, a.ID)[WindowClash]; has {
		t.Error("acknowledged flag reported")
	}
	if _, has := kinds(flags, b.ID)[WindowClash]; !has {
		t.Error("counterpart's own flag suppressed")
	}
	// Prose edit does not lapse.
	b.Description = "more"
	write(t, ws, b)
	if flags, _ := Check(env(t, ws), nil); len(kinds(flags, a.ID)) != 0 {
		t.Error("prose edit lapsed the acknowledgement")
	}
	// Projection edit lapses; the acknowledgement stays.
	b.Placement = placement("2026-09-15T10:45:00+10:00", "PT1H")
	write(t, ws, b)
	flags, _ = Check(env(t, ws), nil)
	if _, has := kinds(flags, a.ID)[WindowClash]; !has {
		t.Error("lapsed flag not reported")
	}
	e := env(t, ws)
	obj, _ := e.G.Get(a.ID)
	if len(obj.(*model.Intention).Acknowledgements) != 1 {
		t.Error("acknowledgement removed")
	}
	// A flag with no counterpart is acknowledged by kind alone.
	nowhere := intention("Nowhere", "PT1H", "2026-09-20", "2026-09-20T20:00:00+10:00")
	nowhere.Acknowledgements = []model.Acknowledgement{{Kind: WindowClash, Reason: "no supply yet", Source: model.Source{Author: ada}, Timestamp: now}}
	write(t, ws, nowhere)
	if flags, _ := Check(env(t, ws), []string{nowhere.ID}); len(flags) != 0 {
		t.Errorf("bare acknowledgement not honoured: %+v", flags)
	}
}

func TestInconsistencyAndCycle(t *testing.T) {
	ws := newWS(t)
	// One two-hour occasion; two two-hour intentions each fit alone, never together.
	write(t, ws, availability("PT4H", "2026-09-15", "09:00/11:00"))
	x := intention("X", "PT2H", "2026-09-15", "")
	y := intention("Y", "PT2H", "2026-09-15", "")
	write(t, ws, x, y)
	flags, err := Check(env(t, ws), nil)
	if err != nil {
		t.Fatal(err)
	}
	if kinds(flags, x.ID)[IntentionInconsistency].Counterpart != y.ID || kinds(flags, y.ID)[IntentionInconsistency].Counterpart != x.ID {
		t.Errorf("inconsistency: %+v", flags)
	}
	e := env(t, ws)
	for _, id := range []string{x.ID, y.ID} {
		o, _ := e.G.Get(id)
		if o.GetRetired() != nil {
			t.Error("check retired something")
		}
	}
	// Cycle.
	c1, c2 := intention("C1", "", "", ""), intention("C2", "", "", "")
	c1.Serves = []model.Ref{{ID: c2.ID, Role: model.RoleInOrderTo}}
	c2.Serves = []model.Ref{{ID: c1.ID, Role: model.RoleInOrderTo}}
	write(t, ws, c1, c2)
	flags, _ = Check(env(t, ws), []string{c1.ID})
	if f := kinds(flags, c1.ID)[Cycle]; f.Kind == "" || f.Counterpart != "" {
		t.Errorf("cycle: %+v", flags)
	}
	// Second run on an unchanged workspace reports the same flags.
	again, _ := Check(env(t, ws), []string{c1.ID})
	if len(again) != len(flags) {
		t.Error("check not stable")
	}
}
