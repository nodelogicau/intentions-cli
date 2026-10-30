package model

import (
	"fmt"
	"sort"
	"strings"
)

// GroundKind says how an intention's serves graph ends.
type GroundKind int

const (
	// Served: a firm, active terminus of the intention's own subject was reached.
	Served GroundKind = iota
	// Draft: the only termini reached are tentative, so nothing grounds it yet.
	Draft
	// Foreign: the only termini reached belong to another subject.
	Foreign
	// Ends: every path ends on something that is not a terminus of any kind.
	Ends
)

// Ground is the outcome of walking an intention's serves graph outward.
type Ground struct {
	Kind GroundKind
	// Terminus is the firm terminus reached, when Kind is Served.
	Terminus string
	// Drafts are the tentative termini of the subject reached.
	Drafts []string
	// Foreign are termini of other subjects reached.
	Foreign []string
	// Retired are the retired termini reached, which ground nothing; a
	// superseded one is not among them, since the walk follows its successor.
	Retired []string
	// Ends are the intentions the chain ends on that are not termini: a
	// scheduled intention with no serves, or a dangling id.
	Ends []string
}

// Grounding walks the serves graph outward from o through every role and
// reports what it reaches. A firm, active terminus of o's own subject wins
// outright. The walk passes through retired intentions: one retired as
// superseded hands the walk to its superseded_by, any other retired
// intention is walked through its own serves, and a retired terminus grounds
// nothing. An intention already visited, by serves or by superseded_by, ends
// that path: a serves cycle is a validation error in its own right and is
// refused at write, and a supersession loop simply grounds nothing.
func Grounding(g Graph, o *Intention) Ground {
	var gr Ground
	seen := map[string]bool{o.ID: true}
	var walk func(refs []Ref) bool
	var reach func(id string) bool
	reach = func(id string) bool {
		if seen[id] {
			return false
		}
		seen[id] = true
		obj, ok := g.Get(id)
		t, isIntention := obj.(*Intention)
		if !ok || !isIntention {
			gr.Ends = append(gr.Ends, id)
			return false
		}
		if t.Retired != nil && t.Retired.Kind == "superseded" && t.Retired.SupersededBy != "" {
			return reach(t.Retired.SupersededBy)
		}
		if t.IsTerminus() {
			switch {
			case t.Retired != nil:
				gr.Retired = append(gr.Retired, t.ID)
			case t.Subject != o.Subject:
				gr.Foreign = append(gr.Foreign, t.ID)
			case t.Stability != "firm":
				gr.Drafts = append(gr.Drafts, t.ID)
			default:
				gr.Terminus = t.ID
				return true
			}
			return false
		}
		if len(t.Serves) == 0 {
			gr.Ends = append(gr.Ends, t.ID)
			return false
		}
		return walk(t.Serves)
	}
	walk = func(refs []Ref) bool {
		for _, r := range refs {
			if reach(r.ID) {
				return true
			}
		}
		return false
	}
	if walk(o.Serves) {
		gr.Kind = Served
		return gr
	}
	switch {
	case len(gr.Drafts) > 0:
		gr.Kind = Draft
	case len(gr.Foreign) > 0:
		gr.Kind = Foreign
	default:
		gr.Kind = Ends
	}
	sort.Strings(gr.Drafts)
	sort.Strings(gr.Foreign)
	sort.Strings(gr.Retired)
	sort.Strings(gr.Ends)
	return gr
}

// FirmTermini returns the ids of the active, firm termini of subject among
// the intentions given, sorted: the termini an unserved intention could serve.
func FirmTermini(intentions []*Intention, subject string) []string {
	var out []string
	for _, in := range intentions {
		if in.Retired == nil && in.Subject == subject && in.IsTerminus() && in.Stability == "firm" {
			out = append(out, in.ID)
		}
	}
	sort.Strings(out)
	return out
}

// Unserved reports why o reaches no firm terminus of its own subject, in a
// message that names the fix, or "" when it is served or is itself a
// terminus. termini are the firm termini of o's subject the workspace holds
// (see FirmTermini), which the message offers when there are any.
func Unserved(g Graph, o *Intention, termini []string) string {
	if o.IsTerminus() {
		return ""
	}
	gr := Grounding(g, o)
	if gr.Kind == Served {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "reaches no firm terminus of %s", o.Subject)
	switch gr.Kind {
	case Draft:
		fmt.Fprintf(&b, "; it reaches only the draft terminus %s, which grounds nothing until the person firms it with `intentions intention firm %s` (an author, no harness)", strings.Join(gr.Drafts, ", "), gr.Drafts[0])
	case Foreign:
		fmt.Fprintf(&b, "; it reaches only %s, which belongs to another subject, and a terminus grounds only its own subject's intentions", strings.Join(gr.Foreign, ", "))
		if len(gr.Retired) > 0 {
			fmt.Fprintf(&b, ", and the retired terminus %s, which grounds nothing", strings.Join(gr.Retired, ", "))
		}
	default:
		if len(gr.Retired) > 0 {
			fmt.Fprintf(&b, "; it reaches only the retired terminus %s, which grounds nothing", strings.Join(gr.Retired, ", "))
		} else if len(o.Serves) == 0 {
			b.WriteString("; it serves nothing")
		} else if len(gr.Ends) > 0 {
			fmt.Fprintf(&b, "; its chain ends on %s, which serves nothing", strings.Join(gr.Ends, ", "))
		}
	}
	if len(termini) > 0 {
		fmt.Fprintf(&b, ". Add --serves %s:for-the-sake-of, or serve an intention that reaches a terminus", termini[0])
	} else if gr.Kind != Draft {
		b.WriteString(". The workspace holds no firm terminus of this subject: add one first, titled as who the person is, and have them firm it")
	}
	return b.String()
}
