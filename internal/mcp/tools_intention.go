package mcp

import (
	"context"
	"fmt"
	"strings"

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

func (s *Server) registerTools() {
	sdk.AddTool(s.srv, &sdk.Tool{Name: "intention_add", Annotations: additive,
		Description: "Record what a person means to do: a duration and a window, never a slot. Always tentative; a harness may draft but may not make it firm here (see intention_firm). List first (intention_list) so you edit an existing intention rather than write a second one for the same thing. Writes one YAML file for a person to review; results equal `intentions intention add --json`."},
		s.intentionAdd)
	sdk.AddTool(s.srv, &sdk.Tool{Name: "intention_edit", Annotations: additive,
		Description: "Edit an intention in place: fill the plan in as it becomes definite. Each field given replaces its own; each window anchor given replaces its own; `clear` removes optional fields. Prose edits leave the version unchanged. Refuses a retired intention, a subject change, a cycle in serves, and firm by a harness without policy. Results equal `intentions intention edit --json`."},
		s.intentionEdit)
	sdk.AddTool(s.srv, &sdk.Tool{Name: "intention_firm", Annotations: additive,
		Description: "Set stability to firm. A person's act needs nothing; a harness (you, when this session identifies one) must pass `policy`, a terminus of the subject carrying auto_firm whose terms the intention satisfies, and the file then records firmed_under. Refused otherwise: never firm because the person sounds sure."},
		s.intentionFirm)
	sdk.AddTool(s.srv, &sdk.Tool{Name: "intention_retire", Annotations: additive,
		Description: "Append a retirement record: fulfilled, abandoned, or superseded (with superseded_by). Nothing is deleted; the file stays and every reference to it still resolves. A retired intention refuses further edits."},
		s.intentionRetire)
	sdk.AddTool(s.srv, &sdk.Tool{Name: "intention_show", Annotations: readOnly,
		Description: "One intention with its computed version, resolved serves targets, and current consistency flags. Results equal `intentions intention show --json`."},
		s.intentionShow)
	sdk.AddTool(s.srv, &sdk.Tool{Name: "intention_list", Annotations: readOnly,
		Description: "Every intention (active by default), filtered by subject, activity, stability, recurring, placed, unplaced, instances_of, or retired. Call this before intention_add."},
		s.intentionList)
	sdk.AddTool(s.srv, &sdk.Tool{Name: "availability_add", Annotations: additive,
		Description: "Record capacity: a standing statement that a particular (a person, a room, anything with a URI) has a duration of capacity within a window, optionally for certain activities (conditional), at certain places, recurring by cadence. Capacity is a fact about the person: record what they told you, never what an empty calendar suggests and never to make a resolution succeed. Only for the people this workspace tracks, not for external parties."},
		s.availabilityAdd)
	sdk.AddTool(s.srv, &sdk.Tool{Name: "availability_renew", Annotations: additive,
		Description: "Advance valid_until on the same availability, when the person reconfirms it. Never moves it earlier."},
		s.availabilityRenew)
	sdk.AddTool(s.srv, &sdk.Tool{Name: "availability_supersede", Annotations: additive,
		Description: "Change an availability's terms (window, clock, duration, conditional, location) by creating a new one from it and retiring the old as superseded pointing at the new. The only way terms change; every reference to the old id stays valid."},
		s.availabilitySupersede)
	sdk.AddTool(s.srv, &sdk.Tool{Name: "availability_retire", Annotations: additive,
		Description: "Append a retirement record to an availability: retracted, or superseded with superseded_by. Nothing is deleted."},
		s.availabilityRetire)
	sdk.AddTool(s.srv, &sdk.Tool{Name: "availability_list", Annotations: readOnly,
		Description: "Every availability (active by default), filtered by subject, conditional, scope, or retired, each with its effective validity horizon."},
		s.availabilityList)
	s.registerCommitmentTools()
	sdk.AddTool(s.srv, &sdk.Tool{Name: "generate", Annotations: additive,
		Description: "Materialise instances of recurring intentions over a horizon (default resolver.horizon from now). Idempotent on (recurring, occurrence); a retired instance is never regenerated. Use it when the person asks to plan a named period, narrowed with `horizon` and `recurring`; `resolve` already materialises what its own range needs, so creating a recurring intention should not be followed by a generate."},
		s.generate)
	sdk.AddTool(s.srv, &sdk.Tool{Name: "resolve", Annotations: additive,
		Description: "Rank candidate placements for one intention against the availability of its subject and every party: rank 1 displaces nothing, rank 2 only tentative things, rank 3 something firm or accepted; then by preference, then earliest. Chooses nothing. Writes only the instances it generates over its range. A party the workspace holds no availability for constrains no supply and is returned in `presumed`, whose time the placement assumes without evidence; say so when you put the candidates to the person, then call select with their choice."},
		s.resolveTool)
	sdk.AddTool(s.srv, &sdk.Tool{Name: "select", Annotations: additive,
		Description: "The recorded act of selection. `candidate` is the person's choice (1-based into the full ranked set). `policy` is the only way a harness selects alone: a terminus of the subject carrying auto_select that covers the intention, and only when a rank-1 candidate exists. Writes the RESOLUTION record, the placement, and a commitment with every party tentative when there are parties. Displaced objects are listed and never changed. `replace` re-resolves a placed intention."},
		s.selectTool)
	sdk.AddTool(s.srv, &sdk.Tool{Name: "unresolved", Annotations: readOnly,
		Description: "Every active intention still without a placement and what stands in its way, soonest deadline first: ready (candidate count, best rank), blocked (on a relational target with no placement), no_candidates (the resolver's reason), incomplete (duration or window missing), unresolvable (a serves cycle). A dry resolution; writes nothing. Instances of recurring intentions appear here once something has materialised them. Results equal `intentions unresolved --json`."},
		s.unresolvedTool)
	sdk.AddTool(s.srv, &sdk.Tool{Name: "check", Annotations: readOnly,
		Description: "The consistency check: window-clash, condition-mismatch, location-mismatch, expired-ground, intention-inconsistency, party-declined, cycle. Flags are computed, never stored, never decisions; one already acknowledged against the counterpart's current version is suppressed. Optionally scoped to ids."},
		s.checkTool)
	sdk.AddTool(s.srv, &sdk.Tool{Name: "acknowledge", Annotations: additive,
		Description: "Record that the person has seen a specific flag against a specific state of its counterpart and is proceeding anyway. Only on their word. Lapses when the counterpart's projection changes; never a way to silence a flag for good."},
		s.acknowledgeTool)
	sdk.AddTool(s.srv, &sdk.Tool{Name: "bounds", Annotations: readOnly,
		Description: "What a window means in clock time: the intervals an object's window (or an ad hoc calendar/clock) admits in the resolver's timezone. Writes nothing."},
		s.boundsTool)
	sdk.AddTool(s.srv, &sdk.Tool{Name: "validate", Annotations: readOnly,
		Description: "Check the whole workspace against the format: errors, warnings, info. Results equal `intentions validate --json`; ok is false when any error is found."},
		s.validateTool)
	sdk.AddTool(s.srv, &sdk.Tool{Name: "workspace_status", Annotations: readOnly,
		Description: "The bound workspace: root, default subject, object counts, validate summary, flag count, and, inside a git checkout, workspace files not yet committed (read-only; never runs a git command that writes)."},
		s.workspaceStatus)
}

// --- intention inputs -------------------------------------------------------

type servesIn struct {
	ID   string `json:"id" jsonschema:"the target intention"`
	Role string `json:"role" jsonschema:"in-order-to | for-the-sake-of | instance-of"`
}

type intentionIn struct {
	Title       string     `json:"title,omitempty" jsonschema:"prose title (required on add)"`
	Description string     `json:"description,omitempty" jsonschema:"prose"`
	Subject     string     `json:"subject,omitempty" jsonschema:"URI of the particular whose intention this is (add only; default: the workspace's defaults.subject)"`
	Duration    any        `json:"duration,omitempty" jsonschema:"ISO 8601 duration such as PT90M, or {nominal, min, max}"`
	Window      *windowIn  `json:"window,omitempty" jsonschema:"the bounds as the person expressed them: calendar, clock, relative; never a computed start"`
	Stability   string     `json:"stability,omitempty" jsonschema:"tentative (default) or firm; firm by a harness needs policy"`
	Activity    string     `json:"activity,omitempty" jsonschema:"lowercase kebab-case term matched against availability conditional; reuse terms already in the workspace"`
	Location    []string   `json:"location,omitempty" jsonschema:"URIs at one of which this must happen"`
	Parties     []string   `json:"parties,omitempty" jsonschema:"URIs of other particulars whose availability must be satisfied"`
	Serves      []servesIn `json:"serves,omitempty" jsonschema:"outbound references: in-order-to a means to an end, for-the-sake-of a terminus"`
	Cadence     string     `json:"cadence,omitempty" jsonschema:"RRULE with date-level parts only; makes this recurring and needs a calendar anchor"`
	Preference  string     `json:"preference,omitempty" jsonschema:"earliest | latest | adjacent | spread"`
	AutoSelect  string     `json:"auto_select,omitempty" jsonschema:"policy condition on a terminus: max_duration=PT30M,stability=tentative"`
	AutoFirm    string     `json:"auto_firm,omitempty" jsonschema:"policy condition on a terminus: max_duration=PT30M"`
	Reference   string     `json:"reference,omitempty" jsonschema:"informal pointer to a DKF claim; never validated"`
	Policy      string     `json:"policy,omitempty" jsonschema:"id of the terminus carrying auto_firm that authorises a harness to set firm"`
	Timestamp   string     `json:"timestamp,omitempty" jsonschema:"RFC 3339 assertion time; default now"`
	Source      *sourceIn  `json:"source,omitempty"`
}

type intentionEditIn struct {
	ID string `json:"id" jsonschema:"the intention to edit"`
	intentionIn
	Clear []string `json:"clear,omitempty" jsonschema:"optional fields to remove: description, duration, window, calendar, clock, relative, activity, location, parties, serves, cadence, preference, auto_select, auto_firm, reference"`
}

// apply merges an input into an intention. Fields present replace their own.
func (s *Server) apply(in intentionIn, clear []string, o *model.Intention, g *store.Graph, ctx temporal.Context) error {
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
	if in.Duration != nil {
		d, err := durationIn(in.Duration)
		if err != nil {
			return err
		}
		o.Duration = d
	}
	if cleared("duration") {
		o.Duration = nil
	}
	if cleared("window") {
		o.Window = nil
	} else if o.Window != nil {
		w := *o.Window
		if cleared("calendar") {
			w.Calendar = nil
		}
		if cleared("clock") {
			w.Clock = nil
		}
		if cleared("relative") {
			w.Relative = nil
		}
		if w.IsZero() {
			o.Window = nil
		} else {
			o.Window = &w
		}
	}
	w, err := window(in.Window, o.Window, ctx, g)
	if err != nil {
		return err
	}
	o.Window = w
	if in.Stability != "" {
		o.Stability = in.Stability
		if o.Stability != "firm" {
			o.FirmedUnder = ""
		}
	}
	if in.Activity != "" {
		o.Activity = in.Activity
	}
	if cleared("activity") {
		o.Activity = ""
	}
	if in.Location != nil {
		if err := checkURIs("location", in.Location); err != nil {
			return err
		}
		o.Location = model.SortStrings(in.Location)
	}
	if cleared("location") {
		o.Location = nil
	}
	if in.Parties != nil {
		if err := checkURIs("parties", in.Parties); err != nil {
			return err
		}
		o.Parties = model.SortStrings(in.Parties)
	}
	if cleared("parties") {
		o.Parties = nil
	}
	if in.Serves != nil {
		refs := make([]model.Ref, 0, len(in.Serves))
		for _, r := range in.Serves {
			refs = append(refs, model.Ref{ID: r.ID, Role: r.Role})
		}
		o.Serves = model.SortRefs(refs)
	}
	if cleared("serves") || o.Serves == nil {
		o.Serves = []model.Ref{}
	}
	if in.Cadence != "" {
		c, err := cadenceIn(in.Cadence)
		if err != nil {
			return err
		}
		o.Cadence = c
	}
	if cleared("cadence") {
		o.Cadence = nil
	}
	if in.Preference != "" {
		o.Preference = in.Preference
	}
	if cleared("preference") {
		o.Preference = ""
	}
	if in.AutoSelect != "" {
		p, err := policyIn("auto_select", in.AutoSelect)
		if err != nil {
			return err
		}
		o.AutoSelect = p
	}
	if cleared("auto_select") {
		o.AutoSelect = nil
	}
	if in.AutoFirm != "" {
		p, err := policyIn("auto_firm", in.AutoFirm)
		if err != nil {
			return err
		}
		o.AutoFirm = p
	}
	if cleared("auto_firm") {
		o.AutoFirm = nil
	}
	if in.Reference != "" {
		o.Reference = in.Reference
	}
	if cleared("reference") {
		o.Reference = ""
	}
	return nil
}

// checkWrite applies the workspace-level write policy and settles firmed_under.
func checkWrite(g *store.Graph, before, o *model.Intention, act model.Source, policy string) error {
	if before != nil {
		if err := model.CheckNotRetired(before); err != nil {
			return err
		}
		if err := model.CheckSubjectUnchanged(before.Subject, o.Subject); err != nil {
			return err
		}
	}
	if err := model.CheckServes(g, o.ID, o.Serves); err != nil {
		return err
	}
	if o.Stability == "firm" && (before == nil || before.Stability != "firm") {
		target := o
		if before != nil {
			target = before
		}
		fu, err := model.CheckFirm(g, target, act, policy)
		if err != nil {
			return err
		}
		o.FirmedUnder = fu
	}
	if o.Stability != "firm" {
		o.FirmedUnder = ""
	}
	return nil
}

func (s *Server) flagsOn(e resolve.Env, id string) []consistency.Flag {
	all, err := consistency.Check(e, []string{id})
	if err != nil {
		return []consistency.Flag{}
	}
	out := []consistency.Flag{}
	for _, f := range all {
		if f.Subject == id {
			out = append(out, f)
		}
	}
	return out
}

func (s *Server) intentionAdd(ctx context.Context, req *sdk.CallToolRequest, in intentionIn) (*sdk.CallToolResult, any, error) {
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
		return errResult(apperr.Refused("an author is required: pass source.author, or set defaults.source.author in intentions.yaml; a speech act with no speaker is not one")), nil, nil
	}
	o := &model.Intention{ID: model.MintID(model.TypeIntention), Stability: "tentative", Source: src, Timestamp: ts, Acknowledgements: []model.Acknowledgement{}, Serves: []model.Ref{}}
	o.Subject = Clean(in.Subject)
	if o.Subject == "" {
		o.Subject = s.ws.Config.Defaults.Subject
	}
	if o.Subject == "" {
		return errResult(apperr.Refused("subject is required: pass subject, or set defaults.subject in intentions.yaml; this workspace has no default")), nil, nil
	}
	if err := s.apply(in, nil, o, g, s.ws.Config.Context(at)); err != nil {
		return errResult(err), nil, nil
	}
	if ps := model.Check(o); len(ps) > 0 {
		return errResult(apperr.Invalid("%s", ps.Error())), nil, nil
	}
	if err := checkWrite(g, nil, o, src, in.Policy); err != nil {
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
	if in.Policy != "" && o.Stability == "firm" {
		out["policy"] = in.Policy
	}
	return okResult(fmt.Sprintf("Created %s (%s)", o.ID, o.Version)), out, nil
}

func (s *Server) intentionEdit(ctx context.Context, req *sdk.CallToolRequest, in intentionEditIn) (*sdk.CallToolResult, any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	g, err := s.load()
	if err != nil {
		return errResult(err), nil, nil
	}
	before, err := getIntention(g, in.ID)
	if err != nil {
		return errResult(err), nil, nil
	}
	at, err := now("")
	if err != nil {
		return errResult(err), nil, nil
	}
	src := s.source(req, in.Source)
	o := *before
	o.Serves = append([]model.Ref(nil), before.Serves...)
	if err := s.apply(in.intentionIn, in.Clear, &o, g, s.ws.Config.Context(at)); err != nil {
		return errResult(err), nil, nil
	}
	if ps := model.Check(&o); len(ps) > 0 {
		return errResult(apperr.Invalid("%s", ps.Error())), nil, nil
	}
	if err := checkWrite(g, before, &o, src, in.Policy); err != nil {
		return errResult(err), nil, nil
	}
	prev, _ := projection.Version(before)
	if err := s.write(&o); err != nil {
		return errResult(err), nil, nil
	}
	g.Add(&o)
	out, err := render.Object(&o)
	if err != nil {
		return errResult(err), nil, nil
	}
	out["previous_version"] = prev
	out["projection_changed"] = prev != o.Version
	if in.Policy != "" && o.Stability == "firm" {
		out["policy"] = in.Policy
	}
	out["flags"] = s.flagsOn(resolve.NewEnv(s.ws, g, at), o.ID)
	return okResult(fmt.Sprintf("Edited %s (%s -> %s)", o.ID, prev, o.Version)), out, nil
}

type firmIn struct {
	ID     string    `json:"id" jsonschema:"the intention to firm"`
	Policy string    `json:"policy,omitempty" jsonschema:"id of the terminus carrying auto_firm; required when the source carries a harness"`
	Source *sourceIn `json:"source,omitempty"`
}

func (s *Server) intentionFirm(ctx context.Context, req *sdk.CallToolRequest, in firmIn) (*sdk.CallToolResult, any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	g, err := s.load()
	if err != nil {
		return errResult(err), nil, nil
	}
	before, err := getIntention(g, in.ID)
	if err != nil {
		return errResult(err), nil, nil
	}
	if err := model.CheckNotRetired(before); err != nil {
		return errResult(err), nil, nil
	}
	if before.Stability == "firm" {
		return errResult(apperr.Refused("%s is already firm", before.ID)), nil, nil
	}
	src := s.source(req, in.Source)
	fu, err := model.CheckFirm(g, before, src, in.Policy)
	if err != nil {
		return errResult(err), nil, nil
	}
	o := *before
	o.Stability, o.FirmedUnder = "firm", fu
	prev, _ := projection.Version(before)
	if err := s.write(&o); err != nil {
		return errResult(err), nil, nil
	}
	out, err := render.Object(&o)
	if err != nil {
		return errResult(err), nil, nil
	}
	out["previous_version"] = prev
	out["source"] = src
	if in.Policy != "" {
		out["policy"] = in.Policy
	}
	return okResult(fmt.Sprintf("Firmed %s (%s -> %s)", o.ID, prev, o.Version)), out, nil
}

type retireIn struct {
	ID           string    `json:"id"`
	Kind         string    `json:"kind" jsonschema:"intentions: fulfilled | abandoned | superseded; availability: retracted | superseded"`
	SupersededBy string    `json:"superseded_by,omitempty" jsonschema:"the replacing object; required with kind superseded"`
	Reason       string    `json:"reason,omitempty"`
	Timestamp    string    `json:"timestamp,omitempty"`
	Source       *sourceIn `json:"source,omitempty"`
}

func (s *Server) retire(req *sdk.CallToolRequest, in retireIn, want model.Type) (*sdk.CallToolResult, any, error) {
	if in.Kind == "" {
		return errResult(apperr.Usage("kind is required: %s", strings.Join(model.RetirementKinds(want), ", "))), nil, nil
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
	if obj.GetType() != want {
		return errResult(apperr.Usage("%s is a %s, not a %s", in.ID, obj.GetType(), want)), nil, nil
	}
	at, err := now("")
	if err != nil {
		return errResult(err), nil, nil
	}
	ts, err := timestamp(in.Timestamp, at)
	if err != nil {
		return errResult(err), nil, nil
	}
	r := model.Retired{Kind: in.Kind, Reason: in.Reason, SupersededBy: in.SupersededBy, Source: s.source(req, in.Source), Timestamp: ts}
	if err := model.CheckRetirement(g, obj, r); err != nil {
		return errResult(err), nil, nil
	}
	prev, _ := projection.Version(obj)
	var updated model.Object
	switch o := obj.(type) {
	case *model.Intention:
		c := *o
		c.Retired = &r
		updated = &c
	case *model.Availability:
		c := *o
		c.Retired = &r
		updated = &c
	}
	if err := s.write(updated); err != nil {
		return errResult(err), nil, nil
	}
	out, err := render.Object(updated)
	if err != nil {
		return errResult(err), nil, nil
	}
	out["previous_version"] = prev
	out["retired"] = map[string]any{"kind": in.Kind, "superseded_by": in.SupersededBy, "reason": in.Reason}
	return okResult(fmt.Sprintf("Retired %s as %s", in.ID, in.Kind)), out, nil
}

func (s *Server) intentionRetire(ctx context.Context, req *sdk.CallToolRequest, in retireIn) (*sdk.CallToolResult, any, error) {
	return s.retire(req, in, model.TypeIntention)
}

type showIn struct {
	ID  string `json:"id"`
	Now string `json:"now,omitempty" jsonschema:"RFC 3339; the instant the check runs at (default now)"`
}

func (s *Server) intentionShow(ctx context.Context, req *sdk.CallToolRequest, in showIn) (*sdk.CallToolResult, any, error) {
	g, err := s.ws.Load()
	if err != nil {
		return errResult(err), nil, nil
	}
	o, err := getIntention(g, in.ID)
	if err != nil {
		return errResult(err), nil, nil
	}
	at, err := now(in.Now)
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
	out["flags"] = s.flagsOn(resolve.NewEnv(s.ws, g, at), o.ID)
	return okResult(fmt.Sprintf("%s %s (%s)", o.ID, o.Title, o.Stability)), out, nil
}

type listIn struct {
	Subject     string `json:"subject,omitempty"`
	Activity    string `json:"activity,omitempty"`
	Stability   string `json:"stability,omitempty"`
	Recurring   bool   `json:"recurring,omitempty"`
	Placed      bool   `json:"placed,omitempty"`
	Unplaced    bool   `json:"unplaced,omitempty"`
	InstancesOf string `json:"instances_of,omitempty"`
	Retired     bool   `json:"retired,omitempty" jsonschema:"list retired intentions instead of active"`
}

func (s *Server) intentionList(ctx context.Context, req *sdk.CallToolRequest, in listIn) (*sdk.CallToolResult, any, error) {
	g, err := s.ws.Load()
	if err != nil {
		return errResult(err), nil, nil
	}
	var items []*model.Intention
	for _, o := range g.Intentions() {
		if (o.Retired != nil) != in.Retired || (in.Subject != "" && o.Subject != in.Subject) || (in.Activity != "" && o.Activity != in.Activity) || (in.Stability != "" && o.Stability != in.Stability) {
			continue
		}
		if (in.Recurring && !o.IsRecurring()) || (in.Placed && o.Placement == nil) || (in.Unplaced && o.Placement != nil) || (in.InstancesOf != "" && o.InstanceOf() != in.InstancesOf) {
			continue
		}
		items = append(items, o)
	}
	list, err := render.Objects(items)
	if err != nil {
		return errResult(err), nil, nil
	}
	return okResult(fmt.Sprintf("%d intentions", len(list))), map[string]any{"intentions": list, "count": len(list)}, nil
}
