package store

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/nodelogicau/intentions-cli/internal/apperr"
	"github.com/nodelogicau/intentions-cli/internal/model"
	"github.com/nodelogicau/intentions-cli/internal/projection"
)

// ObjectChange is one object a migration rewrites: its version before and
// after, and whether its file changes at all.
type ObjectChange struct {
	ID            string `json:"id"`
	Type          string `json:"type"`
	VersionBefore string `json:"version_before"`
	VersionAfter  string `json:"version_after"`
	Rewritten     bool   `json:"rewritten"`
}

// AckChange is one acknowledgement whose counterpart_version a migration
// carries across, so that nothing lapses for a change of shape.
type AckChange struct {
	ID          string `json:"id"`
	Counterpart string `json:"counterpart"`
	Before      string `json:"before"`
	After       string `json:"after"`
}

// Plan is what a migration would do, or did.
type Plan struct {
	FormatBefore     string         `json:"format_before"`
	FormatAfter      string         `json:"format_after"`
	Unserved         []string       `json:"unserved,omitempty"`
	Objects          []ObjectChange `json:"objects"`
	Acknowledgements []AckChange    `json:"acknowledgements"`
	Applied          bool           `json:"applied"`
}

// Rewritten counts the objects whose file changes.
func (p Plan) Rewritten() int {
	n := 0
	for _, o := range p.Objects {
		if o.Rewritten {
			n++
		}
	}
	return n
}

// Migrate moves the workspace from intentions/0.1 to intentions/0.2, the
// person's act. It refuses while any intention is unserved under the 0.2
// rule, naming them. Otherwise, in order: every object is re-projected and
// re-encoded under 0.2 (the origin shape, the availability names, every
// version); every acknowledgement whose counterpart_version equalled its
// counterpart's version before the migration is carried to the new one,
// an already-lapsed one left alone; the index is rebuilt; and format is
// rewritten last, so an interruption leaves a 0.1 workspace the next run
// finishes. With check set nothing is written and the plan is returned.
func (w *Workspace) Migrate(check bool) (Plan, error) {
	plan := Plan{FormatBefore: w.Config.Format, FormatAfter: model.Format02}
	if w.Config.Format == model.Format02 {
		return plan, nil
	}
	if w.Config.Format != model.Format01 {
		return plan, apperr.Refused("%s is %s; this binary migrates only from %s", ConfigFile, w.Config.Format, model.Format01)
	}
	g, err := w.Load()
	if err != nil {
		return plan, err
	}
	if len(g.Unreadable) > 0 {
		return plan, apperr.Refused("%s cannot be read: %s; fix the file before migrating", g.Unreadable[0].Path, g.Unreadable[0].Err)
	}
	// The 0.2 rule: every intention reaches a firm terminus. Migration is
	// refused while any does not, so the walk-up comes first.
	termini := map[string][]string{}
	for _, in := range g.Intentions() {
		if in.Retired != nil || in.IsTerminus() {
			continue
		}
		if _, ok := termini[in.Subject]; !ok {
			termini[in.Subject] = model.FirmTermini(g.Intentions(), in.Subject)
		}
		if msg := model.Unserved(g, in, termini[in.Subject]); msg != "" {
			plan.Unserved = append(plan.Unserved, in.ID)
		}
	}
	sort.Strings(plan.Unserved)
	if len(plan.Unserved) > 0 {
		return plan, apperr.Refused("%d intention(s) reach no firm terminus, which %s refuses: %s. Walk them up first (`intentions validate` names the fix for each), then migrate", len(plan.Unserved), model.Format02, strings.Join(plan.Unserved, ", "))
	}
	// Versions before, bytes before; then everything under 0.2.
	before := map[string]string{}
	bytesBefore := map[string][]byte{}
	for _, id := range g.Order {
		obj := g.Objects[id]
		before[id] = projection.MustVersion(obj)
		bytesBefore[id], _ = model.Encode(obj)
		obj.SetFormat(model.Format02)
	}
	after := map[string]string{}
	for _, id := range g.Order {
		after[id] = projection.MustVersion(g.Objects[id])
	}
	carry := func(id string, acks []model.Acknowledgement) {
		for i := range acks {
			a := &acks[i]
			if a.Counterpart == "" || a.CounterpartVersion == "" {
				continue
			}
			if a.CounterpartVersion == before[a.Counterpart] && before[a.Counterpart] != after[a.Counterpart] {
				plan.Acknowledgements = append(plan.Acknowledgements, AckChange{ID: id, Counterpart: a.Counterpart, Before: a.CounterpartVersion, After: after[a.Counterpart]})
				a.CounterpartVersion = after[a.Counterpart]
			}
		}
	}
	for _, id := range g.Order {
		switch o := g.Objects[id].(type) {
		case *model.Intention:
			carry(id, o.Acknowledgements)
		case *model.Commitment:
			carry(id, o.Acknowledgements)
		}
	}
	for _, id := range g.Order {
		obj := g.Objects[id]
		obj.SetVersion(after[id])
		data, err := model.Encode(obj)
		if err != nil {
			return plan, err
		}
		plan.Objects = append(plan.Objects, ObjectChange{ID: id, Type: string(obj.GetType()), VersionBefore: before[id], VersionAfter: after[id], Rewritten: string(data) != string(bytesBefore[id])})
	}
	if check {
		return plan, nil
	}
	// Write: objects under 0.2, the index, and format last.
	w.Config.Format = model.Format02
	g.Format = model.Format02
	for _, c := range plan.Objects {
		if !c.Rewritten {
			continue
		}
		obj := g.Objects[c.ID]
		data, err := model.Encode(obj)
		if err != nil {
			return plan, err
		}
		if err := atomicWrite(w.Path(obj.GetType(), obj.GetID()), data); err != nil {
			return plan, err
		}
	}
	if err := w.WriteIndex(Rebuild(g)); err != nil {
		return plan, err
	}
	if err := w.WriteConfig(); err != nil {
		return plan, err
	}
	plan.Applied = true
	return plan, nil
}

// WriteConfig writes intentions.yaml from the in-memory configuration,
// preserving keys this implementation does not know.
func (w *Workspace) WriteConfig() error {
	data, err := w.Config.Marshal()
	if err != nil {
		return err
	}
	return atomicWrite(filepath.Join(w.Root, ConfigFile), data)
}

// String summarises the plan for text output.
func (p Plan) String() string {
	var b strings.Builder
	if p.FormatBefore == p.FormatAfter {
		fmt.Fprintf(&b, "already %s; nothing to do\n", p.FormatAfter)
		return b.String()
	}
	verb := "would rewrite"
	if p.Applied {
		verb = "rewrote"
	}
	fmt.Fprintf(&b, "%s -> %s: %s %d of %d object files, carried %d acknowledgement(s) across\n", p.FormatBefore, p.FormatAfter, verb, p.Rewritten(), len(p.Objects), len(p.Acknowledgements))
	for _, o := range p.Objects {
		if o.Rewritten {
			fmt.Fprintf(&b, "  %s  %s -> %s\n", o.ID, short(o.VersionBefore), short(o.VersionAfter))
		}
	}
	for _, a := range p.Acknowledgements {
		fmt.Fprintf(&b, "  %s acknowledges %s  %s -> %s\n", a.ID, a.Counterpart, short(a.Before), short(a.After))
	}
	return b.String()
}

func short(v string) string {
	if len(v) > 19 {
		return v[:19] + "…"
	}
	return v
}
