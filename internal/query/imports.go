package query

import (
	"fmt"
	"sort"
	"strings"

	"github.com/nodelogicau/intentions-cli/internal/model"
	"github.com/nodelogicau/intentions-cli/internal/store"
	"github.com/nodelogicau/intentions-cli/internal/temporal"
)

// CommitmentFindings reports what validate would say about an unretired
// imported commitment as it relates to an intention: an adoption that cannot
// stand (error), an adopted event that has moved (warning), or an event
// directed at no one (info). Nil when nothing applies.
func CommitmentFindings(g *store.Graph, c *model.Commitment) []Finding {
	if c.Retired != nil || c.Origin != model.OriginImport {
		return nil
	}
	var out []Finding
	if c.Intention != "" {
		obj, ok := g.Get(c.Intention)
		in, isIn := obj.(*model.Intention)
		if !ok || !isIn {
			return nil // dangling or mistyped: the referential pass reports it
		}
		if rule, msg := model.AdoptionProblem(c, in); rule != "" {
			out = append(out, Finding{Severity: SeverityError, Code: rule, ID: c.ID, Message: msg})
		}
		if in.Retired == nil && in.Placement != nil && c.Placement != nil && !temporal.SamePlacement(*c.Placement, *in.Placement) {
			out = append(out, Finding{Severity: SeverityWarning, Code: "adopted_moved", ID: c.ID, Message: fmt.Sprintf(
				"placed at %s for %s, but %s is placed at %s; the event moved since it was adopted, so adopt it again (intentions commitment adopt %s %s) or re-place %s",
				describePlacement(*c.Placement), c.Placement.Duration, in.ID, describePlacement(*in.Placement), c.ID, in.ID, in.ID)})
		}
		return out
	}
	if len(c.Parties) == 1 && (c.External == nil || model.HoldOwner(g.Intentions(), *c.External, "") == "") {
		out = append(out, Finding{Severity: SeverityInfo, Code: "directed_at_no_one", ID: c.ID, Message: fmt.Sprintf(
			"an imported event whose only party is %s: a commitment directed at no one, which may be better recorded as an intention that adopts it (intentions commitment adopt %s <int_id>)", c.Parties[0].URI, c.ID)})
	}
	return out
}

func describePlacement(p temporal.Placement) string {
	if p.Location != "" {
		return p.Start.Raw + " at " + p.Location
	}
	return p.Start.Raw
}

// imports reports the per-commitment findings, every intention two unretired
// commitments name, and a workspace importing without calendar.addresses.
func (v *validator) imports() {
	byIntention := map[string][]string{}
	imported := false
	for _, c := range v.g.Commitments() {
		v.findings = append(v.findings, CommitmentFindings(v.g, c)...)
		if c.Origin == model.OriginImport {
			imported = true
		}
		if c.Retired == nil && c.Intention != "" {
			byIntention[c.Intention] = append(byIntention[c.Intention], c.ID)
		}
	}
	ids := make([]string, 0, len(byIntention))
	for id := range byIntention {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		if cs := byIntention[id]; len(cs) > 1 {
			v.add(SeverityError, "intention_two_commitments", id, "is the intention of %d unretired commitments (%s); one occasion fulfils an intention, so cancel all but one", len(cs), strings.Join(cs, ", "))
		}
	}
	if imported && len(v.ws.Config.Calendar.Addresses) == 0 {
		v.addPath(SeverityInfo, "calendar_addresses_missing", store.ConfigFile, "the workspace holds imported commitments but declares no calendar.addresses, so an import can find the person only by their subject URI; list their calendar addresses under calendar.addresses")
	}
}
