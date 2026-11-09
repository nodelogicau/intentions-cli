package importer

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/nodelogicau/intentions-cli/internal/model"
	"github.com/nodelogicau/intentions-cli/internal/projection"
	"github.com/nodelogicau/intentions-cli/internal/store"
	"github.com/nodelogicau/intentions-cli/internal/temporal"
)

const ben = "https://example.org/people/ben"

func newWS(t *testing.T) *store.Workspace {
	t.Helper()
	cfg := store.NewConfig()
	cfg.Format = model.Format02
	cfg.Resolver.Timezone = "Australia/Melbourne"
	cfg.Defaults.Subject = ben
	cfg.Defaults.Source.Author = ben
	cfg.Calendar.Addresses = map[string][]string{ben: {"mailto:ben@example.org"}}
	ws, _, err := store.Init(t.TempDir(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	return ws
}

func event(t *testing.T, js string) Event {
	t.Helper()
	ev, err := Decode([]byte(js))
	if err != nil {
		t.Fatalf("decode %s: %v", js, err)
	}
	return ev
}

func parties(t *testing.T, ws *store.Workspace, js string) []model.Party {
	t.Helper()
	return Parties(event(t, js), ben, ws.Config.SubjectFor, store.NormaliseAddress)
}

func TestParties(t *testing.T) {
	ws := newWS(t)
	head := `{"system":"icalendar","uid":"u","start":"2026-09-15T10:00:00+10:00","duration":"PT1H"`
	for name, tc := range map[string]struct {
		js   string
		want []model.Party
	}{
		"meeting": {head + `,"organizer":{"address":"mailto:Alex@Example.com"},"attendees":[{"address":"mailto:ben@example.org","status":"ACCEPTED"},{"address":"mailto:sam@example.com","status":"NEEDS-ACTION"}]}`,
			[]model.Party{{URI: ben, Status: "accepted"}, {URI: "mailto:alex@example.com", Status: "accepted"}, {URI: "mailto:sam@example.com", Status: "tentative"}}},
		"ticket": {head + `}`, []model.Party{{URI: ben, Status: "accepted"}}},
		"shared calendar": {head + `,"attendees":[{"address":"mailto:sam@example.com","status":"accepted"}]}`,
			[]model.Party{{URI: ben, Status: "tentative"}, {URI: "mailto:sam@example.com", Status: "accepted"}}},
		"non-participant and delegate": {head + `,"attendees":[{"address":"mailto:ben@example.org","status":"ACCEPTED"},{"address":"mailto:cc@example.com","non_participant":true},{"address":"mailto:a@example.com","status":"DELEGATED"},{"address":"mailto:b@example.com","status":"NEEDS-ACTION"}]}`,
			[]model.Party{{URI: ben, Status: "accepted"}, {URI: "mailto:a@example.com", Status: "declined"}, {URI: "mailto:b@example.com", Status: "tentative"}}},
		"organizer also attends": {head + `,"organizer":{"address":"mailto:alex@example.com"},"attendees":[{"address":"mailto:ALEX@example.com","status":"TENTATIVE"},{"address":"mailto:ben@example.org","status":"declined"}]}`,
			[]model.Party{{URI: ben, Status: "declined"}, {URI: "mailto:alex@example.com", Status: "tentative"}}},
		"subject by URI": {head + `,"attendees":[{"address":"` + ben + `","status":"ACCEPTED"},{"address":"mailto:sam@example.com"}]}`,
			[]model.Party{{URI: ben, Status: "accepted"}, {URI: "mailto:sam@example.com", Status: "tentative"}}},
		"no address": {head + `,"attendees":[{"address":"","status":"ACCEPTED"}]}`, []model.Party{{URI: ben, Status: "tentative"}}},
	} {
		if got := parties(t, ws, tc.js); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%s: %v, want %v", name, got, tc.want)
		}
	}
}

func TestDecode(t *testing.T) {
	for name, js := range map[string]string{
		"unknown key":  `{"system":"icalendar","uid":"u","start":"2026-09-15T10:00:00+10:00","duration":"PT1H","cn":"x"}`,
		"no uid":       `{"system":"icalendar","start":"2026-09-15T10:00:00+10:00","duration":"PT1H"}`,
		"system":       `{"system":"outlook","uid":"u","start":"2026-09-15T10:00:00+10:00","duration":"PT1H"}`,
		"text place":   `{"system":"icalendar","uid":"u","start":"2026-09-15T10:00:00+10:00","duration":"PT1H","location":"Cafe on the corner"}`,
		"status":       `{"system":"icalendar","uid":"u","start":"2026-09-15T10:00:00+10:00","duration":"PT1H","attendees":[{"address":"mailto:a@b.c","status":"MAYBE"}]}`,
		"all-day time": `{"system":"icalendar","uid":"u","start":"2026-09-21","duration":"PT1H"}`,
		"start":        `{"system":"icalendar","uid":"u","start":"tomorrow","duration":"PT1H"}`,
	} {
		if _, err := Decode([]byte(js)); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	ev := event(t, `{"system":"jscalendar","uid":"u","start":"2026-09-21","duration":"P5D","transparent":true}`)
	if p := ev.Placement(); !p.Start.AllDay || p.Duration.String() != "P5D" {
		t.Errorf("all-day: %+v", p)
	}
}

func TestImport(t *testing.T) {
	ws := newWS(t)
	g, err := ws.Load()
	if err != nil {
		t.Fatal(err)
	}
	opts := Options{Subject: ben, Source: model.Source{Author: ben, Harness: "claude"}, Timestamp: time.Date(2026, 9, 8, 4, 0, 0, 0, time.UTC)}
	imp := func(js string) Result {
		t.Helper()
		r, err := Import(ws, g, event(t, js), opts)
		if err != nil {
			t.Fatalf("import %s: %v", js, err)
		}
		return r
	}
	meeting := `{"system":"icalendar","uid":"m","start":"2026-09-15T10:00:00+10:00","duration":"PT1H","title":"Planning","attendees":[{"address":"mailto:ben@example.org","status":"ACCEPTED"},{"address":"mailto:sam@example.com","status":"NEEDS-ACTION"}]}`
	r := imp(meeting)
	if r.Outcome != Created || r.Commitment.Origin != model.OriginImport || r.Commitment.Intention != "" || r.Commitment.External.UID != "m" || len(r.Commitment.Parties) != 2 {
		t.Fatalf("created: %+v", r.Commitment)
	}
	id := r.Commitment.ID
	if r := imp(meeting); r.Outcome != Unchanged || r.Commitment.ID != id {
		t.Errorf("unchanged: %+v", r)
	}

	// A reply arrives by re-import: the version moves, the adoption and the
	// acknowledgements stay.
	c := *r.Commitment
	c.Intention = "int_01a06d10-4c2e-7a91-b3f0-2d8e1a7c5b44"
	c.Acknowledgements = []model.Acknowledgement{{Kind: "window-clash", Counterpart: "int_01a06d10-4c2e-7a91-b3f0-2d8e1a7c5b44", CounterpartVersion: "sha256:x", Source: model.Source{Author: ben}, Timestamp: opts.Timestamp}}
	if err := ws.WriteObject(&c); err != nil {
		t.Fatal(err)
	}
	g.Add(&c)
	r = imp(strings.Replace(meeting, `"NEEDS-ACTION"`, `"ACCEPTED"`, 1))
	if r.Outcome != Updated || r.Commitment.ID != id || r.Commitment.Intention != c.Intention || len(r.Commitment.Acknowledgements) != 1 || r.PreviousVersion == r.Commitment.Version {
		t.Errorf("updated: %+v", r)
	}
	// A moved single event updates the same commitment.
	if r := imp(strings.Replace(meeting, "10:00:00", "11:00:00", 1)); r.Outcome != Updated || r.Commitment.ID != id || r.Commitment.Placement.Start.Raw != "2026-09-15T11:00:00+10:00" {
		t.Errorf("moved: %+v", r)
	}

	// Recurring occurrences: new by start, moved by recurrence_id.
	occ := func(start, rid string) string {
		return `{"system":"icalendar","uid":"w","recurrence_id":"` + rid + `","start":"` + start + `","duration":"PT1H"}`
	}
	a := imp(occ("2026-09-16T09:00:00+10:00", "2026-09-16T09:00:00+10:00"))
	b := imp(occ("2026-09-23T09:00:00+10:00", "2026-09-23T09:00:00+10:00"))
	if a.Outcome != Created || b.Outcome != Created || a.Commitment.ID == b.Commitment.ID {
		t.Fatalf("occurrences: %v %v", a.Outcome, b.Outcome)
	}
	moved := imp(occ("2026-09-24T09:00:00+10:00", "2026-09-23T09:00:00+10:00"))
	if moved.Outcome != Updated || moved.Commitment.ID != b.Commitment.ID {
		t.Errorf("moved occurrence: %+v", moved)
	}
	if again := imp(occ("2026-09-24T09:00:00+10:00", "2026-09-23T09:00:00+10:00")); again.Outcome != Unchanged || again.Commitment.ID != b.Commitment.ID {
		t.Errorf("moved occurrence found at its new start: %+v", again)
	}
	// Without recurrence_id, two carriers and no start match is ambiguous.
	if _, err := Import(ws, g, event(t, `{"system":"icalendar","uid":"w","start":"2026-10-01T09:00:00+10:00","duration":"PT1H"}`), opts); err == nil ||
		!strings.Contains(err.Error(), a.Commitment.ID) || !strings.Contains(err.Error(), "--commitment") || !strings.Contains(err.Error(), "recurrence_id") {
		t.Errorf("ambiguous: %v", err)
	}
	named := opts
	named.Commitment = a.Commitment.ID
	if r, err := Import(ws, g, event(t, `{"system":"icalendar","uid":"w","start":"2026-10-01T09:00:00+10:00","duration":"PT1H"}`), named); err != nil || r.Commitment.ID != a.Commitment.ID {
		t.Errorf("--commitment: %v %v", r, err)
	}
	named.Commitment = id
	if _, err := Import(ws, g, event(t, `{"system":"icalendar","uid":"w","start":"2026-10-01T09:00:00+10:00","duration":"PT1H"}`), named); err == nil {
		t.Error("--commitment naming another uid was accepted")
	}

	// A cancelled event stays cancelled.
	cancelled := *a.Commitment
	cancelled.Retired = &model.Retired{Kind: "cancelled", Source: model.Source{Author: ben}, Timestamp: opts.Timestamp}
	if err := ws.WriteObject(&cancelled); err != nil {
		t.Fatal(err)
	}
	g.Add(&cancelled)
	// cancelled is occurrence a, written back at its original start.
	if r := imp(occ("2026-09-16T09:00:00+10:00", "2026-09-16T09:00:00+10:00")); r.Outcome != Retired || r.Commitment.ID != a.Commitment.ID {
		t.Errorf("retired: %+v", r)
	}

	// The intention's own hold creates nothing.
	s, _ := temporal.ParsePlacementStart("2026-09-17T09:00:00+10:00")
	d, _ := temporal.ParseDurationSpec("PT1H")
	hold := &model.Intention{ID: model.MintID(model.TypeIntention), Subject: ben, Title: "Write", Duration: &d, Stability: "tentative",
		Placement: &temporal.Placement{Start: s, Duration: d}, External: &model.External{System: "icalendar", UID: "h"},
		Source: model.Source{Author: ben}, Timestamp: opts.Timestamp, Acknowledgements: []model.Acknowledgement{}, Serves: []model.Ref{}}
	g.Add(hold)
	if r := imp(`{"system":"icalendar","uid":"h","start":"2026-09-17T09:00:00+10:00","duration":"PT1H"}`); r.Outcome != Recognised || r.Intention != hold.ID || r.Commitment != nil {
		t.Errorf("recognised: %+v", r)
	}
}

func TestTwoImportersAgree(t *testing.T) {
	js := `{"system":"icalendar","uid":"m","start":"2026-09-15T10:00:00+10:00","duration":"PT1H","organizer":{"address":"mailto:Alex@Example.com"},"attendees":[{"address":"mailto:ben@example.org","status":"ACCEPTED"}]}`
	var versions []string
	for i := 0; i < 2; i++ {
		ws := newWS(t)
		g, _ := ws.Load()
		r, err := Import(ws, g, event(t, js), Options{Subject: ben, Source: model.Source{Author: ben}, Timestamp: time.Date(2026, 9, 8, 4, 0, 0, 0, time.UTC)})
		if err != nil {
			t.Fatal(err)
		}
		v, _ := projection.Version(r.Commitment)
		versions = append(versions, v)
	}
	if versions[0] != versions[1] {
		t.Errorf("versions differ: %v", versions)
	}
}
