// Package render builds the JSON-ready result maps both front-ends emit, so
// an MCP tool's structured result is exactly the CLI verb's --json output.
package render

import (
	"github.com/nodelogicau/intentions-cli/internal/consistency"
	"github.com/nodelogicau/intentions-cli/internal/importer"
	"github.com/nodelogicau/intentions-cli/internal/model"
	"github.com/nodelogicau/intentions-cli/internal/projection"
	"github.com/nodelogicau/intentions-cli/internal/query"
	"github.com/nodelogicau/intentions-cli/internal/resolve"
	"github.com/nodelogicau/intentions-cli/internal/store"
	"github.com/nodelogicau/intentions-cli/internal/temporal"
)

// Object is the common shape for a written or shown object: id, type, path,
// computed version, the object as its file renders it, and version_cached
// when the file's value disagrees.
func Object(obj model.Object) (map[string]any, error) {
	m, err := model.ToMap(obj)
	if err != nil {
		return nil, err
	}
	v, err := projection.Version(obj)
	if err != nil {
		return nil, err
	}
	out := map[string]any{
		"id":      obj.GetID(),
		"type":    string(obj.GetType()),
		"path":    store.RelPath(obj.GetType(), obj.GetID()),
		"version": v,
		"object":  m,
	}
	if cached := obj.CachedVersion(); cached != "" && cached != v {
		out["version_cached"] = cached
	}
	return out, nil
}

// Resolution is the shape of a resolve result.
func Resolution(res resolve.Result, id string) map[string]any {
	out := map[string]any{"intention": id, "candidates": res.Candidates, "candidates_considered": res.Considered, "generated": res.Generated}
	if res.RangeText != "" {
		out["range"] = res.RangeText
	}
	if res.Reason != "" {
		out["reason"] = res.Reason
	}
	if res.Blocked != "" {
		out["blocked_on"] = res.Blocked
	}
	if len(res.NoSupply) > 0 {
		out["no_supply"] = res.NoSupply
	}
	if len(res.Presumed) > 0 {
		out["presumed"] = res.Presumed
	}
	if len(res.Excluded) > 0 {
		out["excluded"] = res.Excluded
	}
	return out
}

// Selection is the shape of a select result.
func Selection(sel resolve.Selection, res resolve.Result, flags []consistency.Flag, policy string) map[string]any {
	recMap, _ := model.ToMap(sel.Resolution)
	inMap, _ := model.ToMap(sel.Intention)
	out := map[string]any{"resolution": recMap, "intention": inMap, "candidate": sel.Candidate, "candidates_considered": res.Considered, "flags": flags}
	if sel.Commitment != nil {
		cm, _ := model.ToMap(sel.Commitment)
		out["commitment"] = cm
	}
	if sel.Cancelled != nil {
		cm, _ := model.ToMap(sel.Cancelled)
		out["cancelled_commitment"] = cm
	}
	if sel.Unlinked != nil {
		cm, _ := model.ToMap(sel.Unlinked)
		out["unlinked_commitment"] = cm
	}
	if sel.Adopted != nil {
		cm, _ := model.ToMap(sel.Adopted)
		out["commitment"] = cm
		out["adopted"] = sel.Adopted.ID
	}
	if sel.Replaced != nil {
		out["replaced"] = map[string]any{"start": sel.Replaced.Start.Raw, "duration": sel.Replaced.Duration.String(), "location": sel.Replaced.Location}
	}
	if policy != "" {
		out["policy"] = policy
	}
	return out
}

// Import renders what an import did, for the CLI and MCP alike.
func Import(r importer.Result, flags []consistency.Flag, findings []query.Finding) map[string]any {
	out := map[string]any{"outcome": r.Outcome}
	if r.Commitment != nil {
		cm, _ := model.ToMap(r.Commitment)
		out["object"] = cm
		out["id"] = r.Commitment.ID
		v, _ := projection.Version(r.Commitment)
		out["version"] = v
	}
	if r.Intention != "" {
		out["intention"] = r.Intention
	}
	if r.PreviousVersion != "" {
		out["previous_version"] = r.PreviousVersion
	}
	if flags == nil {
		flags = []consistency.Flag{}
	}
	out["flags"] = flags
	if len(findings) > 0 {
		out["findings"] = findings
	}
	return out
}

// ImportText is the one-line text summary of an import.
func ImportText(r importer.Result) string {
	switch r.Outcome {
	case importer.Recognised:
		return "Recognised the hold of " + r.Intention + "; nothing imported"
	case importer.Retired:
		return r.Commitment.ID + " is retired (" + r.Commitment.Retired.Kind + "); the event was not brought back"
	case importer.Unchanged:
		return "Unchanged " + r.Commitment.ID
	case importer.Updated:
		return "Updated " + r.Commitment.ID + " (" + r.PreviousVersion + " -> " + r.Commitment.Version + ")"
	}
	return "Imported " + r.Commitment.ID + " (" + r.Commitment.Version + ")"
}

// Acknowledgement renders an appended entry.
func Acknowledgement(a model.Acknowledgement) map[string]any {
	return map[string]any{"kind": a.Kind, "counterpart": a.Counterpart, "counterpart_version": a.CounterpartVersion, "reason": a.Reason, "timestamp": temporal.FormatTimestamp(a.Timestamp)}
}

// Objects renders a list of objects as their file maps.
func Objects[T model.Object](objs []T) ([]map[string]any, error) {
	list := make([]map[string]any, 0, len(objs))
	for _, o := range objs {
		m, err := model.ToMap(o)
		if err != nil {
			return nil, err
		}
		list = append(list, m)
	}
	return list, nil
}
