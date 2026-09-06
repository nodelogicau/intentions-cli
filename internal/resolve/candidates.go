package resolve

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/nodelogicau/intentions-cli/internal/apperr"
	"github.com/nodelogicau/intentions-cli/internal/model"
	"github.com/nodelogicau/intentions-cli/internal/temporal"
)

// Candidate is one placement resolution offers.
type Candidate struct {
	Rank      int               `json:"rank"`
	Start     string            `json:"start"`
	End       string            `json:"end"`
	Duration  string            `json:"duration"`
	Location  string            `json:"location,omitempty"`
	Displaces []string          `json:"displaces"`
	Supply    []string          `json:"supply"`
	Interval  temporal.Interval `json:"-"`
	prefKey   float64
}

// Result is what resolution returns.
type Result struct {
	Candidates []Candidate        `json:"candidates"`
	Considered int                `json:"candidates_considered"`
	Range      *temporal.Interval `json:"-"`
	RangeText  string             `json:"range,omitempty"`
	Reason     string             `json:"reason,omitempty"`
	Blocked    string             `json:"blocked_on,omitempty"`
	NoSupply   []string           `json:"no_supply,omitempty"`
	Excluded   []Exclusion        `json:"excluded,omitempty"`
	Generated  []string           `json:"generated"`
	all        []Candidate
}

// Options tune a resolution.
type Options struct {
	Limit       int    // candidates to return; 0 means all
	AllowPlaced bool   // resolve a placed intention (for --replace)
	Exclude     string // a placed object id to ignore for displacement and capacity (the intention itself on replace)
}

// Refuse is a precondition failure: the intention cannot be resolved as it
// stands, and the caller should say so with exit code 2.
func refuse(format string, args ...any) error { return apperr.Refused(format, args...) }

// Resolve computes ranked candidates for one intention. Precondition
// failures are errors; an empty candidate set with a reason is a result.
func Resolve(e Env, in *model.Intention, opts Options) (Result, error) {
	var res Result
	res.Candidates = []Candidate{}
	res.Generated = []string{}
	if in.Retired != nil {
		return res, refuse("%s is retired (%s) and cannot be resolved", in.ID, in.Retired.Kind)
	}
	if in.Duration == nil {
		return res, refuse("%s has no duration; a duration is required before resolution", in.ID)
	}
	if in.Window == nil {
		return res, refuse("%s has no window; a window is required before resolution", in.ID)
	}
	if in.Placement != nil && !opts.AllowPlaced {
		return res, refuse("%s is already placed at %s; pass --replace to re-resolve it", in.ID, in.Placement.Start.Raw)
	}
	if cyc := e.InCycle(in.ID); cyc != nil {
		return res, refuse("%s is in a serves cycle with %s and is unresolvable until the cycle is broken", in.ID, strings.Join(cyc, ", "))
	}
	r, reason := temporal.ResolutionRange(*in.Window, e.Ctx, e.Horizon)
	if reason != "" {
		res.Reason = reason
		return res, nil
	}
	res.Range, res.RangeText = &r, describe(r)

	// Relational anchor: a constraint when the target is placed, blocked otherwise.
	var constraint *temporal.Constraint
	if in.Window.Relative != nil {
		target, ok := e.PlacedInterval(in.Window.Relative.Target)
		if !ok {
			res.Blocked = in.Window.Relative.Target
			res.Reason = fmt.Sprintf("blocked on %s, which has no placement", in.Window.Relative.Target)
			return res, nil
		}
		c := temporal.RelativeBounds(*in.Window.Relative, target)
		constraint = &c
	}

	// Supply per particular, intersected.
	involved := append([]string{in.Subject}, in.Parties...)
	supplies := map[string]Supply{}
	var common []temporal.Interval
	for i, uri := range involved {
		s, err := e.SupplyFor(uri, in, r)
		if err != nil {
			return res, err
		}
		res.Excluded = append(res.Excluded, s.Excluded...)
		if len(s.Occasions) == 0 {
			res.NoSupply = append(res.NoSupply, uri)
			continue
		}
		supplies[uri] = s
		if i == 0 {
			common = s.Intervals()
		} else {
			common = temporal.IntersectSets(common, s.Intervals())
		}
	}
	if len(res.NoSupply) > 0 {
		res.Reason = "no eligible supply for " + strings.Join(res.NoSupply, ", ")
		return res, nil
	}
	// The intention's own clock.
	if in.Window.Clock != nil {
		ctx := e.Ctx
		ctx.Horizon = &r
		clockIvs, err := temporal.Bounds(temporal.Window{Clock: in.Window.Clock}, ctx)
		if err != nil {
			return res, err
		}
		common = temporal.IntersectSets(common, clockIvs)
	}
	common = temporal.Clamp(common, r)
	if len(common) == 0 {
		res.Reason = "supply for " + strings.Join(involved, ", ") + " does not intersect within " + describe(r)
		return res, nil
	}

	// Enumerate on the grid.
	placed := e.PlacedObjects()
	step := e.stepDuration()
	dur := in.Duration.Nominal
	var all []Candidate
	for _, iv := range common {
		for t := iv.Start; ; t = t.Add(step) {
			end := temporal.AddDuration(t, dur)
			if end.After(iv.End) {
				break
			}
			cand := temporal.Interval{Start: t, End: end}
			if constraint != nil && !constraint.Admits(cand) {
				continue
			}
			supply, ok := e.covers(cand, involved, supplies, placed, opts.Exclude)
			if !ok {
				continue
			}
			c := Candidate{Start: fmtTime(t), End: fmtTime(end), Duration: dur.String(), Interval: cand, Supply: supply, Displaces: []string{}}
			c.Location = chooseLocation(in, supplies, supply)
			c.Rank, c.Displaces = rankAgainst(cand, involved, placed, opts.Exclude)
			all = append(all, c)
		}
	}
	if all == nil {
		all = []Candidate{}
	}
	orderCandidates(all, e, in, placed)
	res.Considered = len(all)
	res.all = all
	if opts.Limit > 0 && len(all) > opts.Limit {
		res.Candidates = all[:opts.Limit]
	} else {
		res.Candidates = all
	}
	if len(all) == 0 {
		res.Reason = "no candidate fits: supply intersects but no grid position holds " + dur.String() + " within capacity"
		if constraint != nil {
			res.Reason += " and the relational anchor"
		}
	}
	return res, nil
}

// All returns every candidate before the limit was applied.
func (r Result) All() []Candidate { return r.all }

// covers checks that every involved particular has an occasion containing
// the candidate with capacity to spare, and returns the supplying
// availability ids.
func (e Env) covers(cand temporal.Interval, involved []string, supplies map[string]Supply, placed []Placed, exclude string) ([]string, bool) {
	seen := map[string]bool{}
	var ids []string
	need := cand.Length()
	for _, uri := range involved {
		found := false
		for _, o := range supplies[uri].Occasions {
			if !o.Interval.Contains(cand) {
				continue
			}
			if need > o.Capacity || e.Remaining(o, uri, placed, exclude) < need {
				continue
			}
			found = true
			if !seen[o.Availability.ID] {
				seen[o.Availability.ID] = true
				ids = append(ids, o.Availability.ID)
			}
		}
		if !found {
			return nil, false
		}
	}
	sort.Strings(ids)
	return ids, true
}

// rankAgainst assigns the reconsideration cost: 1 displaces nothing, 2 only
// tentative, 3 something firm or accepted. Where a rung names a commitment's
// status it means the resolving subject's own party entry, never another
// party's; involved[0] is that subject. Displaced ids are sorted.
func rankAgainst(cand temporal.Interval, involved []string, placed []Placed, exclude string) (int, []string) {
	subject := ""
	if len(involved) > 0 {
		subject = involved[0]
	}
	rank := 1
	var ids []string
	for _, p := range placed {
		if p.ID == exclude {
			continue
		}
		shares := false
		for _, uri := range involved {
			if p.Involves(uri) {
				shares = true
				break
			}
		}
		if !shares || !p.Interval.Overlaps(cand) {
			continue
		}
		ids = append(ids, p.ID)
		if p.FirmFor(subject) {
			rank = 3
		} else if rank < 2 {
			rank = 2
		}
	}
	sort.Strings(ids)
	if ids == nil {
		ids = []string{}
	}
	return rank, ids
}

// chooseLocation picks one URI from the intersection of the intention's
// locations and the supplying availabilities' locations, or the first of
// whichever side constrained it.
func chooseLocation(in *model.Intention, supplies map[string]Supply, supply []string) string {
	var supplyLocs []string
	for _, s := range supplies {
		for _, o := range s.Occasions {
			for _, id := range supply {
				if o.Availability.ID == id {
					supplyLocs = append(supplyLocs, o.Availability.Location...)
				}
			}
		}
	}
	supplyLocs = model.SortStrings(supplyLocs)
	intent := model.SortStrings(in.Location)
	switch {
	case len(intent) == 0 && len(supplyLocs) == 0:
		return ""
	case len(intent) == 0:
		return supplyLocs[0]
	case len(supplyLocs) == 0:
		return intent[0]
	}
	for _, a := range intent {
		for _, b := range supplyLocs {
			if a == b {
				return a
			}
		}
	}
	return intent[0]
}

// orderCandidates sorts by rank, then the declared preference (the
// intention's, else its recurring intention's), else earliest.
func orderCandidates(cs []Candidate, e Env, in *model.Intention, placed []Placed) {
	pref := in.Preference
	if pref == "" {
		if rec := in.InstanceOf(); rec != "" {
			if obj, ok := e.G.Get(rec); ok {
				if ri, ok := obj.(*model.Intention); ok {
					pref = ri.Preference
				}
			}
		}
	}
	for i := range cs {
		switch pref {
		case "latest":
			cs[i].prefKey = -float64(cs[i].Interval.Start.Unix())
		case "adjacent":
			cs[i].prefKey = float64(distanceToSameActivity(cs[i].Interval, in, placed))
		case "spread":
			cs[i].prefKey = -float64(distanceToSameActivity(cs[i].Interval, in, placed))
		default:
			cs[i].prefKey = float64(cs[i].Interval.Start.Unix())
		}
	}
	sort.SliceStable(cs, func(i, j int) bool {
		if cs[i].Rank != cs[j].Rank {
			return cs[i].Rank < cs[j].Rank
		}
		if cs[i].prefKey != cs[j].prefKey {
			return cs[i].prefKey < cs[j].prefKey
		}
		return cs[i].Interval.Start.Before(cs[j].Interval.Start)
	})
}

// distanceToSameActivity is the gap between a candidate and the nearest
// placement of the same subject and activity; a large value when none.
func distanceToSameActivity(cand temporal.Interval, in *model.Intention, placed []Placed) time.Duration {
	best := time.Duration(1<<62 - 1)
	if in.Activity == "" {
		return best
	}
	for _, p := range placed {
		if p.Activity != in.Activity || !p.Involves(in.Subject) {
			continue
		}
		var d time.Duration
		switch {
		case p.Interval.End.After(cand.Start) && cand.End.After(p.Interval.Start):
			d = 0
		case !p.Interval.End.After(cand.Start):
			d = cand.Start.Sub(p.Interval.End)
		default:
			d = p.Interval.Start.Sub(cand.End)
		}
		if d < best {
			best = d
		}
	}
	return best
}
