package model

import (
	"strings"
	"testing"

	"github.com/nodelogicau/intentions-cli/internal/temporal"
)

func scheduled(id string, serves ...Ref) *Intention {
	o := intention(id, serves...)
	o.Duration = &temporal.DurationSpec{Nominal: temporal.Duration{Hours: 1}}
	return o
}

func terminus(id, stability string) *Intention {
	o := intention(id)
	o.Stability = stability
	return o
}

func TestGrounding(t *testing.T) {
	firm := terminus("int_t", "firm")
	draft := terminus("int_d", "tentative")
	other := terminus("int_o", "firm")
	other.Subject = "https://other/"
	retired := terminus("int_r", "firm")
	retired.Retired = &Retired{Kind: "abandoned"}
	end := scheduled("int_end")
	means := scheduled("int_m", Ref{ID: "int_t", Role: RoleForTheSakeOf})
	recurring := scheduled("int_rec", Ref{ID: "int_m", Role: RoleInOrderTo})
	g := fakeGraph{objs: map[string]Object{"int_t": firm, "int_d": draft, "int_o": other, "int_r": retired, "int_end": end, "int_m": means, "int_rec": recurring}}
	termini := FirmTermini([]*Intention{firm, draft, other, retired, end, means}, "https://s/")
	if strings.Join(termini, ",") != "int_t" {
		t.Errorf("firm termini: %v", termini)
	}
	cases := []struct {
		name  string
		o     *Intention
		kind  GroundKind
		wants string
	}{
		{"direct", scheduled("int_a", Ref{ID: "int_t", Role: RoleForTheSakeOf}), Served, ""},
		{"through a means", scheduled("int_a", Ref{ID: "int_m", Role: RoleInOrderTo}), Served, ""},
		{"instance through recurring", scheduled("int_a", Ref{ID: "int_rec", Role: RoleInstanceOf}), Served, ""},
		{"serves nothing", scheduled("int_a"), Ends, "serves nothing"},
		{"ends on scheduled", scheduled("int_a", Ref{ID: "int_end", Role: RoleInOrderTo}), Ends, "ends on int_end"},
		{"only a draft", scheduled("int_a", Ref{ID: "int_d", Role: RoleForTheSakeOf}), Draft, "draft terminus int_d"},
		{"another subject", scheduled("int_a", Ref{ID: "int_o", Role: RoleForTheSakeOf}), Foreign, "another subject"},
		{"retired terminus", scheduled("int_a", Ref{ID: "int_r", Role: RoleForTheSakeOf}), Ends, "ends on int_r"},
		{"dangling", scheduled("int_a", Ref{ID: "int_zz", Role: RoleInOrderTo}), Ends, "ends on int_zz"},
		{"draft and firm both reached", scheduled("int_a", Ref{ID: "int_d", Role: RoleForTheSakeOf}, Ref{ID: "int_t", Role: RoleForTheSakeOf}), Served, ""},
	}
	for _, c := range cases {
		gr := Grounding(g, c.o)
		if gr.Kind != c.kind {
			t.Errorf("%s: kind %v, want %v (%+v)", c.name, gr.Kind, c.kind, gr)
		}
		msg := Unserved(g, c.o, termini)
		if c.kind == Served && msg != "" {
			t.Errorf("%s: served but message %q", c.name, msg)
		}
		if c.kind != Served && (!strings.Contains(msg, c.wants) || !strings.Contains(msg, "--serves int_t:for-the-sake-of")) {
			t.Errorf("%s: message %q", c.name, msg)
		}
	}
	// A terminus is never unserved; a draft names the firming command; with
	// no firm terminus the message says to add one first.
	if Unserved(g, firm, termini) != "" || Unserved(g, draft, termini) != "" {
		t.Error("terminus reported unserved")
	}
	if msg := Unserved(g, scheduled("int_a", Ref{ID: "int_d", Role: RoleForTheSakeOf}), nil); !strings.Contains(msg, "intentions intention firm int_d") || strings.Contains(msg, "add one first") {
		t.Errorf("draft message: %q", msg)
	}
	if msg := Unserved(g, scheduled("int_a"), nil); !strings.Contains(msg, "add one first") {
		t.Errorf("no termini message: %q", msg)
	}
	// A cycle is stopped, not walked forever.
	x := scheduled("int_x", Ref{ID: "int_y", Role: RoleInOrderTo})
	y := scheduled("int_y", Ref{ID: "int_x", Role: RoleInOrderTo})
	g.objs["int_x"], g.objs["int_y"] = x, y
	if gr := Grounding(g, scheduled("int_a", Ref{ID: "int_x", Role: RoleInOrderTo})); gr.Kind != Ends {
		t.Errorf("cycle: %+v", gr)
	}
}

func TestFirmTerminusIsThePersons(t *testing.T) {
	d30 := temporal.Duration{Minutes: 30}
	policy := intention("int_p")
	policy.AutoFirm = &Policy{MaxDuration: &d30}
	term := terminus("int_t", "tentative")
	g := fakeGraph{objs: map[string]Object{"int_p": policy, "int_t": term}}
	if fu, err := CheckFirm(g, term, Source{Author: "a"}, ""); err != nil || fu != "" {
		t.Errorf("person firms a terminus: %q %v", fu, err)
	}
	if _, err := CheckFirm(g, term, Source{Author: "a", Harness: "claude"}, ""); err == nil || !strings.Contains(err.Error(), "person's word") {
		t.Errorf("harness firms a terminus: %v", err)
	}
	if _, err := CheckFirm(g, term, Source{Author: "a", Harness: "claude"}, "int_p"); err == nil || !strings.Contains(err.Error(), "no policy applies to a terminus") {
		t.Errorf("policy on a terminus: %v", err)
	}
	// Even a person naming a policy is refused: no policy applies.
	if _, err := CheckFirm(g, term, Source{Author: "a"}, "int_p"); err == nil || !strings.Contains(err.Error(), "no policy applies") {
		t.Errorf("person with policy on a terminus: %v", err)
	}
}
