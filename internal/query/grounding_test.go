package query

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/nodelogicau/intentions-cli/internal/model"
	"github.com/nodelogicau/intentions-cli/internal/store"
	"github.com/nodelogicau/intentions-cli/internal/temporal"
)

func withDuration(o *model.Intention) *model.Intention {
	o.Duration = &temporal.DurationSpec{Nominal: temporal.Duration{Hours: 1}}
	return o
}

func firmTerminus(id string) *model.Intention {
	o := intention(id)
	o.Stability = "firm"
	return o
}

func findingsFor(r Report, id string) map[string]string {
	out := map[string]string{}
	for _, f := range r.Findings {
		if f.ID == id {
			out[f.Code] = f.Severity + ": " + f.Message
		}
	}
	return out
}

func TestEveryIntentionReachesATerminus(t *testing.T) {
	// Chain ends on a scheduled intention: both unserved, the message names the fix.
	ws := newWS(t)
	tt := firmTerminus("int_t")
	b := withDuration(intention("int_b"))
	a := withDuration(intention("int_a"))
	a.Serves = []model.Ref{{ID: "int_b", Role: model.RoleInOrderTo}}
	write(t, ws, tt, a, b)
	r := run(t, ws)
	for _, id := range []string{"int_a", "int_b"} {
		f := findingsFor(r, id)
		if !strings.HasPrefix(f["unserved"], "warning:") || !strings.Contains(f["unserved"], "--serves int_t:for-the-sake-of") {
			t.Errorf("%s: %v", id, f)
		}
	}
	if strings.Contains(findingsFor(r, "int_a")["unserved"], "int_b, which serves nothing") == false {
		t.Errorf("int_a should name where its chain ends: %v", findingsFor(r, "int_a"))
	}
	if r.HasErrors() {
		t.Errorf("warnings are not errors: %v", r.Findings)
	}
	// Serving int_b's end serves int_a too.
	b.Serves = []model.Ref{{ID: "int_t", Role: model.RoleForTheSakeOf}}
	write(t, ws, b)
	r = run(t, ws)
	if has(r, SeverityWarning, "unserved") {
		t.Errorf("served through a means still reported: %v", r.Findings)
	}

	// Only a tentative terminus: unserved names the draft, the draft is info.
	ws = newWS(t)
	d := intention("int_d")
	a = withDuration(intention("int_a"))
	a.Serves = []model.Ref{{ID: "int_d", Role: model.RoleForTheSakeOf}}
	write(t, ws, d, a)
	r = run(t, ws)
	if f := findingsFor(r, "int_a"); !strings.Contains(f["unserved"], "draft terminus int_d") || !strings.Contains(f["unserved"], "intentions intention firm int_d") {
		t.Errorf("draft chain: %v", f)
	}
	if f := findingsFor(r, "int_d"); !strings.HasPrefix(f["draft_terminus"], "info:") {
		t.Errorf("draft terminus: %v", f)
	}
	if r.Counts[SeverityWarning] != 1 || r.Counts[SeverityInfo] < 1 {
		t.Errorf("counts: %v", r.Counts)
	}

	// Instance reaches through the recurring intention.
	ws = newWS(t)
	tt = firmTerminus("int_t")
	rec := withDuration(intention("int_rec"))
	c, _ := temporal.ParseCalendar("2026-09")
	rec.Window = &temporal.Window{Calendar: &c}
	rec.Serves = []model.Ref{{ID: "int_t", Role: model.RoleForTheSakeOf}}
	inst := withDuration(intention("int_i"))
	inst.Serves = []model.Ref{{ID: "int_rec", Role: model.RoleInstanceOf}}
	write(t, ws, tt, rec, inst)
	if r = run(t, ws); has(r, SeverityWarning, "unserved") {
		t.Errorf("instance not served: %v", r.Findings)
	}

	// Another subject's terminus does not count.
	ws = newWS(t)
	p := firmTerminus("int_p")
	p.Subject = "https://example.com/people/priya"
	a = withDuration(intention("int_a"))
	a.Serves = []model.Ref{{ID: "int_p", Role: model.RoleForTheSakeOf}}
	write(t, ws, p, a)
	if f := findingsFor(run(t, ws), "int_a"); !strings.Contains(f["unserved"], "another subject") || !strings.Contains(f["unserved"], "add one first") {
		t.Errorf("foreign terminus: %v", f)
	}

	// Workspace without termini: every intention warns, exit stays clean.
	ws = newWS(t)
	write(t, ws, withDuration(intention("int_a")), withDuration(intention("int_b")))
	r = run(t, ws)
	if r.Counts[SeverityWarning] != 2 || r.HasErrors() {
		t.Errorf("no termini: %v", r.Findings)
	}
	if !strings.Contains(findingsFor(r, "int_a")["unserved"], "holds no firm terminus") {
		t.Errorf("message: %v", findingsFor(r, "int_a"))
	}

	// A cycle is an error, not doubled as unserved; a retired intention is not checked.
	ws = newWS(t)
	x := withDuration(intention("int_x"))
	y := withDuration(intention("int_y"))
	x.Serves = []model.Ref{{ID: "int_y", Role: model.RoleInOrderTo}}
	y.Serves = []model.Ref{{ID: "int_x", Role: model.RoleInOrderTo}}
	z := withDuration(intention("int_z"))
	z.Retired = &model.Retired{Kind: "abandoned", Source: model.Source{Author: "a"}, Timestamp: time.Now()}
	write(t, ws, x, y, z)
	r = run(t, ws)
	if has(r, SeverityWarning, "unserved") || len(findingsFor(r, "int_z")) != 0 {
		t.Errorf("cycle or retired reported unserved: %v", r.Findings)
	}
}

func TestFirmedUnderOnATerminus(t *testing.T) {
	ws := newWS(t)
	pol := firmTerminus("int_pol")
	d30 := temporal.Duration{Minutes: 30}
	pol.AutoFirm = &model.Policy{MaxDuration: &d30}
	tt := firmTerminus("int_t")
	tt.FirmedUnder = "int_pol"
	tt.Source.Harness = "claude"
	write(t, ws, pol, tt)
	r := run(t, ws)
	if f := findingsFor(r, "int_t"); !strings.HasPrefix(f["firmed_under_terminus"], "error:") || f["firmed_under_target"] != "" {
		t.Errorf("terminus with firmed_under: %v", f)
	}
}

func TestWriteFindings(t *testing.T) {
	ws := newWS(t)
	tt := firmTerminus("int_t")
	write(t, ws, tt)
	g, err := ws.Load()
	if err != nil {
		t.Fatal(err)
	}
	// An unwritten intention is checked as it will stand.
	a := withDuration(intention("int_a"))
	if fs := WriteFindings(g, a); len(fs) != 1 || fs[0].Code != "unserved" || !strings.Contains(fs[0].Message, "--serves int_t:for-the-sake-of") {
		t.Errorf("unserved add: %+v", fs)
	}
	a.Serves = []model.Ref{{ID: "int_t", Role: model.RoleForTheSakeOf}}
	if fs := WriteFindings(g, a); fs != nil {
		t.Errorf("served add: %+v", fs)
	}
	d := intention("int_d")
	if fs := WriteFindings(g, d); len(fs) != 1 || fs[0].Code != "draft_terminus" || fs[0].Severity != SeverityInfo {
		t.Errorf("draft add: %+v", fs)
	}
	d.Stability = "firm"
	if fs := WriteFindings(g, d); fs != nil {
		t.Errorf("firm terminus: %+v", fs)
	}
}

func TestSelectorWithdrawn(t *testing.T) {
	ws := newWS(t)
	pol := firmTerminus("int_pol")
	d30 := temporal.Duration{Minutes: 30}
	pol.AutoSelect = &model.Policy{MaxDuration: &d30}
	a := withDuration(intention("int_a"))
	a.Serves = []model.Ref{{ID: "int_pol", Role: model.RoleForTheSakeOf}}
	rec := &model.Resolution{ID: "res_1", Intention: "int_a", Selector: "int_pol", Source: model.Source{Author: "a", Harness: "claude"}, Timestamp: time.Now(), Placement: a.Placement}
	write(t, ws, pol, a)
	if err := ws.WriteObject(rec); err != nil {
		t.Fatal(err)
	}
	if r := run(t, ws); has(r, SeverityError, "selector_not_policy") || has(r, SeverityWarning, "selector_withdrawn") {
		t.Errorf("live policy: %v", r.Findings)
	}
	// Suspended: a warning, never an error; the record is history.
	pol.Stability = "tentative"
	write(t, ws, pol)
	r := run(t, ws)
	if f := findingsFor(r, "res_1"); !strings.HasPrefix(f["selector_withdrawn"], "warning:") || !strings.Contains(f["selector_withdrawn"], "set tentative") || f["selector_not_policy"] != "" {
		t.Errorf("suspended: %v", f)
	}
	// Ended: the same warning, naming retirement.
	pol.Stability = "firm"
	pol.Retired = &model.Retired{Kind: "abandoned", Source: model.Source{Author: "a"}, Timestamp: time.Now()}
	write(t, ws, pol)
	if f := findingsFor(run(t, ws), "res_1"); !strings.Contains(f["selector_withdrawn"], "retired") {
		t.Errorf("retired: %v", f)
	}
	// Not a policy at all is still an error.
	pol.Retired = nil
	pol.AutoSelect = nil
	write(t, ws, pol)
	if f := findingsFor(run(t, ws), "res_1"); !strings.HasPrefix(f["selector_not_policy"], "error:") {
		t.Errorf("not a policy: %v", f)
	}
}

func retire(o *model.Intention, kind, by string) *model.Intention {
	o.Retired = &model.Retired{Kind: kind, SupersededBy: by, Source: o.Source, Timestamp: time.Now()}
	return o
}

func TestGroundingThroughRetirement(t *testing.T) {
	ws := newWS(t)
	// A reason superseded by a firm one passes the ground on.
	old := retire(firmTerminus("int_t"), "superseded", "int_u")
	u := firmTerminus("int_u")
	a := withDuration(intention("int_a"))
	a.Serves = []model.Ref{{ID: "int_t", Role: model.RoleForTheSakeOf}}
	// An abandoned reason grounds nothing, and the message says it is retired.
	gone := retire(firmTerminus("int_g"), "abandoned", "")
	b := withDuration(intention("int_b"))
	b.Serves = []model.Ref{{ID: "int_g", Role: model.RoleForTheSakeOf}}
	// An ended arrangement: its instances stay served and each warns.
	rec := retire(withDuration(intention("int_rec")), "abandoned", "")
	rec.Serves = []model.Ref{{ID: "int_u", Role: model.RoleForTheSakeOf}}
	i1 := withDuration(intention("int_i1"))
	i1.Serves = []model.Ref{{ID: "int_rec", Role: model.RoleInstanceOf}}
	// A fulfilled end is walked through and silent.
	done := retire(withDuration(intention("int_done")), "fulfilled", "")
	done.Serves = []model.Ref{{ID: "int_u", Role: model.RoleForTheSakeOf}}
	c := withDuration(intention("int_c"))
	c.Serves = []model.Ref{{ID: "int_done", Role: model.RoleInOrderTo}}
	write(t, ws, old, u, a, gone, b, rec, i1, done, c)
	r := run(t, ws)
	if f := findingsFor(r, "int_a"); len(f) != 0 {
		t.Errorf("served through a superseded reason: %v", f)
	}
	if f := findingsFor(r, "int_b"); !strings.Contains(f["unserved"], "retired terminus int_g") {
		t.Errorf("abandoned reason: %v", f)
	}
	if f := findingsFor(r, "int_i1"); len(f) != 1 || !strings.HasPrefix(f["abandoned_end"], "warning:") || !strings.Contains(f["abandoned_end"], "int_rec") {
		t.Errorf("ended arrangement: %v", f)
	}
	if f := findingsFor(r, "int_c"); len(f) != 0 {
		t.Errorf("fulfilled end: %v", f)
	}
}

func TestTerminusRetirementErrors(t *testing.T) {
	ws := newWS(t) // 0.1: these are errors under either version
	done := retire(firmTerminus("int_t"), "fulfilled", "")
	sched := withDuration(intention("int_b"))
	sched.Serves = []model.Ref{{ID: "int_u", Role: model.RoleForTheSakeOf}}
	u := firmTerminus("int_u")
	bySched := retire(firmTerminus("int_s"), "superseded", "int_b")
	other := firmTerminus("int_o")
	other.Subject = "https://example.com/people/priya"
	byOther := retire(firmTerminus("int_x"), "superseded", "int_o")
	byTerminus := retire(firmTerminus("int_y"), "superseded", "int_u")
	write(t, ws, done, sched, u, bySched, other, byOther, byTerminus)
	r := run(t, ws)
	if f := findingsFor(r, "int_t"); !strings.HasPrefix(f["terminus_fulfilled"], "error:") {
		t.Errorf("fulfilled terminus: %v", f)
	}
	for _, id := range []string{"int_s", "int_x"} {
		if f := findingsFor(r, id); !strings.HasPrefix(f["terminus_successor"], "error:") {
			t.Errorf("%s: %v", id, f)
		}
	}
	if f := findingsFor(r, "int_y"); len(f) != 0 {
		t.Errorf("superseded by a terminus of its subject: %v", f)
	}
}

func TestRefuseRetirement(t *testing.T) {
	ws := newWS(t)
	tt := firmTerminus("int_t")
	u := firmTerminus("int_u")
	draft := intention("int_d")
	mine := intention("int_m")
	mine.Source.Harness = "claude"
	a := withDuration(intention("int_a"))
	a.Serves = []model.Ref{{ID: "int_t", Role: model.RoleForTheSakeOf}}
	b := withDuration(intention("int_b"))
	b.Serves = []model.Ref{{ID: "int_a", Role: model.RoleInOrderTo}}
	both := withDuration(intention("int_both"))
	both.Serves = []model.Ref{{ID: "int_t", Role: model.RoleForTheSakeOf}, {ID: "int_u", Role: model.RoleForTheSakeOf}}
	onDraft := withDuration(intention("int_od"))
	onDraft.Serves = []model.Ref{{ID: "int_d", Role: model.RoleForTheSakeOf}}
	des := &model.Desire{ID: "des_w", Subject: tt.Subject, Title: "w", Serves: []model.Ref{{ID: "int_u", Role: model.RoleForTheSakeOf}}, Source: tt.Source, Timestamp: time.Now()}
	write(t, ws, tt, u, draft, mine, a, b, both, onDraft, des)
	g, err := ws.Load()
	if err != nil {
		t.Fatal(err)
	}
	person := model.Source{Author: tt.Source.Author}
	harness := model.Source{Author: tt.Source.Author, Harness: "claude"}
	at := func(kind, by string, src model.Source) model.Retired {
		return model.Retired{Kind: kind, SupersededBy: by, Source: src, Timestamp: time.Now()}
	}
	rule := func(err error) string {
		if r, ok := err.(*model.Refusal); ok {
			return r.Rule
		}
		return ""
	}
	// Sole ground of int_a and, through it, int_b; int_both has int_u.
	err = RefuseRetirement(g, tt, at("abandoned", "", person), "EDIT")
	if rule(err) != "ungrounds" || !strings.Contains(err.Error(), "int_a, int_b") || strings.Contains(err.Error(), "int_both") || !strings.Contains(err.Error(), "EDIT") || !strings.Contains(err.Error(), "superseded by int_u") {
		t.Errorf("sole ground: %v", err)
	}
	if err := RefuseRetirement(g, tt, at("superseded", "int_u", person), "EDIT"); err != nil {
		t.Errorf("superseded by a firm terminus: %v", err)
	}
	if err := RefuseRetirement(g, tt, at("superseded", "int_d", person), "EDIT"); rule(err) != "ungrounds" {
		t.Errorf("superseded by a draft: %v", err)
	}
	// int_u: int_both has another ground; a desire never blocks.
	if err := RefuseRetirement(g, u, at("abandoned", "", person), "EDIT"); err != nil {
		t.Errorf("another ground remains: %v", err)
	}
	// Dependents of a draft do not block.
	if err := RefuseRetirement(g, draft, at("abandoned", "", person), "EDIT"); err != nil {
		t.Errorf("draft with dependents: %v", err)
	}
	// The harness rule.
	if err := RefuseRetirement(g, mine, at("abandoned", "", harness), "EDIT"); err != nil {
		t.Errorf("harness withdraws its draft: %v", err)
	}
	if err := RefuseRetirement(g, u, at("abandoned", "", harness), "EDIT"); rule(err) != "harness_retire" || !strings.Contains(err.Error(), "person's act") {
		t.Errorf("harness retires a firm terminus: %v", err)
	}
	if err := RefuseRetirement(g, draft, at("abandoned", "", harness), "EDIT"); rule(err) != "harness_retire" {
		t.Errorf("harness retires the person's draft: %v", err)
	}
	// Not a terminus: nothing to guard here.
	if err := RefuseRetirement(g, a, at("abandoned", "", harness), "EDIT"); err != nil {
		t.Errorf("means: %v", err)
	}
}

func newWS02(t *testing.T) *store.Workspace {
	t.Helper()
	cfg := store.NewConfig()
	cfg.Format = model.Format02
	cfg.Resolver.Timezone = "Australia/Melbourne"
	cfg.Defaults.Subject = "https://example.com/people/ada"
	cfg.Defaults.Source.Author = "https://example.com/people/ada"
	ws, _, err := store.Init(t.TempDir(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	return ws
}

func recurring(id string) *model.Intention {
	o := withDuration(intention(id))
	c, _ := temporal.ParseCadence("FREQ=DAILY")
	cal, _ := temporal.ParseCalendar("2026-09/2026-12")
	o.Cadence, o.Window = &c, &temporal.Window{Calendar: &cal}
	return o
}

func TestHabitValidation(t *testing.T) {
	ws := newWS02(t)
	tt := firmTerminus("int_t")
	h := recurring("int_h")
	h.Habit = true
	h.Serves = []model.Ref{{ID: "int_t", Role: model.RoleForTheSakeOf}}
	inst := withDuration(intention("int_i"))
	inst.Habit = true
	inst.Occurrence = "2026-09-15"
	inst.Serves = []model.Ref{{ID: "int_h", Role: model.RoleInstanceOf}}
	term := intention("int_x")
	term.Habit = true
	write(t, ws, tt, h, inst, term)
	r := run(t, ws)
	if len(findingsFor(r, "int_h")) != 0 {
		t.Errorf("a served habit: %v", findingsFor(r, "int_h"))
	}
	for _, id := range []string{"int_i", "int_x"} {
		if f := findingsFor(r, id); !strings.HasPrefix(f["habit_not_recurring"], "error:") {
			t.Errorf("%s: %v", id, f)
		}
	}
	// The person withdraws the reason: the habit is unserved, at error level under 0.2.
	tt.Stability = "tentative"
	inst.Habit, term.Habit = false, false
	write(t, ws, tt, inst, term)
	if f := findingsFor(run(t, ws), "int_h"); !strings.HasPrefix(f["unserved"], "error:") {
		t.Errorf("a habit outlives its reason: %v", f)
	}
}

func TestHabitUnder01(t *testing.T) {
	ws := newWS(t) // intentions/0.1
	tt := firmTerminus("int_t")
	h := recurring("int_h")
	h.Serves = []model.Ref{{ID: "int_t", Role: model.RoleForTheSakeOf}}
	write(t, ws, tt, h)
	// A merge adds habit to the file without touching its cached version.
	path := filepath.Join(ws.Root, "intentions", "int_h.yaml")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	merged := strings.Replace(string(b), "\ncadence: FREQ=DAILY\n", "\ncadence: FREQ=DAILY\nhabit: true\n", 1)
	if merged == string(b) {
		t.Fatalf("no cadence line:\n%s", b)
	}
	if err := os.WriteFile(path, []byte(merged), 0o644); err != nil {
		t.Fatal(err)
	}
	r := run(t, ws)
	f := findingsFor(r, "int_h")
	if !strings.HasPrefix(f["habit_format"], "error:") || !strings.Contains(f["habit_format"], "intentions migrate") {
		t.Errorf("habit under 0.1: %v", f)
	}
	if _, stale := f["stale_version"]; stale {
		t.Errorf("habit moved the 0.1 version: %v", f)
	}
}

func TestHoldValidation(t *testing.T) {
	for _, ws := range []*store.Workspace{newWS(t), newWS02(t)} {
		tt := firmTerminus("int_t")
		s, _ := temporal.ParsePlacementStart("2026-09-21T09:00:00+10:00")
		placed := func(id string, e *model.External) *model.Intention {
			o := withDuration(intention(id))
			o.Serves = []model.Ref{{ID: "int_t", Role: model.RoleForTheSakeOf}}
			o.Placement = &temporal.Placement{Start: s, Duration: *o.Duration}
			o.External = e
			return o
		}
		a := placed("int_a", &model.External{System: "icalendar", UID: "u"})
		b := placed("int_b", &model.External{System: "icalendar", UID: "u"})
		b.Retired = &model.Retired{Kind: "fulfilled", Source: model.Source{Author: "https://example.com/people/ada"}, Timestamp: time.Now()}
		c := placed("int_c", &model.External{System: "jscalendar", UID: "u"})
		d := placed("int_d", &model.External{System: "icalendar", UID: "p"})
		d.Parties = []string{"https://example.com/people/sam"}
		e := placed("int_e", &model.External{System: "icalendar", UID: "e"})
		e.Placement = nil
		write(t, ws, tt, a, b, c, d, e)
		r := run(t, ws)
		if f := findingsFor(r, "int_a"); len(f) != 0 {
			t.Errorf("%s: the first holder: %v", ws.Config.Format, f)
		}
		if f := findingsFor(r, "int_b"); !strings.HasPrefix(f["hold_duplicate"], "error:") || !strings.Contains(f["hold_duplicate"], "int_a") || len(f) != 1 {
			t.Errorf("%s: a retired duplicate: %v", ws.Config.Format, f)
		}
		if f := findingsFor(r, "int_c"); len(f) != 0 {
			t.Errorf("%s: same uid, other system: %v", ws.Config.Format, f)
		}
		if f := findingsFor(r, "int_d"); !strings.HasPrefix(f["hold_with_parties"], "error:") || !strings.Contains(f["hold_with_parties"], "commitment") {
			t.Errorf("%s: parties: %v", ws.Config.Format, f)
		}
		if f := findingsFor(r, "int_e"); !strings.HasPrefix(f["hold_unplaced"], "error:") {
			t.Errorf("%s: unplaced: %v", ws.Config.Format, f)
		}
	}
}

func TestHoldImportedBeforeRecognised(t *testing.T) {
	ws := newWS02(t)
	tt := firmTerminus("int_t")
	s, _ := temporal.ParsePlacementStart("2026-09-21T09:00:00+10:00")
	placed := func(id, uid string) *model.Intention {
		o := withDuration(intention(id))
		o.Serves = []model.Ref{{ID: "int_t", Role: model.RoleForTheSakeOf}}
		o.Placement = &temporal.Placement{Start: s, Duration: *o.Duration}
		o.External = &model.External{System: "icalendar", UID: uid}
		return o
	}
	ada := model.Source{Author: "https://example.com/people/ada"}
	hour, _ := temporal.ParseDurationSpec("PT1H")
	imported := func(id, system, uid string) *model.Commitment {
		return &model.Commitment{ID: id, Parties: []model.Party{{URI: ada.Author, Status: "accepted"}},
			Placement: &temporal.Placement{Start: s, Duration: hour}, Origin: model.OriginImport,
			External: &model.External{System: system, UID: uid}, Source: ada, Timestamp: time.Now()}
	}
	a := placed("int_a", "u") // own hold imported once
	b := placed("int_b", "r") // a recurring import: three matches
	c := placed("int_c", "f") // retired: only the cancel is offered
	c.Retired = &model.Retired{Kind: "fulfilled", Source: ada, Timestamp: time.Now()}
	d := placed("int_d", "x") // the duplicate is cancelled
	e := placed("int_e", "j") // same uid, other system
	cx := imported("cmt_x", "icalendar", "x")
	cx.Retired = &model.Retired{Kind: "cancelled", Source: ada, Timestamp: time.Now()}
	write(t, ws, tt, a, b, c, d, e,
		imported("cmt_a", "icalendar", "u"),
		imported("cmt_r1", "icalendar", "r"), imported("cmt_r2", "icalendar", "r"), imported("cmt_r3", "icalendar", "r"),
		imported("cmt_c", "icalendar", "f"), cx, imported("cmt_j", "jscalendar", "j"))
	r := run(t, ws)
	if r.HasErrors() {
		t.Errorf("hold_imported is only a warning: %v", r.Findings)
	}
	on := func(id string) []Finding {
		var out []Finding
		for _, f := range r.Findings {
			if f.ID == id && f.Code == "hold_imported" {
				out = append(out, f)
			}
		}
		return out
	}
	if fs := on("int_a"); len(fs) != 1 || fs[0].Severity != SeverityWarning ||
		!strings.Contains(fs[0].Message, "cmt_a") || !strings.Contains(fs[0].Message, "icalendar:u") ||
		!strings.Contains(fs[0].Message, "commitment cancel cmt_a") || !strings.Contains(fs[0].Message, "intention edit int_a --clear-external") {
		t.Errorf("own hold imported: %v", fs)
	}
	if fs := on("int_b"); len(fs) != 3 || !strings.Contains(fs[0].Message, "cmt_r1") || !strings.Contains(fs[2].Message, "cmt_r3") {
		t.Errorf("recurring import: %v", fs)
	}
	if fs := on("int_c"); len(fs) != 1 || !strings.Contains(fs[0].Message, "commitment cancel cmt_c") || strings.Contains(fs[0].Message, "--clear-external") {
		t.Errorf("retired intention: %v", fs)
	}
	if fs := on("int_d"); len(fs) != 0 {
		t.Errorf("cancelled duplicate: %v", fs)
	}
	if fs := on("int_e"); len(fs) != 0 {
		t.Errorf("other system: %v", fs)
	}

	// The write path reports the same finding.
	g, err := ws.Load()
	if err != nil {
		t.Fatal(err)
	}
	if fs := WriteFindings(g, a); len(fs) != 1 || fs[0].Code != "hold_imported" {
		t.Errorf("write findings: %v", fs)
	}
}
