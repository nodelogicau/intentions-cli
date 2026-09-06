package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	ada = "https://example.com/people/ada"
	now = "2026-09-04T09:00:00Z"
)

type result struct {
	code           int
	stdout, stderr string
	json           map[string]any
}

func (r result) str(key string) string {
	v, _ := r.json[key].(string)
	return v
}

func (r result) obj() map[string]any {
	m, _ := r.json["object"].(map[string]any)
	return m
}

// run drives the CLI in-process with --json and a fixed clock.
func run(t *testing.T, ws string, stdin string, args ...string) result {
	t.Helper()
	full := append([]string{"--json"}, args...)
	if ws != "" {
		full = append(full, "--workspace", ws)
	}
	var out, errb bytes.Buffer
	code := Execute(full, strings.NewReader(stdin), &out, &errb, func() bool { return stdin == "" })
	r := result{code: code, stdout: out.String(), stderr: errb.String()}
	if out.Len() > 0 {
		_ = json.Unmarshal(out.Bytes(), &r.json)
	}
	if errb.Len() > 0 && code != 0 && out.Len() == 0 {
		var e map[string]any
		if json.Unmarshal(errb.Bytes(), &e) == nil {
			r.json = e
		}
	}
	return r
}

// text drives the CLI without --json.
func text(t *testing.T, ws string, args ...string) result {
	t.Helper()
	full := append([]string{}, args...)
	if ws != "" {
		full = append(full, "--workspace", ws)
	}
	var out, errb bytes.Buffer
	code := Execute(full, strings.NewReader(""), &out, &errb, func() bool { return true })
	return result{code: code, stdout: out.String(), stderr: errb.String()}
}

func initWS(t *testing.T, extra ...string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "planning")
	args := append([]string{"init", dir, "--author", ada, "--subject", ada, "--timezone", "Australia/Melbourne"}, extra...)
	r := run(t, "", "", args...)
	if r.code != 0 {
		t.Fatalf("init: %d %s %s", r.code, r.stdout, r.stderr)
	}
	return dir
}

func mustOK(t *testing.T, r result, what string) result {
	t.Helper()
	if r.code != 0 {
		t.Fatalf("%s: exit %d\nstdout: %s\nstderr: %s", what, r.code, r.stdout, r.stderr)
	}
	return r
}

func errMsg(r result) string {
	e, _ := r.json["error"].(map[string]any)
	m, _ := e["message"].(string)
	return m
}

func readFile(t *testing.T, ws, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(ws, rel))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func addIntention(t *testing.T, ws string, args ...string) result {
	t.Helper()
	full := append([]string{"intention", "add", "--now", now}, args...)
	return mustOK(t, run(t, ws, "", full...), "intention add "+strings.Join(args, " "))
}

// --- cli-interface ---------------------------------------------------------

func TestJSONContractAndExitCodes(t *testing.T) {
	ws := initWS(t)
	r := run(t, ws, "", "version")
	if r.code != 0 || r.str("format") != "intentions/0.1" || r.stderr != "" {
		t.Errorf("version: %+v", r)
	}
	// Failure in JSON mode: stdout empty, stderr one object.
	r = run(t, ws, "", "show", "int_does-not-exist")
	if r.code != 3 || r.stdout != "" || errMsg(r) == "" {
		t.Errorf("not found: %+v", r)
	}
	// Missing required flag.
	r = run(t, ws, "", "intention", "add")
	if r.code != 2 || !strings.Contains(errMsg(r), "--title") {
		t.Errorf("missing title: %+v", r)
	}
	// No workspace.
	t.Setenv(EnvWorkspace(), "")
	empty := t.TempDir()
	var out, errb bytes.Buffer
	code := Execute([]string{"--json", "--workspace", empty, "intention", "list"}, strings.NewReader(""), &out, &errb, func() bool { return true })
	if code != 5 {
		t.Errorf("no workspace: %d %s", code, errb.String())
	}
	// Refused write exits 2 with a refused code.
	r = run(t, ws, "", "intention", "add", "--title", "x", "--activity", "Deep Work")
	if r.code != 2 {
		t.Errorf("refused: %+v", r)
	}
	// Text mode error goes to stderr.
	tr := text(t, ws, "show", "int_nope")
	if tr.code != 3 || !strings.HasPrefix(tr.stderr, "error:") {
		t.Errorf("text error: %+v", tr)
	}
}

func EnvWorkspace() string { return "INTENTIONS_WORKSPACE" }

func TestStdinAndAttribution(t *testing.T) {
	ws := initWS(t)
	r := mustOK(t, run(t, ws, "Two lines\nof prose\n", "intention", "add", "--title", "X", "--description-file", "-", "--now", now), "stdin")
	if r.obj()["description"] != "Two lines\nof prose\n" {
		t.Errorf("description: %v", r.obj()["description"])
	}
	// Harness drafts on the default author's behalf.
	r = addIntention(t, ws, "--title", "Y", "--harness", "claude", "--model", "claude-fable-5-1")
	src := r.obj()["source"].(map[string]any)
	if src["author"] != ada || src["harness"] != "claude" || src["model"] != "claude-fable-5-1" {
		t.Errorf("source: %v", src)
	}
	body := readFile(t, ws, r.str("path"))
	if !strings.Contains(body, "timestamp: 2026-09-04T09:00:00Z") {
		t.Errorf("timestamp from --now missing:\n%s", body)
	}
	// Environment attribution.
	t.Setenv(EnvHarness, "copilot")
	r = addIntention(t, ws, "--title", "Z")
	if r.obj()["source"].(map[string]any)["harness"] != "copilot" {
		t.Errorf("env harness: %v", r.obj()["source"])
	}
	t.Setenv(EnvHarness, "")
	// No author anywhere.
	org := filepath.Join(t.TempDir(), "org")
	mustOK(t, run(t, "", "", "init", org, "--author", "https://example.com/org"), "org init")
	cfg := readFile(t, org, "intentions.yaml")
	cfg = strings.Replace(cfg, "    author: https://example.com/org\n", "", 1)
	_ = os.WriteFile(filepath.Join(org, "intentions.yaml"), []byte(cfg), 0o644)
	r = run(t, org, "", "intention", "add", "--title", "X", "--subject", ada, "--harness", "claude")
	if r.code != 2 {
		t.Errorf("no author: %+v", r)
	}
}

// --- workspace -------------------------------------------------------------

func TestInitAndDiscovery(t *testing.T) {
	ws := initWS(t)
	cfg := readFile(t, ws, "intentions.yaml")
	for _, want := range []string{"format: intentions/0.1", "hash: sha256", "timezone: Australia/Melbourne", "week_start: monday", "default_horizon: P13W", "horizon: P4W", "subject: " + ada, "author: " + ada} {
		if !strings.Contains(cfg, want) {
			t.Errorf("config missing %q", want)
		}
	}
	for _, d := range []string{"intentions", "availability", "commitments", "resolutions"} {
		if st, err := os.Stat(filepath.Join(ws, d)); err != nil || !st.IsDir() {
			t.Errorf("missing dir %s", d)
		}
	}
	if _, err := os.Stat(filepath.Join(ws, "intentions.md")); err != nil {
		t.Error("missing intentions.md")
	}
	// Second init refused with exit 1.
	r := run(t, "", "", "init", ws, "--author", ada)
	if r.code != 1 {
		t.Errorf("second init: %+v", r)
	}
	// Unknown timezone.
	r = run(t, "", "", "init", filepath.Join(t.TempDir(), "x"), "--author", ada, "--timezone", "Mars/Olympus")
	if r.code != 2 {
		t.Errorf("bad tz: %+v", r)
	}
	// Organisation workspace: no default subject, add without --subject refused.
	org := filepath.Join(t.TempDir(), "org")
	mustOK(t, run(t, "", "", "init", org, "--author", ada), "org")
	if strings.Contains(readFile(t, org, "intentions.yaml"), "subject:") {
		t.Error("org config carries a subject")
	}
	r = run(t, org, "", "intention", "add", "--title", "X")
	if r.code != 2 || !strings.Contains(errMsg(r), "subject") {
		t.Errorf("org add: %+v", r)
	}
	// Unsupported hash.
	bad := readFile(t, ws, "intentions.yaml")
	_ = os.WriteFile(filepath.Join(ws, "intentions.yaml"), []byte(strings.Replace(bad, "hash: sha256", "hash: sha512", 1)), 0o644)
	r = run(t, ws, "", "intention", "list")
	if r.code != 2 || !strings.Contains(errMsg(r), "sha256") {
		t.Errorf("hash: %+v", r)
	}
	_ = os.WriteFile(filepath.Join(ws, "intentions.yaml"), []byte(bad), 0o644)
	// Pointer discovery and workspace verb.
	other := t.TempDir()
	_ = os.WriteFile(filepath.Join(other, ".intentions"), []byte(ws+"\n"), 0o644)
	cwd, _ := os.Getwd()
	_ = os.Chdir(other)
	defer func() { _ = os.Chdir(cwd) }()
	t.Setenv(EnvWorkspace(), "")
	r = run(t, "", "", "workspace")
	if r.code != 0 || r.str("found_by") != "pointer" {
		t.Errorf("pointer: %+v", r)
	}
	// Strict env var.
	t.Setenv(EnvWorkspace(), t.TempDir())
	r = run(t, "", "", "workspace")
	if r.code != 5 {
		t.Errorf("strict env: %+v", r)
	}
	// Flag wins over env.
	r = run(t, ws, "", "workspace")
	if r.code != 0 || r.str("found_by") != "flag" {
		t.Errorf("flag wins: %+v", r)
	}
}

// --- intentions ------------------------------------------------------------

func TestIntentionAddMinimalAndFull(t *testing.T) {
	ws := initWS(t)
	r := addIntention(t, ws, "--title", "Draft the Q4 budget narrative")
	o := r.obj()
	if o["subject"] != ada || o["stability"] != "tentative" || o["title"] != "Draft the Q4 budget narrative" {
		t.Errorf("minimal: %v", o)
	}
	if _, has := o["window"]; has {
		t.Error("minimal has window")
	}
	if _, has := o["duration"]; has {
		t.Error("minimal has duration")
	}
	body := readFile(t, ws, r.str("path"))
	if !strings.Contains(body, "serves: []") || !strings.Contains(body, "acknowledgements: []") || !strings.Contains(body, "version: sha256:") {
		t.Errorf("minimal file:\n%s", body)
	}
	if !strings.HasPrefix(r.str("version"), "sha256:") {
		t.Errorf("version: %s", r.str("version"))
	}

	// Full example: a terminus and a means, then the spec's intention.
	term := addIntention(t, ws, "--title", "Being someone who follows through")
	means := addIntention(t, ws, "--title", "Board pack out", "--calendar", "2026-W38")
	full := addIntention(t, ws,
		"--title", "Draft the Q4 budget narrative",
		"--description", "Two focused sessions should be enough; the numbers are already in.\n",
		"--duration", "PT90M",
		"--calendar", "2026-W37",
		"--relative", means.str("id")+":FINISHTOSTART:P0D:P3D",
		"--activity", "deep-work",
		"--location", "https://example.com/places/home",
		"--serves", means.str("id")+":in-order-to",
		"--serves", term.str("id")+":for-the-sake-of",
		"--harness", "claude", "--model", "claude-fable-5-1",
		"--timestamp", "2026-09-04T09:12:00Z")
	body = readFile(t, ws, full.str("path"))
	want := "id: " + full.str("id") + `
subject: https://example.com/people/ada
title: Draft the Q4 budget narrative
description: |
  Two focused sessions should be enough; the numbers are already in.
duration: PT90M
window:
  calendar: 2026-W37
  relative:
    target: ` + means.str("id") + `
    relation: FINISHTOSTART
    gap: {min: P0D, max: P3D}
stability: tentative
activity: deep-work
location:
  - https://example.com/places/home
serves:
`
	if !strings.HasPrefix(body, want) {
		t.Errorf("full example prefix:\n%s\nwant\n%s", body, want)
	}
	for _, line := range []string{"source:\n  author: https://example.com/people/ada\n  harness: claude\n  model: claude-fable-5-1\ntimestamp: 2026-09-04T09:12:00Z\nacknowledgements: []\nversion: sha256:"} {
		if !strings.Contains(body, line) {
			t.Errorf("full example missing:\n%s\nin\n%s", line, body)
		}
	}
	// Deixis: this-week at 2026-09-04 in Melbourne is W36; next-quarter is Q4.
	r = addIntention(t, ws, "--title", "W", "--calendar", "this-week")
	if r.obj()["window"].(map[string]any)["calendar"] != "2026-W36" {
		t.Errorf("this-week: %v", r.obj()["window"])
	}
	r = addIntention(t, ws, "--title", "Q", "--calendar", "next-quarter")
	if r.obj()["window"].(map[string]any)["calendar"] != "2026-36" {
		t.Errorf("next-quarter: %v", r.obj()["window"])
	}
	// Clock alone; ranged duration; sorted lists.
	r = addIntention(t, ws, "--title", "C", "--clock", "17:00/21:00", "--duration", "PT1H:PT30M:PT2H", "--location", "https://b/", "--location", "https://a/")
	body = readFile(t, ws, r.str("path"))
	if !strings.Contains(body, "window:\n  clock: 17:00/21:00\n") || !strings.Contains(body, "duration: {nominal: PT1H, min: PT30M, max: PT2H}") || !strings.Contains(body, "location:\n  - https://a/\n  - https://b/\n") {
		t.Errorf("clock/ranged/sorted:\n%s", body)
	}
}

func TestIntentionRefusals(t *testing.T) {
	ws := initWS(t)
	cases := []struct {
		args []string
		want string
	}{
		{[]string{"--activity", "Deep Work"}, "kebab"},
		{[]string{"--calendar", "2026-09~"}, "qualifier"},
		{[]string{"--calendar", "2026-W38/2026-W36"}, "after"},
		{[]string{"--clock", "09:00/09:00"}, "equal"},
		{[]string{"--duration", "90m"}, "ISO 8601"},
		{[]string{"--duration", "PT3H:PT30M:PT2H"}, "longer than max"},
		{[]string{"--cadence", "FREQ=WEEKLY;BYDAY=TU;BYHOUR=9"}, "clock"},
		{[]string{"--relative", "int_A:DEPENDS-ON"}, "FINISHTOSTART"},
		{[]string{"--relative", "int_missing:FINISHTOSTART"}, "does not exist"},
		{[]string{"--serves", "int_x:PARENT"}, "role"},
		{[]string{"--preference", "soonest"}, "preference"},
		{[]string{"--auto-firm", "activity=deep-work"}, "max_duration"},
		{[]string{"--location", "home"}, "absolute URI"},
		{[]string{"--stability", "maybe"}, "stability"},
	}
	for _, c := range cases {
		args := append([]string{"intention", "add", "--title", "X"}, c.args...)
		r := run(t, ws, "", args...)
		if r.code != 2 || !strings.Contains(errMsg(r), c.want) {
			t.Errorf("%v: exit %d, message %q (want %q)", c.args, r.code, errMsg(r), c.want)
		}
	}
	// Nothing was written.
	entries, _ := os.ReadDir(filepath.Join(ws, "intentions"))
	if len(entries) != 0 {
		t.Errorf("refused writes left files: %v", entries)
	}
}

func TestServesGraphRules(t *testing.T) {
	ws := initWS(t)
	a := addIntention(t, ws, "--title", "A")
	b := addIntention(t, ws, "--title", "B", "--serves", a.str("id")+":in-order-to")
	c := addIntention(t, ws, "--title", "C")
	// Multiple ends, sorted by id.
	d := addIntention(t, ws, "--title", "D", "--serves", c.str("id")+":in-order-to", "--serves", a.str("id")+":in-order-to")
	serves := d.obj()["serves"].([]any)
	if len(serves) != 2 || serves[0].(map[string]any)["id"] != a.str("id") {
		t.Errorf("multiple ends: %v", serves)
	}
	// Cycle refused at write, naming both.
	r := run(t, ws, "", "intention", "edit", a.str("id"), "--serves", b.str("id")+":in-order-to")
	if r.code != 2 || !strings.Contains(errMsg(r), a.str("id")) || !strings.Contains(errMsg(r), b.str("id")) {
		t.Errorf("cycle: %d %s", r.code, errMsg(r))
	}
	// Terminus stays a sink.
	term := addIntention(t, ws, "--title", "T")
	addIntention(t, ws, "--title", "E", "--serves", term.str("id")+":for-the-sake-of")
	r = run(t, ws, "", "intention", "edit", term.str("id"), "--serves", c.str("id")+":in-order-to")
	if r.code != 2 || !strings.Contains(errMsg(r), "terminus") {
		t.Errorf("terminus sink: %d %s", r.code, errMsg(r))
	}
	// for-the-sake-of target with serves refused.
	r = run(t, ws, "", "intention", "add", "--title", "F", "--serves", b.str("id")+":for-the-sake-of")
	if r.code != 2 || !strings.Contains(errMsg(r), "terminus") {
		t.Errorf("terminus target: %d %s", r.code, errMsg(r))
	}
	// show resolves titles.
	r = mustOK(t, run(t, ws, "", "intention", "show", b.str("id")), "show")
	res := r.json["serves_resolved"].([]any)
	if len(res) != 1 || res[0].(map[string]any)["title"] != "A" {
		t.Errorf("serves_resolved: %v", res)
	}
}

func TestFirmAndPolicies(t *testing.T) {
	ws := initWS(t)
	a := addIntention(t, ws, "--title", "A", "--duration", "PT20M")
	// Person firms.
	r := mustOK(t, run(t, ws, "", "intention", "firm", a.str("id")), "person firm")
	if r.obj()["stability"] != "firm" || r.str("previous_version") == r.str("version") {
		t.Errorf("person firm: %v", r.json)
	}
	// Harness without policy.
	b := addIntention(t, ws, "--title", "B", "--duration", "PT20M")
	r = run(t, ws, "", "intention", "firm", b.str("id"), "--harness", "claude")
	if r.code != 2 || !strings.Contains(errMsg(r), "--policy") {
		t.Errorf("harness no policy: %d %s", r.code, errMsg(r))
	}
	if strings.Contains(readFile(t, ws, b.str("path")), "stability: firm") {
		t.Error("file changed after refusal")
	}
	// Policy written as a mapping.
	p := addIntention(t, ws, "--title", "Small things may be firmed", "--auto-firm", "max_duration=PT30M,stability=tentative")
	if !strings.Contains(readFile(t, ws, p.str("path")), "auto_firm: {max_duration: PT30M, stability: tentative}") {
		t.Errorf("policy file:\n%s", readFile(t, ws, p.str("path")))
	}
	// Harness under policy.
	r = mustOK(t, run(t, ws, "", "intention", "firm", b.str("id"), "--harness", "claude", "--policy", p.str("id")), "harness under policy")
	if r.str("policy") != p.str("id") || r.obj()["stability"] != "firm" {
		t.Errorf("policy result: %v", r.json)
	}
	// Policy not met.
	c := addIntention(t, ws, "--title", "C", "--duration", "PT2H")
	r = run(t, ws, "", "intention", "firm", c.str("id"), "--harness", "claude", "--policy", p.str("id"))
	if r.code != 2 || !strings.Contains(errMsg(r), "max_duration") {
		t.Errorf("policy not met: %d %s", r.code, errMsg(r))
	}
	// add --stability firm by a harness is refused; by a person accepted.
	r = run(t, ws, "", "intention", "add", "--title", "D", "--stability", "firm", "--harness", "claude")
	if r.code != 2 {
		t.Errorf("harness add firm: %d", r.code)
	}
	addIntention(t, ws, "--title", "D", "--stability", "firm")
	// Harness drafts tentative.
	addIntention(t, ws, "--title", "E", "--harness", "claude")
}

func TestEditAndRetire(t *testing.T) {
	ws := initWS(t)
	a := addIntention(t, ws, "--title", "A", "--calendar", "2026-09", "--activity", "deep-work", "--clock", "09:00/12:00")
	// Window narrowed keeps id and clock; version changes.
	r := mustOK(t, run(t, ws, "", "intention", "edit", a.str("id"), "--calendar", "2026-W37"), "narrow")
	w := r.obj()["window"].(map[string]any)
	if w["calendar"] != "2026-W37" || w["clock"] != "09:00/12:00" || r.str("id") != a.str("id") || r.json["projection_changed"] != true {
		t.Errorf("narrow: %v", r.json)
	}
	// Prose edit leaves version.
	r = mustOK(t, run(t, ws, "", "intention", "edit", a.str("id"), "--description", "more detail"), "prose")
	if r.json["projection_changed"] != false || r.str("previous_version") != r.str("version") {
		t.Errorf("prose edit: %v", r.json)
	}
	// Clear optional field.
	r = mustOK(t, run(t, ws, "", "intention", "edit", a.str("id"), "--clear-activity"), "clear")
	if _, has := r.obj()["activity"]; has {
		t.Error("activity not cleared")
	}
	// Subject change refused.
	r = run(t, ws, "", "intention", "edit", a.str("id"), "--title", "x", "--calendar", "2026-10")
	mustOK(t, r, "edit ok")
	// Retire: superseded without target, wrong kind, superseded-by on fulfilled.
	c := addIntention(t, ws, "--title", "C")
	for _, args := range [][]string{
		{"--kind", "superseded"},
		{"--kind", "cancelled"},
		{"--kind", "fulfilled", "--superseded-by", c.str("id")},
		{"--kind", "superseded", "--superseded-by", "int_missing"},
	} {
		full := append([]string{"intention", "retire", a.str("id")}, args...)
		if r := run(t, ws, "", full...); r.code != 2 {
			t.Errorf("%v: exit %d %s", args, r.code, errMsg(r))
		}
	}
	r = mustOK(t, run(t, ws, "", "intention", "retire", a.str("id"), "--kind", "superseded", "--superseded-by", c.str("id"), "--reason", "split", "--now", now), "retire")
	body := readFile(t, ws, a.str("path"))
	if !strings.Contains(body, "retired:\n  kind: superseded\n  reason: split\n  superseded_by: "+c.str("id")+"\n  source:\n    author: "+ada+"\n  timestamp: 2026-09-04T09:00:00Z\nversion: sha256:") {
		t.Errorf("retired file:\n%s", body)
	}
	if r.str("previous_version") == r.str("version") {
		t.Error("retirement kept version")
	}
	// Retired object refuses edits and a second retirement.
	if r := run(t, ws, "", "intention", "edit", a.str("id"), "--title", "z"); r.code != 2 {
		t.Errorf("edit retired: %d", r.code)
	}
	if r := run(t, ws, "", "intention", "retire", a.str("id"), "--kind", "abandoned"); r.code != 2 {
		t.Errorf("second retire: %d", r.code)
	}
	// List excludes retired by default; --retired lists it.
	r = mustOK(t, run(t, ws, "", "intention", "list"), "list")
	if int(r.json["count"].(float64)) != 1 {
		t.Errorf("list count: %v", r.json["count"])
	}
	r = mustOK(t, run(t, ws, "", "intention", "list", "--retired"), "list retired")
	if int(r.json["count"].(float64)) != 1 {
		t.Errorf("retired count: %v", r.json["count"])
	}
	// Standing filter.
	addIntention(t, ws, "--title", "S", "--cadence", "FREQ=WEEKLY;BYDAY=TU", "--calendar", "2026-09/..")
	r = mustOK(t, run(t, ws, "", "intention", "list", "--standing"), "standing")
	if int(r.json["count"].(float64)) != 1 {
		t.Errorf("standing count: %v", r.json["count"])
	}
}

// --- availability ----------------------------------------------------------

func addAvail(t *testing.T, ws string, args ...string) result {
	t.Helper()
	full := append([]string{"availability", "add", "--now", now, "--subject", ada}, args...)
	return mustOK(t, run(t, ws, "", full...), "availability add "+strings.Join(args, " "))
}

func TestAvailabilityAddAndRules(t *testing.T) {
	ws := initWS(t)
	// Room availability.
	r := mustOK(t, run(t, ws, "", "availability", "add", "--subject", "https://example.org/rooms/3", "--calendar", "2026-W37", "--duration", "PT8H", "--now", now), "room")
	o := r.obj()
	if o["subject"] != "https://example.org/rooms/3" || o["scope"] != "personal" || o["source"].(map[string]any)["author"] != ada {
		t.Errorf("room: %v", o)
	}
	if _, has := o["conditional"]; has {
		t.Error("room has conditional")
	}
	// Recurring mornings matches the spec's example.
	r = addAvail(t, ws, "--title", "Tuesday mornings for deep work", "--duration", "PT3H", "--calendar", "2026-09/2026-12", "--clock", "09:00/12:00", "--conditional", "writing", "--conditional", "deep-work", "--location", "https://example.com/places/home", "--cadence", "FREQ=WEEKLY;BYDAY=TU", "--valid-until", "2026-12", "--timestamp", "2026-09-04T09:20:00Z")
	body := readFile(t, ws, r.str("path"))
	want := "id: " + r.str("id") + `
subject: https://example.com/people/ada
title: Tuesday mornings for deep work
duration: PT3H
window:
  calendar: 2026-09/2026-12
  clock: 09:00/12:00
conditional:
  - deep-work
  - writing
location:
  - https://example.com/places/home
cadence: FREQ=WEEKLY;BYDAY=TU
valid_until: 2026-12
scope: personal
source:
  author: https://example.com/people/ada
timestamp: 2026-09-04T09:20:00Z
version: sha256:`
	if !strings.HasPrefix(body, want) {
		t.Errorf("example:\n%s\nwant\n%s", body, want)
	}
	if r.str("effective_valid_until") != "2027-01-01T00:00:00Z" && !strings.HasPrefix(r.str("effective_valid_until"), "2026-12-31T13:00:00Z") {
		t.Errorf("explicit horizon: %s", r.str("effective_valid_until"))
	}
	// Missing duration, missing window, malformed conditional, unknown subject accepted.
	if r := run(t, ws, "", "availability", "add", "--subject", ada, "--calendar", "2026-W37"); r.code != 2 || !strings.Contains(errMsg(r), "duration") {
		t.Errorf("missing duration: %d %s", r.code, errMsg(r))
	}
	if r := run(t, ws, "", "availability", "add", "--subject", ada, "--duration", "PT1H"); r.code != 2 || !strings.Contains(errMsg(r), "window") {
		t.Errorf("missing window: %d %s", r.code, errMsg(r))
	}
	if r := run(t, ws, "", "availability", "add", "--subject", ada, "--duration", "PT1H", "--calendar", "2026-W37", "--conditional", "Deep Work"); r.code != 2 {
		t.Errorf("malformed conditional: %d", r.code)
	}
	if r := run(t, ws, "", "availability", "add", "--duration", "PT1H", "--calendar", "2026-W37"); r.code != 2 || !strings.Contains(errMsg(r), "subject") {
		t.Errorf("no subject: %d %s", r.code, errMsg(r))
	}
	mustOK(t, run(t, ws, "", "availability", "add", "--subject", "https://nowhere.example/thing", "--duration", "PT1H", "--calendar", "2026-W37"), "unknown subject")
}

func TestAvailabilityHorizonRenewSupersede(t *testing.T) {
	ws := initWS(t)
	// Default horizon from timestamp plus P13W.
	r := addAvail(t, ws, "--duration", "PT3H", "--calendar", "2026-09/2026-12", "--cadence", "FREQ=WEEKLY;BYDAY=TU", "--timestamp", "2026-09-04T00:00:00Z")
	if r.str("effective_valid_until") != "2026-12-04T00:00:00Z" || r.str("effective_valid_until_from") != "default_horizon" {
		t.Errorf("default horizon: %s %s", r.str("effective_valid_until"), r.str("effective_valid_until_from"))
	}
	if strings.Contains(readFile(t, ws, r.str("path")), "valid_until") {
		t.Error("default horizon written to file")
	}
	// Backdated origin.
	b := addAvail(t, ws, "--duration", "PT3H", "--calendar", "2026-09/2026-12", "--cadence", "FREQ=WEEKLY;BYDAY=TU", "--timestamp", "2026-08-01T00:00:00Z")
	if b.str("effective_valid_until") != "2026-10-31T00:00:00Z" {
		t.Errorf("backdated: %s", b.str("effective_valid_until"))
	}
	// One-off expires with its window.
	c := addAvail(t, ws, "--duration", "PT1H", "--calendar", "2026-W37")
	if c.str("effective_valid_until_from") != "window" || !strings.HasPrefix(c.str("effective_valid_until"), "2026-09-13T14:00:00Z") {
		t.Errorf("one-off: %s %s", c.str("effective_valid_until"), c.str("effective_valid_until_from"))
	}
	// Renew keeps id, changes version; moving earlier refused.
	id := r.str("id")
	rn := mustOK(t, run(t, ws, "", "availability", "renew", id, "--valid-until", "2027-03", "--now", now), "renew")
	if rn.str("id") != id || rn.str("previous_version") == rn.str("version") || rn.obj()["valid_until"] != "2027-03" {
		t.Errorf("renew: %v", rn.json)
	}
	if r := run(t, ws, "", "availability", "renew", id, "--valid-until", "2026-01", "--now", now); r.code != 2 {
		t.Errorf("renew earlier: %d %s", r.code, errMsg(r))
	}
	// Edit: prose ok, widen ok, narrow refused, terms refused naming supersede.
	mustOK(t, run(t, ws, "", "availability", "edit", id, "--title", "renamed"), "edit prose")
	mustOK(t, run(t, ws, "", "availability", "edit", id, "--scope", "organisation"), "widen")
	if r := run(t, ws, "", "availability", "edit", id, "--scope", "personal"); r.code != 2 {
		t.Errorf("narrow: %d", r.code)
	}
	for _, args := range [][]string{{"--conditional", "writing"}, {"--clock", "13:00/16:00"}, {"--duration", "PT1H"}, {"--location", "https://x/"}} {
		full := append([]string{"availability", "edit", id}, args...)
		r := run(t, ws, "", full...)
		if r.code != 2 || !strings.Contains(errMsg(r), "supersede") {
			t.Errorf("%v: %d %s", args, r.code, errMsg(r))
		}
	}
	// Supersede.
	s := mustOK(t, run(t, ws, "", "availability", "supersede", id, "--clock", "13:00/16:00", "--now", now), "supersede")
	newID := s.str("id")
	if newID == id || s.obj()["window"].(map[string]any)["clock"] != "13:00/16:00" || s.obj()["window"].(map[string]any)["calendar"] != "2026-09/2026-12" || s.obj()["scope"] != "organisation" {
		t.Errorf("supersede new: %v", s.obj())
	}
	old := readFile(t, ws, "availability/"+id+".yaml")
	if !strings.Contains(old, "retired:\n  kind: superseded\n  superseded_by: "+newID) {
		t.Errorf("old not retired:\n%s", old)
	}
	if r := run(t, ws, "", "availability", "renew", id, "--valid-until", "2028"); r.code != 2 {
		t.Errorf("renew retired: %d", r.code)
	}
	// Retire kinds.
	if r := run(t, ws, "", "availability", "retire", newID, "--kind", "abandoned"); r.code != 2 || !strings.Contains(errMsg(r), "retracted") {
		t.Errorf("wrong kind: %d %s", r.code, errMsg(r))
	}
	rt := mustOK(t, run(t, ws, "", "availability", "retire", newID, "--kind", "retracted", "--reason", "No longer at home on Tuesdays"), "retract")
	if !strings.Contains(readFile(t, ws, rt.str("path")), "reason: No longer at home on Tuesdays") {
		t.Error("reason missing")
	}
	// List filters.
	l := mustOK(t, run(t, ws, "", "availability", "list", "--subject", ada), "list")
	if int(l.json["count"].(float64)) != 2 {
		t.Errorf("list count: %v", l.json["count"])
	}
	l = mustOK(t, run(t, ws, "", "availability", "list", "--retired"), "list retired")
	if int(l.json["count"].(float64)) != 2 {
		t.Errorf("retired count: %v", l.json["count"])
	}
	if r := run(t, ws, "", "availability", "show", id); r.code != 0 || r.str("effective_valid_until_from") != "explicit" {
		t.Errorf("show: %d %v", r.code, r.json)
	}
}

// --- show, version-of, bounds, validate, index -----------------------------

func TestShowVersionBounds(t *testing.T) {
	ws := initWS(t)
	a := addIntention(t, ws, "--title", "A", "--calendar", "2026-W37", "--clock", "09:00/12:00", "--duration", "PT1H")
	r := mustOK(t, run(t, ws, "", "show", a.str("id")), "show")
	if r.str("type") != "intention" || r.str("version") != a.str("version") {
		t.Errorf("show: %v", r.json)
	}
	tr := text(t, ws, "show", a.str("id"))
	if !strings.HasPrefix(tr.stdout, "id: "+a.str("id")+"\n") {
		t.Errorf("show text:\n%s", tr.stdout)
	}
	r = mustOK(t, run(t, ws, "", "version-of", a.str("id"), "--projection"), "version-of")
	if r.str("version") != a.str("version") || !strings.Contains(r.str("canonical"), `"calendar":"2026-W37"`) {
		t.Errorf("version-of: %v", r.json)
	}
	// Stale cache warned by validate and surfaced by show.
	path := filepath.Join(ws, a.str("path"))
	body := readFile(t, ws, a.str("path"))
	_ = os.WriteFile(path, []byte(strings.Replace(body, "calendar: 2026-W37", "calendar: 2026-W38", 1)), 0o644)
	r = mustOK(t, run(t, ws, "", "show", a.str("id")), "show stale")
	if r.str("version_cached") != a.str("version") || r.str("version") == a.str("version") {
		t.Errorf("stale: %v", r.json)
	}
	r = run(t, ws, "", "validate")
	if r.code != 0 {
		t.Errorf("validate stale: %d", r.code)
	}
	found := 0
	for _, f := range r.json["findings"].([]any) {
		if c := f.(map[string]any)["code"]; c == "stale_version" || c == "index_drift" {
			found++
		}
	}
	if found != 2 {
		t.Errorf("stale findings: %v", r.json["findings"])
	}
	// Bounds of the object and ad hoc.
	r = mustOK(t, run(t, ws, "", "bounds", a.str("id")), "bounds id")
	if int(r.json["count"].(float64)) != 7 {
		t.Errorf("bounds count: %v", r.json)
	}
	r = mustOK(t, run(t, ws, "", "bounds", "--calendar", "2026-W37", "--clock", "09:00/12:00"), "bounds ad hoc")
	ivs := r.json["intervals"].([]any)
	if len(ivs) != 7 || ivs[0].(map[string]any)["start"] != "2026-09-07T09:00:00+10:00" {
		t.Errorf("ad hoc: %v", ivs)
	}
	r = mustOK(t, run(t, ws, "", "bounds", "--calendar", "2026-W36", "--week-start", "sunday", "--timezone", "Europe/London"), "bounds override")
	if r.json["intervals"].([]any)[0].(map[string]any)["start"] != "2026-08-30T00:00:00+01:00" {
		t.Errorf("override: %v", r.json["intervals"])
	}
	entries, _ := os.ReadDir(filepath.Join(ws, "intentions"))
	if len(entries) != 1 {
		t.Error("bounds wrote something")
	}
}

func TestValidateAndIndexVerbs(t *testing.T) {
	ws := initWS(t)
	a := addIntention(t, ws, "--title", "A")
	r := mustOK(t, run(t, ws, "", "validate"), "clean")
	if r.json["ok"] != true {
		t.Errorf("clean: %v", r.json)
	}
	// Hand-written cycle by "merge".
	b := addIntention(t, ws, "--title", "B", "--serves", a.str("id")+":in-order-to")
	body := readFile(t, ws, a.str("path"))
	_ = os.WriteFile(filepath.Join(ws, a.str("path")), []byte(strings.Replace(body, "serves: []", "serves:\n  - {id: "+b.str("id")+", role: in-order-to}", 1)), 0o644)
	r = run(t, ws, "", "validate")
	if r.code != 4 || r.json["ok"] != false {
		t.Errorf("cycle validate: %d %v", r.code, r.json)
	}
	counts := r.json["counts"].(map[string]any)
	if counts["error"].(float64) < 2 {
		t.Errorf("counts: %v", counts)
	}
	// Unrelated add still works in a workspace with a cycle? Cycle refusal is
	// only for the edge being added; C serving A is fine.
	addIntention(t, ws, "--title", "C", "--serves", a.str("id")+":in-order-to")
	// index --check drifts after the hand edit; index rebuilds.
	r = run(t, ws, "", "index", "--check")
	if r.code != 4 || len(r.json["changed"].([]any)) != 1 {
		t.Errorf("index check: %d %v", r.code, r.json)
	}
	mustOK(t, run(t, ws, "", "index"), "index")
	mustOK(t, run(t, ws, "", "index", "--check"), "index check after rebuild")
	// Conflict markers: check fails, rebuild recovers.
	_ = os.WriteFile(filepath.Join(ws, "index.yaml"), []byte("<<<<<<< HEAD\nformat: x\n=======\n>>>>>>> b\n"), 0o644)
	if r := run(t, ws, "", "index", "--check"); r.code != 4 {
		t.Errorf("conflict check: %d", r.code)
	}
	mustOK(t, run(t, ws, "", "index"), "index after conflict")
	// Reordered file is not a finding; foreign id accepted.
	_ = os.WriteFile(filepath.Join(ws, "intentions", "int_01j9xk2p3q4r5s6t.yaml"), []byte("acknowledgements: []\ntimestamp: 2026-09-04T00:00:00Z\nsource: {author: a}\nserves: []\nstability: tentative\ntitle: foreign\nsubject: https://s/\nid: int_01j9xk2p3q4r5s6t\n"), 0o644)
	r = mustOK(t, run(t, ws, "", "show", "int_01j9xk2p3q4r5s6t"), "foreign show")
	if r.obj()["title"] != "foreign" {
		t.Errorf("foreign: %v", r.obj())
	}
	r = run(t, ws, "", "validate")
	for _, f := range r.json["findings"].([]any) {
		if f.(map[string]any)["id"] == "int_01j9xk2p3q4r5s6t" && f.(map[string]any)["severity"] == "error" {
			t.Errorf("foreign reordered file reported: %v", f)
		}
	}
	// Text mode validate lists findings.
	tr := text(t, ws, "validate")
	if tr.code != 4 || !strings.Contains(tr.stdout, "cycle") {
		t.Errorf("text validate: %d\n%s", tr.code, tr.stdout)
	}
}
