// Package render builds the JSON-ready result maps both front-ends emit, so
// an MCP tool's structured result is exactly the CLI verb's --json output.
package render

import (
	"github.com/nodelogicau/intentions-cli/internal/consistency"
	"github.com/nodelogicau/intentions-cli/internal/model"
	"github.com/nodelogicau/intentions-cli/internal/projection"
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
	if sel.Replaced != nil {
		out["replaced"] = map[string]any{"start": sel.Replaced.Start.Raw, "duration": sel.Replaced.Duration.String(), "location": sel.Replaced.Location}
	}
	if policy != "" {
		out["policy"] = policy
	}
	return out
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
