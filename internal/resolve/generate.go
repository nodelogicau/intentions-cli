package resolve

import (
	"fmt"
	"sort"
	"time"

	"github.com/nodelogicau/intentions-cli/internal/model"
	"github.com/nodelogicau/intentions-cli/internal/temporal"
)

// Skip records an occurrence generation did not materialise and why.
type Skip struct {
	Recurring  string `json:"recurring"`
	Occurrence string `json:"occurrence"`
	Reason     string `json:"reason"`
	Existing   string `json:"existing,omitempty"`
}

// GenerateOptions bound a generation run.
type GenerateOptions struct {
	Range     temporal.Interval // the horizon to generate over
	Only      []string          // recurring intention ids; empty means all
	Source    model.Source
	Timestamp time.Time
}

// Generate materialises instances of every active recurring intention (or of
// the named ones) for each occurrence within the range that has no instance
// yet, active or retired. Instances are written immediately and added to the
// graph.
func Generate(e Env, opts GenerateOptions) ([]*model.Intention, []Skip, error) {
	var created []*model.Intention
	var skipped []Skip
	only := map[string]bool{}
	for _, id := range opts.Only {
		only[id] = true
	}
	for _, rec := range e.G.Intentions() {
		if !rec.IsRecurring() || rec.Retired != nil || rec.InstanceOf() != "" {
			continue
		}
		if len(only) > 0 && !only[rec.ID] {
			continue
		}
		if rec.Window == nil || rec.Window.Calendar == nil {
			skipped = append(skipped, Skip{Recurring: rec.ID, Reason: "no calendar anchor to expand within"})
			continue
		}
		ctx := e.Ctx
		ctx.Horizon = &opts.Range
		days, err := temporal.Expand(*rec.Cadence, *rec.Window, ctx)
		if err != nil {
			return created, skipped, fmt.Errorf("%s: %v", rec.ID, err)
		}
		existing := map[string]*model.Intention{}
		for _, inst := range e.G.Instances(rec.ID) {
			existing[inst.Occurrence] = inst
		}
		for _, day := range days {
			occ := day.String()
			if prior, ok := existing[occ]; ok {
				reason := "instance exists"
				if prior.Retired != nil {
					reason = "instance retired (" + prior.Retired.Kind + ")"
				}
				skipped = append(skipped, Skip{Recurring: rec.ID, Occurrence: occ, Reason: reason, Existing: prior.ID})
				continue
			}
			inst := instanceOf(rec, day, opts)
			if err := e.WS.WriteObject(inst); err != nil {
				return created, skipped, err
			}
			e.G.Add(inst)
			created = append(created, inst)
		}
	}
	sort.Slice(created, func(i, j int) bool { return created[i].ID < created[j].ID })
	return created, skipped, nil
}

// instanceOf builds one instance: the occurrence day as the calendar anchor,
// the recurring intention's clock, and its subject, duration, activity,
// parties and location.
func instanceOf(rec *model.Intention, day temporal.Granule, opts GenerateOptions) *model.Intention {
	d := day
	cal := temporal.Calendar{Start: &d, End: &d}
	w := &temporal.Window{Calendar: &cal}
	if rec.Window != nil && rec.Window.Clock != nil {
		c := *rec.Window.Clock
		w.Clock = &c
	}
	inst := &model.Intention{
		ID: model.MintID(model.TypeIntention), Subject: rec.Subject, Title: rec.Title, Description: rec.Description,
		Window: w, Stability: "tentative", Activity: rec.Activity,
		Location: append([]string(nil), rec.Location...), Parties: append([]string(nil), rec.Parties...),
		Serves: []model.Ref{{ID: rec.ID, Role: model.RoleInstanceOf}}, Occurrence: day.String(),
		Source: opts.Source, Timestamp: opts.Timestamp, Acknowledgements: []model.Acknowledgement{},
	}
	if rec.Duration != nil {
		dd := *rec.Duration
		inst.Duration = &dd
	}
	return inst
}
