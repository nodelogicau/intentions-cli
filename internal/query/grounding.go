package query

import (
	"fmt"

	"github.com/nodelogicau/intentions-cli/internal/model"
	"github.com/nodelogicau/intentions-cli/internal/store"
)

// overlay answers Get for one object with the proposed state and delegates
// everything else to the loaded graph, so a write can be checked as the
// workspace will stand once it lands.
type overlay struct {
	*store.Graph
	obj model.Object
}

func (o overlay) Get(id string) (model.Object, bool) {
	if id == o.obj.GetID() {
		return o.obj, true
	}
	return o.Graph.Get(id)
}

// WriteFindings reports what validate would say about o once it is written
// into g: an unserved warning when it reaches no firm terminus of its own
// subject, a draft info finding when it is a tentative terminus. Nil when
// nothing warrants a finding, so a result can omit the key.
func WriteFindings(g *store.Graph, o *model.Intention) []Finding {
	if o.Retired != nil {
		return nil
	}
	var out []Finding
	if o.IsTerminus() {
		if o.Stability != "firm" {
			out = append(out, Finding{Severity: SeverityInfo, Code: "draft_terminus", ID: o.ID, Message: draftMessage(o.ID)})
		}
		return out
	}
	var termini []string
	for _, id := range model.FirmTermini(g.Intentions(), o.Subject) {
		if id != o.ID {
			termini = append(termini, id)
		}
	}
	if msg := model.Unserved(overlay{g, o}, o, termini); msg != "" {
		out = append(out, Finding{Severity: UnservedSeverity(g.Format), Code: "unserved", ID: o.ID, Message: msg})
	}
	return out
}

// UnservedSeverity is the level an unserved intention is reported at: a
// warning under intentions/0.1, so workspaces written before the rule keep
// validating, and an error under intentions/0.2, where the write is refused.
func UnservedSeverity(format string) string {
	if format == model.Format01 {
		return SeverityWarning
	}
	return SeverityError
}

// RefuseUnserved returns the refusal a write must make when the object it
// would write is unserved and the workspace is intentions/0.2, or nil.
func RefuseUnserved(g *store.Graph, o *model.Intention) error {
	if g.Format == model.Format01 {
		return nil
	}
	for _, f := range WriteFindings(g, o) {
		if f.Code == "unserved" {
			return &model.Refusal{Rule: "unserved", Message: f.Message + " (under " + model.Format02 + " an unserved intention is refused; a 0.1 workspace only warns)"}
		}
	}
	return nil
}

func draftMessage(id string) string {
	return fmt.Sprintf("a draft terminus: it grounds nothing until the person firms it with `intentions intention firm %s` (an author, no harness)", id)
}
