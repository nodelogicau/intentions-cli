package query

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/nodelogicau/intentions-cli/internal/model"
)

func desire(id string, serves ...model.Ref) *model.Desire {
	return &model.Desire{ID: id, Subject: "https://example.com/people/ada", Title: "want", Serves: append([]model.Ref{}, serves...),
		Source: model.Source{Author: "https://example.com/people/ada"}, Timestamp: time.Now()}
}

func TestDesireValidation(t *testing.T) {
	ws := newWS(t)
	draft := intention("int_d")
	sched := withDuration(intention("int_s"))
	priya := firmTerminus("int_p")
	priya.Subject = "https://example.com/people/priya"
	write(t, ws, draft, sched, priya,
		desire("des_ok", model.Ref{ID: "int_d", Role: model.RoleForTheSakeOf}),
		desire("des_bare"),
		desire("des_sched", model.Ref{ID: "int_s", Role: model.RoleForTheSakeOf}),
		desire("des_priya", model.Ref{ID: "int_p", Role: model.RoleForTheSakeOf}),
		desire("des_two"))
	adopted := desire("des_adopted")
	adopted.Retired = &model.Retired{Kind: "adopted", AdoptedAs: "int_s", Source: model.Source{Author: "a"}, Timestamp: time.Now()}
	badAdopt := desire("des_badadopt")
	badAdopt.Retired = &model.Retired{Kind: "adopted", AdoptedAs: "des_ok", Source: model.Source{Author: "a"}, Timestamp: time.Now()}
	badSup := desire("des_badsup")
	badSup.Retired = &model.Retired{Kind: "superseded", SupersededBy: "int_s", Source: model.Source{Author: "a"}, Timestamp: time.Now()}
	write(t, ws, adopted, badAdopt, badSup)
	// A raw file with a forbidden field.
	_ = os.WriteFile(filepath.Join(ws.Root, "desires", "des_win.yaml"), []byte("id: des_win\nsubject: https://example.com/people/ada\ntitle: t\nserves: []\nwindow: {calendar: 2026-W40}\nsource: {author: a}\ntimestamp: 2026-09-04T00:00:00Z\n"), 0o644)
	r := run(t, ws)
	for id, want := range map[string]string{
		"des_ok":       "",
		"des_bare":     "",
		"des_two":      "",
		"des_adopted":  "",
		"des_sched":    "serves_target",
		"des_priya":    "serves_target",
		"des_badadopt": "reference_type",
		"des_badsup":   "reference_type",
		"des_win":      "forbidden_field",
	} {
		f := findingsFor(r, id)
		delete(f, "missing_version")
		if want == "" && len(f) != 0 {
			t.Errorf("%s: unexpected %v", id, f)
		}
		if want != "" && !strings.HasPrefix(f[want], "error:") {
			t.Errorf("%s: want %s, got %v", id, want, f)
		}
	}
	// Desires are never unserved (int_s, a scheduled intention, rightly is);
	// the draft terminus is still a draft.
	for _, f := range r.Findings {
		if f.Code == "unserved" && strings.HasPrefix(f.ID, "des_") {
			t.Errorf("grounding leaked onto a desire: %+v", f)
		}
	}
	if !has(r, SeverityWarning, "unserved") || !has(r, SeverityInfo, "draft_terminus") {
		t.Errorf("expected int_s unserved and int_d a draft: %v", r.Findings)
	}
	// The terminus a desire targets is a sink.
	draft.Serves = []model.Ref{{ID: "int_s", Role: model.RoleInOrderTo}}
	write(t, ws, draft)
	if f := findingsFor(run(t, ws), "int_d"); !strings.HasPrefix(f["terminus"], "error:") {
		t.Errorf("sink rule via desire: %v", f)
	}
}
