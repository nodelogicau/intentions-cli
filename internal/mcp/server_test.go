package mcp

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/nodelogicau/intentions-cli/internal/store"
)

const (
	ada  = "https://example.com/people/ada"
	room = "https://example.com/rooms/3"
	rnow = "2026-09-10T00:00:00Z"
)

type harness struct {
	t  *testing.T
	ws *store.Workspace
	cs *sdk.ClientSession
}

func newHarness(t *testing.T, clientName string, opts Options) *harness {
	t.Helper()
	cfg := store.NewConfig()
	cfg.Defaults.Source.Author = ada
	cfg.Defaults.Subject = ada
	cfg.Resolver.Timezone = "Australia/Melbourne"
	ws, _, err := store.Init(filepath.Join(t.TempDir(), "planning"), cfg)
	if err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{EnvAuthor, EnvHarness, EnvModel} {
		t.Setenv(k, "")
	}
	t.Setenv(EnvNow, rnow)
	opts.Workspace, opts.Version = ws, "test"
	srv := New(opts)
	st, ct := sdk.NewInMemoryTransports()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go func() { _ = srv.Run(ctx, st) }()
	cs, err := sdk.NewClient(&sdk.Implementation{Name: clientName, Version: "1"}, nil).Connect(ctx, ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cs.Close() })
	return &harness{t: t, ws: ws, cs: cs}
}

func (h *harness) call(name string, args map[string]any) (map[string]any, bool, string) {
	h.t.Helper()
	res, err := h.cs.CallTool(context.Background(), &sdk.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		h.t.Fatalf("%s: protocol error: %v", name, err)
	}
	text := ""
	if len(res.Content) > 0 {
		if tc, ok := res.Content[0].(*sdk.TextContent); ok {
			text = tc.Text
		}
	}
	var out map[string]any
	if res.StructuredContent != nil {
		b, _ := json.Marshal(res.StructuredContent)
		_ = json.Unmarshal(b, &out)
	}
	return out, res.IsError, text
}

func (h *harness) ok(name string, args map[string]any) map[string]any {
	h.t.Helper()
	out, isErr, text := h.call(name, args)
	if isErr {
		h.t.Fatalf("%s returned error: %s", name, text)
	}
	return out
}

func (h *harness) fail(name string, args map[string]any, code string) string {
	h.t.Helper()
	out, isErr, text := h.call(name, args)
	if !isErr {
		h.t.Fatalf("%s should have failed, got %v", name, out)
	}
	if got := out["error"].(map[string]any)["code"]; got != code {
		h.t.Errorf("%s: error code %v, want %s (%s)", name, got, code, text)
	}
	return text
}

func (h *harness) file(rel string) string {
	h.t.Helper()
	b, err := os.ReadFile(filepath.Join(h.ws.Root, rel))
	if err != nil {
		h.t.Fatal(err)
	}
	return string(b)
}

func str(m map[string]any, k string) string { s, _ := m[k].(string); return s }
func obj(m map[string]any) map[string]any   { o, _ := m["object"].(map[string]any); return o }
func list(m map[string]any, k string) []any { l, _ := m[k].([]any); return l }

func TestInstructionsPromptAndTools(t *testing.T) {
	h := newHarness(t, "claude-ai", Options{})
	ins := h.cs.InitializeResult().Instructions
	for _, want := range []string{h.ws.Root, "default subject is " + ada, "Tool names are this implementation's", "Workspace conventions (intentions.md)"} {
		if !strings.Contains(ins, want) {
			t.Errorf("instructions lack %q", want)
		}
	}
	ps, err := h.cs.ListPrompts(context.Background(), nil)
	if err != nil || len(ps.Prompts) != 1 || ps.Prompts[0].Name != PromptName {
		t.Fatalf("prompts: %v %v", ps, err)
	}
	pr, err := h.cs.GetPrompt(context.Background(), &sdk.GetPromptParams{Name: PromptName})
	if err != nil || pr.Messages[0].Content.(*sdk.TextContent).Text != ins {
		t.Error("prompt text should equal instructions")
	}
	rs, err := h.cs.ListResources(context.Background(), nil)
	if err != nil || len(rs.Resources) != 1 || !strings.HasPrefix(rs.Resources[0].URI, "file://") {
		t.Fatalf("resources: %v %v", rs, err)
	}
	rr, err := h.cs.ReadResource(context.Background(), &sdk.ReadResourceParams{URI: rs.Resources[0].URI})
	if err != nil || rr.Contents[0].Text != h.file(store.ConventionsFile) {
		t.Errorf("resource read: %v", err)
	}
	tools, _ := h.cs.ListTools(context.Background(), nil)
	names := map[string]*sdk.Tool{}
	for _, tl := range tools.Tools {
		names[tl.Name] = tl
	}
	want := []string{"workspace_status", "intention_add", "intention_edit", "intention_firm", "intention_retire", "intention_show", "intention_list", "availability_add", "availability_renew", "availability_supersede", "availability_retire", "availability_list", "generate", "resolve", "select", "check", "acknowledge", "bounds", "validate"}
	if len(names) != len(want) {
		t.Errorf("%d tools, want %d", len(names), len(want))
	}
	for _, n := range want {
		if names[n] == nil {
			t.Errorf("missing tool %s", n)
		}
	}
	if !names["check"].Annotations.ReadOnlyHint || names["intention_add"].Annotations.DestructiveHint == nil || *names["intention_add"].Annotations.DestructiveHint {
		t.Error("annotations")
	}
	// The window schema is typed, not a string.
	var schema map[string]any
	b, _ := json.Marshal(names["intention_add"].InputSchema)
	_ = json.Unmarshal(b, &schema)
	props := schema["properties"].(map[string]any)
	if props["window"].(map[string]any)["properties"] == nil || props["serves"].(map[string]any)["items"] == nil {
		t.Errorf("input schema: %v", props["window"])
	}
}

func TestIntentionLifecycle(t *testing.T) {
	h := newHarness(t, "claude-ai", Options{})
	a := h.ok("intention_add", map[string]any{"title": "Draft the budget narrative", "duration": "PT90M", "window": map[string]any{"calendar": "2026-W38", "clock": "09:00/12:00"}, "activity": "deep-work"})
	id := str(a, "id")
	if a["created"] != true || !strings.HasPrefix(id, "int_") || obj(a)["stability"] != "tentative" || str(obj(a), "subject") != ada {
		t.Fatalf("add: %v", a)
	}
	src := obj(a)["source"].(map[string]any)
	if src["author"] != ada || src["harness"] != "claude-ai" {
		t.Errorf("attribution: %v", src)
	}
	body := h.file("intentions/" + id + ".yaml")
	if !strings.Contains(body, "timestamp: "+rnow) || !strings.Contains(body, "calendar: 2026-W38") {
		t.Errorf("file:\n%s", body)
	}
	// Ranged duration as an object; window anchors merge; clear removes.
	e := h.ok("intention_edit", map[string]any{"id": id, "duration": map[string]any{"nominal": "PT90M", "min": "PT1H", "max": "PT2H"}, "window": map[string]any{"clock": "10:00/12:00"}, "clear": []string{"activity"}})
	o := obj(e)
	if o["duration"].(map[string]any)["min"] != "PT1H" || o["window"].(map[string]any)["clock"] != "10:00/12:00" || o["window"].(map[string]any)["calendar"] != "2026-W38" || o["activity"] != nil || e["projection_changed"] != true {
		t.Errorf("edit: %v", o)
	}
	// Prose only: version unchanged.
	e2 := h.ok("intention_edit", map[string]any{"id": id, "description": "More words"})
	if e2["projection_changed"] != false || str(e2, "version") != str(e, "version") {
		t.Errorf("prose edit changed version: %v", e2)
	}
	// Show and list.
	s := h.ok("intention_show", map[string]any{"id": id})
	if str(s, "id") != id || s["flags"] == nil {
		t.Errorf("show: %v", s)
	}
	l := h.ok("intention_list", map[string]any{"subject": ada})
	if int(l["count"].(float64)) != 1 {
		t.Errorf("list: %v", l)
	}
	// Harness firm without policy refused; with policy firms and records firmed_under.
	h.fail("intention_firm", map[string]any{"id": id}, "refused")
	term := h.ok("intention_add", map[string]any{"title": "Ship the budget", "auto_firm": "max_duration=PT2H"})
	h.ok("intention_edit", map[string]any{"id": id, "serves": []map[string]any{{"id": str(term, "id"), "role": "for-the-sake-of"}}})
	f := h.ok("intention_firm", map[string]any{"id": id, "policy": str(term, "id")})
	if obj(f)["stability"] != "firm" || obj(f)["firmed_under"] != str(term, "id") {
		t.Errorf("firm: %v", obj(f))
	}
	// Retire, then edits refuse; list retired.
	r := h.ok("intention_retire", map[string]any{"id": id, "kind": "fulfilled", "reason": "done"})
	if obj(r)["retired"].(map[string]any)["kind"] != "fulfilled" {
		t.Errorf("retire: %v", r)
	}
	h.fail("intention_edit", map[string]any{"id": id, "title": "x"}, "refused")
	if l := h.ok("intention_list", map[string]any{"retired": true}); int(l["count"].(float64)) != 1 {
		t.Errorf("retired list: %v", l)
	}
	// Errors carry the CLI's codes.
	h.fail("intention_show", map[string]any{"id": "int_0000"}, "not_found")
	h.fail("intention_add", map[string]any{}, "usage")
	h.fail("intention_add", map[string]any{"title": "bad", "duration": "90 minutes"}, "invalid")
	h.fail("intention_retire", map[string]any{"id": str(term, "id"), "kind": "superseded"}, "refused")
}

func TestAvailabilityLifecycle(t *testing.T) {
	h := newHarness(t, "claude-ai", Options{})
	h.fail("availability_add", map[string]any{"duration": "PT3H", "window": map[string]any{"clock": "09:00/12:00"}}, "usage")
	a := h.ok("availability_add", map[string]any{"subject": ada, "title": "Tuesday mornings", "duration": "PT3H", "window": map[string]any{"calendar": "2026-09/2026-12", "clock": "09:00/12:00"}, "conditional": []string{"deep-work"}, "cadence": "FREQ=WEEKLY;BYDAY=TU"})
	id := str(a, "id")
	if !strings.HasPrefix(id, "avl_") || a["effective_valid_until_from"] != "default_horizon" || obj(a)["scope"] != "personal" {
		t.Fatalf("add: %v", a)
	}
	rn := h.ok("availability_renew", map[string]any{"id": id, "valid_until": "2027-06"})
	if rn["effective_valid_until_from"] != "explicit" {
		t.Errorf("renew: %v", rn)
	}
	h.fail("availability_renew", map[string]any{"id": id, "valid_until": "2026-10"}, "refused")
	sp := h.ok("availability_supersede", map[string]any{"id": id, "duration": "PT2H", "reason": "less time"})
	nid := str(sp, "id")
	if nid == id || obj(sp)["duration"] != "PT2H" || sp["superseded"].(map[string]any)["id"] != id || obj(sp)["window"].(map[string]any)["clock"] != "09:00/12:00" {
		t.Errorf("supersede: %v", sp)
	}
	if !strings.Contains(h.file("availability/"+id+".yaml"), "superseded_by: "+nid) {
		t.Error("old availability not retired as superseded")
	}
	// Scope only widens.
	wide := h.ok("availability_supersede", map[string]any{"id": nid, "scope": "organisation"})
	nid = str(wide, "id")
	h.fail("availability_supersede", map[string]any{"id": nid, "scope": "personal"}, "refused")
	l := h.ok("availability_list", map[string]any{"subject": ada})
	if int(l["count"].(float64)) != 1 || str(list(l, "availability")[0].(map[string]any), "id") != nid {
		t.Errorf("list: %v", l)
	}
	rt := h.ok("availability_retire", map[string]any{"id": nid, "kind": "retracted"})
	if obj(rt)["retired"].(map[string]any)["kind"] != "retracted" {
		t.Errorf("retire: %v", rt)
	}
}

func TestResolutionLoop(t *testing.T) {
	h := newHarness(t, "claude-ai", Options{})
	av := func(args map[string]any) map[string]any {
		args["subject"], args["window"].(map[string]any)["calendar"] = ada, "2026-09/2026-12"
		return h.ok("availability_add", args)
	}
	av(map[string]any{"title": "Tuesday mornings", "duration": "PT3H", "window": map[string]any{"clock": "09:00/12:00"}, "conditional": []string{"deep-work"}, "cadence": "FREQ=WEEKLY;BYDAY=TU"})
	av(map[string]any{"title": "Weekday afternoons", "duration": "PT4H", "window": map[string]any{"clock": "13:00/17:00"}, "conditional": []string{"meeting"}, "cadence": "FREQ=WEEKLY;BYDAY=MO,TU,WE,TH,FR"})
	h.ok("availability_add", map[string]any{"subject": room, "duration": "PT8H", "window": map[string]any{"calendar": "2026-09/2026-12", "clock": "08:00/18:00"}, "cadence": "FREQ=WEEKLY;BYDAY=MO,TU,WE,TH,FR"})

	a := h.ok("intention_add", map[string]any{"title": "Draft the budget narrative", "duration": "PT90M", "window": map[string]any{"calendar": "2026-W38"}, "activity": "deep-work"})
	r := h.ok("resolve", map[string]any{"id": str(a, "id"), "limit": 4})
	cands := list(r, "candidates")
	if len(cands) != 4 || int(r["candidates_considered"].(float64)) != 7 || cands[0].(map[string]any)["start"] != "2026-09-15T09:00:00+10:00" || cands[0].(map[string]any)["rank"].(float64) != 1 {
		t.Fatalf("resolve: %v", r)
	}
	if fi, _ := os.ReadDir(filepath.Join(h.ws.Root, "resolutions")); len(fi) != 0 {
		t.Error("resolve wrote a record")
	}
	h.fail("select", map[string]any{"id": str(a, "id")}, "usage")
	s := h.ok("select", map[string]any{"id": str(a, "id"), "candidate": 1})
	rec := s["resolution"].(map[string]any)
	if rec["selector"] != "person" || rec["intention"] != str(a, "id") || s["intention"].(map[string]any)["placement"].(map[string]any)["start"] != "2026-09-15T09:00:00+10:00" {
		t.Errorf("select: %v", s)
	}
	h.fail("select", map[string]any{"id": str(a, "id"), "candidate": 2}, "refused")
	// A harness under a policy: terminus with auto_select covering a short intention.
	term := h.ok("intention_add", map[string]any{"title": "Keep the board informed", "auto_select": "max_duration=PT1H"})
	b := h.ok("intention_add", map[string]any{"title": "Read the board pack", "duration": "PT1H", "window": map[string]any{"calendar": "2026-W38"}, "activity": "deep-work", "serves": []map[string]any{{"id": str(term, "id"), "role": "for-the-sake-of"}}})
	h.fail("select", map[string]any{"id": str(b, "id"), "policy": str(a, "id")}, "refused")
	sb := h.ok("select", map[string]any{"id": str(b, "id"), "policy": str(term, "id")})
	if sb["resolution"].(map[string]any)["selector"] != str(term, "id") || sb["policy"] != str(term, "id") || sb["candidate"].(map[string]any)["start"] != "2026-09-15T10:30:00+10:00" {
		t.Errorf("policy select: %v", sb)
	}
	// Parties: a commitment with everyone tentative.
	m := h.ok("intention_add", map[string]any{"title": "Review in room 3", "duration": "PT1H", "window": map[string]any{"calendar": "2026-W38"}, "activity": "meeting", "parties": []string{room}})
	sm := h.ok("select", map[string]any{"id": str(m, "id"), "candidate": 1})
	cmt, _ := sm["commitment"].(map[string]any)
	if cmt == nil || len(list(cmt, "parties")) != 2 || cmt["intention"] != str(m, "id") {
		t.Errorf("commitment: %v", sm)
	}
	// Displacement is reported; check and acknowledge round-trip.
	d := h.ok("intention_add", map[string]any{"title": "Two hours", "duration": "PT2H", "window": map[string]any{"calendar": "2026-09-14", "clock": "13:00/15:00"}, "activity": "meeting"})
	rd := h.ok("resolve", map[string]any{"id": str(d, "id")})
	if c0 := list(rd, "candidates")[0].(map[string]any); c0["rank"].(float64) != 2 || len(list(c0, "displaces")) != 1 {
		t.Errorf("displacement: %v", rd)
	}
	sd := h.ok("select", map[string]any{"id": str(d, "id"), "candidate": 1})
	var own map[string]any
	for _, f := range list(sd, "flags") {
		if f := f.(map[string]any); f["subject"] == str(d, "id") && f["kind"] == "window-clash" {
			own = f
		}
	}
	if own == nil {
		t.Fatalf("flags after displacing select: %v", sd)
	}
	ck := h.ok("check", map[string]any{"ids": []string{str(d, "id")}})
	if int(ck["count"].(float64)) == 0 {
		t.Errorf("check: %v", ck)
	}
	cp := str(own, "counterpart")
	ak := h.ok("acknowledge", map[string]any{"id": str(d, "id"), "kind": "window-clash", "counterpart": cp, "reason": "the review can move"})
	if ak["acknowledgement"].(map[string]any)["counterpart_version"] == "" || len(list(ak, "flags")) != 0 {
		t.Errorf("acknowledge: %v", ak)
	}
	h.fail("acknowledge", map[string]any{"id": str(d, "id"), "kind": "nonsense"}, "usage")
	// Generation, bounds, validate, status.
	rec2 := h.ok("intention_add", map[string]any{"title": "Weekly review", "duration": "PT1H", "window": map[string]any{"calendar": "2026-09/2026-12"}, "cadence": "FREQ=WEEKLY;BYDAY=FR"})
	g := h.ok("generate", map[string]any{"horizon": "P3W", "recurring": []string{str(rec2, "id")}})
	if int(g["count"].(float64)) != 3 {
		t.Errorf("generate: %v", g)
	}
	if g2 := h.ok("generate", map[string]any{"horizon": "P3W"}); int(g2["count"].(float64)) != 0 || len(list(g2, "skipped")) != 3 {
		t.Errorf("generate idempotent: %v", g2)
	}
	bd := h.ok("bounds", map[string]any{"calendar": "2026-W38", "clock": "09:00/12:00"})
	if int(bd["count"].(float64)) != 7 || bd["timezone"] != "Australia/Melbourne" {
		t.Errorf("bounds: %v", bd)
	}
	h.fail("bounds", map[string]any{}, "usage")
	v := h.ok("validate", map[string]any{})
	if v["ok"] != true {
		t.Errorf("validate: %v", v)
	}
	ws := h.ok("workspace_status", map[string]any{})
	if str(ws, "root") != h.ws.Root || ws["counts"].(map[string]any)["intention"].(float64) < 8 || ws["counts"].(map[string]any)["resolution"].(float64) != 4 {
		t.Errorf("status: %v", ws)
	}
}

func TestAttributionPrecedence(t *testing.T) {
	// Server flags beat env and workspace; a placeholder counts as absent;
	// the client name is the harness of last resort.
	h := newHarness(t, "Claude Desktop", Options{Author: "${user_config.author}", Model: "claude-fable-5-1"})
	t.Setenv(EnvHarness, "env-harness")
	a := h.ok("intention_add", map[string]any{"title": "A"})
	src := obj(a)["source"].(map[string]any)
	if src["author"] != ada || src["harness"] != "env-harness" || src["model"] != "claude-fable-5-1" {
		t.Errorf("precedence: %v", src)
	}
	t.Setenv(EnvHarness, "")
	b := h.ok("intention_add", map[string]any{"title": "B", "source": map[string]any{"author": "https://example.com/people/bob"}})
	src = obj(b)["source"].(map[string]any)
	if src["author"] != "https://example.com/people/bob" || src["harness"] != "Claude Desktop" {
		t.Errorf("call and client: %v", src)
	}
}

func TestConcurrentWrites(t *testing.T) {
	h := newHarness(t, "claude-ai", Options{})
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			h.ok("intention_add", map[string]any{"title": "Parallel " + string(rune('a'+i))})
		}(i)
	}
	wg.Wait()
	if l := h.ok("intention_list", map[string]any{}); int(l["count"].(float64)) != 8 {
		t.Errorf("after concurrent adds: %v", l["count"])
	}
	if v := h.ok("validate", map[string]any{}); v["ok"] != true {
		t.Errorf("validate: %v", v)
	}
}
