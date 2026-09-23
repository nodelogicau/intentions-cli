package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/nodelogicau/intentions-cli/internal/model"
	"github.com/nodelogicau/intentions-cli/internal/projection"
	"github.com/nodelogicau/intentions-cli/internal/temporal"
)

func TestZeroOneIsByteStableAndMigrates(t *testing.T) {
	ws := initWS(t) // testConfig writes intentions/0.1
	ada := "https://example.com/people/ada"
	src := model.Source{Author: ada}
	at := time.Date(2026, 9, 4, 0, 0, 0, 0, time.UTC)
	term := &model.Intention{ID: "int_t", Subject: ada, Title: "being someone who follows through", Stability: "firm", Serves: []model.Ref{}, Source: src, Timestamp: at, Acknowledgements: []model.Acknowledgement{}}
	d, _ := temporal.ParseDurationSpec("PT1H")
	c, _ := temporal.ParseCalendar("2026-W38")
	av := &model.Availability{ID: "avl_a", Subject: ada, Capacity: &d, Window: &temporal.Window{Calendar: &c}, Activities: []string{"deep-work"}, Scope: "personal", Source: src, Timestamp: at}
	start, _ := temporal.ParsePlacementStart("2026-09-15T10:00:00+10:00")
	pl := &temporal.Placement{Start: start, Duration: d}
	rec := &model.Resolution{ID: "res_r", Intention: "int_a", Placement: pl, Selector: "person", Source: src, Timestamp: at}
	cmt := &model.Commitment{ID: "cmt_c", Parties: []model.Party{{URI: ada, Status: "accepted"}}, Placement: pl, Intention: "int_a", Origin: model.OriginResolution, Resolution: "res_r", Source: src, Timestamp: at, Acknowledgements: []model.Acknowledgement{}}
	for _, o := range []model.Object{term, av, rec, cmt} {
		if err := ws.WriteObject(o); err != nil {
			t.Fatal(err)
		}
	}
	// The 0.1 spellings are on disk.
	avBytes, _ := os.ReadFile(ws.Path(model.TypeAvailability, "avl_a"))
	cmtBytes, _ := os.ReadFile(ws.Path(model.TypeCommitment, "cmt_c"))
	if !strings.Contains(string(avBytes), "\nduration: PT1H\nwindow:") || !strings.Contains(string(avBytes), "\nconditional:\n") || !strings.Contains(string(cmtBytes), "\norigin:\n  resolution: res_r\n") {
		t.Fatalf("0.1 spelling:\n%s\n%s", avBytes, cmtBytes)
	}
	// An intention acknowledging the commitment twice: one current, one already lapsed.
	cmtV := projection.MustVersion(cmt)
	in := &model.Intention{ID: "int_a", Subject: ada, Title: "A", Duration: &d, Window: &temporal.Window{Calendar: &c}, Stability: "tentative", Placement: pl,
		Serves: []model.Ref{{ID: "int_t", Role: model.RoleForTheSakeOf}}, Source: src, Timestamp: at,
		Acknowledgements: []model.Acknowledgement{
			{Kind: "window-clash", Counterpart: "cmt_c", CounterpartVersion: cmtV, Source: src, Timestamp: at},
			{Kind: "window-clash", Counterpart: "cmt_c", CounterpartVersion: "sha256:stale", Source: src, Timestamp: at},
		}}
	if err := ws.WriteObject(in); err != nil {
		t.Fatal(err)
	}
	// Re-saving every object under 0.1 changes nothing.
	before := map[string][]byte{}
	g, _ := ws.Load()
	for _, id := range g.Order {
		before[id], _ = os.ReadFile(ws.Path(g.Objects[id].GetType(), id))
		if v := projection.MustVersion(g.Objects[id]); v != g.Objects[id].CachedVersion() {
			t.Errorf("%s: version moved on load: %s vs %s", id, v, g.Objects[id].CachedVersion())
		}
		if err := ws.WriteObject(g.Objects[id]); err != nil {
			t.Fatal(err)
		}
		after, _ := os.ReadFile(ws.Path(g.Objects[id].GetType(), id))
		if string(after) != string(before[id]) {
			t.Errorf("%s: re-save under 0.1 changed bytes:\n%s\n---\n%s", id, before[id], after)
		}
	}
	// Check writes nothing.
	plan, err := ws.Migrate(true)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Applied || len(plan.Acknowledgements) != 1 || plan.Rewritten() < 2 {
		t.Errorf("plan: %+v", plan)
	}
	if b, _ := os.ReadFile(filepath.Join(ws.Root, ConfigFile)); !strings.Contains(string(b), model.Format01) {
		t.Fatal("check wrote the config")
	}
	// Migrate: the current acknowledgement is carried, the stale one is left.
	plan, err = ws.Migrate(false)
	if err != nil || !plan.Applied {
		t.Fatalf("migrate: %v %+v", err, plan)
	}
	g2, _ := ws.Load()
	if g2.Format != model.Format02 || ws.Config.Format != model.Format02 {
		t.Errorf("format after: %s", g2.Format)
	}
	newCmtV := projection.MustVersion(g2.Objects["cmt_c"])
	if newCmtV == cmtV {
		t.Error("commitment version did not move under 0.2")
	}
	acks := g2.Objects["int_a"].(*model.Intention).Acknowledgements
	if acks[0].CounterpartVersion != newCmtV || acks[1].CounterpartVersion != "sha256:stale" {
		t.Errorf("acknowledgements after: %+v", acks)
	}
	for _, id := range g2.Order {
		if v := projection.MustVersion(g2.Objects[id]); v != g2.Objects[id].CachedVersion() {
			t.Errorf("%s: cached version stale after migration", id)
		}
	}
	avBytes, _ = os.ReadFile(ws.Path(model.TypeAvailability, "avl_a"))
	if !strings.Contains(string(avBytes), "\ncapacity: PT1H\nwindow:") || !strings.Contains(string(avBytes), "\nactivities:\n") {
		t.Errorf("availability under 0.2:\n%s", avBytes)
	}
	// Nothing a person wrote moved: the timestamp and the source are as before.
	if !strings.Contains(string(avBytes), "timestamp: 2026-09-04T00:00:00Z") {
		t.Error("migration touched a timestamp")
	}
	// A second run is a no-op.
	if plan, err := ws.Migrate(false); err != nil || plan.Applied || plan.FormatBefore != model.Format02 {
		t.Errorf("second migrate: %v %+v", err, plan)
	}
}
