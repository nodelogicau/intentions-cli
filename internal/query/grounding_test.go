package query

import (
	"strings"
	"testing"
	"time"

	"github.com/nodelogicau/intentions-cli/internal/model"
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
