// Package mcp is the Model Context Protocol front-end: one stdio server bound
// to one workspace, one tool per CLI operation, results equal to the CLI's
// --json, errors carrying the CLI's codes, and the agent skill delivered as
// the server's instructions.
package mcp

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/nodelogicau/intentions-cli/internal/apperr"
	"github.com/nodelogicau/intentions-cli/internal/model"
	"github.com/nodelogicau/intentions-cli/internal/resolve"
	"github.com/nodelogicau/intentions-cli/internal/store"
	"github.com/nodelogicau/intentions-cli/internal/temporal"
	skill "github.com/nodelogicau/intentions-cli/skills/intentions"
)

// Options configure a server.
type Options struct {
	Workspace *store.Workspace
	Version   string
	// Attribution defaults from server flags (optional).
	Author, Harness, Model string
}

// Server wraps an SDK server bound to a workspace.
type Server struct {
	ws   *store.Workspace
	opts Options
	mu   sync.Mutex // serialises every mutating tool: load, check, write, index
	srv  *sdk.Server
}

// PromptName is the prompt carrying the same text as the instructions.
const PromptName = "intentions-discipline"

// Environment variables the server honours, as the CLI does.
const (
	EnvAuthor  = "INTENTIONS_AUTHOR"
	EnvHarness = "INTENTIONS_HARNESS"
	EnvModel   = "INTENTIONS_MODEL"
	EnvNow     = "INTENTIONS_NOW"
)

// placeholder matches an unsubstituted template such as ${user_config.author},
// which a Desktop client passes literally when the field is blank.
var placeholder = regexp.MustCompile(`^\$\{[^}]*\}$`)

// Clean returns v, or empty when v is blank or an unsubstituted placeholder.
func Clean(v string) string {
	v = strings.TrimSpace(v)
	if placeholder.MatchString(v) {
		return ""
	}
	return v
}

// New builds a server for ws with tools, prompt and instructions registered;
// call Run to serve.
func New(o Options) *Server {
	o.Author, o.Harness, o.Model = Clean(o.Author), Clean(o.Harness), Clean(o.Model)
	s := &Server{ws: o.Workspace, opts: o}
	s.srv = sdk.NewServer(&sdk.Implementation{Name: "intentions", Version: skill.NormaliseVersion(o.Version)}, &sdk.ServerOptions{Instructions: s.instructions()})
	s.registerTools()
	s.registerConventionsResource()
	s.srv.AddPrompt(&sdk.Prompt{Name: PromptName, Description: "The intentions discipline: list before you add, a duration and a window never a slot, draft tentative, resolve then ask, a policy is the only way a harness firms or selects alone."}, func(ctx context.Context, req *sdk.GetPromptRequest) (*sdk.GetPromptResult, error) {
		return &sdk.GetPromptResult{Description: "intentions discipline", Messages: []*sdk.PromptMessage{{Role: "user", Content: &sdk.TextContent{Text: s.instructions()}}}}, nil
	})
	return s
}

// MCP exposes the underlying SDK server.
func (s *Server) MCP() *sdk.Server { return s.srv }

// Run serves one session over t until the client disconnects.
func (s *Server) Run(ctx context.Context, t sdk.Transport) error { return s.srv.Run(ctx, t) }

// Instructions is the text sent at initialize and via the prompt.
func (s *Server) Instructions() string { return s.instructions() }

func (s *Server) instructions() string {
	var b strings.Builder
	fmt.Fprintf(&b, "This intentions server is bound to the workspace at %s", s.ws.Root)
	if subj := s.ws.Config.Defaults.Subject; subj != "" {
		fmt.Fprintf(&b, ", whose default subject is %s", subj)
	}
	b.WriteString(". Everything you write lands as YAML files there for a person to review, typically through a git pull request; nothing is committed for you.\n\n")
	b.WriteString("Tool names are this implementation's (the Intentions Format names none): intention_*, availability_*, commitment_*, generate, resolve, select, unresolved, check, acknowledge, bounds, validate, workspace_status. Every result equals the corresponding CLI verb's --json output.\n")
	b.Write(skill.Body())
	if content, err := os.ReadFile(filepath.Join(s.ws.Root, store.ConventionsFile)); err == nil && len(strings.TrimSpace(string(content))) > 0 {
		b.WriteString("\n\n## Workspace conventions (" + store.ConventionsFile + ")\n\n")
		const maxConventions = 16 * 1024
		if len(content) > maxConventions {
			cut := maxConventions
			for cut < len(content) && !utf8.RuneStart(content[cut]) {
				cut++
			}
			b.Write(content[:cut])
			b.WriteString("\n\n[truncated; read " + store.ConventionsFile + " in the workspace for the rest]")
		} else {
			b.Write(content)
		}
	}
	return b.String()
}

// registerConventionsResource lists intentions.md as a resource when it is
// readable at startup; the content is read at request time.
func (s *Server) registerConventionsResource() {
	abs := filepath.Join(s.ws.Root, store.ConventionsFile)
	if _, err := os.ReadFile(abs); err != nil {
		return
	}
	p := filepath.ToSlash(abs)
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	uri := (&url.URL{Scheme: "file", Path: p}).String()
	s.srv.AddResource(&sdk.Resource{
		URI: uri, Name: store.ConventionsFile, Title: "Workspace conventions", MIMEType: "text/markdown",
		Description: "This workspace's own conventions for agents and people: the activity terms in use and what the workspace is for. Delivered with the server instructions and readable here in full.",
	}, func(ctx context.Context, req *sdk.ReadResourceRequest) (*sdk.ReadResourceResult, error) {
		data, err := os.ReadFile(abs)
		if err != nil {
			return nil, err
		}
		return &sdk.ReadResourceResult{Contents: []*sdk.ResourceContents{{URI: uri, MIMEType: "text/markdown", Text: string(data)}}}, nil
	})
}

// --- shared plumbing ------------------------------------------------------

// sourceIn is the source block as tool input.
type sourceIn struct {
	Author  string `json:"author,omitempty" jsonschema:"the person this is on behalf of, a URI or name; pass it only when it differs from the workspace default"`
	Harness string `json:"harness,omitempty" jsonschema:"the AI harness performing the act (defaults to the connected client's name)"`
	Model   string `json:"model,omitempty" jsonschema:"the model, if known"`
}

type clientInfoer interface{ ClientInfo() *sdk.Implementation }

// source resolves attribution: call, server flags, environment, workspace
// default, then the client's name as the harness.
func (s *Server) source(req clientInfoer, in *sourceIn) model.Source {
	pick := func(vals ...string) string {
		for _, v := range vals {
			if c := Clean(v); c != "" {
				return c
			}
		}
		return ""
	}
	var call sourceIn
	if in != nil {
		call = *in
	}
	client := ""
	if ci := req.ClientInfo(); ci != nil {
		client = ci.Name
	}
	return model.Source{
		Author:  pick(call.Author, s.opts.Author, os.Getenv(EnvAuthor), s.ws.Config.Defaults.Source.Author),
		Harness: pick(call.Harness, s.opts.Harness, os.Getenv(EnvHarness), s.ws.Config.Defaults.Source.Harness, client),
		Model:   pick(call.Model, s.opts.Model, os.Getenv(EnvModel), s.ws.Config.Defaults.Source.Model),
	}
}

// now applies an explicit now, then INTENTIONS_NOW, then the clock.
func now(in string) (time.Time, error) {
	s := Clean(in)
	if s == "" {
		s = os.Getenv(EnvNow)
	}
	if s == "" {
		return time.Now().UTC().Truncate(time.Second), nil
	}
	t, err := temporal.ParseTimestamp(s)
	if err != nil {
		return time.Time{}, apperr.Usage("now: %v", err)
	}
	return t, nil
}

func timestamp(in string, def time.Time) (time.Time, error) {
	if Clean(in) == "" {
		return def, nil
	}
	t, err := temporal.ParseTimestamp(in)
	if err != nil {
		return time.Time{}, apperr.Usage("timestamp: %v", err)
	}
	return t, nil
}

// load reads the workspace, refusing to proceed on unreadable files.
func (s *Server) load() (*store.Graph, error) {
	g, err := s.ws.Load()
	if err != nil {
		return nil, err
	}
	if len(g.Unreadable) > 0 {
		return nil, apperr.Runtime(fmt.Errorf("%s cannot be read: %s; fix the file or run validate", g.Unreadable[0].Path, g.Unreadable[0].Err))
	}
	return g, nil
}

func (s *Server) env(g *store.Graph, at time.Time, step, scope string) (resolve.Env, error) {
	e := resolve.NewEnv(s.ws, g, at)
	if step != "" {
		d, err := temporal.ParseDuration(step)
		if err != nil || d.IsZero() || !d.HasTimePart() {
			return e, apperr.Usage("step must be a positive time-of-day duration such as PT30M")
		}
		e.Step = d
	}
	if scope != "" {
		ok := false
		for _, x := range model.Scopes {
			if x == scope {
				ok = true
			}
		}
		if !ok {
			return e, apperr.Usage("scope must be personal, organisation, or public")
		}
		e.Scope = scope
	}
	return e, nil
}

func getObject(g *store.Graph, id string) (model.Object, error) {
	if _, ok := model.TypeOfID(id); !ok {
		return nil, apperr.NotFound("%q is not an identifier", id)
	}
	obj, ok := g.Get(id)
	if !ok {
		return nil, apperr.NotFound("%s does not exist", id)
	}
	return obj, nil
}

func getIntention(g *store.Graph, id string) (*model.Intention, error) {
	obj, err := getObject(g, id)
	if err != nil {
		return nil, err
	}
	in, ok := obj.(*model.Intention)
	if !ok {
		return nil, apperr.Usage("%s is a %s, not an intention", id, obj.GetType())
	}
	return in, nil
}

func getAvailability(g *store.Graph, id string) (*model.Availability, error) {
	obj, err := getObject(g, id)
	if err != nil {
		return nil, err
	}
	av, ok := obj.(*model.Availability)
	if !ok {
		return nil, apperr.Usage("%s is a %s, not an availability", id, obj.GetType())
	}
	return av, nil
}

// write checks the single-object rules, then writes.
func (s *Server) write(obj model.Object) error {
	if ps := model.Check(obj); len(ps) > 0 {
		return apperr.Invalid("%s", ps.Error())
	}
	return s.ws.WriteObject(obj)
}

// errResult turns a domain error into an isError tool result carrying the
// CLI's error code.
func errResult(err error) *sdk.CallToolResult {
	var refusal *model.Refusal
	if ok := asRefusal(err, &refusal); ok {
		err = apperr.Refused("%s", refusal.Message)
	}
	ae := apperr.Classify(err)
	return &sdk.CallToolResult{
		IsError:           true,
		Content:           []sdk.Content{&sdk.TextContent{Text: ae.ErrCode + ": " + ae.Err.Error()}},
		StructuredContent: map[string]any{"error": map[string]any{"code": ae.ErrCode, "message": ae.Err.Error()}},
	}
}

func asRefusal(err error, target **model.Refusal) bool {
	for err != nil {
		if r, ok := err.(*model.Refusal); ok {
			*target = r
			return true
		}
		u, ok := err.(interface{ Unwrap() error })
		if !ok {
			return false
		}
		err = u.Unwrap()
	}
	return false
}

// okResult pairs a one-line text summary with the structured value.
func okResult(summary string) *sdk.CallToolResult {
	return &sdk.CallToolResult{Content: []sdk.Content{&sdk.TextContent{Text: summary}}}
}

func boolp(b bool) *bool { return &b }

var (
	readOnly = &sdk.ToolAnnotations{ReadOnlyHint: true}
	additive = &sdk.ToolAnnotations{DestructiveHint: boolp(false)}
)

// --- value parsers shared by the tools -------------------------------------

// durationIn accepts a string or {nominal, min, max}.
func durationIn(v any) (*temporal.DurationSpec, error) {
	switch t := v.(type) {
	case nil:
		return nil, nil
	case string:
		if Clean(t) == "" {
			return nil, nil
		}
		d, err := temporal.ParseDurationSpec(t)
		if err != nil {
			return nil, apperr.Invalid("duration: %v", err)
		}
		return &d, nil
	case map[string]any:
		get := func(k string) (string, error) {
			raw, ok := t[k]
			if !ok {
				return "", nil
			}
			s, ok := raw.(string)
			if !ok {
				return "", apperr.Usage("duration.%s must be a string", k)
			}
			return s, nil
		}
		nom, err := get("nominal")
		if err != nil {
			return nil, err
		}
		if nom == "" {
			return nil, apperr.Usage("a ranged duration needs nominal")
		}
		n, err := temporal.ParseDuration(nom)
		if err != nil {
			return nil, apperr.Invalid("duration.nominal: %v", err)
		}
		spec := temporal.DurationSpec{Nominal: n}
		for _, k := range []string{"min", "max"} {
			s, err := get(k)
			if err != nil {
				return nil, err
			}
			if s == "" {
				continue
			}
			d, err := temporal.ParseDuration(s)
			if err != nil {
				return nil, apperr.Invalid("duration.%s: %v", k, err)
			}
			if k == "min" {
				spec.Min = &d
			} else {
				spec.Max = &d
			}
		}
		if err := spec.Validate(); err != nil {
			return nil, apperr.Invalid("duration: %v", err)
		}
		return &spec, nil
	}
	return nil, apperr.Usage("duration must be an ISO 8601 string or {nominal, min, max}")
}

// windowIn is a window as tool input.
type windowIn struct {
	Calendar string      `json:"calendar,omitempty" jsonschema:"an EDTF expression from the admitted subset (2026-W36, 2026-09, ../2026-09, a season or quarter code) or a deictic term (this-week, next-month) resolved at write time"`
	Clock    string      `json:"clock,omitempty" jsonschema:"a time-of-day interval HH:MM/HH:MM in the resolver's timezone; may cross midnight"`
	Relative *relativeIn `json:"relative,omitempty"`
}

type relativeIn struct {
	Target   string `json:"target" jsonschema:"the intention or commitment this is relative to"`
	Relation string `json:"relation" jsonschema:"FINISHTOSTART | FINISHTOFINISH | STARTTOFINISH | STARTTOSTART"`
	Gap      *gapIn `json:"gap,omitempty"`
}

type gapIn struct {
	Min string `json:"min,omitempty" jsonschema:"ISO 8601 duration"`
	Max string `json:"max,omitempty" jsonschema:"ISO 8601 duration"`
}

// window merges a window input into an existing window (nil for a new
// object): each present anchor replaces its own.
func window(in *windowIn, existing *temporal.Window, ctx temporal.Context, g *store.Graph) (*temporal.Window, error) {
	if in == nil {
		return existing, nil
	}
	w := &temporal.Window{}
	if existing != nil {
		*w = *existing
	}
	if in.Calendar != "" {
		c, err := temporal.ParseCalendarOrDeixis(in.Calendar, ctx)
		if err != nil {
			return nil, apperr.Invalid("window.calendar: %v", err)
		}
		w.Calendar = &c
	}
	if in.Clock != "" {
		c, err := temporal.ParseClock(in.Clock)
		if err != nil {
			return nil, apperr.Invalid("window.clock: %v", err)
		}
		w.Clock = &c
	}
	if in.Relative != nil {
		r := temporal.Relative{Target: in.Relative.Target, Relation: temporal.Relation(strings.ToUpper(in.Relative.Relation))}
		if in.Relative.Gap != nil {
			r.Gap = &temporal.Gap{}
			if in.Relative.Gap.Min != "" {
				d, err := temporal.ParseDuration(in.Relative.Gap.Min)
				if err != nil {
					return nil, apperr.Invalid("window.relative.gap.min: %v", err)
				}
				r.Gap.Min = &d
			}
			if in.Relative.Gap.Max != "" {
				d, err := temporal.ParseDuration(in.Relative.Gap.Max)
				if err != nil {
					return nil, apperr.Invalid("window.relative.gap.max: %v", err)
				}
				r.Gap.Max = &d
			}
		}
		if err := r.Validate(); err != nil {
			return nil, apperr.Invalid("window.relative: %v", err)
		}
		target, ok := g.Get(r.Target)
		if !ok {
			return nil, apperr.Refused("window.relative: target %s does not exist", r.Target)
		}
		if t := target.GetType(); t != model.TypeIntention && t != model.TypeCommitment {
			return nil, apperr.Refused("window.relative: target %s is a %s; a relational anchor targets an intention or a commitment", r.Target, t)
		}
		w.Relative = &r
	}
	if w.IsZero() {
		return nil, nil
	}
	return w, nil
}

func cadenceIn(s string) (*temporal.Cadence, error) {
	if Clean(s) == "" {
		return nil, nil
	}
	c, err := temporal.ParseCadence(s)
	if err != nil {
		return nil, apperr.Invalid("cadence: %v", err)
	}
	return &c, nil
}

func policyIn(name string, s string) (*model.Policy, error) {
	p, err := model.ParsePolicy(s)
	if err != nil {
		return nil, apperr.Invalid("%s: %v", name, err)
	}
	return p, nil
}

func checkURIs(name string, vs []string) error {
	for _, v := range vs {
		if !model.ValidURI(v) {
			return apperr.Invalid("%s: %q is not an absolute URI", name, v)
		}
	}
	return nil
}

func has(set []string, v string) bool {
	for _, x := range set {
		if x == v {
			return true
		}
	}
	return false
}
