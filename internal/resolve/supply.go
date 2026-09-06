package resolve

import (
	"fmt"
	"sort"
	"time"

	"github.com/nodelogicau/intentions-cli/internal/model"
	"github.com/nodelogicau/intentions-cli/internal/temporal"
)

// Occasion is one interval an availability offers, with the capacity it
// carries per occasion.
type Occasion struct {
	Interval     temporal.Interval
	Availability *model.Availability
	Capacity     time.Duration
}

// Exclusion says why an availability was not eligible supply.
type Exclusion struct {
	Availability string `json:"availability"`
	Reason       string `json:"reason"`
}

// EffectiveValidUntil is the explicit horizon, else timestamp plus the
// workspace default for a recurring availability, else the end of the
// window's calendar bounds; zero when unbounded.
func (e Env) EffectiveValidUntil(av *model.Availability) time.Time {
	if !av.ValidUntil.IsZero() {
		return av.ValidUntil.Instant(e.Ctx)
	}
	if av.Cadence != nil {
		return temporal.AddDuration(av.Timestamp, e.WS.Config.DefaultHorizon())
	}
	if av.Window != nil && av.Window.Calendar != nil {
		ivs, err := temporal.Bounds(temporal.Window{Calendar: av.Window.Calendar}, e.Ctx)
		if err == nil && len(ivs) > 0 && !ivs[len(ivs)-1].OpenEnd {
			return ivs[len(ivs)-1].End
		}
	}
	return time.Time{}
}

// Expired reports whether the availability's horizon has passed at now.
func (e Env) Expired(av *model.Availability) bool {
	vu := e.EffectiveValidUntil(av)
	return !vu.IsZero() && !vu.After(e.now())
}

// Eligible applies the supply filters for a demand: unretired, unexpired,
// conditional admits the activity, location shares a URI, scope visible.
func (e Env) Eligible(av *model.Availability, in *model.Intention) (bool, string) {
	if av.Retired != nil {
		return false, "retired (" + av.Retired.Kind + ")"
	}
	if e.Expired(av) {
		return false, "expired at " + fmtTime(e.EffectiveValidUntil(av))
	}
	if !ConditionalAdmits(av, in.Activity) {
		return false, fmt.Sprintf("conditional %v does not include activity %q", av.Conditional, in.Activity)
	}
	if !LocationsIntersect(av.Location, in.Location) {
		return false, fmt.Sprintf("location %v shares nothing with the intention's %v", av.Location, in.Location)
	}
	if scopeRank(av.Scope) < scopeRank(e.Scope) {
		return false, fmt.Sprintf("scope %s is narrower than the resolver's %s", av.Scope, e.Scope)
	}
	return true, ""
}

// ConditionalAdmits reports whether an availability's conditional admits the
// activity: absent means anything.
func ConditionalAdmits(av *model.Availability, activity string) bool {
	if len(av.Conditional) == 0 {
		return true
	}
	for _, t := range av.Conditional {
		if t == activity {
			return true
		}
	}
	return false
}

// LocationsIntersect reports whether an availability's location list admits
// an intention's: either absent, or sharing a URI.
func LocationsIntersect(avail, intent []string) bool {
	if len(avail) == 0 || len(intent) == 0 {
		return true
	}
	for _, a := range avail {
		for _, b := range intent {
			if a == b {
				return true
			}
		}
	}
	return false
}

// OccasionsOf returns the availability's occasions within r.
func (e Env) OccasionsOf(av *model.Availability, r temporal.Interval) ([]Occasion, error) {
	if av.Window == nil || av.Duration == nil {
		return nil, nil
	}
	ivs, err := temporal.Occasions(*av.Window, av.Cadence, e.Ctx, r)
	if err != nil {
		return nil, err
	}
	capD := av.Duration.Nominal
	if av.Duration.Max != nil {
		capD = *av.Duration.Max
	}
	out := make([]Occasion, 0, len(ivs))
	for _, iv := range ivs {
		out = append(out, Occasion{Interval: iv, Availability: av, Capacity: goDuration(iv.Start, capD)})
	}
	return out, nil
}

// Supply is a particular's eligible occasions within a range.
type Supply struct {
	Particular string
	Occasions  []Occasion
	Excluded   []Exclusion
}

// Intervals returns the merged intervals of the supply.
func (s Supply) Intervals() []temporal.Interval {
	ivs := make([]temporal.Interval, 0, len(s.Occasions))
	for _, o := range s.Occasions {
		ivs = append(ivs, o.Interval)
	}
	return temporal.Merge(ivs)
}

// SupplyFor collects the eligible occasions of one particular for a demand.
func (e Env) SupplyFor(uri string, in *model.Intention, r temporal.Interval) (Supply, error) {
	s := Supply{Particular: uri}
	for _, av := range e.G.Availabilities() {
		if av.Subject != uri {
			continue
		}
		ok, why := e.Eligible(av, in)
		if !ok {
			s.Excluded = append(s.Excluded, Exclusion{Availability: av.ID, Reason: why})
			continue
		}
		occ, err := e.OccasionsOf(av, r)
		if err != nil {
			return s, fmt.Errorf("%s: %v", av.ID, err)
		}
		if len(occ) == 0 {
			s.Excluded = append(s.Excluded, Exclusion{Availability: av.ID, Reason: "no occasion within " + describe(r)})
			continue
		}
		s.Occasions = append(s.Occasions, occ...)
	}
	sort.Slice(s.Occasions, func(i, j int) bool { return s.Occasions[i].Interval.Start.Before(s.Occasions[j].Interval.Start) })
	return s, nil
}

// Remaining is the occasion's capacity less the time opaque placements of
// the particular already occupy within it.
func (e Env) Remaining(o Occasion, uri string, placed []Placed, exclude string) time.Duration {
	used := time.Duration(0)
	for _, p := range placed {
		if p.ID == exclude || !p.Involves(uri) {
			continue
		}
		if iv, ok := p.Interval.Intersect(o.Interval); ok {
			used += iv.Length()
		}
	}
	return o.Capacity - used
}
