package model

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/nodelogicau/intentions-cli/internal/temporal"
)

// Encode serialises an object deterministically in canonical field order:
// 2-space indent, literal block scalars for multi-line prose, RFC 3339 Z
// timestamps, optional fields omitted, set-valued lists sorted, fields this
// implementation does not know after every specified field, the cached
// version last, no document markers.
func Encode(obj Object) ([]byte, error) {
	var root *yaml.Node
	switch o := obj.(type) {
	case *Intention:
		root = intentionNode(o)
	case *Availability:
		root = availabilityNode(o)
	case *Commitment:
		root = commitmentNode(o)
	case *Resolution:
		root = resolutionNode(o)
	default:
		return nil, fmt.Errorf("cannot encode %T", obj)
	}
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(root); err != nil {
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// --- node builders --------------------------------------------------------

func mapping() *yaml.Node  { return &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"} }
func sequence() *yaml.Node { return &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"} }

// flowMapping renders inline, as the spec's examples do for small records.
func flowMapping() *yaml.Node {
	return &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map", Style: yaml.FlowStyle}
}

func scalar(s string) *yaml.Node {
	n := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: s}
	if strings.Contains(s, "\n") {
		n.Style = yaml.LiteralStyle
	}
	return n
}

func plain(tag, s string) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: tag, Value: s}
}

func addKV(m *yaml.Node, key string, v *yaml.Node) {
	if v == nil {
		return
	}
	m.Content = append(m.Content, plain("!!str", key), v)
}

func addStr(m *yaml.Node, key, v string) {
	if v == "" {
		return
	}
	addKV(m, key, scalar(v))
}

// addStrAlways writes required fields even when empty, so a malformed object
// round-trips visibly rather than silently dropping the key.
func addStrAlways(m *yaml.Node, key, v string) { addKV(m, key, scalar(v)) }

func addStrList(m *yaml.Node, key string, vs []string, always bool) {
	if len(vs) == 0 && !always {
		return
	}
	seq := sequence()
	for _, v := range SortStrings(vs) {
		seq.Content = append(seq.Content, scalar(v))
	}
	addKV(m, key, seq)
}

func addTime(m *yaml.Node, key string, t time.Time) {
	if t.IsZero() {
		return
	}
	addKV(m, key, plain("!!timestamp", temporal.FormatTimestamp(t)))
}

func durationNode(d temporal.Duration) *yaml.Node { return scalar(d.String()) }

func durationSpecNode(s *temporal.DurationSpec) *yaml.Node {
	if s == nil {
		return nil
	}
	if !s.Ranged() {
		return durationNode(s.Nominal)
	}
	m := flowMapping()
	addKV(m, "nominal", durationNode(s.Nominal))
	if s.Min != nil {
		addKV(m, "min", durationNode(*s.Min))
	}
	if s.Max != nil {
		addKV(m, "max", durationNode(*s.Max))
	}
	return m
}

func windowNode(w *temporal.Window) *yaml.Node {
	if w == nil || w.IsZero() {
		return nil
	}
	m := mapping()
	if w.Calendar != nil {
		addKV(m, "calendar", scalar(w.Calendar.String()))
	}
	if w.Relative != nil {
		r := mapping()
		addStr(r, "target", w.Relative.Target)
		addStr(r, "relation", string(w.Relative.Relation))
		if w.Relative.Gap != nil {
			g := flowMapping()
			if w.Relative.Gap.Min != nil {
				addKV(g, "min", durationNode(*w.Relative.Gap.Min))
			}
			if w.Relative.Gap.Max != nil {
				addKV(g, "max", durationNode(*w.Relative.Gap.Max))
			}
			addKV(r, "gap", g)
		}
		addKV(m, "relative", r)
	}
	if w.Clock != nil {
		addKV(m, "clock", scalar(w.Clock.String()))
	}
	return m
}

func placementNode(p *temporal.Placement) *yaml.Node {
	if p == nil {
		return nil
	}
	m := mapping()
	addKV(m, "start", plain("!!timestamp", p.Start.Raw))
	addKV(m, "duration", durationSpecNode(&p.Duration))
	addStr(m, "location", p.Location)
	return m
}

func sourceNode(s Source) *yaml.Node {
	m := mapping()
	addStr(m, "author", s.Author)
	addStr(m, "harness", s.Harness)
	addStr(m, "model", s.Model)
	return m
}

func refsNode(rs []Ref) *yaml.Node {
	seq := sequence()
	for _, r := range SortRefs(rs) {
		m := flowMapping()
		addStrAlways(m, "id", r.ID)
		addStrAlways(m, "role", r.Role)
		seq.Content = append(seq.Content, m)
	}
	return seq
}

func partiesNode(ps []Party) *yaml.Node {
	seq := sequence()
	for _, p := range SortParties(ps) {
		m := mapping()
		addStrAlways(m, "uri", p.URI)
		addStrAlways(m, "status", p.Status)
		seq.Content = append(seq.Content, m)
	}
	return seq
}

func policyNode(p *Policy) *yaml.Node {
	if p == nil {
		return nil
	}
	m := flowMapping()
	if p.MaxDuration != nil {
		addKV(m, "max_duration", durationNode(*p.MaxDuration))
	}
	addStr(m, "stability", p.Stability)
	return m
}

func retiredNode(r *Retired) *yaml.Node {
	if r == nil {
		return nil
	}
	m := mapping()
	addStrAlways(m, "kind", r.Kind)
	addStr(m, "reason", r.Reason)
	addStr(m, "superseded_by", r.SupersededBy)
	addKV(m, "source", sourceNode(r.Source))
	addTime(m, "timestamp", r.Timestamp)
	return m
}

func acknowledgementsNode(as []Acknowledgement) *yaml.Node {
	seq := sequence()
	for _, a := range as {
		m := mapping()
		addStrAlways(m, "kind", a.Kind)
		addStr(m, "counterpart", a.Counterpart)
		addStr(m, "counterpart_version", a.CounterpartVersion)
		addStr(m, "reason", a.Reason)
		addKV(m, "source", sourceNode(a.Source))
		addTime(m, "timestamp", a.Timestamp)
		seq.Content = append(seq.Content, m)
	}
	return seq
}

func addExtras(m *yaml.Node, extras []Extra, version string) {
	for _, e := range extras {
		addKV(m, e.Key, e.Node)
	}
	addStr(m, "version", version)
}

func cadenceNode(c *temporal.Cadence) *yaml.Node {
	if c == nil {
		return nil
	}
	return scalar(c.Raw)
}

// --- objects --------------------------------------------------------------

func intentionNode(o *Intention) *yaml.Node {
	m := mapping()
	addStrAlways(m, "id", o.ID)
	addStrAlways(m, "subject", o.Subject)
	addStrAlways(m, "title", o.Title)
	addStr(m, "description", o.Description)
	addKV(m, "duration", durationSpecNode(o.Duration))
	addKV(m, "window", windowNode(o.Window))
	addStrAlways(m, "stability", o.Stability)
	addStr(m, "activity", o.Activity)
	addStrList(m, "location", o.Location, false)
	addStrList(m, "parties", o.Parties, false)
	addKV(m, "serves", refsNode(o.Serves))
	addKV(m, "cadence", cadenceNode(o.Cadence))
	addStr(m, "occurrence", o.Occurrence)
	addKV(m, "placement", placementNode(o.Placement))
	addStr(m, "preference", o.Preference)
	addKV(m, "auto_select", policyNode(o.AutoSelect))
	addKV(m, "auto_firm", policyNode(o.AutoFirm))
	addStr(m, "reference", o.Reference)
	addKV(m, "source", sourceNode(o.Source))
	addTime(m, "timestamp", o.Timestamp)
	addKV(m, "acknowledgements", acknowledgementsNode(o.Acknowledgements))
	addKV(m, "retired", retiredNode(o.Retired))
	addExtras(m, o.Extras, o.Version)
	return m
}

func availabilityNode(o *Availability) *yaml.Node {
	m := mapping()
	addStrAlways(m, "id", o.ID)
	addStrAlways(m, "subject", o.Subject)
	addStr(m, "title", o.Title)
	addStr(m, "description", o.Description)
	addKV(m, "duration", durationSpecNode(o.Duration))
	addKV(m, "window", windowNode(o.Window))
	addStrList(m, "conditional", o.Conditional, false)
	addStrList(m, "location", o.Location, false)
	addKV(m, "cadence", cadenceNode(o.Cadence))
	addStr(m, "valid_until", o.ValidUntil.String())
	addStrAlways(m, "scope", o.Scope)
	addKV(m, "source", sourceNode(o.Source))
	addTime(m, "timestamp", o.Timestamp)
	addKV(m, "retired", retiredNode(o.Retired))
	addExtras(m, o.Extras, o.Version)
	return m
}

func commitmentNode(o *Commitment) *yaml.Node {
	m := mapping()
	addStrAlways(m, "id", o.ID)
	addKV(m, "parties", partiesNode(o.Parties))
	addKV(m, "placement", placementNode(o.Placement))
	addStr(m, "intention", o.Intention)
	if o.Origin.Import {
		addKV(m, "origin", scalar("import"))
	} else {
		om := mapping()
		addStrAlways(om, "resolution", o.Origin.Resolution)
		addKV(m, "origin", om)
	}
	if o.Transparent != nil {
		addKV(m, "transparent", plain("!!bool", strconv.FormatBool(*o.Transparent)))
	}
	if o.External != nil {
		em := mapping()
		addStrAlways(em, "system", o.External.System)
		addStrAlways(em, "uid", o.External.UID)
		addKV(m, "external", em)
	}
	addStr(m, "title", o.Title)
	addStr(m, "description", o.Description)
	addKV(m, "source", sourceNode(o.Source))
	addTime(m, "timestamp", o.Timestamp)
	addKV(m, "acknowledgements", acknowledgementsNode(o.Acknowledgements))
	addKV(m, "retired", retiredNode(o.Retired))
	addExtras(m, o.Extras, o.Version)
	return m
}

func resolutionNode(o *Resolution) *yaml.Node {
	m := mapping()
	addStrAlways(m, "id", o.ID)
	addStrAlways(m, "intention", o.Intention)
	addKV(m, "placement", placementNode(o.Placement))
	addStrAlways(m, "selector", o.Selector)
	if o.CandidatesConsidered != nil {
		addKV(m, "candidates_considered", plain("!!int", strconv.Itoa(*o.CandidatesConsidered)))
	}
	addStrList(m, "displaced", o.Displaced, false)
	addKV(m, "source", sourceNode(o.Source))
	addTime(m, "timestamp", o.Timestamp)
	addExtras(m, o.Extras, o.Version)
	return m
}

// ToMap renders an object as a JSON-ready map via its canonical YAML, so the
// JSON a verb emits carries exactly the fields the file does.
func ToMap(obj Object) (map[string]any, error) {
	data, err := Encode(obj)
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return m, nil
}
