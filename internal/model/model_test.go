package model

import (
	"bytes"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/nodelogicau/intentions-cli/internal/temporal"
)

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestGoldenRoundTrip(t *testing.T) {
	cases := map[string]Type{
		"intention.yaml":    TypeIntention,
		"availability.yaml": TypeAvailability,
		"commitment.yaml":   TypeCommitment,
		"resolution.yaml":   TypeResolution,
		"retired.yaml":      TypeIntention,
	}
	for name, typ := range cases {
		in := readFixture(t, name)
		obj, probs, err := Decode(in, typ)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if len(probs) > 0 {
			t.Fatalf("%s: problems %v", name, probs)
		}
		if ps := Check(obj); len(ps) > 0 {
			t.Errorf("%s: check %v", name, ps)
		}
		out, err := Encode(obj)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(in, out) {
			t.Errorf("%s: round trip differs\n--- in\n%s\n--- out\n%s", name, in, out)
		}
		// Second pass is byte-identical too.
		obj2, _, _ := Decode(out, typ)
		out2, _ := Encode(obj2)
		if !bytes.Equal(out, out2) {
			t.Errorf("%s: second pass differs", name)
		}
	}
}

func TestReorderedAndExtras(t *testing.T) {
	in := []byte(`notes: keep me
timestamp: 2026-09-04T09:12:00Z
title: Reordered
window:
  clock: 09:00/12:00
  calendar: 2026-W36
source:
  model: m
  author: https://example.com/people/ada
stability: tentative
serves: []
acknowledgements: []
location: [https://b.example/, https://a.example/]
subject: https://example.com/people/ada
id: int_01a06d10-4c2e-7a91-b3f0-2d8e1a7c5b44
version: sha256:stale
`)
	obj, probs, err := Decode(in, TypeIntention)
	if err != nil || len(probs) > 0 {
		t.Fatalf("%v %v", err, probs)
	}
	if ps := Check(obj); len(ps) > 0 {
		t.Fatalf("check: %v", ps)
	}
	out, _ := Encode(obj)
	want := `id: int_01a06d10-4c2e-7a91-b3f0-2d8e1a7c5b44
subject: https://example.com/people/ada
title: Reordered
window:
  calendar: 2026-W36
  clock: 09:00/12:00
stability: tentative
location:
  - https://a.example/
  - https://b.example/
serves: []
source:
  author: https://example.com/people/ada
  model: m
timestamp: 2026-09-04T09:12:00Z
acknowledgements: []
notes: keep me
version: sha256:stale
`
	if string(out) != want {
		t.Errorf("canonical re-emit:\n%s\nwant\n%s", out, want)
	}
}

func TestDecodeErrors(t *testing.T) {
	if _, _, err := Decode([]byte("- a\n- b\n"), TypeIntention); err == nil {
		t.Error("sequence document accepted")
	}
	if _, _, err := Decode([]byte("id: a\nid: b\n"), TypeIntention); err == nil {
		t.Error("duplicate key accepted")
	}
	obj, probs, err := Decode([]byte("id: int_x\nduration: 90m\nwindow:\n  calendar: 2026-09~\ncadence: FREQ=DAILY;BYHOUR=9\n"), TypeIntention)
	if err != nil || obj == nil {
		t.Fatal(err)
	}
	codes := map[string]bool{}
	for _, p := range probs {
		codes[p.Field] = true
	}
	for _, f := range []string{"duration", "window.calendar", "cadence"} {
		if !codes[f] {
			t.Errorf("missing problem for %s: %v", f, probs)
		}
	}
}

func TestCheckRules(t *testing.T) {
	base := func() *Intention {
		return &Intention{ID: "int_a", Subject: "https://example.com/x", Title: "t", Stability: "tentative", Serves: []Ref{},
			Source: Source{Author: "https://example.com/x"}, Timestamp: time.Now(), Acknowledgements: []Acknowledgement{}}
	}
	if ps := Check(base()); len(ps) > 0 {
		t.Fatalf("base: %v", ps)
	}
	fieldOf := func(ps Problems) string {
		var fs []string
		for _, p := range ps {
			fs = append(fs, p.Field+":"+p.Code)
		}
		sort.Strings(fs)
		return strings.Join(fs, ",")
	}
	o := base()
	o.Activity = "Deep Work"
	if got := fieldOf(Check(o)); got != "activity:malformed_term" {
		t.Errorf("term: %s", got)
	}
	o = base()
	o.Stability = "maybe"
	if got := fieldOf(Check(o)); got != "stability:invalid" {
		t.Errorf("stability: %s", got)
	}
	o = base()
	o.Source = Source{Harness: "claude"}
	if got := fieldOf(Check(o)); got != "source.author:no_author" {
		t.Errorf("author: %s", got)
	}
	o = base()
	o.Location = []string{"home"}
	if got := fieldOf(Check(o)); got != "location[0]:invalid" {
		t.Errorf("uri: %s", got)
	}
	o = base()
	o.Serves = []Ref{{ID: "int_b", Role: "PARENT"}}
	if got := fieldOf(Check(o)); got != "serves[0].role:invalid" {
		t.Errorf("role: %s", got)
	}
	o = base()
	o.Retired = &Retired{Kind: "superseded", Timestamp: time.Now()}
	if got := fieldOf(Check(o)); got != "retired.superseded_by:superseded_by" {
		t.Errorf("superseded: %s", got)
	}
	o = base()
	o.Retired = &Retired{Kind: "cancelled", Timestamp: time.Now()}
	if got := fieldOf(Check(o)); got != "retired.kind:invalid" {
		t.Errorf("kind: %s", got)
	}
	o = base()
	o.Preference = "soonest"
	if got := fieldOf(Check(o)); got != "preference:invalid" {
		t.Errorf("preference: %s", got)
	}
	o = base()
	s, _ := temporal.ParsePlacementStart("2026-09-21")
	o.Placement = &temporal.Placement{Start: s, Duration: temporal.DurationSpec{Nominal: temporal.Duration{Hours: 4}}}
	if got := fieldOf(Check(o)); got != "placement:all_day_duration" {
		t.Errorf("all-day: %s", got)
	}
	// Commitment: transparent on a resolution-born commitment.
	tr := true
	c := &Commitment{ID: "cmt_a", Parties: []Party{{URI: "https://x/", Status: "accepted"}}, Placement: &temporal.Placement{Start: s, Duration: temporal.DurationSpec{Nominal: temporal.Duration{Days: 1}}},
		Origin: Origin{Resolution: "res_a"}, Transparent: &tr, Timestamp: time.Now(), Acknowledgements: []Acknowledgement{}}
	if got := fieldOf(Check(c)); got != "transparent:transparent_origin" {
		t.Errorf("transparent: %s", got)
	}
	c.Origin = Origin{Import: true}
	if ps := Check(c); len(ps) > 0 {
		t.Errorf("import transparent: %v", ps)
	}
}

// fakeGraph is a minimal Graph for policy tests.
type fakeGraph struct {
	objs map[string]Object
}

func (g fakeGraph) Get(id string) (Object, bool) { o, ok := g.objs[id]; return o, ok }
func (g fakeGraph) Inbound(id string) []Inbound {
	var in []Inbound
	for _, o := range g.objs {
		if i, ok := o.(*Intention); ok {
			for _, r := range i.Serves {
				if r.ID == id {
					in = append(in, Inbound{From: i.ID, Role: r.Role})
				}
			}
		}
	}
	return in
}

func intention(id string, serves ...Ref) *Intention {
	return &Intention{ID: id, Subject: "https://s/", Stability: "tentative", Serves: serves}
}

func TestServesPolicy(t *testing.T) {
	g := fakeGraph{objs: map[string]Object{}}
	a := intention("int_a", Ref{ID: "int_b", Role: RoleInOrderTo})
	b := intention("int_b", Ref{ID: "int_t", Role: RoleForTheSakeOf})
	tt := intention("int_t")
	for _, o := range []*Intention{a, b, tt} {
		g.objs[o.ID] = o
	}
	// b -> a would close a cycle.
	err := CheckServes(g, "int_b", []Ref{{ID: "int_t", Role: RoleForTheSakeOf}, {ID: "int_a", Role: RoleInOrderTo}})
	if err == nil || !strings.Contains(err.Error(), "cycle") || !strings.Contains(err.Error(), "int_a") {
		t.Errorf("cycle: %v", err)
	}
	// Terminus cannot gain serves.
	err = CheckServes(g, "int_t", []Ref{{ID: "int_a", Role: RoleInOrderTo}})
	if err == nil || !strings.Contains(err.Error(), "terminus") {
		t.Errorf("terminus sink: %v", err)
	}
	// for-the-sake-of target with serves refused.
	err = CheckServes(g, "int_t2", []Ref{{ID: "int_b", Role: RoleForTheSakeOf}})
	if err == nil || !strings.Contains(err.Error(), "terminus") {
		t.Errorf("terminus target: %v", err)
	}
	if err := CheckServes(g, "int_c", []Ref{{ID: "int_missing", Role: RoleInOrderTo}}); err == nil {
		t.Error("dangling accepted")
	}
	if err := CheckServes(g, "int_c", []Ref{{ID: "int_a", Role: RoleInOrderTo}, {ID: "int_b", Role: RoleInOrderTo}}); err != nil {
		t.Errorf("multiple ends: %v", err)
	}
}

func TestFirmPolicy(t *testing.T) {
	d30 := temporal.Duration{Minutes: 30}
	policy := intention("int_p")
	policy.AutoFirm = &Policy{MaxDuration: &d30}
	target := intention("int_x")
	target.Duration = &temporal.DurationSpec{Nominal: temporal.Duration{Minutes: 20}}
	g := fakeGraph{objs: map[string]Object{"int_p": policy, "int_x": target}}
	if err := CheckFirm(g, target, Source{Author: "a"}, ""); err != nil {
		t.Errorf("person: %v", err)
	}
	if err := CheckFirm(g, target, Source{Author: "a", Harness: "claude"}, ""); err == nil {
		t.Error("harness without policy accepted")
	}
	if err := CheckFirm(g, target, Source{Author: "a", Harness: "claude"}, "int_p"); err != nil {
		t.Errorf("harness under policy: %v", err)
	}
	target.Duration = &temporal.DurationSpec{Nominal: temporal.Duration{Hours: 2}}
	if err := CheckFirm(g, target, Source{Author: "a", Harness: "claude"}, "int_p"); err == nil || !strings.Contains(err.Error(), "max_duration") {
		t.Errorf("policy not met: %v", err)
	}
}

func TestAvailabilityTerms(t *testing.T) {
	d := temporal.DurationSpec{Nominal: temporal.Duration{Hours: 3}}
	c, _ := temporal.ParseCalendar("2026-09")
	old := &Availability{ID: "avl_a", Duration: &d, Window: &temporal.Window{Calendar: &c}, Conditional: []string{"deep-work"}, Scope: "personal"}
	edited := *old
	edited.Title = "renamed"
	if err := CheckAvailabilityTerms(old, &edited); err != nil {
		t.Errorf("prose: %v", err)
	}
	edited = *old
	edited.Conditional = []string{"writing"}
	if err := CheckAvailabilityTerms(old, &edited); err == nil || !strings.Contains(err.Error(), "supersede") {
		t.Errorf("conditional: %v", err)
	}
	k, _ := temporal.ParseClock("13:00/16:00")
	edited = *old
	edited.Window = &temporal.Window{Calendar: &c, Clock: &k}
	if err := CheckAvailabilityTerms(old, &edited); err == nil {
		t.Error("clock change accepted")
	}
	if err := CheckScope("organisation", "personal"); err == nil {
		t.Error("narrowing accepted")
	}
	if err := CheckScope("personal", "public"); err != nil {
		t.Errorf("widening: %v", err)
	}
}

func TestMintIDs(t *testing.T) {
	var ids []string
	for i := 0; i < 50; i++ {
		id := MintID(TypeIntention)
		if !StrictID(id) {
			t.Fatalf("not strict: %s", id)
		}
		ids = append(ids, id)
	}
	if !sort.StringsAreSorted(ids) {
		t.Error("burst-minted ids are not in order")
	}
	if typ, ok := TypeOfID("avl_01j9xk2p3q4r5s6t"); !ok || typ != TypeAvailability {
		t.Error("lenient parse failed")
	}
	if _, ok := TypeOfID("xyz_1"); ok {
		t.Error("unknown prefix accepted")
	}
}
