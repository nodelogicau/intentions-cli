package model

import (
	"fmt"
	"regexp"
	"strings"
)

// Field-level rules shared by every write path and by validate. They look at
// one object in isolation; anything that needs the rest of the workspace is
// in policy.go or in the query package.

var (
	reTerm = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
	reURI  = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9+.-]*:\S+$`)
)

// ValidTerm reports whether s is a lowercase kebab-case activity term.
func ValidTerm(s string) bool { return reTerm.MatchString(s) }

// ValidURI reports whether s is an absolute URI: a scheme followed by a colon
// and something.
func ValidURI(s string) bool { return reURI.MatchString(s) }

func oneOf(v string, vs []string) bool {
	for _, x := range vs {
		if x == v {
			return true
		}
	}
	return false
}

// RetirementKinds returns the admitted kinds for a type.
func RetirementKinds(t Type) []string {
	switch t {
	case TypeIntention:
		return IntentionRetirementKinds
	case TypeAvailability:
		return AvailabilityRetirementKinds
	case TypeCommitment:
		return CommitmentRetirementKinds
	}
	return nil
}

type checker struct {
	ps Problems
}

func (c *checker) add(code, field, format string, args ...any) {
	c.ps = append(c.ps, Problem{Code: code, Field: field, Message: fmt.Sprintf(format, args...)})
}

func (c *checker) require(field, v string) {
	if strings.TrimSpace(v) == "" {
		c.add("missing", field, "%s is required", field)
	}
}

func (c *checker) enum(field, v string, vs []string, required bool) {
	if v == "" {
		if required {
			c.add("missing", field, "%s is required: one of %s", field, strings.Join(vs, ", "))
		}
		return
	}
	if !oneOf(v, vs) {
		c.add("invalid", field, "%q is not admitted; must be one of %s", v, strings.Join(vs, ", "))
	}
}

func (c *checker) uri(field, v string) {
	if v != "" && !ValidURI(v) {
		c.add("invalid", field, "%q is not an absolute URI", v)
	}
}

func (c *checker) uris(field string, vs []string) {
	for i, v := range vs {
		c.uri(fmt.Sprintf("%s[%d]", field, i), v)
	}
}

func (c *checker) term(field, v string) {
	if v != "" && !ValidTerm(v) {
		c.add("malformed_term", field, "%q is not a lowercase kebab-case term", v)
	}
}

func (c *checker) source(field string, s Source, needAuthor bool) {
	if needAuthor && strings.TrimSpace(s.Author) == "" {
		c.add("no_author", field+".author", "an author is required: a speech act with no speaker is not one")
	}
}

func (c *checker) retired(t Type, r *Retired) {
	if r == nil {
		return
	}
	c.enum("retired.kind", r.Kind, RetirementKinds(t), true)
	switch {
	case r.Kind == "superseded" && r.SupersededBy == "":
		c.add("superseded_by", "retired.superseded_by", "superseded_by is required when kind is superseded")
	case r.Kind != "superseded" && r.SupersededBy != "":
		c.add("superseded_by", "retired.superseded_by", "superseded_by is admitted only when kind is superseded")
	}
	if r.SupersededBy != "" && !ValidID(r.SupersededBy) {
		c.add("invalid", "retired.superseded_by", "%q is not an identifier", r.SupersededBy)
	}
	if r.Timestamp.IsZero() {
		c.add("missing", "retired.timestamp", "a retirement record needs a timestamp")
	}
}

func (c *checker) policy(field string, p *Policy) {
	if p == nil {
		return
	}
	c.enum(field+".stability", p.Stability, Stabilities, false)
	if p.MaxDuration == nil && p.Stability == "" {
		c.add("invalid", field, "a policy condition needs at least one term")
	}
}

func (c *checker) refs(field string, rs []Ref) {
	for i, r := range rs {
		f := fmt.Sprintf("%s[%d]", field, i)
		c.require(f+".id", r.ID)
		if r.ID != "" && !ValidID(r.ID) {
			c.add("invalid", f+".id", "%q is not an identifier", r.ID)
		}
		c.enum(f+".role", r.Role, Roles, true)
	}
}

// Check applies every single-object rule and returns the problems found.
func Check(obj Object) Problems {
	c := &checker{}
	if obj.GetID() != "" && !ValidID(obj.GetID()) {
		c.add("invalid", "id", "%q is not an identifier", obj.GetID())
	}
	c.require("id", obj.GetID())
	switch o := obj.(type) {
	case *Intention:
		c.intention(o)
	case *Availability:
		c.availability(o)
	case *Commitment:
		c.commitment(o)
	case *Resolution:
		c.resolution(o)
	}
	return c.ps
}

func (c *checker) intention(o *Intention) {
	c.require("subject", o.Subject)
	c.uri("subject", o.Subject)
	c.require("title", o.Title)
	c.enum("stability", o.Stability, Stabilities, true)
	c.term("activity", o.Activity)
	c.uris("location", o.Location)
	c.uris("parties", o.Parties)
	if o.Serves == nil {
		c.add("missing", "serves", "serves is required, possibly empty")
	}
	c.refs("serves", o.Serves)
	if o.Occurrence != "" && o.InstanceOf() == "" {
		c.add("invalid", "occurrence", "occurrence is admitted only on a generated instance carrying an instance-of reference")
	}
	c.enum("preference", o.Preference, Preferences, false)
	c.policy("auto_select", o.AutoSelect)
	c.policy("auto_firm", o.AutoFirm)
	if o.HasCondition() && !o.IsTerminus() {
		what := "a window, a duration or a serves entry"
		if o.Cadence != nil {
			what = "a cadence"
		}
		c.add("policy_not_terminus", "auto_firm", "auto_select and auto_firm are admitted on termini only; this intention has %s", what)
	}
	if o.FirmedUnder != "" {
		if o.Stability != "firm" {
			c.add("firmed_under_not_firm", "firmed_under", "firmed_under is admitted only on a firm intention")
		}
		if !ValidID(o.FirmedUnder) {
			c.add("invalid", "firmed_under", "%q is not an identifier", o.FirmedUnder)
		}
	}
	if o.Stability == "firm" && o.Source.Harness != "" && o.FirmedUnder == "" {
		c.add("harness_firm", "firmed_under", "stability is firm and the source carries harness %q but no firmed_under names the policy; a harness may not firm without one", o.Source.Harness)
	}
	if o.Cadence != nil && (o.Window == nil || o.Window.Calendar == nil) {
		c.add("cadence_anchor", "cadence", "a cadence needs a window with a calendar anchor to expand within")
	}
	if o.Window != nil {
		if err := o.Window.Validate(); err != nil {
			c.add("invalid", "window", "%v", err)
		}
	}
	if o.Duration != nil {
		if err := o.Duration.Validate(); err != nil {
			c.add("ranged_duration", "duration", "%v", err)
		}
	}
	if o.Placement != nil {
		c.uri("placement.location", o.Placement.Location)
		if err := o.Placement.Validate(); err != nil {
			c.add("all_day_duration", "placement", "%v", err)
		}
	}
	c.source("source", o.Source, true)
	if o.Timestamp.IsZero() {
		c.add("missing", "timestamp", "timestamp is required")
	}
	if o.Acknowledgements == nil {
		c.add("missing", "acknowledgements", "acknowledgements is required, possibly empty")
	}
	c.retired(TypeIntention, o.Retired)
}

func (c *checker) availability(o *Availability) {
	c.require("subject", o.Subject)
	c.uri("subject", o.Subject)
	if o.Duration == nil {
		c.add("missing", "duration", "duration is required")
	} else if err := o.Duration.Validate(); err != nil {
		c.add("ranged_duration", "duration", "%v", err)
	}
	if o.Window == nil {
		c.add("missing", "window", "window is required")
	} else if err := o.Window.Validate(); err != nil {
		c.add("invalid", "window", "%v", err)
	}
	if o.Cadence != nil && (o.Window == nil || o.Window.Calendar == nil) {
		c.add("cadence_anchor", "cadence", "a cadence needs a window with a calendar anchor to expand within")
	}
	for i, t := range o.Conditional {
		c.term(fmt.Sprintf("conditional[%d]", i), t)
	}
	c.uris("location", o.Location)
	c.enum("scope", o.Scope, Scopes, true)
	c.source("source", o.Source, true)
	if o.Timestamp.IsZero() {
		c.add("missing", "timestamp", "timestamp is required")
	}
	c.retired(TypeAvailability, o.Retired)
}

func (c *checker) commitment(o *Commitment) {
	if len(o.Parties) == 0 {
		c.add("missing", "parties", "parties is required")
	}
	for i, p := range o.Parties {
		f := fmt.Sprintf("parties[%d]", i)
		c.require(f+".uri", p.URI)
		c.uri(f+".uri", p.URI)
		c.enum(f+".status", p.Status, PartyStatuses, true)
	}
	if o.Placement == nil {
		c.add("missing", "placement", "a commitment always has a placement")
	} else {
		c.uri("placement.location", o.Placement.Location)
		if err := o.Placement.Validate(); err != nil {
			c.add("all_day_duration", "placement", "%v", err)
		}
	}
	if o.Intention != "" && !ValidID(o.Intention) {
		c.add("invalid", "intention", "%q is not an identifier", o.Intention)
	}
	switch {
	case o.Origin.Import && o.Origin.Resolution != "":
		c.add("invalid", "origin", "origin is either import or a resolution, not both")
	case !o.Origin.Import && o.Origin.Resolution == "":
		c.add("missing", "origin", "origin is required: import or {resolution: res_…}")
	case o.Origin.Resolution != "" && !ValidID(o.Origin.Resolution):
		c.add("invalid", "origin.resolution", "%q is not an identifier", o.Origin.Resolution)
	}
	if o.IsTransparent() && !o.Origin.Import {
		c.add("transparent_origin", "transparent", "a commitment born of a resolution consumed supply to exist and cannot be transparent")
	}
	if o.External != nil {
		c.enum("external.system", o.External.System, ExternalSystems, true)
		c.require("external.uid", o.External.UID)
	}
	c.source("source", o.Source, false)
	if o.Timestamp.IsZero() {
		c.add("missing", "timestamp", "timestamp is required")
	}
	if o.Acknowledgements == nil {
		c.add("missing", "acknowledgements", "acknowledgements is required, possibly empty")
	}
	c.retired(TypeCommitment, o.Retired)
}

func (c *checker) resolution(o *Resolution) {
	c.require("intention", o.Intention)
	if o.Intention != "" && !ValidID(o.Intention) {
		c.add("invalid", "intention", "%q is not an identifier", o.Intention)
	}
	if o.Placement == nil {
		c.add("missing", "placement", "a resolution records a placement")
	} else if err := o.Placement.Validate(); err != nil {
		c.add("all_day_duration", "placement", "%v", err)
	}
	c.require("selector", o.Selector)
	if o.Selector != "" && o.Selector != "person" && !ValidID(o.Selector) {
		c.add("invalid", "selector", "%q must be person or the id of the policy that authorised selection", o.Selector)
	}
	for i, d := range o.Displaced {
		if !ValidID(d) {
			c.add("invalid", fmt.Sprintf("displaced[%d]", i), "%q is not an identifier", d)
		}
	}
	c.source("source", o.Source, false)
	if o.Timestamp.IsZero() {
		c.add("missing", "timestamp", "timestamp is required")
	}
}
