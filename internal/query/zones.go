package query

import (
	"fmt"

	"github.com/nodelogicau/intentions-cli/internal/model"
	"github.com/nodelogicau/intentions-cli/internal/store"
	"github.com/nodelogicau/intentions-cli/internal/temporal"
)

// ZoneFindings reports a placement start whose zone this binary does not
// know, or whose written offset is not the zone's offset at that instant.
// intention is the intention a replacing selection would re-place. A start
// in UTC carries no local offset to check.
func ZoneFindings(id, intention string, p *temporal.Placement) []Finding {
	if p == nil || p.Start.AllDay || p.Start.Zone == "" || p.Start.UTCStart() {
		return nil
	}
	zone := p.Start.Zone
	loc, err := temporal.LoadZone(zone)
	if err != nil {
		return []Finding{{Severity: SeverityWarning, Code: "zone_unknown", ID: id, Message: fmt.Sprintf(
			"placement.start %s names the zone %s, which this binary's tz database does not know; the instant is taken from the offset", p.Start.Raw, zone)}}
	}
	written := p.Start.Time.Format("-07:00")
	actual := p.Start.Time.In(loc).Format("-07:00")
	if written == actual {
		return nil
	}
	fix := "a re-import of the event"
	if intention != "" {
		fix = fmt.Sprintf("a replacing selection (intentions select %s --replace)", intention)
	}
	return []Finding{{Severity: SeverityWarning, Code: "zone_offset_mismatch", ID: id, Message: fmt.Sprintf(
		"placement.start %s is written at %s, but %s is at %s at that instant; the zone's rules may have changed since it was written, and the instant stands as written until %s re-places it",
		p.Start.Raw, written, zone, actual, fix)}}
}

// zones reports ZoneFindings over every intention, commitment and
// resolution placement.
func (v *validator) zones() {
	for _, in := range v.g.Intentions() {
		v.findings = append(v.findings, ZoneFindings(in.ID, in.ID, in.Placement)...)
	}
	for _, c := range v.g.Commitments() {
		v.findings = append(v.findings, ZoneFindings(c.ID, c.Intention, c.Placement)...)
	}
	for _, r := range v.g.Resolutions() {
		v.findings = append(v.findings, ZoneFindings(r.ID, r.Intention, r.Placement)...)
	}
}

// ImportFindings are the findings an import result carries: the
// commitment's own, as validate would report them.
func ImportFindings(g *store.Graph, c *model.Commitment) []Finding {
	return append(CommitmentFindings(g, c), ZoneFindings(c.ID, c.Intention, c.Placement)...)
}
