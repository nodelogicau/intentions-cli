package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/nodelogicau/intentions-cli/internal/apperr"
	"github.com/nodelogicau/intentions-cli/internal/model"
	"github.com/nodelogicau/intentions-cli/internal/temporal"
)

func testConfig() Config {
	c := NewConfig()
	c.Resolver.Timezone = "Australia/Melbourne"
	c.Defaults.Subject = "https://example.com/people/ada"
	c.Defaults.Source.Author = "https://example.com/people/ada"
	return c
}

func initWS(t *testing.T) *Workspace {
	t.Helper()
	dir := t.TempDir()
	ws, created, err := Init(dir, testConfig())
	if err != nil {
		t.Fatal(err)
	}
	if len(created) != 7 {
		t.Fatalf("created %v", created)
	}
	return ws
}

func sample(id string) *model.Intention {
	d, _ := temporal.ParseDurationSpec("PT1H")
	c, _ := temporal.ParseCalendar("2026-W37")
	return &model.Intention{ID: id, Subject: "https://example.com/people/ada", Title: "t", Duration: &d, Window: &temporal.Window{Calendar: &c},
		Stability: "tentative", Serves: []model.Ref{}, Source: model.Source{Author: "a"}, Timestamp: time.Date(2026, 9, 4, 0, 0, 0, 0, time.UTC), Acknowledgements: []model.Acknowledgement{}}
}

func TestInitAndConfig(t *testing.T) {
	ws := initWS(t)
	data, _ := os.ReadFile(filepath.Join(ws.Root, ConfigFile))
	for _, want := range []string{"format: intentions/0.1", "hash: sha256", "timezone: Australia/Melbourne", "week_start: monday", "default_horizon: P13W", "horizon: P4W", "subject: https://example.com/people/ada", "author: https://example.com/people/ada"} {
		if !strings.Contains(string(data), want) {
			t.Errorf("config missing %q:\n%s", want, data)
		}
	}
	if _, _, err := Init(ws.Root, testConfig()); err == nil {
		t.Error("second init accepted")
	}
	// Unknown key preserved through a rewrite.
	data = append(data, []byte("custom: 1\n")...)
	cfg, err := ParseConfig(data)
	if err != nil {
		t.Fatal(err)
	}
	cfg.Defaults.Source.Author = "https://example.com/people/bob"
	out, _ := cfg.Marshal()
	if !strings.Contains(string(out), "custom: 1") || !strings.Contains(string(out), "people/bob") {
		t.Errorf("rewrite lost keys:\n%s", out)
	}
	bad := testConfig()
	bad.Hash = "sha512"
	if err := bad.Validate(); err == nil || !strings.Contains(err.Error(), "sha256") {
		t.Errorf("hash: %v", err)
	}
	bad = testConfig()
	bad.Resolver.Timezone = "Mars/Olympus"
	if err := bad.Validate(); err == nil {
		t.Error("bad timezone accepted")
	}
}

func TestDiscovery(t *testing.T) {
	ws := initWS(t)
	nested := filepath.Join(ws.Root, "a", "b")
	_ = os.MkdirAll(nested, 0o755)
	inner, _, err := Init(filepath.Join(ws.Root, "a"), testConfig())
	if err != nil {
		t.Fatal(err)
	}
	found, res, err := DiscoverFrom(nested)
	if err != nil || found.Root != inner.Root || res.FoundBy != "marker" {
		t.Errorf("nearest: %v %v %v", found, res, err)
	}
	// Pointer file.
	other := t.TempDir()
	_ = os.MkdirAll(filepath.Join(other, "deep"), 0o755)
	if _, err := WritePointer(other, ws.Root); err != nil {
		t.Fatal(err)
	}
	found, res, err = DiscoverFrom(filepath.Join(other, "deep"))
	if err != nil || found.Root != ws.Root || res.FoundBy != "pointer" {
		t.Errorf("pointer: %v %v %v", found, res, err)
	}
	// Relative pointer.
	rel := t.TempDir()
	_, _, _ = Init(filepath.Join(rel, "planning"), testConfig())
	_, _ = WritePointer(rel, "planning")
	found, _, err = DiscoverFrom(rel)
	if err != nil || found.Root != filepath.Join(rel, "planning") {
		t.Errorf("relative pointer: %v %v", found, err)
	}
	// Strict env var.
	empty := t.TempDir()
	t.Setenv(EnvWorkspace, empty)
	_, _, err = Discover("")
	if ee := apperr.Classify(err); ee == nil || ee.Code != apperr.ExitNoWorkspace {
		t.Errorf("env strict: %v", err)
	}
	// Flag wins.
	found, res, err = Discover(ws.Root)
	if err != nil || found.Root != ws.Root || res.FoundBy != "flag" {
		t.Errorf("flag: %v %v", res, err)
	}
	// Nothing anywhere.
	t.Setenv(EnvWorkspace, "")
	if _, _, err := DiscoverFrom(t.TempDir()); err == nil {
		t.Error("expected no workspace")
	}
}

func TestWriteReadLoadIndex(t *testing.T) {
	ws := initWS(t)
	a := sample(model.MintID(model.TypeIntention))
	if err := ws.WriteObject(a); err != nil {
		t.Fatal(err)
	}
	if a.Version == "" {
		t.Error("version not stamped")
	}
	b := sample(model.MintID(model.TypeIntention))
	b.Serves = []model.Ref{{ID: a.ID, Role: model.RoleInOrderTo}}
	if err := ws.WriteObject(b); err != nil {
		t.Fatal(err)
	}
	got, probs, err := ws.ReadObject(a.ID)
	if err != nil || len(probs) > 0 || got.GetID() != a.ID || got.CachedVersion() != a.Version {
		t.Errorf("read: %v %v %v", got, probs, err)
	}
	if _, _, err := ws.ReadObject("int_missing"); apperr.Classify(err).Code != apperr.ExitNotFound {
		t.Errorf("missing: %v", err)
	}
	g, err := ws.Load()
	if err != nil || len(g.Order) != 2 {
		t.Fatalf("load: %v %v", g, err)
	}
	if in := g.Inbound(a.ID); len(in) != 1 || in[0].From != b.ID {
		t.Errorf("inbound: %v", in)
	}
	ix, _ := ws.ReadIndex()
	if len(ix.Entries) != 2 || ix.Entries[0].ID != a.ID || ix.Entries[1].Refs[0] != a.ID || ix.Entries[0].Path != "intentions/"+a.ID+".yaml" {
		t.Errorf("index: %+v", ix.Entries)
	}
	if d := Compare(ix, Rebuild(g)); !d.Empty() {
		t.Errorf("fresh index drifts: %+v", d)
	}
	// Hand edit drifts.
	path := ws.Path(model.TypeIntention, a.ID)
	data, _ := os.ReadFile(path)
	data = []byte(strings.Replace(string(data), "2026-W37", "2026-W38", 1))
	_ = os.WriteFile(path, data, 0o644)
	g, _ = ws.Load()
	if d := Compare(ix, Rebuild(g)); len(d.Changed) != 1 || d.Changed[0] != a.ID {
		t.Errorf("drift: %+v", d)
	}
	// Conflict markers: read fails, rebuild succeeds.
	_ = os.WriteFile(filepath.Join(ws.Root, IndexFile), []byte("<<<<<<< HEAD\nformat: x\n=======\n>>>>>>> other\n"), 0o644)
	if _, err := ws.ReadIndex(); err == nil {
		t.Error("conflict markers parsed")
	}
	if err := ws.WriteIndex(Rebuild(g)); err != nil {
		t.Fatal(err)
	}
	ix, err = ws.ReadIndex()
	if err != nil || len(ix.Entries) != 2 {
		t.Errorf("rebuilt: %v %v", ix, err)
	}
	// Unknown entry type survives an upsert.
	ix.Entries = append(ix.Entries, Entry{ID: "prs_x", Type: "presence", Path: "presence/prs_x.yaml"})
	ix.Upsert(EntryFor(a))
	found := false
	for _, e := range ix.Entries {
		if e.Type == "presence" {
			found = true
		}
	}
	if !found {
		t.Error("foreign entry dropped by upsert")
	}
	// Unreadable file is recorded, not fatal.
	_ = os.WriteFile(ws.Path(model.TypeIntention, "int_bad"), []byte("- not\n- a mapping\n"), 0o644)
	g, err = ws.Load()
	if err != nil || len(g.Unreadable) != 1 || len(g.Order) != 2 {
		t.Errorf("unreadable: %v %v", g.Unreadable, err)
	}
}

func TestAtomicWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "f.yaml")
	if err := atomicWrite(path, []byte("one\n")); err != nil {
		t.Fatal(err)
	}
	if err := atomicWrite(path, []byte("two\n")); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(path)
	if string(b) != "two\n" {
		t.Errorf("got %q", b)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Errorf("temp file left behind: %v", entries)
	}
	// A write into a missing directory fails and leaves nothing.
	if err := atomicWrite(filepath.Join(dir, "missing", "f.yaml"), []byte("x")); err == nil {
		t.Error("expected failure")
	}
}
