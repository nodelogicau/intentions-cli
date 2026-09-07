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
	"github.com/nodelogicau/intentions-cli/internal/temporal"
)

type generateIn struct {
	Horizon   string    `json:"horizon,omitempty" jsonschema:"ISO 8601 duration from now (default resolver.horizon)"`
	Recurring []string  `json:"recurring,omitempty" jsonschema:"only these recurring intentions"`
	Now       string    `json:"now,omitempty"`
	Source    *sourceIn `json:"source,omitempty"`
}

func (s *Server) generate(ctx context.Context, req *sdk.CallToolRequest, in generateIn) (*sdk.CallToolResult, any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	g, err := s.load()
	if err != nil {
		return errResult(err), nil, nil
	}
	at, err := now(in.Now)
	if err != nil {
		return errResult(err), nil, nil
	}
	h := s.ws.Config.GenerationHorizon()
	if in.Horizon != "" {
		if h, err = temporal.ParseDuration(in.Horizon); err != nil {
			return errResult(apperr.Usage("horizon: %v", err)), nil, nil
		}
	}
	for _, id := range in.Recurring {
		if _, err := getIntention(g, id); err != nil {
			return errResult(err), nil, nil
		}
	}
	e := resolve.NewEnv(s.ws, g, at)
	r := temporal.Interval{Start: at, End: temporal.AddDuration(at, h)}
	created, skipped, err := resolve.Generate(e, resolve.GenerateOptions{Range: r, Only: in.Recurring, Source: s.source(req, in.Source), Timestamp: at})
	if err != nil {
		return errResult(err), nil, nil
	}
	list, err := render.Objects(created)
	if err != nil {
		return errResult(err), nil, nil
	}
	if skipped == nil {
		skipped = []resolve.Skip{}
	}
	out := map[string]any{"created": list, "skipped": skipped, "count": len(created), "range": map[string]any{"start": r.Start.Format(time.RFC3339), "end": r.End.Format(time.RFC3339)}}
	return okResult(fmt.Sprintf("%d created, %d skipped", len(created), len(skipped))), out, nil
}

// generateForRange runs generation over the intention's range before ranking.
func (s *Server) generateForRange(e resolve.Env, in *model.Intention, src model.Source) ([]string, error) {
	generated := []string{}
	if in.Window == nil {
		return generated, nil
	}
	r, reason := temporal.ResolutionRange(*in.Window, e.Ctx, e.Horizon)
	if reason != "" {
		return generated, nil
	}
	created, _, err := resolve.Generate(e, resolve.GenerateOptions{Range: r, Source: src, Timestamp: e.Ctx.Now})
	if err != nil {
		return nil, err
	}
	for _, c := range created {
		generated = append(generated, c.ID)
	}
	return generated, nil
}

type resolveIn struct {
	ID     string    `json:"id" jsonschema:"the intention to resolve"`
	Limit  int       `json:"limit,omitempty" jsonschema:"candidates to return (default 20); candidates_considered keeps the full count"`
	Step   string    `json:"step,omitempty" jsonschema:"candidate grid, e.g. PT30M (default resolver.step)"`
	Scope  string    `json:"scope,omitempty" jsonschema:"the resolver's own scope (default resolver.scope)"`
	Now    string    `json:"now,omitempty"`
	Source *sourceIn `json:"source,omitempty"`
}

func (s *Server) resolveTool(ctx context.Context, req *sdk.CallToolRequest, in resolveIn) (*sdk.CallToolResult, any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	g, err := s.load()
	if err != nil {
		return errResult(err), nil, nil
	}
	target, err := getIntention(g, in.ID)
	if err != nil {
		return errResult(err), nil, nil
	}
	at, err := now(in.Now)
	if err != nil {
		return errResult(err), nil, nil
	}
	e, err := s.env(g, at, in.Step, in.Scope)
	if err != nil {
		return errResult(err), nil, nil
	}
	generated, err := s.generateForRange(e, target, s.source(req, in.Source))
	if err != nil {
		return errResult(err), nil, nil
	}
	limit := in.Limit
	if limit == 0 {
		limit = 20
	}
	res, err := resolve.Resolve(e, target, resolve.Options{Limit: limit})
	if err != nil {
		return errResult(err), nil, nil
	}
	res.Generated = generated
	summary := fmt.Sprintf("%d of %d candidates for %s", len(res.Candidates), res.Considered, target.ID)
	if res.Reason != "" {
		summary = "no candidates: " + res.Reason
	}
	return okResult(summary), render.Resolution(res, target.ID), nil
}

type selectIn struct {
	ID        string    `json:"id" jsonschema:"the intention to place"`
	Candidate int       `json:"candidate,omitempty" jsonschema:"1-based index into the full ranked set: the person's choice"`
	Policy    string    `json:"policy,omitempty" jsonschema:"id of the terminus carrying auto_select that authorises a harness to take the top rank-1 candidate"`
	Replace   bool      `json:"replace,omitempty" jsonschema:"re-resolve a placed intention, clearing its placement"`
	Scope     string    `json:"scope,omitempty"`
	Now       string    `json:"now,omitempty"`
	Source    *sourceIn `json:"source,omitempty"`
}

func (s *Server) selectTool(ctx context.Context, req *sdk.CallToolRequest, in selectIn) (*sdk.CallToolResult, any, error) {
	if (in.Candidate == 0) == (in.Policy == "") {
		return errResult(apperr.Usage("pass exactly one of candidate (a person's choice) or policy (a harness under a policy)")), nil, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	g, err := s.load()
	if err != nil {
		return errResult(err), nil, nil
	}
	target, err := getIntention(g, in.ID)
	if err != nil {
		return errResult(err), nil, nil
	}
	at, err := now(in.Now)
	if err != nil {
		return errResult(err), nil, nil
	}
	e, err := s.env(g, at, "", in.Scope)
	if err != nil {
		return errResult(err), nil, nil
	}
	src := s.source(req, in.Source)
	if _, err := s.generateForRange(e, target, src); err != nil {
		return errResult(err), nil, nil
	}
	sel, res, err := resolve.Select(e, target, resolve.SelectOptions{Candidate: in.Candidate, Policy: in.Policy, Source: src, Timestamp: at, Replace: in.Replace})
	if err != nil {
		return errResult(err), nil, nil
	}
	ids := []string{sel.Intention.ID, sel.Resolution.ID}
	if sel.Commitment != nil {
		ids = append(ids, sel.Commitment.ID)
	}
	flags, err := consistency.Check(e, ids)
	if err != nil {
		return errResult(err), nil, nil
	}
	return okResult(sel.Describe()), render.Selection(sel, res, flags, in.Policy), nil
}

type checkIn struct {
	IDs   []string `json:"ids,omitempty" jsonschema:"restrict to flags whose subject or counterpart is one of these"`
	Scope string   `json:"scope,omitempty"`
	Now   string   `json:"now,omitempty"`
}

func (s *Server) checkTool(ctx context.Context, req *sdk.CallToolRequest, in checkIn) (*sdk.CallToolResult, any, error) {
	g, err := s.load()
	if err != nil {
		return errResult(err), nil, nil
	}
	for _, id := range in.IDs {
		if _, err := getObject(g, id); err != nil {
			return errResult(err), nil, nil
		}
	}
	at, err := now(in.Now)
	if err != nil {
		return errResult(err), nil, nil
	}
	e, err := s.env(g, at, "", in.Scope)
	if err != nil {
		return errResult(err), nil, nil
	}
	flags, err := consistency.Check(e, in.IDs)
	if err != nil {
		return errResult(err), nil, nil
	}
	counts := map[string]int{}
	for _, f := range flags {
		counts[f.Kind]++
	}
	return okResult(fmt.Sprintf("%d flags", len(flags))), map[string]any{"flags": flags, "count": len(flags), "counts": counts}, nil
}

type acknowledgeIn struct {
	ID          string    `json:"id" jsonschema:"the intention or commitment the flag was reported on"`
	Kind        string    `json:"kind" jsonschema:"window-clash | condition-mismatch | location-mismatch | expired-ground | intention-inconsistency | party-declined | cycle"`
	Counterpart string    `json:"counterpart,omitempty" jsonschema:"the other object the flag named"`
	Reason      string    `json:"reason,omitempty" jsonschema:"why the person is proceeding anyway"`
	Now         string    `json:"now,omitempty"`
	Source      *sourceIn `json:"source,omitempty"`
}

func (s *Server) acknowledgeTool(ctx context.Context, req *sdk.CallToolRequest, in acknowledgeIn) (*sdk.CallToolResult, any, error) {
	if !consistency.ValidKind(in.Kind) {
		return errResult(apperr.Usage("kind must be one of %s", strings.Join(consistency.Kinds, ", "))), nil, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	g, err := s.load()
	if err != nil {
		return errResult(err), nil, nil
	}
	obj, err := getObject(g, in.ID)
	if err != nil {
		return errResult(err), nil, nil
	}
	at, err := now(in.Now)
	if err != nil {
		return errResult(err), nil, nil
	}
	e := resolve.NewEnv(s.ws, g, at)
	ack := model.Acknowledgement{Kind: in.Kind, Counterpart: in.Counterpart, Reason: in.Reason, Source: s.source(req, in.Source), Timestamp: at}
	if in.Counterpart != "" {
		cp, err := getObject(g, in.Counterpart)
		if err != nil {
			return errResult(err), nil, nil
		}
		ack.CounterpartVersion, _ = projection.Version(cp)
	} else if in.Kind != consistency.Cycle {
		flags, _ := consistency.Check(e, []string{obj.GetID()})
		bare := false
		for _, f := range flags {
			if f.Subject == obj.GetID() && f.Kind == in.Kind && f.Counterpart == "" {
				bare = true
			}
		}
		if !bare {
			return errResult(apperr.Usage("kind %s names a counterpart; pass counterpart", in.Kind)), nil, nil
		}
	}
	var updated model.Object
	switch o := obj.(type) {
	case *model.Intention:
		c := *o
		c.Acknowledgements = append(append([]model.Acknowledgement(nil), o.Acknowledgements...), ack)
		updated = &c
	case *model.Commitment:
		c := *o
		c.Acknowledgements = append(append([]model.Acknowledgement(nil), o.Acknowledgements...), ack)
		updated = &c
	default:
		return errResult(apperr.Usage("%s is a %s; only intentions and commitments carry acknowledgements", obj.GetID(), obj.GetType())), nil, nil
	}
	if err := s.ws.WriteObject(updated); err != nil {
		return errResult(err), nil, nil
	}
	g.Add(updated)
	out, err := render.Object(updated)
	if err != nil {
		return errResult(err), nil, nil
	}
	out["acknowledgement"] = render.Acknowledgement(ack)
	out["flags"] = s.flagsOn(e, updated.GetID())
	return okResult(fmt.Sprintf("Acknowledged %s on %s", in.Kind, updated.GetID())), out, nil
}

type boundsIn struct {
	ID         string `json:"id,omitempty" jsonschema:"an intention or availability whose window to bound"`
	Calendar   string `json:"calendar,omitempty" jsonschema:"an ad hoc calendar anchor (EDTF or deictic)"`
	Clock      string `json:"clock,omitempty"`
	Timezone   string `json:"timezone,omitempty" jsonschema:"override resolver.timezone"`
	Hemisphere string `json:"hemisphere,omitempty"`
	Now        string `json:"now,omitempty"`
}

func (s *Server) boundsTool(ctx context.Context, req *sdk.CallToolRequest, in boundsIn) (*sdk.CallToolResult, any, error) {
	at, err := now(in.Now)
	if err != nil {
		return errResult(err), nil, nil
	}
	tctx := s.ws.Config.Context(at)
	if in.Timezone != "" {
		loc, err := time.LoadLocation(in.Timezone)
		if err != nil {
			return errResult(apperr.Usage("timezone %q is not a known IANA zone", in.Timezone)), nil, nil
		}
		tctx.Location = loc
	}
	if in.Hemisphere != "" {
		h, err := temporal.ParseHemisphere(in.Hemisphere)
		if err != nil {
			return errResult(apperr.Usage("hemisphere: %v", err)), nil, nil
		}
		tctx.Hemisphere = h
	}
	var win temporal.Window
	if in.ID != "" {
		obj, _, err := s.ws.ReadObject(in.ID)
		if err != nil {
			return errResult(err), nil, nil
		}
		switch o := obj.(type) {
		case *model.Intention:
			if o.Window != nil {
				win = *o.Window
			}
		case *model.Availability:
			if o.Window != nil {
				win = *o.Window
			}
		default:
			return errResult(apperr.Usage("%s is a %s and has no window", in.ID, obj.GetType())), nil, nil
		}
	}
	if in.Calendar != "" {
		c, err := temporal.ParseCalendarOrDeixis(in.Calendar, tctx)
		if err != nil {
			return errResult(apperr.Invalid("calendar: %v", err)), nil, nil
		}
		win.Calendar = &c
	}
	if in.Clock != "" {
		c, err := temporal.ParseClock(in.Clock)
		if err != nil {
			return errResult(apperr.Invalid("clock: %v", err)), nil, nil
		}
		win.Clock = &c
	}
	if win.IsZero() {
		return errResult(apperr.Usage("nothing to bound: pass an id with a window, or calendar and/or clock")), nil, nil
	}
	ivs, err := temporal.Bounds(win, tctx)
	if err != nil {
		return errResult(apperr.Usage("%v", err)), nil, nil
	}
	list := make([]map[string]any, 0, len(ivs))
	for _, iv := range ivs {
		m := map[string]any{"start": nil, "end": nil}
		if !iv.OpenStart {
			m["start"] = iv.Start.Format(time.RFC3339)
		}
		if !iv.OpenEnd {
			m["end"] = iv.End.Format(time.RFC3339)
		}
		list = append(list, m)
	}
	out := map[string]any{"window": win.String(), "timezone": tctx.Location.String(), "hemisphere": string(tctx.Hemisphere), "intervals": list, "count": len(list)}
	if in.ID != "" {
		out["id"] = in.ID
	}
	return okResult(fmt.Sprintf("%d intervals for %s in %s", len(list), win, tctx.Location)), out, nil
}

type unresolvedIn struct {
	Subject string `json:"subject,omitempty" jsonschema:"only intentions of this subject"`
	Scope   string `json:"scope,omitempty"`
	Now     string `json:"now,omitempty"`
}

func (s *Server) unresolvedTool(ctx context.Context, req *sdk.CallToolRequest, in unresolvedIn) (*sdk.CallToolResult, any, error) {
	g, err := s.load()
	if err != nil {
		return errResult(err), nil, nil
	}
	at, err := now(in.Now)
	if err != nil {
		return errResult(err), nil, nil
	}
	e, err := s.env(g, at, "", in.Scope)
	if err != nil {
		return errResult(err), nil, nil
	}
	entries, err := resolve.Unresolved(e, resolve.UnresolvedOptions{Subject: in.Subject})
	if err != nil {
		return errResult(err), nil, nil
	}
	out := resolve.UnresolvedResult(entries)
	return okResult(fmt.Sprintf("%d unresolved: %v", len(entries), out["counts"])), out, nil
}
