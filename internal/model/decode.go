package model

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/nodelogicau/intentions-cli/internal/temporal"
)

// Problem is a value-level defect found while reading or checking an object.
type Problem struct {
	Code    string `json:"code"`
	Field   string `json:"field,omitempty"`
	Message string `json:"message"`
}

func (p Problem) Error() string {
	if p.Field != "" {
		return p.Field + ": " + p.Message
	}
	return p.Message
}

// Problems is a list of problems that is itself an error.
type Problems []Problem

func (ps Problems) Error() string {
	msgs := make([]string, len(ps))
	for i, p := range ps {
		msgs[i] = p.Error()
	}
	return strings.Join(msgs, "; ")
}

// Err returns the list as an error, or nil when empty.
func (ps Problems) Err() error {
	if len(ps) == 0 {
		return nil
	}
	return ps
}

type decoder struct {
	problems Problems
}

func (d *decoder) problem(code, field, format string, args ...any) {
	d.problems = append(d.problems, Problem{Code: code, Field: field, Message: fmt.Sprintf(format, args...)})
}

// Decode parses one object file of the given type. A document that is not a
// single mapping, or carries a duplicate key, is a hard error. Value-level
// defects (an unparseable duration, an unknown enum) are returned as
// Problems alongside the partially populated object so that validation can
// report every one of them.
func Decode(data []byte, t Type) (Object, Problems, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, nil, fmt.Errorf("yaml: %v", err)
	}
	if doc.Kind != yaml.DocumentNode || len(doc.Content) != 1 || doc.Content[0].Kind != yaml.MappingNode {
		return nil, nil, fmt.Errorf("file must contain exactly one YAML mapping")
	}
	root := doc.Content[0]
	seen := map[string]bool{}
	for i := 0; i+1 < len(root.Content); i += 2 {
		k := root.Content[i].Value
		if seen[k] {
			return nil, nil, fmt.Errorf("duplicate key %q", k)
		}
		seen[k] = true
	}
	d := &decoder{}
	var obj Object
	switch t {
	case TypeIntention:
		obj = d.intention(root)
	case TypeAvailability:
		obj = d.availability(root)
	case TypeCommitment:
		obj = d.commitment(root)
	case TypeResolution:
		obj = d.resolution(root)
	default:
		return nil, nil, fmt.Errorf("unknown type %q", t)
	}
	return obj, d.problems, nil
}

// --- node readers ---------------------------------------------------------

func pairs(m *yaml.Node, fn func(key string, v *yaml.Node) bool) []Extra {
	var extras []Extra
	for i := 0; i+1 < len(m.Content); i += 2 {
		k, v := m.Content[i], m.Content[i+1]
		if !fn(k.Value, v) {
			extras = append(extras, Extra{Key: k.Value, Node: v})
		}
	}
	return extras
}

func (d *decoder) str(field string, v *yaml.Node) string {
	if v.Kind != yaml.ScalarNode {
		d.problem("invalid", field, "must be a scalar")
		return ""
	}
	if v.Tag == "!!null" {
		return ""
	}
	return v.Value
}

func (d *decoder) strList(field string, v *yaml.Node) []string {
	if v.Kind == yaml.ScalarNode && v.Tag == "!!null" {
		return nil
	}
	if v.Kind != yaml.SequenceNode {
		d.problem("invalid", field, "must be a list")
		return nil
	}
	out := make([]string, 0, len(v.Content))
	for i, c := range v.Content {
		out = append(out, d.str(fmt.Sprintf("%s[%d]", field, i), c))
	}
	return out
}

func (d *decoder) mapping(field string, v *yaml.Node) *yaml.Node {
	if v.Kind != yaml.MappingNode {
		d.problem("invalid", field, "must be a mapping")
		return nil
	}
	return v
}

func (d *decoder) timestamp(field string, v *yaml.Node) time.Time {
	s := d.str(field, v)
	if s == "" {
		return time.Time{}
	}
	t, err := temporal.ParseTimestamp(s)
	if err != nil {
		d.problem("unparseable", field, "%v", err)
		return time.Time{}
	}
	return t
}

func (d *decoder) boolean(field string, v *yaml.Node) *bool {
	s := strings.ToLower(d.str(field, v))
	switch s {
	case "true":
		b := true
		return &b
	case "false":
		b := false
		return &b
	}
	d.problem("invalid", field, "must be true or false")
	return nil
}

func (d *decoder) integer(field string, v *yaml.Node) *int {
	s := d.str(field, v)
	n, err := strconv.Atoi(s)
	if err != nil {
		d.problem("invalid", field, "must be an integer")
		return nil
	}
	return &n
}

func (d *decoder) source(field string, v *yaml.Node) Source {
	m := d.mapping(field, v)
	if m == nil {
		return Source{}
	}
	var s Source
	pairs(m, func(k string, v *yaml.Node) bool {
		switch k {
		case "author":
			s.Author = d.str(field+".author", v)
		case "harness":
			s.Harness = d.str(field+".harness", v)
		case "model":
			s.Model = d.str(field+".model", v)
		default:
			return false
		}
		return true
	})
	return s
}

func (d *decoder) duration(field string, v *yaml.Node) *temporal.Duration {
	s := d.str(field, v)
	if s == "" {
		return nil
	}
	dur, err := temporal.ParseDuration(s)
	if err != nil {
		d.problem("unparseable", field, "%v", err)
		return nil
	}
	return &dur
}

func (d *decoder) durationSpec(field string, v *yaml.Node) *temporal.DurationSpec {
	switch v.Kind {
	case yaml.ScalarNode:
		dur := d.duration(field, v)
		if dur == nil {
			return nil
		}
		return &temporal.DurationSpec{Nominal: *dur}
	case yaml.MappingNode:
		var spec temporal.DurationSpec
		hasNominal := false
		pairs(v, func(k string, c *yaml.Node) bool {
			switch k {
			case "nominal":
				if n := d.duration(field+".nominal", c); n != nil {
					spec.Nominal, hasNominal = *n, true
				}
			case "min":
				spec.Min = d.duration(field+".min", c)
			case "max":
				spec.Max = d.duration(field+".max", c)
			default:
				d.problem("invalid", field+"."+k, "unknown duration field")
			}
			return true
		})
		if !hasNominal {
			d.problem("invalid", field, "a ranged duration needs nominal")
			return nil
		}
		if err := spec.Validate(); err != nil {
			d.problem("invalid", field, "%v", err)
		}
		return &spec
	}
	d.problem("invalid", field, "must be a duration or {nominal, min, max}")
	return nil
}

func (d *decoder) window(field string, v *yaml.Node) *temporal.Window {
	m := d.mapping(field, v)
	if m == nil {
		return nil
	}
	w := &temporal.Window{}
	pairs(m, func(k string, c *yaml.Node) bool {
		switch k {
		case "calendar":
			s := d.str(field+".calendar", c)
			cal, err := temporal.ParseCalendar(s)
			if err != nil {
				d.problem("unparseable", field+".calendar", "%v", err)
			} else {
				w.Calendar = &cal
			}
		case "clock":
			s := d.str(field+".clock", c)
			ck, err := temporal.ParseClock(s)
			if err != nil {
				d.problem("unparseable", field+".clock", "%v", err)
			} else {
				w.Clock = &ck
			}
		case "relative":
			w.Relative = d.relative(field+".relative", c)
		default:
			d.problem("invalid", field+"."+k, "unknown window anchor; admitted anchors are calendar, relative, clock")
		}
		return true
	})
	return w
}

func (d *decoder) relative(field string, v *yaml.Node) *temporal.Relative {
	m := d.mapping(field, v)
	if m == nil {
		return nil
	}
	r := &temporal.Relative{}
	pairs(m, func(k string, c *yaml.Node) bool {
		switch k {
		case "target":
			r.Target = d.str(field+".target", c)
		case "relation":
			r.Relation = temporal.Relation(d.str(field+".relation", c))
		case "gap":
			gm := d.mapping(field+".gap", c)
			if gm != nil {
				r.Gap = &temporal.Gap{}
				pairs(gm, func(gk string, gc *yaml.Node) bool {
					switch gk {
					case "min":
						r.Gap.Min = d.duration(field+".gap.min", gc)
					case "max":
						r.Gap.Max = d.duration(field+".gap.max", gc)
					default:
						d.problem("invalid", field+".gap."+gk, "unknown gap field")
					}
					return true
				})
			}
		default:
			d.problem("invalid", field+"."+k, "unknown relative anchor field")
		}
		return true
	})
	if err := r.Validate(); err != nil {
		d.problem("invalid", field, "%v", err)
	}
	return r
}

func (d *decoder) cadence(field string, v *yaml.Node) *temporal.Cadence {
	s := d.str(field, v)
	if s == "" {
		return nil
	}
	c, err := temporal.ParseCadence(s)
	if err != nil {
		d.problem("unparseable", field, "%v", err)
		return nil
	}
	return &c
}

func (d *decoder) placement(field string, v *yaml.Node) *temporal.Placement {
	m := d.mapping(field, v)
	if m == nil {
		return nil
	}
	p := &temporal.Placement{}
	hasStart, hasDuration := false, false
	pairs(m, func(k string, c *yaml.Node) bool {
		switch k {
		case "start":
			s, err := temporal.ParsePlacementStart(d.str(field+".start", c))
			if err != nil {
				d.problem("unparseable", field+".start", "%v", err)
			} else {
				p.Start, hasStart = s, true
			}
		case "duration":
			if spec := d.durationSpec(field+".duration", c); spec != nil {
				p.Duration, hasDuration = *spec, true
			}
		case "location":
			p.Location = d.str(field+".location", c)
		default:
			d.problem("invalid", field+"."+k, "unknown placement field")
		}
		return true
	})
	if !hasStart {
		d.problem("missing", field+".start", "a placement needs a start")
	}
	if !hasDuration {
		d.problem("missing", field+".duration", "a placement needs a duration")
	}
	if hasStart && hasDuration {
		if err := p.Validate(); err != nil {
			d.problem("invalid", field, "%v", err)
		}
	}
	return p
}

func (d *decoder) refs(field string, v *yaml.Node) []Ref {
	if v.Kind == yaml.ScalarNode && v.Tag == "!!null" {
		return []Ref{}
	}
	if v.Kind != yaml.SequenceNode {
		d.problem("invalid", field, "must be a list of {id, role}")
		return nil
	}
	out := make([]Ref, 0, len(v.Content))
	for i, c := range v.Content {
		f := fmt.Sprintf("%s[%d]", field, i)
		m := d.mapping(f, c)
		if m == nil {
			continue
		}
		var r Ref
		pairs(m, func(k string, cv *yaml.Node) bool {
			switch k {
			case "id":
				r.ID = d.str(f+".id", cv)
			case "role":
				r.Role = d.str(f+".role", cv)
			default:
				d.problem("invalid", f+"."+k, "unknown reference field")
			}
			return true
		})
		out = append(out, r)
	}
	return out
}

func (d *decoder) parties(field string, v *yaml.Node) []Party {
	if v.Kind != yaml.SequenceNode {
		d.problem("invalid", field, "must be a list of {uri, status}")
		return nil
	}
	out := make([]Party, 0, len(v.Content))
	for i, c := range v.Content {
		f := fmt.Sprintf("%s[%d]", field, i)
		m := d.mapping(f, c)
		if m == nil {
			continue
		}
		var p Party
		pairs(m, func(k string, cv *yaml.Node) bool {
			switch k {
			case "uri":
				p.URI = d.str(f+".uri", cv)
			case "status":
				p.Status = d.str(f+".status", cv)
			default:
				d.problem("invalid", f+"."+k, "unknown party field")
			}
			return true
		})
		out = append(out, p)
	}
	return out
}

func (d *decoder) policy(field string, v *yaml.Node) *Policy {
	m := d.mapping(field, v)
	if m == nil {
		return nil
	}
	p := &Policy{}
	pairs(m, func(k string, c *yaml.Node) bool {
		switch k {
		case "max_duration":
			p.MaxDuration = d.duration(field+".max_duration", c)
		case "stability":
			p.Stability = d.str(field+".stability", c)
		default:
			d.problem("invalid", field+"."+k, "unknown policy term; admitted terms are max_duration, stability")
		}
		return true
	})
	return p
}

func (d *decoder) retired(field string, v *yaml.Node) *Retired {
	m := d.mapping(field, v)
	if m == nil {
		return nil
	}
	r := &Retired{}
	pairs(m, func(k string, c *yaml.Node) bool {
		switch k {
		case "kind":
			r.Kind = d.str(field+".kind", c)
		case "reason":
			r.Reason = d.str(field+".reason", c)
		case "superseded_by":
			r.SupersededBy = d.str(field+".superseded_by", c)
		case "source":
			r.Source = d.source(field+".source", c)
		case "timestamp":
			r.Timestamp = d.timestamp(field+".timestamp", c)
		default:
			d.problem("invalid", field+"."+k, "unknown retirement field")
		}
		return true
	})
	return r
}

func (d *decoder) acknowledgements(field string, v *yaml.Node) []Acknowledgement {
	if v.Kind == yaml.ScalarNode && v.Tag == "!!null" {
		return []Acknowledgement{}
	}
	if v.Kind != yaml.SequenceNode {
		d.problem("invalid", field, "must be a list")
		return nil
	}
	out := make([]Acknowledgement, 0, len(v.Content))
	for i, c := range v.Content {
		f := fmt.Sprintf("%s[%d]", field, i)
		m := d.mapping(f, c)
		if m == nil {
			continue
		}
		var a Acknowledgement
		pairs(m, func(k string, cv *yaml.Node) bool {
			switch k {
			case "kind":
				a.Kind = d.str(f+".kind", cv)
			case "counterpart":
				a.Counterpart = d.str(f+".counterpart", cv)
			case "counterpart_version":
				a.CounterpartVersion = d.str(f+".counterpart_version", cv)
			case "reason":
				a.Reason = d.str(f+".reason", cv)
			case "source":
				a.Source = d.source(f+".source", cv)
			case "timestamp":
				a.Timestamp = d.timestamp(f+".timestamp", cv)
			default:
				d.problem("invalid", f+"."+k, "unknown acknowledgement field")
			}
			return true
		})
		out = append(out, a)
	}
	return out
}

// --- objects --------------------------------------------------------------

func (d *decoder) intention(root *yaml.Node) *Intention {
	o := &Intention{Serves: nil}
	o.Extras = pairs(root, func(k string, v *yaml.Node) bool {
		switch k {
		case "id":
			o.ID = d.str(k, v)
		case "subject":
			o.Subject = d.str(k, v)
		case "title":
			o.Title = d.str(k, v)
		case "description":
			o.Description = d.str(k, v)
		case "duration":
			o.Duration = d.durationSpec(k, v)
		case "window":
			o.Window = d.window(k, v)
		case "stability":
			o.Stability = d.str(k, v)
		case "activity":
			o.Activity = d.str(k, v)
		case "location":
			o.Location = d.strList(k, v)
		case "parties":
			o.Parties = d.strList(k, v)
		case "serves":
			o.Serves = d.refs(k, v)
		case "cadence":
			o.Cadence = d.cadence(k, v)
		case "occurrence":
			o.Occurrence = d.str(k, v)
		case "placement":
			o.Placement = d.placement(k, v)
		case "preference":
			o.Preference = d.str(k, v)
		case "auto_select":
			o.AutoSelect = d.policy(k, v)
		case "auto_firm":
			o.AutoFirm = d.policy(k, v)
		case "reference":
			o.Reference = d.str(k, v)
		case "source":
			o.Source = d.source(k, v)
		case "timestamp":
			o.Timestamp = d.timestamp(k, v)
		case "acknowledgements":
			o.Acknowledgements = d.acknowledgements(k, v)
		case "retired":
			o.Retired = d.retired(k, v)
		case "version":
			o.Version = d.str(k, v)
		default:
			return false
		}
		return true
	})
	return o
}

func (d *decoder) availability(root *yaml.Node) *Availability {
	o := &Availability{}
	o.Extras = pairs(root, func(k string, v *yaml.Node) bool {
		switch k {
		case "id":
			o.ID = d.str(k, v)
		case "subject":
			o.Subject = d.str(k, v)
		case "title":
			o.Title = d.str(k, v)
		case "description":
			o.Description = d.str(k, v)
		case "duration":
			o.Duration = d.durationSpec(k, v)
		case "window":
			o.Window = d.window(k, v)
		case "conditional":
			o.Conditional = d.strList(k, v)
		case "location":
			o.Location = d.strList(k, v)
		case "cadence":
			o.Cadence = d.cadence(k, v)
		case "valid_until":
			s := d.str(k, v)
			if s != "" {
				vu, err := temporal.ParseValidUntil(s)
				if err != nil {
					d.problem("unparseable", k, "%v", err)
				} else {
					o.ValidUntil = vu
				}
			}
		case "scope":
			o.Scope = d.str(k, v)
		case "source":
			o.Source = d.source(k, v)
		case "timestamp":
			o.Timestamp = d.timestamp(k, v)
		case "retired":
			o.Retired = d.retired(k, v)
		case "version":
			o.Version = d.str(k, v)
		default:
			return false
		}
		return true
	})
	return o
}

func (d *decoder) commitment(root *yaml.Node) *Commitment {
	o := &Commitment{}
	o.Extras = pairs(root, func(k string, v *yaml.Node) bool {
		switch k {
		case "id":
			o.ID = d.str(k, v)
		case "parties":
			o.Parties = d.parties(k, v)
		case "placement":
			o.Placement = d.placement(k, v)
		case "intention":
			o.Intention = d.str(k, v)
		case "origin":
			switch v.Kind {
			case yaml.ScalarNode:
				if d.str(k, v) == "import" {
					o.Origin.Import = true
				} else {
					d.problem("invalid", k, "must be import or {resolution: res_…}")
				}
			case yaml.MappingNode:
				pairs(v, func(ok string, ov *yaml.Node) bool {
					if ok == "resolution" {
						o.Origin.Resolution = d.str(k+".resolution", ov)
					} else {
						d.problem("invalid", k+"."+ok, "unknown origin field")
					}
					return true
				})
			default:
				d.problem("invalid", k, "must be import or {resolution: res_…}")
			}
		case "transparent":
			o.Transparent = d.boolean(k, v)
		case "external":
			m := d.mapping(k, v)
			if m != nil {
				o.External = &External{}
				pairs(m, func(ek string, ev *yaml.Node) bool {
					switch ek {
					case "system":
						o.External.System = d.str(k+".system", ev)
					case "uid":
						o.External.UID = d.str(k+".uid", ev)
					default:
						d.problem("invalid", k+"."+ek, "unknown external field")
					}
					return true
				})
			}
		case "title":
			o.Title = d.str(k, v)
		case "description":
			o.Description = d.str(k, v)
		case "source":
			o.Source = d.source(k, v)
		case "timestamp":
			o.Timestamp = d.timestamp(k, v)
		case "acknowledgements":
			o.Acknowledgements = d.acknowledgements(k, v)
		case "retired":
			o.Retired = d.retired(k, v)
		case "version":
			o.Version = d.str(k, v)
		default:
			return false
		}
		return true
	})
	return o
}

func (d *decoder) resolution(root *yaml.Node) *Resolution {
	o := &Resolution{}
	o.Extras = pairs(root, func(k string, v *yaml.Node) bool {
		switch k {
		case "id":
			o.ID = d.str(k, v)
		case "intention":
			o.Intention = d.str(k, v)
		case "placement":
			o.Placement = d.placement(k, v)
		case "selector":
			o.Selector = d.str(k, v)
		case "candidates_considered":
			o.CandidatesConsidered = d.integer(k, v)
		case "displaced":
			o.Displaced = d.strList(k, v)
		case "source":
			o.Source = d.source(k, v)
		case "timestamp":
			o.Timestamp = d.timestamp(k, v)
		case "version":
			o.Version = d.str(k, v)
		default:
			return false
		}
		return true
	})
	return o
}
