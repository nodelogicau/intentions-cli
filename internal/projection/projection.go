// Package projection computes an object's version: the sha256 of the RFC 8785
// canonical JSON of its scheduling projection, the fields that bear on
// consistency, normalised first. The field sets are frozen for intentions/0.1.
package projection

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gowebpki/jcs"

	"github.com/nodelogicau/intentions-cli/internal/model"
	"github.com/nodelogicau/intentions-cli/internal/temporal"
)

// Fields lists the projection field set per type for intentions/0.1. It is
// frozen: a change to any set is a new format version.
var Fields = map[model.Type][]string{
	model.TypeIntention:    {"subject", "duration", "window", "stability", "activity", "location", "parties", "serves", "cadence", "occurrence", "placement", "retired.kind"},
	model.TypeAvailability: {"subject", "duration", "window", "conditional", "location", "cadence", "valid_until", "retired.kind"},
	model.TypeCommitment:   {"parties", "placement", "intention", "origin", "transparent", "external", "retired.kind"},
	model.TypeResolution:   {"intention", "placement", "selector", "displaced"},
}

// Algorithm is the only hash this format version admits.
const Algorithm = "sha256"

// Project builds the normalised projection as a JSON-ready map.
func Project(obj model.Object) map[string]any {
	p := map[string]any{}
	put := func(k string, v any) {
		if v == nil {
			return
		}
		switch x := v.(type) {
		case string:
			if x == "" {
				return
			}
		case []string:
			if len(x) == 0 {
				return
			}
		case []any:
			if len(x) == 0 {
				return
			}
		case map[string]any:
			if len(x) == 0 {
				return
			}
		}
		p[k] = v
	}
	switch o := obj.(type) {
	case *model.Intention:
		put("subject", o.Subject)
		put("duration", durationSpec(o.Duration))
		put("window", window(o.Window))
		put("stability", o.Stability)
		put("activity", o.Activity)
		put("location", model.SortStrings(o.Location))
		put("parties", model.SortStrings(o.Parties))
		put("serves", refs(o.Serves))
		put("cadence", cadence(o.Cadence))
		put("occurrence", granule(o.Occurrence))
		put("placement", placement(o.Placement))
		put("retired", retired(o.Retired))
	case *model.Availability:
		put("subject", o.Subject)
		put("duration", durationSpec(o.Duration))
		put("window", window(o.Window))
		put("conditional", model.SortStrings(o.Conditional))
		put("location", model.SortStrings(o.Location))
		put("cadence", cadence(o.Cadence))
		put("valid_until", validUntil(o.ValidUntil))
		put("retired", retired(o.Retired))
	case *model.Commitment:
		put("parties", parties(o.Parties))
		put("placement", placement(o.Placement))
		put("intention", o.Intention)
		if o.Origin.Import {
			put("origin", "import")
		} else if o.Origin.Resolution != "" {
			put("origin", map[string]any{"resolution": o.Origin.Resolution})
		}
		if o.IsTransparent() {
			put("transparent", true)
		}
		if o.External != nil {
			put("external", map[string]any{"system": o.External.System, "uid": o.External.UID})
		}
		put("retired", retired(o.Retired))
	case *model.Resolution:
		put("intention", o.Intention)
		put("placement", placement(o.Placement))
		put("selector", o.Selector)
		put("displaced", model.SortStrings(o.Displaced))
	}
	return p
}

// Canonical returns the RFC 8785 serialisation of the projection.
func Canonical(obj model.Object) ([]byte, error) {
	raw, err := json.Marshal(Project(obj))
	if err != nil {
		return nil, err
	}
	return jcs.Transform(raw)
}

// Version returns the object's version string, sha256:<hex>.
func Version(obj model.Object) (string, error) {
	c, err := Canonical(obj)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(c)
	return Algorithm + ":" + hex.EncodeToString(sum[:]), nil
}

// MustVersion is Version for callers that have already validated the object.
func MustVersion(obj model.Object) string {
	v, err := Version(obj)
	if err != nil {
		panic(fmt.Sprintf("projection: %v", err))
	}
	return v
}

// --- normalisers ----------------------------------------------------------

func durationSpec(s *temporal.DurationSpec) any {
	if s == nil {
		return nil
	}
	n := s.Normalise()
	if !n.Ranged() {
		return n.Nominal.String()
	}
	m := map[string]any{"nominal": n.Nominal.String()}
	if n.Min != nil {
		m["min"] = n.Min.String()
	}
	if n.Max != nil {
		m["max"] = n.Max.String()
	}
	return m
}

func window(w *temporal.Window) any {
	if w == nil || w.IsZero() {
		return nil
	}
	m := map[string]any{}
	if w.Calendar != nil {
		m["calendar"] = w.Calendar.Normalise().String()
	}
	if w.Relative != nil {
		r := map[string]any{"target": w.Relative.Target, "relation": string(w.Relative.Relation)}
		if w.Relative.Gap != nil {
			g := map[string]any{}
			if w.Relative.Gap.Min != nil {
				g["min"] = w.Relative.Gap.Min.Normalise().String()
			}
			if w.Relative.Gap.Max != nil {
				g["max"] = w.Relative.Gap.Max.Normalise().String()
			}
			if len(g) > 0 {
				r["gap"] = g
			}
		}
		m["relative"] = r
	}
	if w.Clock != nil {
		m["clock"] = w.Clock.String()
	}
	return m
}

func placement(p *temporal.Placement) any {
	if p == nil {
		return nil
	}
	m := map[string]any{}
	if p.Start.AllDay {
		m["start"] = p.Start.Day.String()
	} else if !p.Start.Time.IsZero() {
		m["start"] = p.Start.Time.UTC().Format(temporal.TimeLayout)
	}
	m["duration"] = durationSpec(&p.Duration)
	if p.Location != "" {
		m["location"] = p.Location
	}
	return m
}

func refs(rs []model.Ref) any {
	if len(rs) == 0 {
		return nil
	}
	out := make([]any, 0, len(rs))
	for _, r := range model.SortRefs(rs) {
		out = append(out, map[string]any{"id": r.ID, "role": r.Role})
	}
	return out
}

func parties(ps []model.Party) any {
	if len(ps) == 0 {
		return nil
	}
	out := make([]any, 0, len(ps))
	for _, p := range model.SortParties(ps) {
		out = append(out, map[string]any{"uri": p.URI, "status": p.Status})
	}
	return out
}

func cadence(c *temporal.Cadence) any {
	if c == nil {
		return nil
	}
	return strings.ToUpper(strings.TrimSpace(c.Raw))
}

func granule(s string) any {
	if s == "" {
		return nil
	}
	if g, err := temporal.ParseGranule(s); err == nil {
		return g.String()
	}
	return s
}

func validUntil(v temporal.ValidUntil) any {
	switch {
	case v.IsZero():
		return nil
	case v.Calendar != nil:
		return v.Calendar.Normalise().String()
	default:
		return v.Time.UTC().Format(temporal.TimeLayout)
	}
}

func retired(r *model.Retired) any {
	if r == nil {
		return nil
	}
	return map[string]any{"kind": r.Kind}
}

// Stamp computes and caches the version on the object.
func Stamp(obj model.Object) (string, error) {
	v, err := Version(obj)
	if err != nil {
		return "", err
	}
	obj.SetVersion(v)
	return v, nil
}

var _ = time.Now
