package resolve

import (
	"sort"
	"time"

	"github.com/nodelogicau/intentions-cli/internal/model"
)

// AdoptOptions describe the act.
type AdoptOptions struct {
	Source    model.Source
	Timestamp time.Time
	Replace   bool
}

// Adopt records that the imported commitment c is the occasion of in: a
// selection by the person whose placement is c's. It writes a RESOLUTION
// record (selector person, one candidate considered), gives in that
// placement, and names in on c. The placement need not be a candidate;
// consistency reports what it lacks.
func Adopt(e Env, c *model.Commitment, in *model.Intention, opts AdoptOptions) (Selection, error) {
	var sel Selection
	if err := model.CheckAdopt(c, in, e.G.Commitments(), opts.Replace); err != nil {
		return sel, err
	}
	if in.Placement != nil {
		p := *in.Placement
		sel.Replaced = &p
	}
	placement := *c.Placement
	iv := placement.Bounds(e.Ctx)

	// What else the occasion overlaps on the subject's time: never the
	// commitment itself, nor the intention's own earlier placement.
	var placed []Placed
	for _, p := range e.PlacedObjects() {
		if p.ID != c.ID && p.ID != in.ID {
			placed = append(placed, p)
		}
	}
	rank, displaced := rankAgainst(iv, []string{in.Subject}, placed, "")

	// The subject's availabilities with an occasion containing it.
	supply := []string{}
	if s, err := e.SupplyFor(in.Subject, in, iv); err == nil {
		seen := map[string]bool{}
		for _, o := range s.Occasions {
			if o.Interval.Contains(iv) && !seen[o.Availability.ID] {
				seen[o.Availability.ID] = true
				supply = append(supply, o.Availability.ID)
			}
		}
		sort.Strings(supply)
	}

	one := 1
	rec := &model.Resolution{
		ID: model.MintID(model.TypeResolution), Intention: in.ID, Placement: &placement, Selector: "person",
		CandidatesConsidered: &one, Displaced: displaced, Supply: supply, Presumed: []string{}, Source: opts.Source, Timestamp: opts.Timestamp,
	}
	if err := e.WS.WriteObject(rec); err != nil {
		return sel, err
	}
	updated := *in
	updated.Placement = &placement
	updated.Timestamp = opts.Timestamp
	if err := e.WS.WriteObject(&updated); err != nil {
		return sel, err
	}
	adopted := *c
	adopted.Intention = in.ID
	adopted.Timestamp = opts.Timestamp
	if err := e.WS.WriteObject(&adopted); err != nil {
		return sel, err
	}
	e.G.Add(rec)
	e.G.Add(&updated)
	e.G.Add(&adopted)
	sel.Resolution, sel.Intention, sel.Adopted = rec, &updated, &adopted
	sel.Candidate = Candidate{Start: placement.Start.Raw, Interval: iv, Rank: rank, Displaces: displaced, Supply: supply, Location: placement.Location}
	return sel, nil
}
