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
	for _, want := range []string{"format: intentions/0.1", "hash: sha256", "timezone: Australia/Melbourne", "default_horizon: P13W", "horizon: P4W", "subject: " + ada, "author: " + ada} {
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
	if strings.Contains(cfg, "week_start") {
		t.Error("init wrote week_start")
	}
	if r := run(t, "", "", "init", filepath.Join(t.TempDir(), "y"), "--author", ada, "--week-start", "sunday"); r.code != 2 {
		t.Errorf("--week-start still accepted: %d", r.code)
	}
	south := filepath.Join(t.TempDir(), "south")
	mustOK(t, run(t, "", "", "init", south, "--author", ada, "--hemisphere", "south"), "south init")
	if !strings.Contains(readFile(t, south, "intentions.yaml"), "hemisphere: south") {
		t.Error("hemisphere not written")
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
	want := "id: " + full.str("id") + "\nversion: " + full.str("version") + `
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
	for _, line := range []string{"source:\n  author: https://example.com/people/ada\n  harness: claude\n  model: claude-fable-5-1\ntimestamp: 2026-09-04T09:12:00Z\nacknowledgements: []\n"} {
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
	// Harness under policy: firmed_under written after stability, version moves only for the stability edit.
	r = mustOK(t, run(t, ws, "", "intention", "firm", b.str("id"), "--harness", "claude", "--policy", p.str("id")), "harness under policy")
	if r.str("policy") != p.str("id") || r.obj()["stability"] != "firm" || r.obj()["firmed_under"] != p.str("id") {
		t.Errorf("policy result: %v", r.json)
	}
	body := readFile(t, ws, b.str("path"))
	if !strings.Contains(body, "stability: firm\nfirmed_under: "+p.str("id")+"\n") {
		t.Errorf("firmed_under placement:\n%s", body)
	}
	if !strings.HasPrefix(body, "id: "+b.str("id")+"\nversion: sha256:") {
		t.Errorf("version not second:\n%s", body)
	}
	// A person's own firming leaves firmed_under absent, clearing a previous one.
	mustOK(t, run(t, ws, "", "intention", "edit", b.str("id"), "--stability", "tentative"), "unfirm")
	if strings.Contains(readFile(t, ws, b.str("path")), "firmed_under") {
		t.Error("firmed_under survived unfirming")
	}
	r = mustOK(t, run(t, ws, "", "intention", "firm", b.str("id")), "person re-firm")
	if _, has := r.obj()["firmed_under"]; has {
		t.Error("person firming wrote firmed_under")
	}
	// A scheduled intention cannot be a policy.
	sched := addIntention(t, ws, "--title", "Scheduled", "--calendar", "2026-W37")
	d := addIntention(t, ws, "--title", "D2", "--duration", "PT10M")
	r = run(t, ws, "", "intention", "firm", d.str("id"), "--harness", "claude", "--policy", sched.str("id"))
	if r.code != 2 || !strings.Contains(errMsg(r), "terminus") {
		t.Errorf("scheduled policy: %d %s", r.code, errMsg(r))
	}
	// Conditions on termini only; a cadence needs a calendar anchor.
	if r := run(t, ws, "", "intention", "add", "--title", "X", "--calendar", "2026-W37", "--auto-select", "max_duration=PT30M"); r.code != 2 || !strings.Contains(errMsg(r), "termini") {
		t.Errorf("condition on scheduled: %d %s", r.code, errMsg(r))
	}
	if r := run(t, ws, "", "intention", "add", "--title", "X", "--cadence", "FREQ=WEEKLY;BYDAY=TU", "--calendar", "2026-09/..", "--auto-firm", "max_duration=PT30M"); r.code != 2 {
		t.Errorf("condition on recurring: %d", r.code)
	}
	if r := run(t, ws, "", "intention", "add", "--title", "X", "--cadence", "FREQ=WEEKLY;BYDAY=TU", "--clock", "09:00/12:00"); r.code != 2 || !strings.Contains(errMsg(r), "calendar anchor") {
		t.Errorf("cadence without anchor: %d %s", r.code, errMsg(r))
	}
	if r := run(t, ws, "", "availability", "add", "--subject", ada, "--duration", "PT3H", "--clock", "09:00/12:00", "--cadence", "FREQ=WEEKLY;BYDAY=TU"); r.code != 2 || !strings.Contains(errMsg(r), "calendar anchor") {
		t.Errorf("availability cadence without anchor: %d %s", r.code, errMsg(r))
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
	if !strings.Contains(body, "retired:\n  kind: superseded\n  reason: split\n  superseded_by: "+c.str("id")+"\n  source:\n    author: "+ada+"\n  timestamp: 2026-09-04T09:00:00Z\n") || !strings.HasPrefix(body, "id: "+a.str("id")+"\nversion: "+r.str("version")+"\n") {
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
	r = mustOK(t, run(t, ws, "", "intention", "list", "--recurring"), "recurring")
	if int(r.json["count"].(float64)) != 1 {
		t.Errorf("recurring count: %v", r.json["count"])
	}
	if r := run(t, ws, "", "intention", "list", "--standing"); r.code != 2 {
		t.Errorf("--standing still accepted: %d", r.code)
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
	want := "id: " + r.str("id") + "\nversion: " + r.str("version") + `
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
`
	if body != want {
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
	r = mustOK(t, run(t, ws, "", "bounds", "--calendar", "2026-W36", "--timezone", "Europe/London"), "bounds override")
	if r.json["intervals"].([]any)[0].(map[string]any)["start"] != "2026-08-31T00:00:00+01:00" {
		t.Errorf("override: %v", r.json["intervals"])
	}
	if r := run(t, ws, "", "bounds", "--calendar", "2026-W36", "--week-start", "sunday"); r.code != 2 {
		t.Errorf("--week-start still accepted: %d", r.code)
	}
	// Explicit season codes ignore the workspace hemisphere.
	r = mustOK(t, run(t, ws, "", "bounds", "--calendar", "2026-29"), "southern spring")
	if r.json["intervals"].([]any)[0].(map[string]any)["start"] != "2026-09-01T00:00:00+10:00" {
		t.Errorf("explicit southern spring: %v", r.json["intervals"])
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

// TestLegacyWorkspace reads a v0.1.0-shaped workspace: version last, an
// explicit transparent: false, week_start in the configuration. It must load,
// validate with only the expected findings, and be re-emitted canonically on
// edit.
func TestLegacyWorkspace(t *testing.T) {
	ws := initWS(t)
	cfg := readFile(t, ws, "intentions.yaml")
	_ = os.WriteFile(filepath.Join(ws, "intentions.yaml"), []byte(strings.Replace(cfg, "resolver:\n", "resolver:\n  week_start: monday\n", 1)), 0o644)
	legacy := `id: int_01a06d10-4c2e-7a91-b3f0-2d8e1a7c5b44
subject: https://example.com/people/ada
title: Legacy
window:
  calendar: 2026-W37
stability: tentative
serves: []
source:
  author: https://example.com/people/ada
timestamp: 2026-09-04T09:12:00Z
acknowledgements: []
version: sha256:0000000000000000000000000000000000000000000000000000000000000000
`
	_ = os.WriteFile(filepath.Join(ws, "intentions", "int_01a06d10-4c2e-7a91-b3f0-2d8e1a7c5b44.yaml"), []byte(legacy), 0o644)
	cmt := `id: cmt_01a06d15-3f8a-7d61-8c2b-9a4e6f1d3b05
parties:
  - uri: https://example.com/people/ada
    status: accepted
placement:
  start: 2026-09-15T10:00:00+10:00
  duration: PT1H
origin: import
transparent: false
source:
  author: https://example.com/people/ada
timestamp: 2026-09-08T14:02:00Z
acknowledgements: []
`
	_ = os.WriteFile(filepath.Join(ws, "commitments", "cmt_01a06d15-3f8a-7d61-8c2b-9a4e6f1d3b05.yaml"), []byte(cmt), 0o644)
	mustOK(t, run(t, ws, "", "index"), "index")
	r := mustOK(t, run(t, ws, "", "validate"), "legacy validates")
	got := map[string]int{}
	for _, f := range r.json["findings"].([]any) {
		got[f.(map[string]any)["code"].(string)]++
	}
	if got["stale_version"] != 1 || got["missing_version"] != 1 || got["resolver_unknown_key"] != 1 || r.json["ok"] != true {
		t.Errorf("legacy findings: %v", got)
	}
	// Bounds still work with the stale key present, and weeks are ISO.
	b := mustOK(t, run(t, ws, "", "bounds", "int_01a06d10-4c2e-7a91-b3f0-2d8e1a7c5b44"), "bounds")
	if b.json["intervals"].([]any)[0].(map[string]any)["start"] != "2026-09-07T00:00:00+10:00" {
		t.Errorf("legacy bounds: %v", b.json["intervals"])
	}
	// An edit re-emits canonically: version second, correct.
	e := mustOK(t, run(t, ws, "", "intention", "edit", "int_01a06d10-4c2e-7a91-b3f0-2d8e1a7c5b44", "--description", "touched"), "edit legacy")
	body := readFile(t, ws, e.str("path"))
	if !strings.HasPrefix(body, "id: int_01a06d10-4c2e-7a91-b3f0-2d8e1a7c5b44\nversion: "+e.str("version")+"\nsubject:") {
		t.Errorf("legacy re-emit:\n%s", body)
	}
	if strings.Count(body, "version:") != 1 {
		t.Errorf("version written twice:\n%s", body)
	}
	// The commitment shown by the generic verb re-emits without transparent.
	s := mustOK(t, run(t, ws, "", "show", "cmt_01a06d15-3f8a-7d61-8c2b-9a4e6f1d3b05"), "show legacy commitment")
	if _, has := s.obj()["transparent"]; has {
		t.Error("transparent: false re-emitted")
	}
}

// --- skill ---------------------------------------------------------------

func TestSkillShowAndInstall(t *testing.T) {
	work := t.TempDir()
	cwd, _ := os.Getwd()
	_ = os.Chdir(work)
	defer func() { _ = os.Chdir(cwd) }()
	t.Setenv("HOME", filepath.Join(work, "home"))
	t.Setenv("USERPROFILE", filepath.Join(work, "home")) // os.UserHomeDir on Windows

	// show needs no workspace and renders the embedded skill.
	show := text(t, "", "skill", "show")
	if show.code != 0 || !strings.HasPrefix(show.stdout, "---\n") || !strings.Contains(show.stdout, "name: intentions") {
		t.Fatalf("show: %d %q", show.code, show.stdout[:60])
	}
	sj := mustOK(t, run(t, "", "", "skill", "show"), "show json")
	if sj.str("content") != show.stdout || sj.str("version") != "dev" {
		t.Errorf("show json: %v", sj.json["version"])
	}
	// Fresh install, idempotent reinstall, check.
	r := mustOK(t, run(t, "", "", "skill", "install"), "install")
	target := filepath.Join(work, ".claude", "skills", "intentions", "SKILL.md")
	gotPath, _ := filepath.EvalSymlinks(r.str("path"))
	wantPath, _ := filepath.EvalSymlinks(target)
	if r.json["created"] != true || gotPath != wantPath {
		t.Errorf("install: %v", r.json)
	}
	if b, _ := os.ReadFile(target); string(b) != show.stdout {
		t.Error("installed content differs from show")
	}
	if r := run(t, "", "", "skill", "install"); r.code != 0 || r.json["unchanged"] != true {
		t.Errorf("reinstall: %v", r.json)
	}
	if r := run(t, "", "", "skill", "install", "--check"); r.code != 0 || r.str("status") != "ok" {
		t.Errorf("check: %d %v", r.code, r.json)
	}
	// A foreign file is protected; --force replaces it; a drifted own file is updated.
	_ = os.WriteFile(target, []byte("# mine\n"), 0o644)
	if r := run(t, "", "", "skill", "install"); r.code != 1 || !strings.Contains(errMsg(r), "--force") {
		t.Errorf("foreign: %d %s", r.code, errMsg(r))
	}
	if r := run(t, "", "", "skill", "install", "--check"); r.code != 4 || r.str("status") != "foreign" {
		t.Errorf("check foreign: %d %v", r.code, r.json)
	}
	if r := run(t, "", "", "skill", "install", "--force"); r.code != 0 || r.json["updated"] != true {
		t.Errorf("force: %v", r.json)
	}
	b, _ := os.ReadFile(target)
	_ = os.WriteFile(target, []byte(strings.Replace(string(b), "## The loop", "## The loop!", 1)), 0o644)
	if r := run(t, "", "", "skill", "install", "--check"); r.code != 4 || r.str("status") != "differs" {
		t.Errorf("check differs: %d %v", r.code, r.json)
	}
	if r := run(t, "", "", "skill", "install"); r.code != 0 || r.json["updated"] != true {
		t.Errorf("update own: %v", r.json)
	}
	// Presets: copilot, agents (user), cursor, agents-md, several at once, --dir.
	mustOK(t, run(t, "", "", "skill", "install", "--harness", "copilot"), "copilot")
	if _, err := os.Stat(filepath.Join(work, ".github", "skills", "intentions", "SKILL.md")); err != nil {
		t.Error("copilot target missing")
	}
	mustOK(t, run(t, "", "", "skill", "install", "--harness", "agents", "--user"), "agents user")
	if _, err := os.Stat(filepath.Join(work, "home", ".agents", "skills", "intentions", "SKILL.md")); err != nil {
		t.Error("agents user target missing")
	}
	mustOK(t, run(t, "", "", "skill", "install", "--harness", "cursor"), "cursor")
	if c, _ := os.ReadFile(filepath.Join(work, ".cursor", "rules", "intentions.mdc")); !strings.Contains(string(c), "alwaysApply: false") {
		t.Error("cursor rule shape")
	}
	_ = os.WriteFile(filepath.Join(work, "AGENTS.md"), []byte("# Project\n\nBuild with make.\n"), 0o644)
	mustOK(t, run(t, "", "", "skill", "install", "--harness", "agents-md"), "agents-md")
	am, _ := os.ReadFile(filepath.Join(work, "AGENTS.md"))
	if !strings.HasPrefix(string(am), "# Project\n\nBuild with make.\n\n<!-- intentions:skill:start") || !strings.HasSuffix(string(am), "<!-- intentions:skill:end -->\n") {
		t.Errorf("agents-md splice:\n%s", am[:200])
	}
	multi := mustOK(t, run(t, "", "", "skill", "install", "--harness", "claude", "--harness", "cursor"), "multi")
	if len(multi.json["targets"].([]any)) != 2 {
		t.Errorf("multi targets: %v", multi.json["targets"])
	}
	mustOK(t, run(t, "", "", "skill", "install", "--dir", filepath.Join(work, "tmp", "skills")), "dir")
	if _, err := os.Stat(filepath.Join(work, "tmp", "skills", "SKILL.md")); err != nil {
		t.Error("dir target missing")
	}
	// Conflicting flags.
	for _, args := range [][]string{{"--user", "--dir", "x"}, {"--harness", "cursor", "--user"}, {"--file", "AGENTS.md"}, {"--harness", "nope"}} {
		full := append([]string{"skill", "install"}, args...)
		if r := run(t, "", "", full...); r.code != 2 {
			t.Errorf("%v: exit %d", args, r.code)
		}
	}
	// Copilot duplicate warning: three project locations now hold a skill.
	w := mustOK(t, run(t, "", "", "skill", "install", "--harness", "agents"), "agents project")
	if ws, _ := w.json["warnings"].([]any); len(ws) == 0 {
		t.Error("expected a duplicate-location warning")
	}
}

// --- resolution loop ---------------------------------------------------------

const rnow = "2026-09-10T00:00:00Z"

func TestResolutionLoop(t *testing.T) {
	ws := initWS(t)
	av := func(args ...string) result {
		full := append([]string{"availability", "add", "--now", rnow, "--subject", ada}, args...)
		return mustOK(t, run(t, ws, "", full...), "availability add")
	}
	add := func(args ...string) result {
		full := append([]string{"intention", "add", "--now", rnow}, args...)
		return mustOK(t, run(t, ws, "", full...), "intention add")
	}
	av("--title", "Tuesday mornings", "--duration", "PT3H", "--calendar", "2026-09/2026-12", "--clock", "09:00/12:00", "--conditional", "deep-work", "--cadence", "FREQ=WEEKLY;BYDAY=TU")
	av("--title", "Weekday afternoons", "--duration", "PT4H", "--calendar", "2026-09/2026-12", "--clock", "13:00/17:00", "--conditional", "meeting", "--cadence", "FREQ=WEEKLY;BYDAY=MO,TU,WE,TH,FR")
	mustOK(t, run(t, ws, "", "availability", "add", "--now", rnow, "--subject", "https://example.com/rooms/3", "--duration", "PT8H", "--calendar", "2026-09/2026-12", "--clock", "08:00/18:00", "--cadence", "FREQ=WEEKLY;BYDAY=MO,TU,WE,TH,FR"), "room")

	// Resolve: rank-1 candidates inside Tuesday morning, 15-minute grid, limit keeps the count.
	a := add("--title", "Draft the budget narrative", "--duration", "PT90M", "--calendar", "2026-W38", "--activity", "deep-work")
	r := mustOK(t, run(t, ws, "", "resolve", a.str("id"), "--limit", "4", "--now", rnow), "resolve")
	cands := r.json["candidates"].([]any)
	if len(cands) != 4 || int(r.json["candidates_considered"].(float64)) != 7 || cands[0].(map[string]any)["start"] != "2026-09-15T09:00:00+10:00" || cands[0].(map[string]any)["rank"].(float64) != 1 {
		t.Fatalf("resolve: %v", r.json)
	}
	if fi, _ := os.ReadDir(filepath.Join(ws, "resolutions")); len(fi) != 0 {
		t.Error("resolve wrote a record")
	}
	// Refusals: no duration; placed without --replace later.
	noDur := add("--title", "No duration", "--calendar", "2026-W38")
	if r := run(t, ws, "", "resolve", noDur.str("id"), "--now", rnow); r.code != 2 || !strings.Contains(errMsg(r), "duration") {
		t.Errorf("no duration: %d %s", r.code, errMsg(r))
	}
	// Select by a person.
	s := mustOK(t, run(t, ws, "", "select", a.str("id"), "--candidate", "1", "--now", rnow), "select")
	rec := s.json["resolution"].(map[string]any)
	if rec["selector"] != "person" || rec["intention"] != a.str("id") || rec["candidates_considered"].(float64) != 7 || len(rec["supply"].([]any)) != 1 {
		t.Errorf("record: %v", rec)
	}
	if s.json["intention"].(map[string]any)["placement"].(map[string]any)["start"] != "2026-09-15T09:00:00+10:00" {
		t.Errorf("placement: %v", s.json["intention"])
	}
	body := readFile(t, ws, "resolutions/"+rec["id"].(string)+".yaml")
	if !strings.HasPrefix(body, "id: res_") || !strings.Contains(body, "\ntimestamp: 2026-09-10T00:00:00Z\nsupply:\n  - avl_") {
		t.Errorf("record file:\n%s", body)
	}
	if r := run(t, ws, "", "select", a.str("id"), "--candidate", "2", "--now", rnow); r.code != 2 || !strings.Contains(errMsg(r), "--replace") {
		t.Errorf("placed without replace: %d %s", r.code, errMsg(r))
	}
	// Capacity: the 3h morning holds 90m + 1h; a second 90m no longer fits there.
	b := add("--title", "Read the board pack", "--duration", "PT1H", "--calendar", "2026-W38", "--activity", "deep-work")
	sb := mustOK(t, run(t, ws, "", "select", b.str("id"), "--candidate", "1", "--now", rnow), "select b")
	if sb.json["candidate"].(map[string]any)["rank"].(float64) != 1 || sb.json["candidate"].(map[string]any)["start"] != "2026-09-15T10:30:00+10:00" {
		t.Errorf("capacity-aware placement: %v", sb.json["candidate"])
	}
	c := add("--title", "One more", "--duration", "PT90M", "--calendar", "2026-09-15", "--activity", "deep-work")
	rc := mustOK(t, run(t, ws, "", "resolve", c.str("id"), "--now", rnow), "resolve c")
	if len(rc.json["candidates"].([]any)) != 0 || !strings.Contains(rc.str("reason"), "capacity") {
		t.Errorf("capacity exhausted: %v", rc.json)
	}
	// Parties create a commitment with everyone tentative.
	m := add("--title", "Review in room 3", "--duration", "PT1H", "--calendar", "2026-W38", "--activity", "meeting", "--party", "https://example.com/rooms/3")
	sm := mustOK(t, run(t, ws, "", "select", m.str("id"), "--candidate", "1", "--now", rnow), "select meeting")
	cmt := sm.json["commitment"].(map[string]any)
	parties := cmt["parties"].([]any)
	if len(parties) != 2 || parties[0].(map[string]any)["status"] != "tentative" || cmt["origin"].(map[string]any)["resolution"] != sm.json["resolution"].(map[string]any)["id"] || cmt["intention"] != m.str("id") {
		t.Errorf("commitment: %v", cmt)
	}
	if len(sm.json["flags"].([]any)) != 0 {
		t.Errorf("fresh selection flagged: %v", sm.json["flags"])
	}
	// Displacement: a 2h intention in a 2h window over the tentative commitment.
	cmtStart := cmt["placement"].(map[string]any)["start"].(string) // 2026-09-14T13:00:00+10:00
	day, hour := cmtStart[:10], cmtStart[11:16]
	d := add("--title", "Displacer", "--duration", "PT2H", "--calendar", day, "--clock", hour+"/15:00", "--activity", "meeting")
	rd := mustOK(t, run(t, ws, "", "resolve", d.str("id"), "--now", rnow), "resolve displacer")
	top := rd.json["candidates"].([]any)[0].(map[string]any)
	if top["rank"].(float64) != 2 || top["displaces"].([]any)[0] != cmt["id"] {
		t.Errorf("displacing candidate: %v", top)
	}
	sd := mustOK(t, run(t, ws, "", "select", d.str("id"), "--candidate", "1", "--now", rnow), "select displacer")
	if sd.json["resolution"].(map[string]any)["displaced"].([]any)[0] != cmt["id"] || len(sd.json["flags"].([]any)) != 2 {
		t.Errorf("displacement recorded and flagged: %v %v", sd.json["resolution"], sd.json["flags"])
	}
	after := mustOK(t, run(t, ws, "", "show", cmt["id"].(string)), "show displaced")
	if after.obj()["placement"].(map[string]any)["start"] != cmtStart || after.obj()["parties"].([]any)[0].(map[string]any)["status"] != "tentative" {
		t.Error("displaced commitment was changed")
	}
	// Check on demand, scoped and with --fail-on-flags.
	ck := mustOK(t, run(t, ws, "", "check", "--now", rnow), "check")
	if int(ck.json["count"].(float64)) != 2 {
		t.Errorf("check count: %v", ck.json)
	}
	if r := run(t, ws, "", "check", d.str("id"), "--fail-on-flags", "--now", rnow); r.code != 4 || int(r.json["count"].(float64)) != 2 {
		t.Errorf("scoped check: %d %v", r.code, r.json)
	}
	// Acknowledge on the displacer suppresses its flag; the counterpart keeps its own.
	if r := run(t, ws, "", "acknowledge", d.str("id"), "--kind", "overlap"); r.code != 2 {
		t.Errorf("unknown kind: %d", r.code)
	}
	if r := run(t, ws, "", "acknowledge", d.str("id"), "--kind", "window-clash", "--now", rnow); r.code != 2 || !strings.Contains(errMsg(r), "counterpart") {
		t.Errorf("missing counterpart: %d %s", r.code, errMsg(r))
	}
	dv := mustOK(t, run(t, ws, "", "version-of", d.str("id")), "version-of d")
	ack := mustOK(t, run(t, ws, "", "acknowledge", d.str("id"), "--kind", "window-clash", "--counterpart", cmt["id"].(string), "--reason", "The review can move", "--now", rnow), "acknowledge")
	entry := ack.json["acknowledgement"].(map[string]any)
	cv := mustOK(t, run(t, ws, "", "version-of", cmt["id"].(string)), "version-of")
	if entry["counterpart_version"] != cv.str("version") || ack.str("version") != dv.str("version") || len(ack.json["flags"].([]any)) != 0 {
		t.Errorf("acknowledgement: %v flags %v", entry, ack.json["flags"])
	}
	ck = mustOK(t, run(t, ws, "", "check", "--now", rnow), "check after ack")
	if int(ck.json["count"].(float64)) != 1 || ck.json["flags"].([]any)[0].(map[string]any)["subject"] != cmt["id"] {
		t.Errorf("suppression: %v", ck.json)
	}
	// A prose edit on the counterpart's intention keeps the suppression; show carries flags.
	mustOK(t, run(t, ws, "", "intention", "edit", m.str("id"), "--description", "prose", "--now", rnow), "prose edit")
	if r := mustOK(t, run(t, ws, "", "intention", "show", d.str("id"), "--now", rnow), "show"); len(r.json["flags"].([]any)) != 0 {
		t.Errorf("prose edit lapsed: %v", r.json["flags"])
	}
	// Replace: a new record, the old placement reported.
	rep := mustOK(t, run(t, ws, "", "select", a.str("id"), "--candidate", "2", "--replace", "--now", rnow), "replace")
	if rep.json["replaced"].(map[string]any)["start"] != "2026-09-15T09:00:00+10:00" {
		t.Errorf("replace: %v", rep.json["replaced"])
	}
	if fi, _ := os.ReadDir(filepath.Join(ws, "resolutions")); len(fi) != 5 {
		t.Errorf("records: %d", len(fi))
	}
	// Policy selection by a harness; refused for a person and for a non-covering policy.
	p := add("--title", "Small things", "--auto-select", "max_duration=PT30M")
	q := add("--title", "Quick", "--duration", "PT15M", "--calendar", "2026-W39", "--activity", "meeting")
	if r := run(t, ws, "", "select", q.str("id"), "--policy", p.str("id"), "--now", rnow); r.code != 2 {
		t.Errorf("person with policy: %d", r.code)
	}
	sq := mustOK(t, run(t, ws, "", "select", q.str("id"), "--policy", p.str("id"), "--harness", "claude", "--now", rnow), "policy select")
	if sq.json["resolution"].(map[string]any)["selector"] != p.str("id") || sq.json["resolution"].(map[string]any)["source"].(map[string]any)["harness"] != "claude" {
		t.Errorf("policy record: %v", sq.json["resolution"])
	}
	long := add("--title", "Long", "--duration", "PT2H", "--calendar", "2026-W39", "--activity", "meeting")
	if r := run(t, ws, "", "select", long.str("id"), "--policy", p.str("id"), "--harness", "claude", "--now", rnow); r.code != 2 || !strings.Contains(errMsg(r), "max_duration") {
		t.Errorf("policy not covering: %d %s", r.code, errMsg(r))
	}
	// Generate: instances, idempotent, resolve generates first.
	rec2 := add("--title", "Weekly 1:1", "--duration", "PT30M", "--calendar", "2026-09/2026-12", "--clock", "14:00/15:00", "--cadence", "FREQ=WEEKLY;BYDAY=TU")
	g := mustOK(t, run(t, ws, "", "generate", "--horizon", "P2W", "--now", rnow), "generate")
	if int(g.json["count"].(float64)) != 2 || g.json["created"].([]any)[0].(map[string]any)["occurrence"] != "2026-09-15" {
		t.Errorf("generate: %v", g.json)
	}
	g2 := mustOK(t, run(t, ws, "", "generate", "--horizon", "P2W", "--now", rnow), "generate again")
	if int(g2.json["count"].(float64)) != 0 || len(g2.json["skipped"].([]any)) != 2 {
		t.Errorf("idempotent: %v", g2.json)
	}
	li := mustOK(t, run(t, ws, "", "intention", "list", "--instances-of", rec2.str("id")), "instances")
	if int(li.json["count"].(float64)) != 2 {
		t.Errorf("instances: %v", li.json["count"])
	}
	later := add("--title", "Later", "--duration", "PT1H", "--calendar", "2026-W40", "--activity", "deep-work")
	rl := mustOK(t, run(t, ws, "", "resolve", later.str("id"), "--now", rnow), "resolve generates")
	if len(rl.json["generated"].([]any)) != 1 {
		t.Errorf("resolve generated: %v", rl.json["generated"])
	}
	// Placed and unplaced filters; validate stays clean; index matches.
	pl := mustOK(t, run(t, ws, "", "intention", "list", "--placed"), "placed")
	up := mustOK(t, run(t, ws, "", "intention", "list", "--unplaced"), "unplaced")
	if int(pl.json["count"].(float64)) != 5 || int(up.json["count"].(float64)) < 5 {
		t.Errorf("placed %v unplaced %v", pl.json["count"], up.json["count"])
	}
	mustOK(t, run(t, ws, "", "validate"), "validate")
	mustOK(t, run(t, ws, "", "index", "--check"), "index check")
}

func TestValidationForResolution(t *testing.T) {
	ws := initWS(t)
	rec := addIntention(t, ws, "--title", "Weekly", "--duration", "PT30M", "--calendar", "2026-09/2026-09-20", "--cadence", "FREQ=WEEKLY;BYDAY=TU")
	// An instance outside the recurring window, a placement outside its window, a selector that is not a policy.
	_ = os.WriteFile(filepath.Join(ws, "intentions", "int_inst.yaml"), []byte("id: int_inst\nsubject: "+ada+"\ntitle: t\nstability: tentative\nserves:\n  - {id: "+rec.str("id")+", role: instance-of}\noccurrence: 2027-03-02\nwindow: {calendar: 2027-03-02}\nsource: {author: a}\ntimestamp: 2026-09-04T00:00:00Z\nacknowledgements: []\n"), 0o644)
	_ = os.WriteFile(filepath.Join(ws, "intentions", "int_out.yaml"), []byte("id: int_out\nsubject: "+ada+"\ntitle: t\nstability: tentative\nserves: []\nwindow: {calendar: 2026-W37}\nplacement: {start: 2026-09-21T10:00:00+10:00, duration: PT1H}\nsource: {author: a}\ntimestamp: 2026-09-04T00:00:00Z\nacknowledgements: []\n"), 0o644)
	_ = os.WriteFile(filepath.Join(ws, "resolutions", "res_bad.yaml"), []byte("id: res_bad\nintention: int_out\nplacement: {start: 2026-09-21T10:00:00+10:00, duration: PT1H}\nselector: "+rec.str("id")+"\nsource: {author: a}\ntimestamp: 2026-09-04T00:00:00Z\n"), 0o644)
	r := run(t, ws, "", "validate")
	got := map[string]bool{}
	for _, f := range r.json["findings"].([]any) {
		got[f.(map[string]any)["code"].(string)] = true
	}
	for _, code := range []string{"occurrence_outside_window", "placement_outside_window", "selector_not_policy"} {
		if !got[code] {
			t.Errorf("missing %s: %v", code, got)
		}
	}
}

func TestUnresolved(t *testing.T) {
	ws := initWS(t)
	mustOK(t, run(t, ws, "", "availability", "add", "--now", rnow, "--subject", ada, "--duration", "PT3H", "--calendar", "2026-09/2026-12", "--clock", "09:00/12:00", "--conditional", "deep-work", "--cadence", "FREQ=WEEKLY;BYDAY=TU"), "availability")
	add := func(args ...string) result {
		full := append([]string{"intention", "add", "--now", rnow}, args...)
		return mustOK(t, run(t, ws, "", full...), "intention add")
	}
	a := add("--title", "Ready", "--duration", "PT90M", "--calendar", "2026-W38", "--activity", "deep-work")
	b := add("--title", "Blocked", "--duration", "PT1H", "--relative", a.str("id")+":FINISHTOSTART", "--activity", "deep-work")
	c := add("--title", "Incomplete", "--calendar", "2026-W38")
	past := add("--title", "Passed", "--duration", "PT1H", "--calendar", "2026-W30", "--activity", "deep-work")
	add("--title", "Terminus")
	retired := add("--title", "Retired", "--duration", "PT1H", "--calendar", "2026-W38")
	mustOK(t, run(t, ws, "", "intention", "retire", retired.str("id"), "--kind", "abandoned", "--now", rnow), "retire")
	rec := add("--title", "Weekly", "--duration", "PT1H", "--calendar", "2026-09/2026-12", "--cadence", "FREQ=WEEKLY;BYDAY=TU", "--activity", "deep-work")
	mustOK(t, run(t, ws, "", "generate", "--horizon", "P2W", "--now", rnow), "generate")
	other := mustOK(t, run(t, ws, "", "intention", "add", "--now", rnow, "--title", "Someone else", "--subject", "https://example.com/people/bob", "--duration", "PT1H", "--calendar", "2026-W38"), "other")

	r := mustOK(t, run(t, ws, "", "unresolved", "--now", rnow), "unresolved")
	entries := r.json["entries"].([]any)
	status := map[string]map[string]any{}
	for _, e := range entries {
		m := e.(map[string]any)
		status[m["id"].(string)] = m
	}
	if len(entries) != 7 || int(r.json["count"].(float64)) != 7 {
		t.Fatalf("entries: %d %v", len(entries), r.json["counts"])
	}
	if e := status[a.str("id")]; e["status"] != "ready" || e["candidates"].(float64) != 7 || e["best_rank"].(float64) != 1 || e["deadline"] == nil {
		t.Errorf("ready: %v", e)
	}
	if e := status[b.str("id")]; e["status"] != "blocked" || e["blocked_on"] != a.str("id") {
		t.Errorf("blocked: %v", e)
	}
	if e := status[c.str("id")]; e["status"] != "incomplete" || !strings.Contains(e["reason"].(string), "duration") {
		t.Errorf("incomplete: %v", e)
	}
	if e := status[past.str("id")]; e["status"] != "no_candidates" || !strings.Contains(e["reason"].(string), "before now") {
		t.Errorf("passed: %v", e)
	}
	if e := status[other.str("id")]; e["status"] != "no_candidates" || !strings.Contains(e["reason"].(string), "no eligible supply") {
		t.Errorf("no supply: %v", e)
	}
	if _, ok := status[rec.str("id")]; ok || status[retired.str("id")] != nil {
		t.Error("recurring parent or retired listed")
	}
	instances := 0
	for id := range status {
		if status[id]["title"] == "Weekly" {
			instances++
		}
	}
	if instances != 2 {
		t.Errorf("instances listed: %d", instances)
	}
	// Order: the Tuesday instances (deadlines on their days) before W38's end; incomplete or passed last.
	first, last := entries[0].(map[string]any), entries[len(entries)-1].(map[string]any)
	if first["title"] != "Weekly" || last["status"] != "incomplete" && last["status"] != "no_candidates" {
		t.Errorf("order: first %v last %v", first["title"], last["status"])
	}
	counts := r.json["counts"].(map[string]any)
	if counts["ready"].(float64) != 3 || counts["blocked"].(float64) != 1 || counts["incomplete"].(float64) != 1 || counts["no_candidates"].(float64) != 2 {
		t.Errorf("counts: %v", counts)
	}
	// Subject filter, and a placed intention drops out.
	if r := mustOK(t, run(t, ws, "", "unresolved", "--subject", "https://example.com/people/bob", "--now", rnow), "subject"); int(r.json["count"].(float64)) != 1 {
		t.Errorf("subject filter: %v", r.json)
	}
	mustOK(t, run(t, ws, "", "select", a.str("id"), "--candidate", "1", "--now", rnow), "select")
	r = mustOK(t, run(t, ws, "", "unresolved", "--now", rnow), "after select")
	if int(r.json["count"].(float64)) != 6 || r.json["counts"].(map[string]any)["blocked"] != nil {
		t.Errorf("after placing A: %v", r.json["counts"])
	}
	tx := text(t, ws, "unresolved", "--now", rnow)
	if !strings.Contains(tx.stdout, "6 unresolved") || !strings.Contains(tx.stdout, "  incomplete\n  no duration") {
		t.Errorf("text:\n%s", tx.stdout)
	}
	if fi, _ := os.ReadDir(filepath.Join(ws, "resolutions")); len(fi) != 1 {
		t.Error("unresolved wrote a record")
	}
}

func TestWorkspacePointer(t *testing.T) {
	root := t.TempDir()
	real, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	root = real
	wsDir := filepath.Join(root, "planning")
	mustOK(t, run(t, "", "", "init", wsDir, "--author", ada, "--subject", ada, "--timezone", "Australia/Melbourne"), "init")

	// Relative pointer at the repository root, and discovery through it.
	r := mustOK(t, run(t, "", "", "workspace", "pointer", wsDir, "--at", root), "pointer")
	if r.str("target") != "planning" || r.json["relative"] != true || r.str("root") != wsDir {
		t.Fatalf("pointer: %v", r.json)
	}
	body := readFileAt(t, filepath.Join(root, ".intentions"))
	if strings.TrimSpace(body) != "planning" {
		t.Errorf("pointer file: %q", body)
	}
	sub := filepath.Join(root, "src", "pkg")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	w := mustOK(t, runIn(t, sub, "workspace"), "discover through pointer")
	if w.str("found_by") != "pointer" || w.str("root") != wsDir {
		t.Errorf("discovery: %v", w.json)
	}

	// Idempotent; a different target is refused until --force.
	mustOK(t, run(t, "", "", "workspace", "pointer", wsDir, "--at", root), "rewrite same")
	other := filepath.Join(root, "other")
	mustOK(t, run(t, "", "", "init", other, "--author", ada, "--timezone", "UTC"), "init other")
	if r := run(t, "", "", "workspace", "pointer", other, "--at", root); r.code != 1 || !strings.Contains(errMsg(r), "--force") {
		t.Errorf("clobber refused: %d %s", r.code, errMsg(r))
	}
	if strings.TrimSpace(readFileAt(t, filepath.Join(root, ".intentions"))) != "planning" {
		t.Error("refused write changed the file")
	}
	f := mustOK(t, run(t, "", "", "workspace", "pointer", other, "--at", root, "--force"), "force")
	if f.str("target") != "other" {
		t.Errorf("force: %v", f.json)
	}

	// Absolute when the workspace lies outside the pointer's tree.
	outside := t.TempDir()
	away := mustOK(t, run(t, "", "", "workspace", "pointer", wsDir, "--at", outside), "outside")
	if away.json["relative"] != false || !filepath.IsAbs(away.str("target")) {
		t.Errorf("outside: %v", away.json)
	}
	tx := textIn(t, outside, "workspace", "pointer", wsDir, "--at", outside)
	if !strings.Contains(tx.stdout, "machine-specific") {
		t.Errorf("absolute note missing:\n%s", tx.stdout)
	}

	// Usage guards.
	if r := run(t, "", "", "workspace", "pointer", wsDir, "--at", wsDir); r.code != 2 || !strings.Contains(errMsg(r), "the workspace itself") {
		t.Errorf("pointer at the workspace: %d %s", r.code, errMsg(r))
	}
	if r := run(t, "", "", "workspace", "pointer", wsDir, "--at", filepath.Join(root, "nope")); r.code != 2 || !strings.Contains(errMsg(r), "not a directory") {
		t.Errorf("--at missing: %d %s", r.code, errMsg(r))
	}
	nowhere := t.TempDir()
	if r := runIn(t, nowhere, "workspace", "pointer", "--at", nowhere); r.code == 0 {
		t.Error("no argument outside a workspace should fail")
	}
}

func TestInitPointer(t *testing.T) {
	root := t.TempDir()
	real, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	root = real
	r := mustOK(t, runIn(t, root, "init", "./planning", "--pointer", "--author", ada, "--timezone", "UTC"), "init --pointer")
	if r.str("pointer") != filepath.Join(root, ".intentions") {
		t.Fatalf("pointer: %v", r.json)
	}
	if strings.TrimSpace(readFileAt(t, filepath.Join(root, ".intentions"))) != "planning" {
		t.Errorf("pointer file: %q", readFileAt(t, filepath.Join(root, ".intentions")))
	}
	w := mustOK(t, runIn(t, root, "workspace"), "workspace")
	if w.str("found_by") != "pointer" || w.str("root") != filepath.Join(root, "planning") {
		t.Errorf("discovery: %v", w.json)
	}
	// --pointer needs a directory other than the current one.
	if r := runIn(t, t.TempDir(), "init", "--pointer", "--author", ada); r.code != 2 || !strings.Contains(errMsg(r), "--pointer needs") {
		t.Errorf("pointer without dir: %d %s", r.code, errMsg(r))
	}
}

// runIn drives the CLI with the working directory set to dir.
func runIn(t *testing.T, dir string, args ...string) result {
	t.Helper()
	restore := chdir(t, dir)
	defer restore()
	return run(t, "", "", args...)
}

func textIn(t *testing.T, dir string, args ...string) result {
	t.Helper()
	restore := chdir(t, dir)
	defer restore()
	return text(t, "", args...)
}

func chdir(t *testing.T, dir string) func() {
	t.Helper()
	prev, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	return func() { _ = os.Chdir(prev) }
}

func readFileAt(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// commitmentFixture builds a workspace holding one tentative commitment
// between ada and the meeting room, and returns the ids.
func commitmentFixture(t *testing.T) (ws, cmt, intention string) {
	t.Helper()
	ws = initWS(t)
	mustOK(t, run(t, ws, "", "availability", "add", "--now", rnow, "--subject", ada, "--duration", "PT4H", "--calendar", "2026-09/2026-12", "--clock", "13:00/17:00", "--conditional", "meeting", "--cadence", "FREQ=WEEKLY;BYDAY=MO,TU,WE,TH,FR"), "ada supply")
	mustOK(t, run(t, ws, "", "availability", "add", "--now", rnow, "--subject", room, "--duration", "PT8H", "--calendar", "2026-09/2026-12", "--clock", "08:00/18:00", "--cadence", "FREQ=WEEKLY;BYDAY=MO,TU,WE,TH,FR"), "room supply")
	m := mustOK(t, run(t, ws, "", "intention", "add", "--now", rnow, "--title", "Review in room 3", "--duration", "PT1H", "--calendar", "2026-W38", "--activity", "meeting", "--party", room), "intention")
	s := mustOK(t, run(t, ws, "", "select", m.str("id"), "--candidate", "1", "--now", rnow), "select")
	return ws, s.json["commitment"].(map[string]any)["id"].(string), m.str("id")
}

const room = "https://example.com/rooms/3"

func TestCommitmentAnswer(t *testing.T) {
	ws, cmt, _ := commitmentFixture(t)
	// The owner's own entry is the default.
	r := mustOK(t, run(t, ws, "", "commitment", "accept", cmt, "--now", rnow), "accept")
	if r.str("party") != ada || r.str("status") != "accepted" {
		t.Fatalf("accept: %v", r.json)
	}
	parties := r.obj()["parties"].([]any)
	got := map[string]string{}
	for _, p := range parties {
		p := p.(map[string]any)
		got[p["uri"].(string)] = p["status"].(string)
	}
	if got[ada] != "accepted" || got[room] != "tentative" {
		t.Errorf("only the answering party changes: %v", got)
	}
	if r.str("version") == r.str("previous_version") {
		t.Error("a status change must change the version")
	}
	// Idempotence is refused, not written.
	if r := run(t, ws, "", "commitment", "accept", cmt, "--now", rnow); r.code != 2 || !strings.Contains(errMsg(r), "already accepted") {
		t.Errorf("re-accept: %d %s", r.code, errMsg(r))
	}
	// Declining is the same act with the other status.
	d := mustOK(t, run(t, ws, "", "commitment", "decline", cmt, "--now", rnow), "decline")
	if d.obj()["parties"].([]any)[0].(map[string]any)["status"] != "declined" && d.obj()["parties"].([]any)[1].(map[string]any)["status"] != "declined" {
		t.Errorf("decline: %v", d.obj()["parties"])
	}
	// A party the commitment does not list.
	if r := run(t, ws, "", "commitment", "accept", cmt, "--party", "https://example.com/people/bob", "--now", rnow); r.code != 2 || !strings.Contains(errMsg(r), "not a party") {
		t.Errorf("stranger: %d %s", r.code, errMsg(r))
	}
	// Another party of the commitment may be answered for explicitly.
	mustOK(t, run(t, ws, "", "commitment", "accept", cmt, "--party", room, "--now", rnow), "room accepts")
	// A cancelled commitment refuses every answer.
	mustOK(t, run(t, ws, "", "commitment", "cancel", cmt, "--now", rnow), "cancel")
	if r := run(t, ws, "", "commitment", "accept", cmt, "--now", rnow); r.code != 2 || !strings.Contains(errMsg(r), "is retired (cancelled)") {
		t.Errorf("answer after cancel: %d %s", r.code, errMsg(r))
	}
}

func TestCommitmentCancelFreesTheIntention(t *testing.T) {
	ws, cmt, in := commitmentFixture(t)
	before := mustOK(t, run(t, ws, "", "show", in), "intention before")
	if before.obj()["placement"] == nil {
		t.Fatal("fixture intention is not placed")
	}
	r := mustOK(t, run(t, ws, "", "commitment", "cancel", cmt, "--reason", "the room fell through", "--now", rnow), "cancel")
	if r.obj()["retired"].(map[string]any)["kind"] != "cancelled" || r.obj()["retired"].(map[string]any)["reason"] != "the room fell through" {
		t.Errorf("retirement: %v", r.obj()["retired"])
	}
	freed := r.json["freed"].(map[string]any)
	if freed["id"] != in {
		t.Fatalf("freed: %v", freed)
	}
	after := mustOK(t, run(t, ws, "", "show", in), "intention after")
	o := after.obj()
	if o["placement"] != nil || o["retired"] != nil || o["window"].(map[string]any)["calendar"] != "2026-W38" || o["stability"] != "tentative" {
		t.Errorf("freed intention: %v", o)
	}
	// It is resolvable again, and unresolved lists it.
	u := mustOK(t, run(t, ws, "", "unresolved", "--now", rnow), "unresolved")
	found := false
	for _, e := range u.json["entries"].([]any) {
		if e.(map[string]any)["id"] == in {
			found = true
		}
	}
	if !found {
		t.Errorf("freed intention absent from unresolved: %v", u.json)
	}
	mustOK(t, run(t, ws, "", "resolve", in, "--now", rnow), "resolve again")
	// Cancellation is terminal, and the resolution record is untouched.
	if r := run(t, ws, "", "commitment", "cancel", cmt, "--now", rnow); r.code != 2 || !strings.Contains(errMsg(r), "already retired") {
		t.Errorf("re-cancel: %d %s", r.code, errMsg(r))
	}
	if fi, _ := os.ReadDir(filepath.Join(ws, "resolutions")); len(fi) != 1 {
		t.Error("cancel touched the resolution record")
	}
}

func TestCommitmentShowAndList(t *testing.T) {
	ws, cmt, in := commitmentFixture(t)
	s := mustOK(t, run(t, ws, "", "commitment", "show", cmt, "--now", rnow), "show")
	if s.json["intention_resolved"].(map[string]any)["id"] != in || s.json["flags"] == nil {
		t.Errorf("show: %v", s.json)
	}
	if l := mustOK(t, run(t, ws, "", "commitment", "list"), "list"); int(l.json["count"].(float64)) != 1 {
		t.Errorf("list: %v", l.json)
	}
	if l := mustOK(t, run(t, ws, "", "commitment", "list", "--party", ada, "--status", "tentative"), "filter"); int(l.json["count"].(float64)) != 1 {
		t.Errorf("party+status filter: %v", l.json)
	}
	mustOK(t, run(t, ws, "", "commitment", "accept", cmt, "--now", rnow), "accept")
	if l := mustOK(t, run(t, ws, "", "commitment", "list", "--party", ada, "--status", "tentative"), "filter after accept"); int(l.json["count"].(float64)) != 0 {
		t.Errorf("status filter after accept: %v", l.json)
	}
	if l := mustOK(t, run(t, ws, "", "commitment", "list", "--intention", in), "by intention"); int(l.json["count"].(float64)) != 1 {
		t.Errorf("intention filter: %v", l.json)
	}
	if r := run(t, ws, "", "commitment", "list", "--status", "maybe"); r.code != 2 {
		t.Errorf("bad status: %d", r.code)
	}
	// Cancelled ones are listed only on request.
	mustOK(t, run(t, ws, "", "commitment", "cancel", cmt, "--now", rnow), "cancel")
	if l := mustOK(t, run(t, ws, "", "commitment", "list"), "active"); int(l.json["count"].(float64)) != 0 {
		t.Errorf("cancelled still active: %v", l.json)
	}
	if l := mustOK(t, run(t, ws, "", "commitment", "list", "--cancelled"), "cancelled"); int(l.json["count"].(float64)) != 1 {
		t.Errorf("cancelled list: %v", l.json)
	}
	if r := run(t, ws, "", "commitment", "show", in); r.code != 2 || !strings.Contains(errMsg(r), "not a commitment") {
		t.Errorf("show an intention: %d %s", r.code, errMsg(r))
	}
}

// writeImport hand-writes an imported commitment: there is no import verb
// yet, and an import is the only commitment with no intention behind it.
func writeImport(t *testing.T, ws, id, start, duration, status string, extra string) string {
	t.Helper()
	body := "id: " + id + "\nparties:\n  - uri: " + ada + "\n    status: " + status +
		"\nplacement:\n  start: " + start + "\n  duration: " + duration +
		"\norigin: import\ntitle: Imported\n" + extra +
		"source:\n  author: " + ada + "\ntimestamp: " + rnow + "\nacknowledgements: []\n"
	if err := os.WriteFile(filepath.Join(ws, "commitments", id+".yaml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	mustOK(t, run(t, ws, "", "index"), "reindex after import")
	return id
}

// TestPartyOccupancyIsPerParty covers capacity: a decline frees the decliner
// and nobody else, and the subject's own decline frees nothing while their
// intention's placement stands.
func TestPartyOccupancyIsPerParty(t *testing.T) {
	ws, cmt, in := commitmentFixture(t)
	// The fixture placed a one-hour meeting at 13:00 on 2026-09-14, leaving
	// three of ada's four meeting hours that day.
	fits := func(what string, dur string) bool {
		t.Helper()
		x := mustOK(t, run(t, ws, "", "intention", "add", "--now", rnow, "--title", what, "--duration", dur, "--calendar", "2026-09-14", "--activity", "meeting"), "add "+what)
		r := mustOK(t, run(t, ws, "", "resolve", x.str("id"), "--now", rnow), "resolve "+what)
		mustOK(t, run(t, ws, "", "intention", "retire", x.str("id"), "--kind", "abandoned", "--now", rnow), "tidy "+what)
		return len(r.json["candidates"].([]any)) > 0
	}
	if fits("four hours", "PT4H") {
		t.Fatal("the placed hour should already have consumed capacity")
	}
	if !fits("three hours", "PT3H") {
		t.Fatal("three of four hours should remain")
	}
	// The subject declines their own arrangement: the intention's placement
	// still consumes the hour, so nothing is freed.
	mustOK(t, run(t, ws, "", "commitment", "decline", cmt, "--now", rnow), "subject declines")
	if fits("four hours after decline", "PT4H") {
		t.Error("a subject's decline freed an hour their own placement still occupies")
	}
	// Cancelling clears the placement, and only then is the hour free.
	mustOK(t, run(t, ws, "", "commitment", "cancel", cmt, "--now", rnow), "cancel")
	mustOK(t, run(t, ws, "", "intention", "retire", in, "--kind", "abandoned", "--now", rnow), "retire freed intention")
	if !fits("four hours after cancel", "PT4H") {
		t.Error("cancelling did not free the hour")
	}
}

// TestDeclinedImportFreesTheHour covers the other origin: an import names no
// intention, so declining it frees the decliner's time outright.
func TestDeclinedImportFreesTheHour(t *testing.T) {
	ws := initWS(t)
	mustOK(t, run(t, ws, "", "availability", "add", "--now", rnow, "--subject", ada, "--duration", "PT4H", "--calendar", "2026-09/2026-12", "--clock", "13:00/17:00", "--conditional", "meeting", "--cadence", "FREQ=WEEKLY;BYDAY=MO,TU,WE,TH,FR"), "supply")
	fits := func(what, dur string) bool {
		t.Helper()
		x := mustOK(t, run(t, ws, "", "intention", "add", "--now", rnow, "--title", what, "--duration", dur, "--calendar", "2026-09-14", "--activity", "meeting"), "add")
		r := mustOK(t, run(t, ws, "", "resolve", x.str("id"), "--now", rnow), "resolve")
		mustOK(t, run(t, ws, "", "intention", "retire", x.str("id"), "--kind", "abandoned", "--now", rnow), "tidy")
		return len(r.json["candidates"].([]any)) > 0
	}
	id := writeImport(t, ws, "cmt_01a07700-0000-7000-8000-00000000ab01", "2026-09-14T13:00:00+10:00", "PT1H", "tentative", "")
	if fits("four hours", "PT4H") {
		t.Fatal("a tentative import should consume the hour")
	}
	mustOK(t, run(t, ws, "", "commitment", "decline", id, "--now", rnow), "decline the import")
	if !fits("four hours after decline", "PT4H") {
		t.Error("declining an import did not free the hour")
	}
	// And nothing is flagged, because no plan disagrees with it.
	c := mustOK(t, run(t, ws, "", "check", "--now", rnow), "check")
	for _, f := range c.json["flags"].([]any) {
		if f.(map[string]any)["kind"] == "party-declined" {
			t.Errorf("an import raised party-declined: %v", f)
		}
	}
}

// TestRankReadsTheResolvingSubject covers the ladder: the rungs read the
// subject's own entry, and a commitment that does not occupy them is neither
// displaced nor allowed to raise a rank.
func TestRankReadsTheResolvingSubject(t *testing.T) {
	ws, cmt, _ := commitmentFixture(t)
	rankAt13 := func(what string) (float64, []any) {
		t.Helper()
		x := mustOK(t, run(t, ws, "", "intention", "add", "--now", rnow, "--title", what, "--duration", "PT1H", "--calendar", "2026-09-14", "--clock", "13:00/14:00", "--activity", "meeting"), "add")
		r := mustOK(t, run(t, ws, "", "resolve", x.str("id"), "--now", rnow), "resolve")
		mustOK(t, run(t, ws, "", "intention", "retire", x.str("id"), "--kind", "abandoned", "--now", rnow), "tidy")
		cands := r.json["candidates"].([]any)
		if len(cands) == 0 {
			t.Fatalf("%s: no candidate to rank: %v", what, r.json)
		}
		c := cands[0].(map[string]any)
		return c["rank"].(float64), c["displaces"].([]any)
	}
	// The subject is tentative on the commitment: rung 2.
	if rank, disp := rankAt13("against tentative"); rank != 2 || len(disp) == 0 {
		t.Errorf("tentative subject: rank %v displaces %v", rank, disp)
	}
	// The room accepts; the subject is still tentative, so the rung is still 2.
	mustOK(t, run(t, ws, "", "commitment", "accept", cmt, "--party", room, "--now", rnow), "room accepts")
	if rank, _ := rankAt13("counterparty accepted"); rank != 2 {
		t.Errorf("another party's acceptance raised the subject's rung: %v", rank)
	}
	// The subject accepts: rung 3.
	mustOK(t, run(t, ws, "", "commitment", "accept", cmt, "--now", rnow), "subject accepts")
	if rank, _ := rankAt13("subject accepted"); rank != 3 {
		t.Errorf("subject accepted: rank %v", rank)
	}
}

func TestDeclinedImportIsNotDisplaced(t *testing.T) {
	ws := initWS(t)
	mustOK(t, run(t, ws, "", "availability", "add", "--now", rnow, "--subject", ada, "--duration", "PT4H", "--calendar", "2026-09/2026-12", "--clock", "13:00/17:00", "--conditional", "meeting", "--cadence", "FREQ=WEEKLY;BYDAY=MO,TU,WE,TH,FR"), "supply")
	id := writeImport(t, ws, "cmt_01a07700-0000-7000-8000-00000000ab02", "2026-09-14T13:00:00+10:00", "PT1H", "declined", "")
	x := mustOK(t, run(t, ws, "", "intention", "add", "--now", rnow, "--title", "Over the declined hour", "--duration", "PT1H", "--calendar", "2026-09-14", "--clock", "13:00/14:00", "--activity", "meeting"), "add")
	r := mustOK(t, run(t, ws, "", "resolve", x.str("id"), "--now", rnow), "resolve")
	c := r.json["candidates"].([]any)[0].(map[string]any)
	if c["rank"].(float64) != 1 || len(c["displaces"].([]any)) != 0 {
		t.Errorf("a declined commitment was displaced: rank %v displaces %v", c["rank"], c["displaces"])
	}
	_ = id
}

// TestPartyDeclinedFlag covers the seventh kind.
func TestPartyDeclinedFlag(t *testing.T) {
	ws, cmt, in := commitmentFixture(t)
	find := func(r result, subject string) map[string]any {
		t.Helper()
		for _, f := range r.json["flags"].([]any) {
			f := f.(map[string]any)
			if f["kind"] == "party-declined" && f["subject"] == subject {
				return f
			}
		}
		return nil
	}
	// No decline, no flag.
	if find(mustOK(t, run(t, ws, "", "check", "--now", rnow), "check"), cmt) != nil {
		t.Fatal("flagged with nothing declined")
	}
	// The subject declines their own placed plan: on both objects.
	mustOK(t, run(t, ws, "", "commitment", "decline", cmt, "--now", rnow), "subject declines")
	c := mustOK(t, run(t, ws, "", "check", "--now", rnow), "check")
	onCmt, onInt := find(c, cmt), find(c, in)
	if onCmt == nil || onInt == nil {
		t.Fatalf("flag missing on one side: %v", c.json["flags"])
	}
	if onCmt["counterpart"] != in || onInt["counterpart"] != cmt {
		t.Errorf("counterparts: %v %v", onCmt, onInt)
	}
	if !strings.Contains(onCmt["detail"].(string), "the subject") {
		t.Errorf("detail should name the subject: %q", onCmt["detail"])
	}
	// Acknowledging suppresses it; the placement and the status stand.
	mustOK(t, run(t, ws, "", "acknowledge", in, "--kind", "party-declined", "--counterpart", cmt, "--reason", "going ahead anyway", "--now", rnow), "acknowledge")
	if find(mustOK(t, run(t, ws, "", "check", "--now", rnow), "check"), in) != nil {
		t.Error("acknowledgement did not suppress the flag")
	}
	if find(mustOK(t, run(t, ws, "", "check", "--now", rnow), "check"), cmt) == nil {
		t.Error("acknowledging one side silenced the other")
	}
	// A reversal changes the commitment's projection: the acknowledgement
	// lapses, and the flag is gone anyway because nobody is declined.
	mustOK(t, run(t, ws, "", "commitment", "accept", cmt, "--now", rnow), "reverse")
	after := mustOK(t, run(t, ws, "", "check", "--now", rnow), "check")
	if find(after, in) != nil || find(after, cmt) != nil {
		t.Errorf("flag survived the reversal: %v", after.json["flags"])
	}
	// A counterparty's decline reads differently.
	mustOK(t, run(t, ws, "", "commitment", "decline", cmt, "--party", room, "--now", rnow), "room declines")
	f := find(mustOK(t, run(t, ws, "", "check", "--now", rnow), "check"), cmt)
	if f == nil || !strings.Contains(f["detail"].(string), "a counterparty") {
		t.Errorf("counterparty detail: %v", f)
	}
	// Cancelling clears the placement, so the disagreement is gone.
	mustOK(t, run(t, ws, "", "commitment", "cancel", cmt, "--now", rnow), "cancel")
	if find(mustOK(t, run(t, ws, "", "check", "--now", rnow), "check"), cmt) != nil {
		t.Error("flag survived cancellation")
	}
}

// --- align-with-resolution-additions ---------------------------------------

func TestOnePlanningHorizon(t *testing.T) {
	ws := initWS(t)
	body := readFile(t, ws, "intentions.yaml")
	if strings.Contains(body, "generation:") || !strings.Contains(body, "horizon: P4W") {
		t.Errorf("init wrote a generation section:\n%s", body)
	}
	if r := run(t, "", "", "init", filepath.Join(t.TempDir(), "x"), "--author", ada, "--generation-horizon", "P2W"); r.code != 2 {
		t.Errorf("--generation-horizon should be unknown: %d", r.code)
	}
	// resolver.horizon governs generation as well as resolution.
	ws2 := initWS(t, "--horizon", "P1W")
	mustOK(t, run(t, ws2, "", "availability", "add", "--now", rnow, "--subject", ada, "--duration", "PT2H", "--calendar", "2026-09/2026-12", "--clock", "09:00/11:00", "--cadence", "FREQ=WEEKLY;BYDAY=TU"), "supply")
	rec := mustOK(t, run(t, ws2, "", "intention", "add", "--now", rnow, "--title", "Weekly", "--duration", "PT1H", "--calendar", "2026-09/2026-12", "--cadence", "FREQ=WEEKLY;BYDAY=TU"), "recurring")
	g := mustOK(t, run(t, ws2, "", "generate", "--now", rnow), "generate")
	if int(g.json["count"].(float64)) != 1 {
		t.Errorf("a one-week horizon should generate one instance, got %v", g.json["count"])
	}
	_ = rec
	// A stale generation.horizon is ignored and reported.
	cfg := filepath.Join(ws, "intentions.yaml")
	old := readFileAt(t, cfg)
	if err := os.WriteFile(cfg, []byte(old+"generation:\n  horizon: P2W\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	v := mustOK(t, run(t, ws, "", "validate"), "validate")
	found := false
	for _, f := range v.json["findings"].([]any) {
		f := f.(map[string]any)
		if f["severity"] == "info" && strings.Contains(f["message"].(string), "generation.horizon") {
			found = true
		}
	}
	if !found {
		t.Errorf("stale generation.horizon not reported: %v", v.json["findings"])
	}
	if v.json["ok"] != true {
		t.Error("a stale key should not be an error")
	}
}

func TestRangedDurationShrinksToFit(t *testing.T) {
	ws := initWS(t)
	// One hour of supply on the Monday, nothing else that week.
	mustOK(t, run(t, ws, "", "availability", "add", "--now", rnow, "--subject", ada, "--duration", "PT1H", "--calendar", "2026-09-14", "--clock", "09:00/10:00"), "supply")
	// A nominal ninety minutes that will not fit, with a floor of one hour.
	a := mustOK(t, run(t, ws, "", "intention", "add", "--now", rnow, "--title", "Ranged", "--duration", "PT90M:PT1H:PT2H", "--calendar", "2026-09-14"), "add")
	r := mustOK(t, run(t, ws, "", "resolve", a.str("id"), "--now", rnow), "resolve")
	cands := r.json["candidates"].([]any)
	if len(cands) == 0 {
		t.Fatalf("a ranged duration should be offered shorter: %v", r.json)
	}
	for _, c := range cands {
		if c.(map[string]any)["duration"] != "PT1H" {
			t.Errorf("candidate duration: %v", c)
		}
	}
	// Selecting places what was offered, not the nominal.
	s := mustOK(t, run(t, ws, "", "select", a.str("id"), "--candidate", "1", "--now", rnow), "select")
	if s.json["intention"].(map[string]any)["placement"].(map[string]any)["duration"] != "PT1H" {
		t.Errorf("placement duration: %v", s.json["intention"])
	}
	// A nominal that fits is never shortened.
	b := mustOK(t, run(t, ws, "", "intention", "add", "--now", rnow, "--title", "Fits", "--duration", "PT30M:PT15M:PT1H", "--calendar", "2026-09-15", "--clock", "09:00/10:00"), "add b")
	mustOK(t, run(t, ws, "", "availability", "add", "--now", rnow, "--subject", ada, "--duration", "PT1H", "--calendar", "2026-09-15", "--clock", "09:00/10:00"), "supply b")
	rb := mustOK(t, run(t, ws, "", "resolve", b.str("id"), "--now", rnow), "resolve b")
	for _, c := range rb.json["candidates"].([]any) {
		if c.(map[string]any)["duration"] != "PT30M" {
			t.Errorf("nominal should be kept: %v", c)
		}
	}
	// Nothing below min is offered.
	c := mustOK(t, run(t, ws, "", "intention", "add", "--now", rnow, "--title", "Too long", "--duration", "PT4H:PT3H:PT5H", "--calendar", "2026-09-14"), "add c")
	rc := mustOK(t, run(t, ws, "", "resolve", c.str("id"), "--now", rnow), "resolve c")
	if len(rc.json["candidates"].([]any)) != 0 || !strings.Contains(rc.str("reason"), "PT3H") {
		t.Errorf("below min should not be offered: %v", rc.json)
	}
}

func TestPersonalAvailabilityIsNotSharedSupply(t *testing.T) {
	ws := initWS(t)
	rob := "https://example.com/people/rob"
	mustOK(t, run(t, ws, "", "availability", "add", "--now", rnow, "--subject", rob, "--duration", "PT4H", "--calendar", "2026-09-14", "--clock", "09:00/13:00"), "rob's personal supply")
	// Ada's own intention cannot draw on Rob's personal capacity.
	a := mustOK(t, run(t, ws, "", "intention", "add", "--now", rnow, "--title", "Ada alone", "--duration", "PT1H", "--calendar", "2026-09-14"), "add")
	r := mustOK(t, run(t, ws, "", "resolve", a.str("id"), "--now", rnow), "resolve")
	if len(r.json["candidates"].([]any)) != 0 {
		t.Fatalf("Rob's personal availability was used as Ada's supply: %v", r.json)
	}
	// Naming Rob a party makes it visible as his supply, and Ada still needs her own.
	b := mustOK(t, run(t, ws, "", "intention", "add", "--now", rnow, "--title", "With Rob", "--duration", "PT1H", "--calendar", "2026-09-14", "--party", rob), "add b")
	rb := mustOK(t, run(t, ws, "", "resolve", b.str("id"), "--now", rnow), "resolve b")
	if !strings.Contains(rb.str("reason"), ada) {
		t.Errorf("Rob's supply should now be visible, leaving Ada without: %v", rb.json)
	}
	mustOK(t, run(t, ws, "", "availability", "add", "--now", rnow, "--subject", ada, "--duration", "PT4H", "--calendar", "2026-09-14", "--clock", "09:00/13:00"), "ada's supply")
	rb2 := mustOK(t, run(t, ws, "", "resolve", b.str("id"), "--now", rnow), "resolve b2")
	if len(rb2.json["candidates"].([]any)) == 0 {
		t.Errorf("a party's personal availability should be visible: %v", rb2.json)
	}
	// Widening the scope makes it supply for anyone the resolver may see.
	mustOK(t, run(t, ws, "", "availability", "add", "--now", rnow, "--subject", "https://example.com/rooms/9", "--duration", "PT8H", "--calendar", "2026-09-15", "--clock", "09:00/17:00", "--scope", "organisation"), "shared room")
	c := mustOK(t, run(t, ws, "", "intention", "add", "--now", rnow, "--title", "Uses the shared room", "--duration", "PT1H", "--calendar", "2026-09-15", "--party", "https://example.com/rooms/9"), "add c")
	mustOK(t, run(t, ws, "", "availability", "add", "--now", rnow, "--subject", ada, "--duration", "PT8H", "--calendar", "2026-09-15", "--clock", "09:00/17:00"), "ada tuesday")
	if rc := mustOK(t, run(t, ws, "", "resolve", c.str("id"), "--now", rnow), "resolve c"); len(rc.json["candidates"].([]any)) == 0 {
		t.Errorf("organisation scope should be visible: %v", rc.json)
	}
}

func TestReplacementCarriesTheCommitment(t *testing.T) {
	ws, cmt, in := commitmentFixture(t)
	mustOK(t, run(t, ws, "", "commitment", "accept", cmt, "--now", rnow), "accept")
	r := mustOK(t, run(t, ws, "", "select", in, "--candidate", "2", "--replace", "--now", rnow), "replace")
	// The old commitment is cancelled, naming the new resolution.
	old := mustOK(t, run(t, ws, "", "show", cmt), "old commitment").obj()
	ret, _ := old["retired"].(map[string]any)
	if ret == nil || ret["kind"] != "cancelled" || !strings.Contains(ret["reason"].(string), r.json["resolution"].(map[string]any)["id"].(string)) {
		t.Fatalf("old commitment: %v", old["retired"])
	}
	// It still records who had accepted.
	accepted := false
	for _, p := range old["parties"].([]any) {
		if p.(map[string]any)["status"] == "accepted" {
			accepted = true
		}
	}
	if !accepted {
		t.Error("the retired file should still show who had accepted")
	}
	// A fresh commitment carries the new placement with everyone tentative.
	fresh, _ := r.json["commitment"].(map[string]any)
	if fresh == nil || fresh["id"] == cmt {
		t.Fatalf("no fresh commitment: %v", r.json["commitment"])
	}
	for _, p := range fresh["parties"].([]any) {
		if p.(map[string]any)["status"] != "tentative" {
			t.Errorf("fresh commitment party: %v", p)
		}
	}
	if fresh["placement"].(map[string]any)["start"] != r.json["candidate"].(map[string]any)["start"] {
		t.Errorf("fresh commitment placement: %v", fresh["placement"])
	}
	if r.json["cancelled_commitment"].(map[string]any)["id"] != cmt {
		t.Errorf("result should name the cancelled commitment: %v", r.json["cancelled_commitment"])
	}
	// Only one live commitment remains.
	if l := mustOK(t, run(t, ws, "", "commitment", "list"), "list"); int(l.json["count"].(float64)) != 1 {
		t.Errorf("live commitments: %v", l.json)
	}
}
