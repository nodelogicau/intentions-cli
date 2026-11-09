package importer

import (
	"reflect"
	"strings"
	"time"

	"github.com/nodelogicau/intentions-cli/internal/apperr"
	"github.com/nodelogicau/intentions-cli/internal/model"
	"github.com/nodelogicau/intentions-cli/internal/projection"
	"github.com/nodelogicau/intentions-cli/internal/store"
	"github.com/nodelogicau/intentions-cli/internal/temporal"
)

// Outcomes of an import.
const (
	Created    = "created"
	Updated    = "updated"
	Unchanged  = "unchanged"
	Recognised = "recognised" // the event is an intention's own hold
	Retired    = "retired"    // the event's commitment is retired and stays so
)

// Options describe the act.
type Options struct {
	Subject    string // the particular the import is made for
	Commitment string // the commitment to update, overriding the match
	Source     model.Source
	Timestamp  time.Time
}

// Result is what an import did.
type Result struct {
	Outcome         string
	Commitment      *model.Commitment // created, updated, unchanged, retired
	Intention       string            // recognised: the hold's intention
	PreviousVersion string            // updated
}

// Import brings ev into the workspace for opts.Subject.
func Import(ws *store.Workspace, g *store.Graph, ev Event, opts Options) (Result, error) {
	var res Result
	link := ev.External()
	if owner := model.HoldOwner(g.Intentions(), link, ""); owner != "" {
		res.Outcome, res.Intention = Recognised, owner
		return res, nil
	}
	match, err := findMatch(g, ev, opts.Commitment)
	if err != nil {
		return res, err
	}
	parties := Parties(ev, opts.Subject, ws.Config.SubjectFor, store.NormaliseAddress)
	placement := ev.Placement()
	var transparent *bool
	if ev.Transparent {
		t := true
		transparent = &t
	}
	if match == nil {
		c := &model.Commitment{
			ID: model.MintID(model.TypeCommitment), Parties: parties, Placement: &placement, Origin: model.OriginImport,
			Transparent: transparent, External: &link, Title: ev.Title, Source: opts.Source, Timestamp: opts.Timestamp,
			Acknowledgements: []model.Acknowledgement{},
		}
		if ps := model.Check(c); len(ps) > 0 {
			return res, apperr.Invalid("%s", ps.Error())
		}
		if err := ws.WriteObject(c); err != nil {
			return res, err
		}
		g.Add(c)
		res.Outcome, res.Commitment = Created, c
		return res, nil
	}
	if match.Retired != nil {
		res.Outcome, res.Commitment = Retired, match
		return res, nil
	}
	if reflect.DeepEqual(match.Parties, parties) && temporal.SamePlacement(*match.Placement, placement) && match.Placement.Start.Raw == placement.Start.Raw &&
		match.IsTransparent() == ev.Transparent && match.Title == ev.Title {
		res.Outcome, res.Commitment = Unchanged, match
		return res, nil
	}
	prev, _ := projection.Version(match)
	updated := *match
	updated.Parties, updated.Placement, updated.Transparent, updated.Title = parties, &placement, transparent, ev.Title
	updated.Source, updated.Timestamp = opts.Source, opts.Timestamp
	if ps := model.Check(&updated); len(ps) > 0 {
		return res, apperr.Invalid("%s", ps.Error())
	}
	if err := ws.WriteObject(&updated); err != nil {
		return res, err
	}
	g.Add(&updated)
	res.Outcome, res.Commitment, res.PreviousVersion = Updated, &updated, prev
	return res, nil
}

// findMatch picks the commitment an occurrence updates: the one named, the
// only one carrying the link, or of several the one at the same start.
func findMatch(g *store.Graph, ev Event, named string) (*model.Commitment, error) {
	link := ev.External()
	if named != "" {
		obj, ok := g.Get(named)
		c, isC := obj.(*model.Commitment)
		if !ok || !isC {
			return nil, apperr.NotFound("%s is not a commitment in this workspace", named)
		}
		if c.External == nil || *c.External != link {
			return nil, apperr.Refused("%s does not carry %s:%s; --commitment names the commitment this event updates", named, link.System, link.UID)
		}
		return c, nil
	}
	var same []*model.Commitment
	for _, c := range g.Commitments() {
		if c.External != nil && *c.External == link {
			same = append(same, c)
		}
	}
	sameStart := func(a, b temporal.PlacementStart) bool {
		return a.AllDay == b.AllDay && (a.AllDay && a.Raw == b.Raw || !a.AllDay && a.Time.Equal(b.Time))
	}
	start := ev.Placement().Start
	if ev.recurrence != nil {
		// An occurrence of a recurring event: the one already at this start,
		// else the one still at the occurrence's original start (moved since),
		// else a new occurrence.
		for _, want := range []temporal.PlacementStart{start, *ev.recurrence} {
			for _, c := range same {
				if sameStart(c.Placement.Start, want) {
					return c, nil
				}
			}
		}
		return nil, nil
	}
	switch len(same) {
	case 0:
		return nil, nil
	case 1:
		return same[0], nil // a single event, moved or not
	}
	var ids []string
	for _, c := range same {
		ids = append(ids, c.ID)
		if sameStart(c.Placement.Start, start) {
			return c, nil
		}
	}
	return nil, apperr.Refused("%d commitments carry %s:%s and none starts at %s (%s); pass recurrence_id for an occurrence of a recurring event, or --commitment <id> to say which this event updates", len(same), link.System, link.UID, ev.Start, strings.Join(ids, ", "))
}
