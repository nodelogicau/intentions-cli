package resolve

import (
	"fmt"
	"time"

	"github.com/nodelogicau/intentions-cli/internal/model"
	"github.com/nodelogicau/intentions-cli/internal/temporal"
)

// Selection is what a selection wrote.
type Selection struct {
	Resolution *model.Resolution
	Intention  *model.Intention
	Commitment *model.Commitment
	Cancelled  *model.Commitment // a live commitment the replacement had to cancel
	Replaced   *temporal.Placement
	Candidate  Candidate
}

// SelectOptions describe the act.
type SelectOptions struct {
	Candidate int          // 1-based index into the result's candidates (person)
	Policy    string       // policy id (harness); selects the top rank-1 candidate
	Source    model.Source // the hand performing the act
	Timestamp time.Time
	Replace   bool
	Limit     int
}

// Select performs a selection: it recomputes the candidates, takes the
// chosen one, writes the RESOLUTION record, the placement on the intention,
// and a COMMITMENT when the intention has parties. Displaced objects are
// listed on the record and never changed.
func Select(e Env, in *model.Intention, opts SelectOptions) (Selection, Result, error) {
	var sel Selection
	if in.Placement != nil {
		if !opts.Replace {
			return sel, Result{}, refuse("%s is already placed at %s; pass --replace to select a new placement", in.ID, in.Placement.Start.Raw)
		}
		p := *in.Placement
		sel.Replaced = &p
	}
	exclude := ""
	if in.Placement != nil {
		exclude = in.ID
	}
	res, err := Resolve(e, in, Options{Limit: opts.Limit, AllowPlaced: opts.Replace, Exclude: exclude})
	if err != nil {
		return sel, res, err
	}
	all := res.All()
	if len(all) == 0 {
		return sel, res, refuse("no candidates: %s", res.Reason)
	}
	var chosen Candidate
	selector := "person"
	switch {
	case opts.Policy != "":
		if opts.Source.Harness == "" {
			return sel, res, refuse("--policy is for a harness acting under a policy the person holds; a person selects a candidate directly")
		}
		policy, err := model.LookupPolicy(e.G, opts.Policy, in.Subject)
		if err != nil {
			return sel, res, err
		}
		if policy.AutoSelect == nil {
			return sel, res, refuse("policy %s carries no auto_select condition", opts.Policy)
		}
		if err := model.Satisfies(policy.AutoSelect, in); err != nil {
			return sel, res, refuse("policy %s does not cover %s: %v", opts.Policy, in.ID, err)
		}
		if all[0].Rank != 1 {
			return sel, res, refuse("policy %s may select only a candidate that displaces nothing, and every candidate for %s displaces something; put the candidates to the person", opts.Policy, in.ID)
		}
		chosen = all[0]
		selector = opts.Policy
	default:
		if opts.Candidate < 1 || opts.Candidate > len(all) {
			return sel, res, refuse("--candidate %d is out of range; %d candidates were considered (use resolve to list them)", opts.Candidate, len(all))
		}
		chosen = all[opts.Candidate-1]
	}
	sel.Candidate = chosen

	start, _ := temporal.ParsePlacementStart(chosen.Interval.Start.In(e.Ctx.Location).Format(time.RFC3339))
	offered := in.Duration.Nominal
	if d, err := temporal.ParseDuration(chosen.Duration); err == nil {
		offered = d // a ranged duration may have been offered shorter
	}
	placement := &temporal.Placement{Start: start, Duration: temporal.DurationSpec{Nominal: offered}, Location: chosen.Location}
	considered := res.Considered
	rec := &model.Resolution{
		ID: model.MintID(model.TypeResolution), Intention: in.ID, Placement: placement, Selector: selector,
		CandidatesConsidered: &considered, Displaced: chosen.Displaces, Source: opts.Source, Timestamp: opts.Timestamp, Supply: chosen.Supply,
	}
	if err := e.WS.WriteObject(rec); err != nil {
		return sel, res, err
	}
	updated := *in
	updated.Placement = placement
	if err := e.WS.WriteObject(&updated); err != nil {
		return sel, res, err
	}
	sel.Resolution, sel.Intention = rec, &updated
	e.G.Add(rec)
	e.G.Add(&updated)

	// A placement its parties agreed to cannot move under them. Replacing
	// one cancels the commitment that rested on it, naming the new
	// resolution as the reason, and a fresh commitment is written below with
	// everyone tentative again; the retired file keeps who had accepted.
	if sel.Replaced != nil {
		for _, c := range e.G.Commitments() {
			if c.Retired != nil || c.Intention != in.ID || c.ID == "" {
				continue
			}
			cancelled := *c
			cancelled.Retired = &model.Retired{Kind: "cancelled", Reason: "superseded by " + rec.ID, Source: opts.Source, Timestamp: opts.Timestamp}
			if err := e.WS.WriteObject(&cancelled); err != nil {
				return sel, res, err
			}
			e.G.Add(&cancelled)
			sel.Cancelled = &cancelled
			break
		}
	}

	if len(in.Parties) > 0 {
		parties := []model.Party{{URI: in.Subject, Status: "tentative"}}
		for _, p := range in.Parties {
			parties = append(parties, model.Party{URI: p, Status: "tentative"})
		}
		cmt := &model.Commitment{
			ID: model.MintID(model.TypeCommitment), Parties: model.SortParties(parties), Placement: placement, Intention: in.ID,
			Origin: model.Origin{Resolution: rec.ID}, Title: in.Title, Source: opts.Source, Timestamp: opts.Timestamp, Acknowledgements: []model.Acknowledgement{},
		}
		if err := e.WS.WriteObject(cmt); err != nil {
			return sel, res, err
		}
		sel.Commitment = cmt
		e.G.Add(cmt)
	}
	return sel, res, nil
}

// Describe renders a selection for a text result.
func (s Selection) Describe() string {
	out := fmt.Sprintf("Selected candidate %s (rank %d) for %s: resolution %s", s.Candidate.Start, s.Candidate.Rank, s.Intention.ID, s.Resolution.ID)
	if s.Commitment != nil {
		out += ", commitment " + s.Commitment.ID
	}
	if s.Cancelled != nil {
		out += " (cancelling " + s.Cancelled.ID + ")"
	}
	if len(s.Candidate.Displaces) > 0 {
		out += fmt.Sprintf("; displaces %v", s.Candidate.Displaces)
	}
	if s.Replaced != nil {
		out += "; replaced placement at " + s.Replaced.Start.Raw
	}
	return out
}
