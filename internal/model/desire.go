package model

import (
	"fmt"
	"strings"
	"time"

	"github.com/nodelogicau/intentions-cli/internal/temporal"
)

// Desire is a want the person has expressed and not yet committed to: the
// rung below intention. It carries none of what makes an intention an
// intention, so nothing plans, checks or grounds it. Canonical order: id,
// version, subject, title, description, activity, location, serves,
// reference, source, timestamp, retired.
type Desire struct {
	format      string
	ID          string
	Subject     string
	Title       string
	Description string
	Activity    string
	Location    []string
	Parties     []string // URIs of other particulars the want involves, a hint outside the projection
	Serves      []Ref
	Reference   string
	Source      Source
	Timestamp   time.Time
	Retired     *Retired
	Version     string
	Extras      []Extra
}

// DesireRetirementKinds are the admitted kinds; adopted is written only by
// adoption, never by a retirement act.
var DesireRetirementKinds = []string{"abandoned", "superseded", "adopted"}

// desireForbidden are an intention's temporal and deontic fields, which a
// desire may not carry. The decoder names each rather than keeping it as
// an unknown extra, so validation reports it and a write is refused.
var desireForbidden = []string{"duration", "window", "stability", "firmed_under", "cadence", "occurrence", "placement", "preference", "auto_select", "auto_firm", "acknowledgements"}

func (o *Desire) GetID() string         { return o.ID }
func (o *Desire) GetType() Type         { return TypeDesire }
func (o *Desire) GetSource() Source     { return o.Source }
func (o *Desire) GetRetired() *Retired  { return o.Retired }
func (o *Desire) SubjectURI() string    { return o.Subject }
func (o *Desire) CachedVersion() string { return o.Version }
func (o *Desire) Format() string        { return formatOf(o.format) }
func (o *Desire) SetFormat(f string)    { o.format = f }
func (o *Desire) SetVersion(v string)   { o.Version = v }
func (o *Desire) Refs() []string {
	var ids []string
	for _, r := range o.Serves {
		ids = append(ids, r.ID)
	}
	if o.Retired != nil {
		ids = append(ids, o.Retired.SupersededBy, o.Retired.AdoptedAs)
	}
	return uniqueSorted(ids)
}

func (c *checker) desire(o *Desire) {
	c.require("subject", o.Subject)
	c.uri("subject", o.Subject)
	c.require("title", o.Title)
	c.term("activity", o.Activity)
	c.uris("location", o.Location)
	c.uris("parties", o.Parties)
	if o.Serves == nil {
		c.add("missing", "serves", "serves is required, possibly empty")
	}
	for i, r := range o.Serves {
		f := fmt.Sprintf("serves[%d]", i)
		c.require(f+".id", r.ID)
		if r.ID != "" && !ValidID(r.ID) {
			c.add("invalid", f+".id", "%q is not an identifier", r.ID)
		}
		if r.Role != RoleForTheSakeOf {
			c.add("serves_role", f+".role", "a desire serves only for-the-sake-of; %q is not admitted", r.Role)
		}
	}
	c.source("source", o.Source, true)
	if o.Timestamp.IsZero() {
		c.add("missing", "timestamp", "timestamp is required")
	}
	c.retired(TypeDesire, o.Retired)
}

// CheckDesireServes refuses serves entries a desire may not carry: any role
// but for-the-sake-of, and any target that is not an active terminus of the
// desire's own subject. The terminus may be tentative: a draft want may
// point at a draft self, and nothing rests on either.
func CheckDesireServes(g Graph, o *Desire) error {
	for _, r := range o.Serves {
		if r.Role != RoleForTheSakeOf {
			return refuse("serves_role", "a desire serves only for-the-sake-of; %q is not admitted", r.Role)
		}
		target, ok := g.Get(r.ID)
		if !ok {
			return refuse("dangling", "serves target %s does not exist", r.ID)
		}
		t, isIntention := target.(*Intention)
		if !isIntention || !t.IsTerminus() {
			return refuse("serves_target", "serves target %s is not a terminus: a desire is for the sake of who the person is, an intention with no window, duration or serves", r.ID)
		}
		if t.Retired != nil {
			return refuse("serves_target", "serves target %s is retired", r.ID)
		}
		if t.Subject != o.Subject {
			return refuse("serves_target", "serves target %s belongs to %s, not to %s; a terminus grounds only its own subject's wants", r.ID, t.Subject, o.Subject)
		}
	}
	return nil
}

// Adopt builds the intention a desire becomes and the retirement record that
// points at it. The intention is tentative, carries the desire's subject,
// title, description, serves, reference, activity and location, the
// duration and window supplied, and the act's source. It refuses a retired
// desire, and refuses a bare adoption, one whose result would be a terminus:
// an adopted want is a plan, and a self is declared, not adopted.
func Adopt(d *Desire, duration *temporal.DurationSpec, window *temporal.Window, act Source, at time.Time) (*Intention, Retired, error) {
	if d.Retired != nil {
		return nil, Retired{}, refuse("retired", "%s is retired (%s) and cannot be adopted", d.ID, d.Retired.Kind)
	}
	if len(d.Serves) == 0 && duration == nil && window == nil {
		var missing []string
		missing = append(missing, "a why (edit the desire with --serves <terminus>:for-the-sake-of)", "a when (--duration, or --calendar, --clock or --relative)")
		return nil, Retired{}, refuse("bare_adoption", "adopting %s as it stands would write a terminus, a self titled as a task; a want becomes a plan with %s", d.ID, strings.Join(missing, " or "))
	}
	in := &Intention{
		ID:               MintID(TypeIntention),
		Subject:          d.Subject,
		Title:            d.Title,
		Description:      d.Description,
		Duration:         duration,
		Window:           window,
		Stability:        "tentative",
		Activity:         d.Activity,
		Location:         append([]string(nil), d.Location...),
		Parties:          append([]string(nil), d.Parties...),
		Serves:           append([]Ref{}, d.Serves...),
		Reference:        d.Reference,
		Source:           act,
		Timestamp:        at,
		Acknowledgements: []Acknowledgement{},
	}
	r := Retired{Kind: "adopted", AdoptedAs: in.ID, Source: act, Timestamp: at}
	return in, r, nil
}
