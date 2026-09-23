package model

import (
	"strings"
	"testing"
	"time"

	"github.com/nodelogicau/intentions-cli/internal/temporal"
)

func TestDesireDecodeEncode(t *testing.T) {
	src := "id: des_a\nsubject: https://s/\ntitle: call the accountant\nactivity: admin\nserves:\n  - {id: int_t, role: for-the-sake-of}\nsource: {author: a}\ntimestamp: 2026-09-04T00:00:00Z\n"
	obj, probs, err := Decode([]byte(src), TypeDesire)
	if err != nil || len(probs) != 0 {
		t.Fatalf("decode: %v %v", err, probs)
	}
	d := obj.(*Desire)
	if d.Title != "call the accountant" || d.Activity != "admin" || len(d.Serves) != 1 || d.Serves[0].Role != RoleForTheSakeOf {
		t.Errorf("fields: %+v", d)
	}
	out, err := Encode(d)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(out), "id: des_a\nsubject: https://s/\ntitle: call the accountant\nactivity: admin\nserves:\n") || !strings.Contains(string(out), "\nsource:\n") {
		t.Errorf("canonical order:\n%s", out)
	}
	if ps := Check(d); len(ps) != 0 {
		t.Errorf("check: %v", ps)
	}
	// A forbidden field is named, not kept.
	obj, probs, err = Decode([]byte(src+"window: {calendar: 2026-W40}\nstability: firm\n"), TypeDesire)
	if err != nil {
		t.Fatal(err)
	}
	codes := map[string]bool{}
	for _, p := range probs {
		codes[p.Code+":"+p.Field] = true
	}
	if !codes["forbidden_field:window"] || !codes["forbidden_field:stability"] || len(obj.(*Desire).Extras) != 0 {
		t.Errorf("forbidden fields: %v extras=%v", probs, obj.(*Desire).Extras)
	}
	// An unknown key is still an extra.
	obj, probs, _ = Decode([]byte(src+"notes: x\n"), TypeDesire)
	if len(probs) != 0 || len(obj.(*Desire).Extras) != 1 {
		t.Errorf("unknown key: %v %v", probs, obj.(*Desire).Extras)
	}
	// Ids and types.
	if typ, ok := TypeOfID("des_01j9xk2p3q4r5s6t"); !ok || typ != TypeDesire {
		t.Errorf("TypeOfID: %v %v", typ, ok)
	}
	if !strings.HasPrefix(MintID(TypeDesire), "des_") || TypeDesire.Dir() != "desires" {
		t.Error("prefix or dir")
	}
}

func TestDesireRules(t *testing.T) {
	term := intention("int_t")
	other := intention("int_o")
	other.Subject = "https://other/"
	sched := intention("int_s")
	sched.Duration = &temporal.DurationSpec{Nominal: temporal.Duration{Hours: 1}}
	g := fakeGraph{objs: map[string]Object{"int_t": term, "int_o": other, "int_s": sched}}
	d := &Desire{ID: "des_a", Subject: "https://s/", Title: "t", Source: Source{Author: "a"}, Timestamp: time.Now(), Serves: []Ref{}}
	cases := []struct {
		serves []Ref
		want   string
	}{
		{[]Ref{{ID: "int_t", Role: RoleForTheSakeOf}}, ""},
		{[]Ref{{ID: "int_t", Role: RoleInOrderTo}}, "for-the-sake-of"},
		{[]Ref{{ID: "int_s", Role: RoleForTheSakeOf}}, "not a terminus"},
		{[]Ref{{ID: "int_o", Role: RoleForTheSakeOf}}, "belongs to"},
		{[]Ref{{ID: "int_zz", Role: RoleForTheSakeOf}}, "does not exist"},
	}
	for _, c := range cases {
		d.Serves = c.serves
		err := CheckDesireServes(g, d)
		if c.want == "" && err != nil {
			t.Errorf("%v: %v", c.serves, err)
		}
		if c.want != "" && (err == nil || !strings.Contains(err.Error(), c.want)) {
			t.Errorf("%v: want %q, got %v", c.serves, c.want, err)
		}
	}
	// A tentative terminus is fine for a desire.
	term.Stability = "tentative"
	d.Serves = []Ref{{ID: "int_t", Role: RoleForTheSakeOf}}
	if err := CheckDesireServes(g, d); err != nil {
		t.Errorf("draft terminus: %v", err)
	}
	// Retirement: adopted never by hand; superseded_by must be a desire.
	if err := CheckRetirement(g, d, Retired{Kind: "adopted", Source: Source{Author: "a"}, Timestamp: time.Now()}); err == nil || !strings.Contains(err.Error(), "adoption") {
		t.Errorf("adopted by hand: %v", err)
	}
	if err := CheckRetirement(g, d, Retired{Kind: "superseded", SupersededBy: "int_t", Source: Source{Author: "a"}, Timestamp: time.Now()}); err == nil || !strings.Contains(err.Error(), "not a desire") {
		t.Errorf("superseded by an intention: %v", err)
	}
	if err := CheckRetirement(g, d, Retired{Kind: "abandoned", Source: Source{Author: "a"}, Timestamp: time.Now()}); err != nil {
		t.Errorf("abandoned: %v", err)
	}
	// Structural check on the record shape.
	d.Retired = &Retired{Kind: "adopted", Source: Source{Author: "a"}, Timestamp: time.Now()}
	if ps := Check(d); !strings.Contains(ps.Error(), "adopted_as") {
		t.Errorf("adopted without pointer: %v", ps)
	}
	d.Retired = nil
}

func TestAdopt(t *testing.T) {
	act := Source{Author: "a", Harness: "claude"}
	at := time.Now()
	d := &Desire{ID: "des_a", Subject: "https://s/", Title: "call the accountant", Description: "soon", Activity: "admin", Location: []string{"geo:1,2"}, Reference: "dkf:x", Source: Source{Author: "a"}, Timestamp: at, Serves: []Ref{}}
	// Bare: refused, naming a why or a when.
	if _, _, err := Adopt(d, nil, nil, act, at); err == nil || !strings.Contains(err.Error(), "a why") || !strings.Contains(err.Error(), "a when") {
		t.Errorf("bare adoption: %v", err)
	}
	// A when and no why: accepted.
	dur := &temporal.DurationSpec{Nominal: temporal.Duration{Minutes: 30}}
	in, r, err := Adopt(d, dur, nil, act, at)
	if err != nil {
		t.Fatal(err)
	}
	if in.Stability != "tentative" || in.Title != d.Title || in.Description != "soon" || in.Activity != "admin" || in.Reference != "dkf:x" || len(in.Location) != 1 || in.Duration != dur || in.Source != act || !strings.HasPrefix(in.ID, "int_") {
		t.Errorf("adopted intention: %+v", in)
	}
	if r.Kind != "adopted" || r.AdoptedAs != in.ID || r.Source != act {
		t.Errorf("retirement: %+v", r)
	}
	// A why and no when: accepted, serves copied.
	d.Serves = []Ref{{ID: "int_t", Role: RoleForTheSakeOf}}
	in, _, err = Adopt(d, nil, nil, act, at)
	if err != nil || len(in.Serves) != 1 || in.IsTerminus() {
		t.Errorf("why and no when: %v %+v", err, in)
	}
	// Retired: refused.
	d.Retired = &Retired{Kind: "abandoned"}
	if _, _, err := Adopt(d, dur, nil, act, at); err == nil || !strings.Contains(err.Error(), "retired") {
		t.Errorf("retired: %v", err)
	}
}
