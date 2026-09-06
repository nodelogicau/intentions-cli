// Package consistency computes the flags that say how the prospective graph
// fails to hang together. It never decides and never writes: flags are
// recomputed on every check, and what persists is the person's
// acknowledgement, which suppresses a flag for exactly the counterpart state
// it named.
package consistency

import (
	"fmt"
	"sort"
	"strings"

	"github.com/nodelogicau/intentions-cli/internal/model"
	"github.com/nodelogicau/intentions-cli/internal/projection"
	"github.com/nodelogicau/intentions-cli/internal/resolve"
	"github.com/nodelogicau/intentions-cli/internal/temporal"
)

// Flag kinds.
const (
	WindowClash            = "window-clash"
	ConditionMismatch      = "condition-mismatch"
	LocationMismatch       = "location-mismatch"
	ExpiredGround          = "expired-ground"
	IntentionInconsistency = "intention-inconsistency"
	PartyDeclined          = "party-declined"
	Cycle                  = "cycle"
)

// Kinds lists the seven flag kinds.
var Kinds = []string{WindowClash, ConditionMismatch, LocationMismatch, ExpiredGround, IntentionInconsistency, PartyDeclined, Cycle}

// ValidKind reports whether k is a flag kind.
func ValidKind(k string) bool {
	for _, x := range Kinds {
		if x == k {
			return true
		}
	}
	return false
}

// Flag is one finding of the check.
type Flag struct {
	Kind               string `json:"kind"`
	Subject            string `json:"subject"`
	Counterpart        string `json:"counterpart,omitempty"`
	CounterpartVersion string `json:"counterpart_version,omitempty"`
	Detail             string `json:"detail"`
}

// Check computes every flag over the workspace, suppresses those a current
// acknowledgement covers, and returns them sorted by subject, kind,
// counterpart. When ids is non-empty only flags whose subject or counterpart
// is one of them are returned.
func Check(e resolve.Env, ids []string) ([]Flag, error) {
	var flags []Flag
	flags = append(flags, cycles(e)...)
	placed := e.PlacedObjects()
	for _, p := range placed {
		flags = append(flags, clashes(e, p, placed)...)
		fs, err := grounds(e, p)
		if err != nil {
			return nil, err
		}
		flags = append(flags, fs...)
	}
	flags = append(flags, declines(e)...)
	inc, err := inconsistencies(e)
	if err != nil {
		return nil, err
	}
	flags = append(flags, inc...)
	flags = suppress(e, flags)
	if len(ids) > 0 {
		want := map[string]bool{}
		for _, id := range ids {
			want[id] = true
		}
		var kept []Flag
		for _, f := range flags {
			if want[f.Subject] || (f.Counterpart != "" && want[f.Counterpart]) {
				kept = append(kept, f)
			}
		}
		flags = kept
	}
	sort.Slice(flags, func(i, j int) bool {
		a, b := flags[i], flags[j]
		if a.Subject != b.Subject {
			return a.Subject < b.Subject
		}
		if a.Kind != b.Kind {
			return a.Kind < b.Kind
		}
		return a.Counterpart < b.Counterpart
	})
	if flags == nil {
		flags = []Flag{}
	}
	return flags, nil
}

// declines reports party-declined: a party has said no while the intention
// the commitment fulfils is still placed, so the person has refused a thing
// they still intend at that hour. An imported commitment names no intention
// and disagrees with nothing. The flag decides nothing; the person cancels,
// replaces, or acknowledges and goes ahead without them.
func declines(e resolve.Env) []Flag {
	var out []Flag
	for _, c := range e.G.Commitments() {
		if c.Retired != nil || c.Intention == "" {
			continue
		}
		obj, ok := e.G.Get(c.Intention)
		if !ok {
			continue
		}
		in, ok := obj.(*model.Intention)
		if !ok || in.Retired != nil || in.Placement == nil {
			continue
		}
		cv, iv := versionOf(e, c.ID), versionOf(e, in.ID)
		for _, p := range c.Parties {
			if p.Status != "declined" {
				continue
			}
			who := "a counterparty"
			if p.URI == in.Subject {
				who = "the subject"
			}
			detail := fmt.Sprintf("%s (%s) has declined while %s is still placed at %s", p.URI, who, in.ID, in.Placement.Start.Raw)
			out = append(out, Flag{Kind: PartyDeclined, Subject: c.ID, Counterpart: in.ID, CounterpartVersion: iv, Detail: detail})
			out = append(out, Flag{Kind: PartyDeclined, Subject: in.ID, Counterpart: c.ID, CounterpartVersion: cv, Detail: detail})
		}
	}
	return out
}

func versionOf(e resolve.Env, id string) string {
	obj, ok := e.G.Get(id)
	if !ok {
		return ""
	}
	v, _ := projection.Version(obj)
	return v
}

func cycles(e resolve.Env) []Flag {
	var out []Flag
	for _, in := range e.G.Intentions() {
		if cyc := e.InCycle(in.ID); cyc != nil {
			out = append(out, Flag{Kind: Cycle, Subject: in.ID, Detail: "in a serves cycle with " + strings.Join(cyc, ", ") + "; unresolvable until the cycle is broken"})
		}
	}
	return out
}

// clashes reports every opaque placed object sharing a particular whose
// interval overlaps p's, on p; the symmetric flag arises when the other
// object is visited.
func clashes(e resolve.Env, p resolve.Placed, placed []resolve.Placed) []Flag {
	var out []Flag
	for _, q := range placed {
		if q.ID == p.ID {
			continue
		}
		shares := ""
		for _, u := range p.Particulars {
			if q.Involves(u) {
				shares = u
				break
			}
		}
		if shares == "" || !p.Interval.Overlaps(q.Interval) {
			continue
		}
		out = append(out, Flag{Kind: WindowClash, Subject: p.ID, Counterpart: q.ID, CounterpartVersion: versionOf(e, q.ID), Detail: fmt.Sprintf("overlaps %s (%s to %s) on %s's time", q.ID, fmtT(q.Interval.Start), fmtT(q.Interval.End), shares)})
	}
	return out
}

func fmtT(t interface{ Format(string) string }) string { return t.Format("2006-01-02T15:04:05Z07:00") }

// grounds checks what the placement rests on for each involved particular:
// no containing availability at all is a window-clash with no counterpart;
// containing availabilities that all fail are reported by their failure.
func grounds(e resolve.Env, p resolve.Placed) ([]Flag, error) {
	var out []Flag
	activity, intentLoc := p.Activity, []string(nil)
	if in, ok := p.Object.(*model.Intention); ok {
		intentLoc = in.Location
	} else if c, ok := p.Object.(*model.Commitment); ok && c.Intention != "" {
		if obj, ok := e.G.Get(c.Intention); ok {
			if ii, ok := obj.(*model.Intention); ok {
				intentLoc = ii.Location
			}
		}
	}
	// The placement's location must be one of the intention's, when it has any.
	if len(intentLoc) > 0 && p.Location != "" && !contains(intentLoc, p.Location) {
		out = append(out, Flag{Kind: LocationMismatch, Subject: p.ID, Detail: fmt.Sprintf("placement location %s is outside the intention's %v", p.Location, intentLoc)})
	}
	for _, uri := range p.Particulars {
		var containing []*model.Availability
		for _, av := range e.G.Availabilities() {
			if av.Subject != uri || av.Window == nil {
				continue
			}
			occ, err := e.OccasionsOf(av, p.Interval)
			if err != nil {
				return nil, err
			}
			for _, o := range occ {
				if o.Interval.Contains(p.Interval) {
					containing = append(containing, av)
					break
				}
			}
		}
		if len(containing) == 0 {
			out = append(out, Flag{Kind: WindowClash, Subject: p.ID, Detail: fmt.Sprintf("falls where %s has no availability at all", uri)})
			continue
		}
		type verdict struct {
			av   *model.Availability
			kind string
			why  string
		}
		var verdicts []verdict
		supported := false
		for _, av := range containing {
			switch {
			case av.Retired != nil:
				verdicts = append(verdicts, verdict{av, ExpiredGround, "retired (" + av.Retired.Kind + ")"})
			case e.Expired(av):
				verdicts = append(verdicts, verdict{av, ExpiredGround, "expired at " + fmtT(e.EffectiveValidUntil(av))})
			case !resolve.ConditionalAdmits(av, activity):
				verdicts = append(verdicts, verdict{av, ConditionMismatch, fmt.Sprintf("conditional %v does not include activity %q", av.Conditional, activity)})
			case len(av.Location) > 0 && p.Location != "" && !contains(av.Location, p.Location):
				verdicts = append(verdicts, verdict{av, LocationMismatch, fmt.Sprintf("placement location %s is outside the availability's %v", p.Location, av.Location)})
			default:
				supported = true
			}
		}
		if supported {
			continue
		}
		for _, v := range verdicts {
			out = append(out, Flag{Kind: v.kind, Subject: p.ID, Counterpart: v.av.ID, CounterpartVersion: versionOf(e, v.av.ID), Detail: v.av.ID + " contains the placement but " + v.why})
		}
	}
	return out, nil
}

func contains(vs []string, v string) bool {
	for _, x := range vs {
		if x == v {
			return true
		}
	}
	return false
}

// inconsistencies is the pairwise check: two active unplaced intentions of
// one subject that each have candidates alone but no non-overlapping pair.
func inconsistencies(e resolve.Env) ([]Flag, error) {
	type cand struct {
		in  *model.Intention
		ivs []temporal.Interval
	}
	bySubject := map[string][]cand{}
	for _, in := range e.G.Intentions() {
		if in.Retired != nil || in.Placement != nil || in.Duration == nil || in.Window == nil || in.IsRecurring() {
			continue
		}
		if e.InCycle(in.ID) != nil {
			continue
		}
		res, err := resolve.Resolve(e, in, resolve.Options{})
		if err != nil || len(res.All()) == 0 {
			continue
		}
		var ivs []temporal.Interval
		for _, c := range res.All() {
			ivs = append(ivs, c.Interval)
		}
		bySubject[in.Subject] = append(bySubject[in.Subject], cand{in, ivs})
	}
	var out []Flag
	for _, cs := range bySubject {
		for i := 0; i < len(cs); i++ {
			for j := i + 1; j < len(cs); j++ {
				if compatible(cs[i].ivs, cs[j].ivs) {
					continue
				}
				a, b := cs[i].in, cs[j].in
				out = append(out,
					Flag{Kind: IntentionInconsistency, Subject: a.ID, Counterpart: b.ID, CounterpartVersion: versionOf(e, b.ID), Detail: "no pair of candidates places both " + a.ID + " and " + b.ID + " within their windows (pairwise)"},
					Flag{Kind: IntentionInconsistency, Subject: b.ID, Counterpart: a.ID, CounterpartVersion: versionOf(e, a.ID), Detail: "no pair of candidates places both " + b.ID + " and " + a.ID + " within their windows (pairwise)"})
			}
		}
	}
	return out, nil
}

func compatible(a, b []temporal.Interval) bool {
	for _, x := range a {
		for _, y := range b {
			if !x.Overlaps(y) {
				return true
			}
		}
	}
	return false
}

// suppress drops flags the subject has acknowledged against the
// counterpart's current version.
func suppress(e resolve.Env, flags []Flag) []Flag {
	var out []Flag
	for _, f := range flags {
		if !acknowledged(e, f) {
			out = append(out, f)
		}
	}
	return out
}

func acknowledged(e resolve.Env, f Flag) bool {
	obj, ok := e.G.Get(f.Subject)
	if !ok {
		return false
	}
	var acks []model.Acknowledgement
	switch o := obj.(type) {
	case *model.Intention:
		acks = o.Acknowledgements
	case *model.Commitment:
		acks = o.Acknowledgements
	}
	for _, a := range acks {
		if a.Kind != f.Kind || a.Counterpart != f.Counterpart {
			continue
		}
		if f.Counterpart == "" || a.CounterpartVersion == f.CounterpartVersion {
			return true
		}
	}
	return false
}
