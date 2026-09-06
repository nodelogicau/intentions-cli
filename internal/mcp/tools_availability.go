package mcp

import (
	"context"
	"fmt"
	"strings"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/nodelogicau/intentions-cli/internal/apperr"
	"github.com/nodelogicau/intentions-cli/internal/consistency"
	"github.com/nodelogicau/intentions-cli/internal/model"
	"github.com/nodelogicau/intentions-cli/internal/projection"
	"github.com/nodelogicau/intentions-cli/internal/render"
	"github.com/nodelogicau/intentions-cli/internal/resolve"
	"github.com/nodelogicau/intentions-cli/internal/store"
	"github.com/nodelogicau/intentions-cli/internal/temporal"
)

type availabilityIn struct {
	Subject     string    `json:"subject,omitempty" jsonschema:"URI of the particular whose availability this is (required on add; no default applies)"`
	Title       string    `json:"title,omitempty"`
	Description string    `json:"description,omitempty"`
	Duration    any       `json:"duration,omitempty" jsonschema:"capacity offered per occasion: ISO 8601 duration or {nominal, min, max} (required on add)"`
	Window      *windowIn `json:"window,omitempty" jsonschema:"calendar and/or clock; a cadence needs a calendar anchor"`
	Conditional []string  `json:"conditional,omitempty" jsonschema:"activity terms this supply is good for; absent means anything"`
	Location    []string  `json:"location,omitempty" jsonschema:"URIs at which this capacity holds; absent means anywhere"`
	Cadence     string    `json:"cadence,omitempty" jsonschema:"RRULE with date-level parts only"`
	ValidUntil  string    `json:"valid_until,omitempty" jsonschema:"an EDTF expression or RFC 3339 datetime"`
	Scope       string    `json:"scope,omitempty" jsonschema:"personal (default) | organisation | public; only ever widened"`
	Timestamp   string    `json:"timestamp,omitempty"`
	Source      *sourceIn `json:"source,omitempty"`
}

func (s *Server) applyAvailability(in availabilityIn, o *model.Availability, g *store.Graph, ctx temporal.Context) error {
	if in.Subject != "" {
		o.Subject = in.Subject
	}
	if in.Title != "" {
		o.Title = in.Title
	}
	if in.Description != "" {
		o.Description = in.Description
	}
	if in.Duration != nil {
		d, err := durationIn(in.Duration)
		if err != nil {
			return err
		}
		o.Duration = d
	}
	w, err := window(in.Window, o.Window, ctx, g)
	if err != nil {
		return err
	}
	if w != nil {
		o.Window = w
	}
	if in.Conditional != nil {
		o.Conditional = model.SortStrings(in.Conditional)
	}
	if in.Location != nil {
		if err := checkURIs("location", in.Location); err != nil {
			return err
		}
		o.Location = model.SortStrings(in.Location)
	}
	if in.Cadence != "" {
		c, err := cadenceIn(in.Cadence)
		if err != nil {
			return err
		}
		o.Cadence = c
	}
	if in.ValidUntil != "" {
		v, err := temporal.ParseValidUntil(in.ValidUntil)
		if err != nil {
			return apperr.Invalid("valid_until: %v", err)
		}
		o.ValidUntil = v
	}
	if in.Scope != "" {
		o.Scope = in.Scope
	}
	if o.Scope == "" {
		o.Scope = "personal"
	}
	return nil
}

func (s *Server) effectiveValidUntil(e resolve.Env, o *model.Availability) (string, string) {
	if !o.ValidUntil.IsZero() {
		return o.ValidUntil.Instant(e.Ctx).UTC().Format(temporal.TimeLayout), "explicit"
	}
	if o.Cadence != nil {
		return temporal.AddDuration(o.Timestamp, s.ws.Config.DefaultHorizon()).UTC().Format(temporal.TimeLayout), "default_horizon"
	}
	if t := e.EffectiveValidUntil(o); !t.IsZero() {
		return t.UTC().Format(temporal.TimeLayout), "window"
	}
	return "", "unbounded"
}

func (s *Server) availabilityResult(e resolve.Env, o *model.Availability, withFlags bool) (map[string]any, error) {
	out, err := render.Object(o)
	if err != nil {
		return nil, err
	}
	t, how := s.effectiveValidUntil(e, o)
	if t != "" {
		out["effective_valid_until"] = t
	} else {
		out["effective_valid_until"] = nil
	}
	out["effective_valid_until_from"] = how
	if withFlags {
		flags, err := consistency.Check(e, []string{o.ID})
		if err != nil {
			flags = []consistency.Flag{}
		}
		out["flags"] = flags
	}
	return out, nil
}

func (s *Server) availabilityAdd(ctx context.Context, req *sdk.CallToolRequest, in availabilityIn) (*sdk.CallToolResult, any, error) {
	if Clean(in.Subject) == "" {
		return errResult(apperr.Usage("subject is required: availability names its particular explicitly")), nil, nil
	}
	if in.Duration == nil {
		return errResult(apperr.Usage("duration is required: the capacity offered per occasion")), nil, nil
	}
	if in.Window == nil {
		return errResult(apperr.Usage("a window is required: calendar and/or clock")), nil, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	g, err := s.load()
	if err != nil {
		return errResult(err), nil, nil
	}
	at, err := now("")
	if err != nil {
		return errResult(err), nil, nil
	}
	ts, err := timestamp(in.Timestamp, at)
	if err != nil {
		return errResult(err), nil, nil
	}
	src := s.source(req, in.Source)
	if src.Author == "" {
		return errResult(apperr.Refused("an author is required: pass source.author, or set defaults.source.author in intentions.yaml")), nil, nil
	}
	o := &model.Availability{ID: model.MintID(model.TypeAvailability), Source: src, Timestamp: ts}
	if err := s.applyAvailability(in, o, g, s.ws.Config.Context(at)); err != nil {
		return errResult(err), nil, nil
	}
	if err := s.write(o); err != nil {
		return errResult(err), nil, nil
	}
	g.Add(o)
	out, err := s.availabilityResult(resolve.NewEnv(s.ws, g, at), o, true)
	if err != nil {
		return errResult(err), nil, nil
	}
	out["created"] = true
	return okResult(fmt.Sprintf("Created %s (%s)", o.ID, o.Version)), out, nil
}

type renewIn struct {
	ID         string    `json:"id"`
	ValidUntil string    `json:"valid_until" jsonschema:"the new horizon: an EDTF expression or RFC 3339 datetime; never earlier than the current one"`
	Source     *sourceIn `json:"source,omitempty"`
}

func (s *Server) availabilityRenew(ctx context.Context, req *sdk.CallToolRequest, in renewIn) (*sdk.CallToolResult, any, error) {
	if Clean(in.ValidUntil) == "" {
		return errResult(apperr.Usage("valid_until is required")), nil, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	g, err := s.load()
	if err != nil {
		return errResult(err), nil, nil
	}
	before, err := getAvailability(g, in.ID)
	if err != nil {
		return errResult(err), nil, nil
	}
	if err := model.CheckNotRetired(before); err != nil {
		return errResult(err), nil, nil
	}
	v, err := temporal.ParseValidUntil(in.ValidUntil)
	if err != nil {
		return errResult(apperr.Invalid("valid_until: %v", err)), nil, nil
	}
	at, _ := now("")
	e := resolve.NewEnv(s.ws, g, at)
	cur, how := s.effectiveValidUntil(e, before)
	if cur != "" {
		curT, _ := time.Parse(temporal.TimeLayout, cur)
		if v.Instant(e.Ctx).Before(curT) {
			return errResult(apperr.Refused("renewal moves valid_until earlier than the current horizon %s (%s); renewal only advances it", cur, how)), nil, nil
		}
	}
	o := *before
	o.ValidUntil = v
	prev, _ := projection.Version(before)
	if err := s.write(&o); err != nil {
		return errResult(err), nil, nil
	}
	g.Add(&o)
	out, err := s.availabilityResult(e, &o, true)
	if err != nil {
		return errResult(err), nil, nil
	}
	out["previous_version"] = prev
	return okResult(fmt.Sprintf("Renewed %s until %s", o.ID, o.ValidUntil)), out, nil
}

type supersedeIn struct {
	ID string `json:"id" jsonschema:"the availability whose terms change"`
	availabilityIn
	Reason string `json:"reason,omitempty" jsonschema:"recorded on the old availability's retirement"`
}

func (s *Server) availabilitySupersede(ctx context.Context, req *sdk.CallToolRequest, in supersedeIn) (*sdk.CallToolResult, any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	g, err := s.load()
	if err != nil {
		return errResult(err), nil, nil
	}
	old, err := getAvailability(g, in.ID)
	if err != nil {
		return errResult(err), nil, nil
	}
	if err := model.CheckNotRetired(old); err != nil {
		return errResult(err), nil, nil
	}
	at, err := now("")
	if err != nil {
		return errResult(err), nil, nil
	}
	ts, err := timestamp(in.Timestamp, at)
	if err != nil {
		return errResult(err), nil, nil
	}
	src := s.source(req, in.Source)
	if src.Author == "" {
		return errResult(apperr.Refused("an author is required")), nil, nil
	}
	n := *old
	n.ID, n.Source, n.Timestamp, n.Retired, n.Version, n.Extras = model.MintID(model.TypeAvailability), src, ts, nil, "", nil
	n.Conditional = append([]string(nil), old.Conditional...)
	n.Location = append([]string(nil), old.Location...)
	if old.Window != nil {
		w := *old.Window
		n.Window = &w
	}
	in.Subject = ""
	if err := s.applyAvailability(in.availabilityIn, &n, g, s.ws.Config.Context(at)); err != nil {
		return errResult(err), nil, nil
	}
	if err := model.CheckScope(old.Scope, n.Scope); err != nil {
		return errResult(err), nil, nil
	}
	if err := s.write(&n); err != nil {
		return errResult(err), nil, nil
	}
	g.Add(&n)
	r := model.Retired{Kind: "superseded", Reason: in.Reason, SupersededBy: n.ID, Source: src, Timestamp: ts}
	if err := model.CheckRetirement(g, old, r); err != nil {
		return errResult(err), nil, nil
	}
	retired := *old
	retired.Retired = &r
	if err := s.write(&retired); err != nil {
		return errResult(err), nil, nil
	}
	g.Add(&retired)
	out, err := s.availabilityResult(resolve.NewEnv(s.ws, g, at), &n, true)
	if err != nil {
		return errResult(err), nil, nil
	}
	out["created"] = true
	out["superseded"] = map[string]any{"id": old.ID, "version": retired.Version}
	return okResult(fmt.Sprintf("Created %s; retired %s as superseded by it", n.ID, old.ID)), out, nil
}

func (s *Server) availabilityRetire(ctx context.Context, req *sdk.CallToolRequest, in retireIn) (*sdk.CallToolResult, any, error) {
	return s.retire(req, in, model.TypeAvailability)
}

type availabilityListIn struct {
	Subject     string `json:"subject,omitempty"`
	Conditional string `json:"conditional,omitempty"`
	Scope       string `json:"scope,omitempty"`
	Retired     bool   `json:"retired,omitempty"`
	Now         string `json:"now,omitempty"`
}

func (s *Server) availabilityList(ctx context.Context, req *sdk.CallToolRequest, in availabilityListIn) (*sdk.CallToolResult, any, error) {
	g, err := s.ws.Load()
	if err != nil {
		return errResult(err), nil, nil
	}
	at, err := now(in.Now)
	if err != nil {
		return errResult(err), nil, nil
	}
	e := resolve.NewEnv(s.ws, g, at)
	list := []map[string]any{}
	for _, o := range g.Availabilities() {
		if (o.Retired != nil) != in.Retired || (in.Subject != "" && o.Subject != in.Subject) || (in.Scope != "" && o.Scope != in.Scope) || (in.Conditional != "" && !has(o.Conditional, in.Conditional)) {
			continue
		}
		m, err := model.ToMap(o)
		if err != nil {
			return errResult(err), nil, nil
		}
		t, how := s.effectiveValidUntil(e, o)
		if t != "" {
			m["effective_valid_until"] = t
		}
		m["effective_valid_until_from"] = how
		list = append(list, m)
	}
	summary := strings.TrimSpace(fmt.Sprintf("%d availability", len(list)))
	return okResult(summary), map[string]any{"availability": list, "count": len(list)}, nil
}
