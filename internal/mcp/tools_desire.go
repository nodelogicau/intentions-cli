package mcp

import (
	"context"
	"fmt"
	"strings"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/nodelogicau/intentions-cli/internal/apperr"
	"github.com/nodelogicau/intentions-cli/internal/model"
	"github.com/nodelogicau/intentions-cli/internal/projection"
	"github.com/nodelogicau/intentions-cli/internal/render"
	"github.com/nodelogicau/intentions-cli/internal/store"
	"github.com/nodelogicau/intentions-cli/internal/temporal"
)

func (s *Server) registerDesireTools() {
	sdk.AddTool(s.srv, &sdk.Tool{Name: "desire_add", Annotations: additive, InputSchema: requiring[desireIn]("title"),
		Description: "Record a want the person expressed and has not committed to: the inbox, the rung below intention. No why and no when are required; a passing remark (\"call the accountant\") goes here, never into intention_add. Record only what the person said they wanted, never what you inferred. List first (desire_list) so you edit an existing want rather than write a second. Results equal `intentions desire add --json`."},
		s.desireAdd)
	sdk.AddTool(s.srv, &sdk.Tool{Name: "desire_edit", Annotations: additive,
		Description: "Edit a desire in place; `clear` removes optional fields. Only a change of terminus (serves) moves the version. Refuses a retired desire. Results equal `intentions desire edit --json`."},
		s.desireEdit)
	sdk.AddTool(s.srv, &sdk.Tool{Name: "desire_adopt", Annotations: additive,
		Description: "Turn a want into a plan: write a tentative intention carrying the desire's title, description, serves, activity and location plus the duration and window you supply, then retire the desire as adopted naming it. Adopt when the want has a why (it serves a terminus) or a when (duration or window); a bare adoption is refused, because it would write a terminus, a self titled as a task. The result carries both objects and the intention's findings; an unserved warning there is the moment to ask the person to firm their terminus. Results equal `intentions desire adopt --json`."},
		s.desireAdopt)
	sdk.AddTool(s.srv, &sdk.Tool{Name: "desire_retire", Annotations: additive,
		Description: "Append a retirement record to a desire: abandoned (the person let it go) or superseded (with superseded_by naming another desire). Never adopted: only desire_adopt writes that, with the intention it names. Results equal `intentions desire retire --json`."},
		s.desireRetire)
	sdk.AddTool(s.srv, &sdk.Tool{Name: "desire_show", Annotations: readOnly,
		Description: "One desire with its computed version and the terminus it serves. Results equal `intentions desire show --json`."},
		s.desireShow)
	sdk.AddTool(s.srv, &sdk.Tool{Name: "desire_list", Annotations: readOnly,
		Description: "The inbox: every desire (active by default), filtered by subject, activity, or retired. Read it each session. Results equal `intentions desire list --json`."},
		s.desireList)
}

type desireIn struct {
	Title       string     `json:"title,omitempty" jsonschema:"what the person said they wanted (required on add)"`
	Description string     `json:"description,omitempty" jsonschema:"prose"`
	Subject     string     `json:"subject,omitempty" jsonschema:"URI of the particular whose desire this is (add only; default: the workspace's defaults.subject)"`
	Activity    string     `json:"activity,omitempty" jsonschema:"activity term, a hint carried onto the intention on adoption"`
	Location    []string   `json:"location,omitempty" jsonschema:"URIs, hints carried onto the intention on adoption"`
	Serves      []servesIn `json:"serves,omitempty" jsonschema:"for-the-sake-of only, each naming a terminus of the subject, which may be a draft"`
	Reference   string     `json:"reference,omitempty" jsonschema:"informal pointer to a DKF claim"`
	Timestamp   string     `json:"timestamp,omitempty"`
	Source      *sourceIn  `json:"source,omitempty"`
}

type desireEditIn struct {
	ID string `json:"id" jsonschema:"the desire to edit"`
	desireIn
	Clear []string `json:"clear,omitempty" jsonschema:"optional fields to remove: description, activity, location, serves, reference"`
}

type adoptIn struct {
	ID        string    `json:"id" jsonschema:"the desire to adopt"`
	Duration  any       `json:"duration,omitempty" jsonschema:"ISO 8601 duration such as PT30M, or {nominal, min, max}"`
	Window    *windowIn `json:"window,omitempty" jsonschema:"the bounds as the person expressed them: calendar, clock, relative"`
	Timestamp string    `json:"timestamp,omitempty"`
	Source    *sourceIn `json:"source,omitempty"`
}

type desireShowIn struct {
	ID string `json:"id" jsonschema:"the desire to show"`
}

type desireListIn struct {
	Subject  string `json:"subject,omitempty"`
	Activity string `json:"activity,omitempty"`
	Retired  bool   `json:"retired,omitempty" jsonschema:"list retired desires instead of active"`
}

func getDesire(g *store.Graph, id string) (*model.Desire, error) {
	obj, err := getObject(g, id)
	if err != nil {
		return nil, err
	}
	d, ok := obj.(*model.Desire)
	if !ok {
		return nil, apperr.Usage("%s is a %s, not a desire", id, obj.GetType())
	}
	return d, nil
}

func applyDesire(in desireIn, clear []string, o *model.Desire) error {
	cleared := func(n string) bool { return has(clear, n) }
	if in.Title != "" {
		o.Title = in.Title
	}
	if in.Description != "" {
		o.Description = in.Description
	}
	if cleared("description") {
		o.Description = ""
	}
	if in.Activity != "" {
		o.Activity = in.Activity
	}
	if cleared("activity") {
		o.Activity = ""
	}
	if in.Location != nil {
		o.Location = model.SortStrings(in.Location)
	}
	if cleared("location") {
		o.Location = nil
	}
	if in.Serves != nil {
		refs := make([]model.Ref, 0, len(in.Serves))
		for _, r := range in.Serves {
			refs = append(refs, model.Ref{ID: Clean(r.ID), Role: Clean(r.Role)})
		}
		o.Serves = refs
	}
	if cleared("serves") {
		o.Serves = []model.Ref{}
	}
	if in.Reference != "" {
		o.Reference = in.Reference
	}
	if cleared("reference") {
		o.Reference = ""
	}
	return nil
}

func (s *Server) desireAdd(ctx context.Context, req *sdk.CallToolRequest, in desireIn) (*sdk.CallToolResult, any, error) {
	if strings.TrimSpace(in.Title) == "" {
		return errResult(apperr.Usage("title is required")), nil, nil
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
	o := &model.Desire{ID: model.MintID(model.TypeDesire), Source: src, Timestamp: ts, Serves: []model.Ref{}}
	o.Subject = Clean(in.Subject)
	if o.Subject == "" {
		o.Subject = s.ws.Config.Defaults.Subject
	}
	if o.Subject == "" {
		return errResult(apperr.Refused("subject is required: pass subject, or set defaults.subject in intentions.yaml; this workspace has no default")), nil, nil
	}
	if err := applyDesire(in, nil, o); err != nil {
		return errResult(err), nil, nil
	}
	if ps := model.Check(o); len(ps) > 0 {
		return errResult(apperr.Invalid("%s", ps.Error())), nil, nil
	}
	if err := model.CheckDesireServes(g, o); err != nil {
		return errResult(err), nil, nil
	}
	if err := s.write(o); err != nil {
		return errResult(err), nil, nil
	}
	out, err := render.Object(o)
	if err != nil {
		return errResult(err), nil, nil
	}
	out["created"] = true
	return okResult(fmt.Sprintf("Created %s (%s)", o.ID, o.Version)), out, nil
}

func (s *Server) desireEdit(ctx context.Context, req *sdk.CallToolRequest, in desireEditIn) (*sdk.CallToolResult, any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	g, err := s.load()
	if err != nil {
		return errResult(err), nil, nil
	}
	before, err := getDesire(g, in.ID)
	if err != nil {
		return errResult(err), nil, nil
	}
	if err := model.CheckNotRetired(before); err != nil {
		return errResult(err), nil, nil
	}
	o := *before
	o.Serves = append([]model.Ref{}, before.Serves...)
	if err := applyDesire(in.desireIn, in.Clear, &o); err != nil {
		return errResult(err), nil, nil
	}
	if ps := model.Check(&o); len(ps) > 0 {
		return errResult(apperr.Invalid("%s", ps.Error())), nil, nil
	}
	if err := model.CheckDesireServes(g, &o); err != nil {
		return errResult(err), nil, nil
	}
	prev, _ := projection.Version(before)
	if err := s.write(&o); err != nil {
		return errResult(err), nil, nil
	}
	out, err := render.Object(&o)
	if err != nil {
		return errResult(err), nil, nil
	}
	out["previous_version"] = prev
	out["projection_changed"] = prev != o.Version
	return okResult(fmt.Sprintf("Edited %s (%s -> %s)", o.ID, prev, o.Version)), out, nil
}

func (s *Server) desireAdopt(ctx context.Context, req *sdk.CallToolRequest, in adoptIn) (*sdk.CallToolResult, any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	g, err := s.load()
	if err != nil {
		return errResult(err), nil, nil
	}
	d, err := getDesire(g, in.ID)
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
	var dur *temporal.DurationSpec
	if in.Duration != nil {
		if dur, err = durationIn(in.Duration); err != nil {
			return errResult(err), nil, nil
		}
	}
	win, err := window(in.Window, nil, s.ws.Config.Context(at), g)
	if err != nil {
		return errResult(err), nil, nil
	}
	o, r, err := model.Adopt(d, dur, win, src, ts)
	if err != nil {
		return errResult(err), nil, nil
	}
	if ps := model.Check(o); len(ps) > 0 {
		return errResult(apperr.Invalid("%s", ps.Error())), nil, nil
	}
	if err := checkWrite(g, nil, o, src, ""); err != nil {
		return errResult(err), nil, nil
	}
	// The intention first: it is the record of the adoption.
	if err := s.write(o); err != nil {
		return errResult(err), nil, nil
	}
	g.Add(o)
	retired := *d
	retired.Retired = &r
	if err := s.write(&retired); err != nil {
		return errResult(err), nil, nil
	}
	inOut, err := render.Object(o)
	if err != nil {
		return errResult(err), nil, nil
	}
	dOut, err := render.Object(&retired)
	if err != nil {
		return errResult(err), nil, nil
	}
	out := map[string]any{"intention": inOut, "desire": dOut, "id": o.ID}
	fs := attachFindings(out, g, o)
	return okResult(fmt.Sprintf("Adopted %s as %s (%s)", d.ID, o.ID, o.Version) + findingsText(fs)), out, nil
}

func (s *Server) desireRetire(ctx context.Context, req *sdk.CallToolRequest, in retireIn) (*sdk.CallToolResult, any, error) {
	return s.retire(req, in, model.TypeDesire)
}

func (s *Server) desireShow(ctx context.Context, req *sdk.CallToolRequest, in desireShowIn) (*sdk.CallToolResult, any, error) {
	g, err := s.ws.Load()
	if err != nil {
		return errResult(err), nil, nil
	}
	o, err := getDesire(g, in.ID)
	if err != nil {
		return errResult(err), nil, nil
	}
	out, err := render.Object(o)
	if err != nil {
		return errResult(err), nil, nil
	}
	resolved := []map[string]any{}
	for _, r := range o.Serves {
		entry := map[string]any{"id": r.ID, "role": r.Role}
		if t, ok := g.Get(r.ID); ok {
			if ti, ok := t.(*model.Intention); ok {
				entry["title"] = ti.Title
				entry["stability"] = ti.Stability
				if ti.Retired != nil {
					entry["retired"] = ti.Retired.Kind
				}
			}
		} else {
			entry["missing"] = true
		}
		resolved = append(resolved, entry)
	}
	out["serves_resolved"] = resolved
	if ld := g.Loaded[o.ID]; ld != nil && len(ld.Problems) > 0 {
		out["problems"] = ld.Problems
	}
	return okResult(fmt.Sprintf("%s %s", o.ID, o.Title)), out, nil
}

func (s *Server) desireList(ctx context.Context, req *sdk.CallToolRequest, in desireListIn) (*sdk.CallToolResult, any, error) {
	g, err := s.ws.Load()
	if err != nil {
		return errResult(err), nil, nil
	}
	var items []*model.Desire
	for _, o := range g.Desires() {
		if (o.Retired != nil) != in.Retired || (in.Subject != "" && o.Subject != in.Subject) || (in.Activity != "" && o.Activity != in.Activity) {
			continue
		}
		items = append(items, o)
	}
	list, err := render.Objects(items)
	if err != nil {
		return errResult(err), nil, nil
	}
	return okResult(fmt.Sprintf("%d desires", len(list))), map[string]any{"desires": list, "count": len(list)}, nil
}
