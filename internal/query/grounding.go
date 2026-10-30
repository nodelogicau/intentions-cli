package query

import (
	"fmt"
	"sort"
	"strings"

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

// RefuseRetirement returns the refusal a retirement of terminus o by r must
// make, or nil; anything but a terminus passes. A firm terminus is the
// person's to retire, and a harness may withdraw only a draft it wrote
// itself. Under either format version, no retirement may leave an intention
// that is served now with no firm terminus: the message names those
// intentions and the two ways out, withdrawing the terminus by setting it
// tentative or superseding it with a firm one. edit is how the front end
// spells setting it tentative.
func RefuseRetirement(g *store.Graph, o *model.Intention, r model.Retired, edit string) error {
	if !o.IsTerminus() {
		return nil
	}
	if r.Source.Harness != "" && (o.Stability == "firm" || o.Source.Harness == "") {
		why := "it is firm"
		if o.Stability != "firm" {
			why = "the person wrote it"
		}
		return &model.Refusal{Rule: "harness_retire", Message: fmt.Sprintf("%s is a terminus and %s, so retiring it is the person's act (an author, no harness); a harness may retire only a tentative terminus it drafted itself", o.ID, why)}
	}
	after := *o
	after.Retired = &r
	var lost []string
	for _, in := range g.Intentions() {
		if in.Retired != nil || in.IsTerminus() || in.Subject != o.Subject {
			continue
		}
		if model.Grounding(g, in).Kind == model.Served && model.Grounding(overlay{g, &after}, in).Kind != model.Served {
			lost = append(lost, in.ID)
		}
	}
	if len(lost) == 0 {
		return nil
	}
	sort.Strings(lost)
	successor := "a firm terminus of the subject"
	for _, id := range model.FirmTermini(g.Intentions(), o.Subject) {
		if id != o.ID {
			successor = id
			break
		}
	}
	return &model.Refusal{Rule: "ungrounds", Message: fmt.Sprintf("retiring %s would leave %s with no firm terminus of %s; to withdraw the reason, set it tentative (%s), or to replace it, retire it as superseded by %s", o.ID, strings.Join(lost, ", "), o.Subject, edit, successor)}
}
