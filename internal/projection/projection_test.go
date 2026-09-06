package projection

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/nodelogicau/intentions-cli/internal/model"
	"github.com/nodelogicau/intentions-cli/internal/temporal"
)

var update = flag.Bool("update", false, "rewrite golden vectors")

func load(t *testing.T, name string, typ model.Type) model.Object {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "model", "testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	obj, probs, err := model.Decode(b, typ)
	if err != nil || len(probs) > 0 {
		t.Fatalf("%s: %v %v", name, err, probs)
	}
	return obj
}

// TestGoldenVectors pins the canonical JSON and version of the spec's example
// objects. The fixtures are suitable for donation to the spec repository as
// conformance vectors: any implementation must produce these bytes.
func TestGoldenVectors(t *testing.T) {
	cases := map[string]model.Type{
		"intention.yaml":    model.TypeIntention,
		"availability.yaml": model.TypeAvailability,
		"commitment.yaml":   model.TypeCommitment,
		"resolution.yaml":   model.TypeResolution,
		"retired.yaml":      model.TypeIntention,
	}
	for name, typ := range cases {
		obj := load(t, name, typ)
		canon, err := Canonical(obj)
		if err != nil {
			t.Fatal(err)
		}
		v, _ := Version(obj)
		if !strings.HasPrefix(v, "sha256:") || len(v) != len("sha256:")+64 {
			t.Errorf("%s: version shape %s", name, v)
		}
		base := strings.TrimSuffix(name, ".yaml")
		jsonPath := filepath.Join("testdata", base+".canonical.json")
		verPath := filepath.Join("testdata", base+".version")
		if *update {
			_ = os.MkdirAll("testdata", 0o755)
			_ = os.WriteFile(jsonPath, canon, 0o644)
			_ = os.WriteFile(verPath, []byte(v+"\n"), 0o644)
			continue
		}
		wantJSON, err := os.ReadFile(jsonPath)
		if err != nil {
			t.Fatalf("%s: missing golden vector; run with -update", name)
		}
		if string(wantJSON) != string(canon) {
			t.Errorf("%s: canonical JSON\n got %s\nwant %s", name, canon, wantJSON)
		}
		wantV, _ := os.ReadFile(verPath)
		if strings.TrimSpace(string(wantV)) != v {
			t.Errorf("%s: version %s want %s", name, v, strings.TrimSpace(string(wantV)))
		}
	}
}

func base() *model.Intention {
	d, _ := temporal.ParseDurationSpec("PT1H")
	c, _ := temporal.ParseCalendar("2026-09")
	return &model.Intention{ID: "int_a", Subject: "https://s/", Title: "t", Description: "d", Duration: &d,
		Window: &temporal.Window{Calendar: &c}, Stability: "tentative", Location: []string{"https://b/", "https://a/"},
		Serves: []model.Ref{}, Source: model.Source{Author: "a", Model: "m"}, Timestamp: time.Now(), Acknowledgements: []model.Acknowledgement{}}
}

func TestProjectionScenarios(t *testing.T) {
	v0 := MustVersion(base())

	o := base()
	o.Description = "changed"
	if MustVersion(o) != v0 {
		t.Error("prose edit changed version")
	}
	o = base()
	o.Source.Model = "other"
	o.Timestamp = time.Now().Add(time.Hour)
	if MustVersion(o) != v0 {
		t.Error("source or timestamp edit changed version")
	}
	o = base()
	o.Acknowledgements = []model.Acknowledgement{{Kind: "window-clash"}}
	if MustVersion(o) != v0 {
		t.Error("acknowledgement changed version")
	}
	o = base()
	w, _ := temporal.ParseCalendar("2026-W37")
	o.Window = &temporal.Window{Calendar: &w}
	if MustVersion(o) == v0 {
		t.Error("window edit kept version")
	}
	o = base()
	o.Retired = &model.Retired{Kind: "fulfilled", Reason: "done", Timestamp: time.Now()}
	v1 := MustVersion(o)
	if v1 == v0 {
		t.Error("retirement kept version")
	}
	o.Retired.Reason = "different reason"
	if MustVersion(o) != v1 {
		t.Error("retirement reason changed version")
	}
	// Equivalent durations agree; days and hours do not.
	o = base()
	d60, _ := temporal.ParseDurationSpec("PT60M")
	o.Duration = &d60
	if MustVersion(o) != v0 {
		t.Error("PT60M and PT1H differ")
	}
	a, b := base(), base()
	d1, _ := temporal.ParseDurationSpec("P1D")
	d24, _ := temporal.ParseDurationSpec("PT24H")
	a.Duration, b.Duration = &d1, &d24
	if MustVersion(a) == MustVersion(b) {
		t.Error("P1D and PT24H agree")
	}
	// List order is irrelevant.
	o = base()
	o.Location = []string{"https://a/", "https://b/"}
	if MustVersion(o) != v0 {
		t.Error("location order changed version")
	}
	// Explicit transparent: false is omitted; true is not.
	f, tr := false, true
	s, _ := temporal.ParsePlacementStart("2026-09-15T10:00:00+10:00")
	pl := &temporal.Placement{Start: s, Duration: d}
	c1 := &model.Commitment{ID: "cmt_a", Parties: []model.Party{{URI: "https://x/", Status: "accepted"}}, Placement: pl, Origin: model.Origin{Import: true}}
	c2 := *c1
	c2.Transparent = &f
	if MustVersion(c1) != MustVersion(&c2) {
		t.Error("transparent: false changed version")
	}
	c3 := *c1
	c3.Transparent = &tr
	if MustVersion(c1) == MustVersion(&c3) {
		t.Error("transparent: true kept version")
	}
	// Offset datetime normalised to UTC.
	canon, _ := Canonical(c1)
	if !strings.Contains(string(canon), `"start":"2026-09-15T00:00:00Z"`) {
		t.Errorf("start not UTC: %s", canon)
	}
}

var d, _ = temporal.ParseDurationSpec("PT1H")
