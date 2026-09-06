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

func newWS(t *testing.T) *store.Workspace {
	t.Helper()
	cfg := store.NewConfig()
	cfg.Resolver.Timezone = "Australia/Melbourne"
	cfg.Defaults.Subject = "https://example.com/people/ada"
	cfg.Defaults.Source.Author = "https://example.com/people/ada"
	ws, _, err := store.Init(t.TempDir(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	return ws
}

func intention(id string) *model.Intention {
	return &model.Intention{ID: id, Subject: "https://example.com/people/ada", Title: "t", Stability: "tentative", Serves: []model.Ref{},
		Source: model.Source{Author: "https://example.com/people/ada"}, Timestamp: time.Now(), Acknowledgements: []model.Acknowledgement{}}
}

func write(t *testing.T, ws *store.Workspace, objs ...model.Object) {
	t.Helper()
	for _, o := range objs {
		if err := ws.WriteObject(o); err != nil {
			t.Fatal(err)
		}
	}
}

func run(t *testing.T, ws *store.Workspace) Report {
	t.Helper()
	g, err := ws.Load()
	if err != nil {
		t.Fatal(err)
	}
	return Validate(ws, g)
}

func codes(r Report, sev string) []string {
	var out []string
	for _, f := range r.Findings {
		if f.Severity == sev {
			out = append(out, f.Code)
		}
	}
	return out
}

func has(r Report, sev, code string) bool {
	for _, c := range codes(r, sev) {
		if c == code {
			return true
		}
	}
	return false
}

func TestCleanWorkspace(t *testing.T) {
	ws := newWS(t)
	write(t, ws, intention("int_a"))
	r := run(t, ws)
	if r.HasErrors() || len(codes(r, SeverityWarning)) > 0 {
		t.Errorf("clean workspace: %+v", r.Findings)
	}
}

func TestCycleAndTerminus(t *testing.T) {
	ws := newWS(t)
	a, b, c := intention("int_a"), intention("int_b"), intention("int_c")
	a.Serves = []model.Ref{{ID: "int_b", Role: model.RoleInOrderTo}}
	b.Serves = []model.Ref{{ID: "int_a", Role: model.RoleInOrderTo}}
	write(t, ws, a, b, c)
	r := run(t, ws)
	n := 0
	for _, f := range r.Findings {
		if f.Code == "cycle" {
			n++
			if !strings.Contains(f.Message, "int_a") || !strings.Contains(f.Message, "int_b") {
				t.Errorf("cycle message: %s", f.Message)
			}
			if f.ID == "int_c" {
				t.Error("int_c reported in cycle")
			}
		}
	}
	if n != 2 {
		t.Errorf("expected 2 cycle findings, got %d: %+v", n, r.Findings)
	}
	// Terminus with outbound reference.
	ws = newWS(t)
	tt := intention("int_t")
	tt.Serves = []model.Ref{{ID: "int_x", Role: model.RoleInOrderTo}}
	x := intention("int_x")
	y := intention("int_y")
	y.Serves = []model.Ref{{ID: "int_t", Role: model.RoleForTheSakeOf}}
	write(t, ws, tt, x, y)
	r = run(t, ws)
	found := false
	for _, f := range r.Findings {
		if f.Code == "terminus" && f.ID == "int_t" {
			found = true
		}
	}
	if !found {
		t.Errorf("terminus not reported: %+v", r.Findings)
	}
}

func TestReferential(t *testing.T) {
	ws := newWS(t)
	a := intention("int_a")
	a.Serves = []model.Ref{{ID: "int_missing", Role: model.RoleInOrderTo}}
	write(t, ws, a)
	r := run(t, ws)
	if !has(r, SeverityError, "dangling") {
		t.Errorf("dangling not reported: %+v", r.Findings)
	}
	// Reference to a retired object is fine.
	ws = newWS(t)
	old := intention("int_old")
	old.Retired = &model.Retired{Kind: "abandoned", Timestamp: time.Now(), Source: model.Source{Author: "a"}}
	b := intention("int_b")
	b.Serves = []model.Ref{{ID: "int_old", Role: model.RoleInOrderTo}}
	write(t, ws, old, b)
	if r := run(t, ws); r.HasErrors() {
		t.Errorf("retired reference reported: %+v", r.Findings)
	}
}

func TestStructuralFromFiles(t *testing.T) {
	ws := newWS(t)
	raw := func(name, body string) {
		_ = os.WriteFile(filepath.Join(ws.Root, "intentions", name), []byte(body), 0o644)
	}
	raw("int_k.yaml", "id: int_k\nsubject: https://s/\ntitle: t\nstability: tentative\nserves: []\nsource: {author: a}\ntimestamp: 2026-09-04T00:00:00Z\nacknowledgements: []\nretired: {kind: cancelled, timestamp: 2026-09-05T00:00:00Z, source: {author: a}}\n")
	raw("int_q.yaml", "id: int_q\nsubject: https://s/\ntitle: t\nstability: tentative\nserves: []\nsource: {author: a}\ntimestamp: 2026-09-04T00:00:00Z\nacknowledgements: []\nwindow: {calendar: 2026-09~}\n")
	raw("int_c.yaml", "id: int_c\nsubject: https://s/\ntitle: t\nstability: tentative\nserves: []\nsource: {author: a}\ntimestamp: 2026-09-04T00:00:00Z\nacknowledgements: []\ncadence: FREQ=DAILY;BYHOUR=9\n")
	raw("int_h.yaml", "id: int_h\nsubject: https://s/\ntitle: t\nstability: tentative\nserves: []\nsource: {harness: claude}\ntimestamp: 2026-09-04T00:00:00Z\nacknowledgements: []\n")
	raw("int_f.yaml", "id: int_f\nsubject: https://s/\ntitle: t\nstability: firm\nserves: []\nsource: {author: a, harness: claude}\ntimestamp: 2026-09-04T00:00:00Z\nacknowledgements: []\n")
	raw("int_m.yaml", "id: int_other\nsubject: https://s/\ntitle: t\nstability: tentative\nserves: []\nsource: {author: a}\ntimestamp: 2026-09-04T00:00:00Z\nacknowledgements: []\n")
	raw("int_bad.yaml", "- nope\n")
	raw("int_fu.yaml", "id: int_fu\nsubject: https://s/\ntitle: t\nstability: firm\nfirmed_under: int_notpolicy\nserves: []\nsource: {author: a, harness: claude}\ntimestamp: 2026-09-04T00:00:00Z\nacknowledgements: []\n")
	raw("int_notpolicy.yaml", "id: int_notpolicy\nsubject: https://s/\ntitle: t\nstability: tentative\nwindow: {calendar: 2026-W37}\nserves: []\nsource: {author: a}\ntimestamp: 2026-09-04T00:00:00Z\nacknowledgements: []\n")
	raw("int_ok.yaml", "id: int_ok\nsubject: https://s/\ntitle: t\nstability: firm\nfirmed_under: int_pol\nserves: []\nsource: {author: a, harness: claude}\ntimestamp: 2026-09-04T00:00:00Z\nacknowledgements: []\n")
	raw("int_pol.yaml", "id: int_pol\nsubject: https://s/\ntitle: p\nstability: tentative\nserves: []\nauto_firm: {max_duration: PT30M}\nsource: {author: a}\ntimestamp: 2026-09-04T00:00:00Z\nacknowledgements: []\n")
	raw("int_cond.yaml", "id: int_cond\nsubject: https://s/\ntitle: t\nstability: tentative\nwindow: {calendar: 2026-W37}\nauto_select: {max_duration: PT30M}\nserves: []\nsource: {author: a}\ntimestamp: 2026-09-04T00:00:00Z\nacknowledgements: []\n")
	raw("int_cad.yaml", "id: int_cad\nsubject: https://s/\ntitle: t\nstability: tentative\nwindow: {clock: 09:00/12:00}\ncadence: FREQ=WEEKLY;BYDAY=TU\nserves: []\nsource: {author: a}\ntimestamp: 2026-09-04T00:00:00Z\nacknowledgements: []\n")
	raw("int_r.yaml", "acknowledgements: []\ntimestamp: 2026-09-04T00:00:00Z\nsource: {author: a}\nserves: []\nstability: tentative\ntitle: reversed\nsubject: https://s/\nid: int_r\n")
	r := run(t, ws)
	byID := map[string][]string{}
	for _, f := range r.Findings {
		byID[f.ID] = append(byID[f.ID], f.Code)
	}
	want := map[string]string{"int_k": "invalid", "int_q": "unparseable", "int_c": "unparseable", "int_h": "no_author", "int_m": "id_mismatch", "int_fu": "firmed_under_target", "int_cond": "policy_not_terminus", "int_cad": "cadence_anchor"}
	for id, code := range want {
		ok := false
		for _, c := range byID[id] {
			if c == code {
				ok = true
			}
		}
		if !ok {
			t.Errorf("%s: want %s, got %v", id, code, byID[id])
		}
	}
	if len(byID["int_r"]) != 1 || byID["int_r"][0] != "missing_version" {
		t.Errorf("reordered file: want only missing_version, got %v", byID["int_r"])
	}
	for _, c := range byID["int_ok"] {
		if c != "missing_version" {
			t.Errorf("authorised firming reported: %v", byID["int_ok"])
		}
	}
	if !has(r, SeverityWarning, "missing_version") {
		t.Error("missing version not warned")
	}
	if !has(r, SeverityError, "harness_firm") {
		t.Error("harness-firmed intention without firmed_under not an error")
	}
	if !has(r, SeverityError, "unreadable") {
		t.Error("unreadable file not reported")
	}
	for _, f := range r.Findings {
		if f.Code == "unparseable" && f.ID == "int_c" && !strings.Contains(f.Message, "clock") {
			t.Errorf("BYHOUR message should name clock: %s", f.Message)
		}
	}
}

func TestInstancesVersionsIndexTerms(t *testing.T) {
	ws := newWS(t)
	s := intention("int_s")
	c, _ := temporal.ParseCadence("FREQ=WEEKLY;BYDAY=TU")
	s.Cadence = &c
	i1, i2 := intention("int_i1"), intention("int_i2")
	for _, i := range []*model.Intention{i1, i2} {
		i.Serves = []model.Ref{{ID: "int_s", Role: model.RoleInstanceOf}}
		i.Occurrence = "2026-09-15"
	}
	i1.Activity = "piano-practice"
	write(t, ws, s, i1, i2)
	r := run(t, ws)
	if !has(r, SeverityError, "duplicate_instance") {
		t.Errorf("duplicate instance not reported: %+v", r.Findings)
	}
	if !has(r, SeverityInfo, "lonely_term") {
		t.Errorf("lonely term not reported")
	}
	// Retire one instance: no duplicate.
	i2.Retired = &model.Retired{Kind: "abandoned", Timestamp: time.Now(), Source: model.Source{Author: "a"}}
	write(t, ws, i2)
	if r := run(t, ws); has(r, SeverityError, "duplicate_instance") {
		t.Error("retired duplicate reported")
	}
	// Hand-edit: stale version and index drift.
	path := ws.Path(model.TypeIntention, "int_i1")
	data, _ := os.ReadFile(path)
	_ = os.WriteFile(path, []byte(strings.Replace(string(data), "stability: tentative", "stability: firm", 1)), 0o644)
	r = run(t, ws)
	if !has(r, SeverityWarning, "stale_version") || !has(r, SeverityWarning, "index_drift") {
		t.Errorf("stale/drift not reported: %+v", r.Findings)
	}
	// Absent conventions file.
	_ = os.Remove(filepath.Join(ws.Root, store.ConventionsFile))
	if r := run(t, ws); !has(r, SeverityInfo, "no_conventions") {
		t.Error("absent conventions not reported")
	}
	// Transparent resolution-born commitment via raw file.
	_ = os.WriteFile(filepath.Join(ws.Root, "commitments", "cmt_t.yaml"), []byte("id: cmt_t\nparties:\n  - uri: https://x/\n    status: accepted\nplacement:\n  start: 2026-09-21\n  duration: PT4H\norigin:\n  resolution: res_x\ntransparent: true\nsource: {author: a}\ntimestamp: 2026-09-04T00:00:00Z\nacknowledgements: []\n"), 0o644)
	r = run(t, ws)
	if !has(r, SeverityError, "transparent_origin") || !has(r, SeverityError, "all_day_duration") || !has(r, SeverityError, "dangling") {
		t.Errorf("commitment errors: %+v", codes(r, SeverityError))
	}
}
