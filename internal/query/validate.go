// Package query holds the read-only computations over a loaded workspace:
// validation and, in later changes, resolution and consistency.
package query

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/nodelogicau/intentions-cli/internal/model"
	"github.com/nodelogicau/intentions-cli/internal/projection"
	"github.com/nodelogicau/intentions-cli/internal/store"
	"github.com/nodelogicau/intentions-cli/internal/temporal"
)

// Severity of a finding.
const (
	SeverityError   = "error"
	SeverityWarning = "warning"
	SeverityInfo    = "info"
)

// Finding is one validation result.
type Finding struct {
	Severity string `json:"severity"`
	Code     string `json:"code"`
	Path     string `json:"path,omitempty"`
	ID       string `json:"id,omitempty"`
	Message  string `json:"message"`
}

// Report is the outcome of a validation run.
type Report struct {
	Findings []Finding      `json:"findings"`
	Counts   map[string]int `json:"counts"`
}

// HasErrors reports whether any finding is an error.
func (r Report) HasErrors() bool { return r.Counts[SeverityError] > 0 }

type validator struct {
	ws       *store.Workspace
	g        *store.Graph
	findings []Finding
}

func (v *validator) add(sev, code, id, format string, args ...any) {
	path := ""
	if ld := v.g.Loaded[id]; ld != nil {
		path = ld.Path
	}
	v.findings = append(v.findings, Finding{Severity: sev, Code: code, Path: path, ID: id, Message: fmt.Sprintf(format, args...)})
}

func (v *validator) addPath(sev, code, path, format string, args ...any) {
	v.findings = append(v.findings, Finding{Severity: sev, Code: code, Path: path, Message: fmt.Sprintf(format, args...)})
}

// Validate checks the whole workspace and returns every finding, errors
// first, then warnings, then info, each group in id order.
func Validate(ws *store.Workspace, g *store.Graph) Report {
	v := &validator{ws: ws, g: g}
	v.unreadable()
	v.structural()
	v.referential()
	v.graph()
	v.instances()
	v.versions()
	v.index()
	v.terms()
	v.conventions()
	sort.SliceStable(v.findings, func(i, j int) bool {
		ri, rj := rank(v.findings[i].Severity), rank(v.findings[j].Severity)
		if ri != rj {
			return ri < rj
		}
		if v.findings[i].ID != v.findings[j].ID {
			return v.findings[i].ID < v.findings[j].ID
		}
		return v.findings[i].Code < v.findings[j].Code
	})
	counts := map[string]int{SeverityError: 0, SeverityWarning: 0, SeverityInfo: 0}
	for _, f := range v.findings {
		counts[f.Severity]++
	}
	if v.findings == nil {
		v.findings = []Finding{}
	}
	return Report{Findings: v.findings, Counts: counts}
}

func rank(sev string) int {
	switch sev {
	case SeverityError:
		return 0
	case SeverityWarning:
		return 1
	}
	return 2
}

func (v *validator) unreadable() {
	for _, u := range v.g.Unreadable {
		v.addPath(SeverityError, "unreadable", u.Path, "%s", u.Err)
	}
}

func (v *validator) structural() {
	for _, id := range v.g.Order {
		ld := v.g.Loaded[id]
		obj := v.g.Objects[id]
		if obj.GetID() != ld.FileID {
			v.add(SeverityError, "id_mismatch", id, "file is named %s but carries id %q", ld.FileID, obj.GetID())
		}
		for _, p := range ld.Problems {
			v.add(SeverityError, p.Code, id, "%s", p.Error())
		}
		for _, p := range model.Check(obj) {
			v.add(SeverityError, p.Code, id, "%s", p.Error())
		}
	}
}

func (v *validator) referential() {
	for _, id := range v.g.Order {
		obj := v.g.Objects[id]
		check := func(field, target string, want ...model.Type) {
			if target == "" {
				return
			}
			t, ok := v.g.Get(target)
			if !ok {
				v.add(SeverityError, "dangling", id, "%s references %s, which resolves to no file", field, target)
				return
			}
			if len(want) > 0 {
				okType := false
				for _, w := range want {
					if t.GetType() == w {
						okType = true
					}
				}
				if !okType {
					v.add(SeverityError, "reference_type", id, "%s references %s, a %s", field, target, t.GetType())
				}
			}
		}
		switch o := obj.(type) {
		case *model.Intention:
			for i, r := range o.Serves {
				check(fmt.Sprintf("serves[%d]", i), r.ID, model.TypeIntention)
			}
			if o.FirmedUnder != "" {
				if _, err := model.LookupPolicy(v.g, o.FirmedUnder, o.Subject); err != nil {
					v.add(SeverityError, "firmed_under_target", id, "firmed_under: %v", err)
				} else if p, _ := v.g.Get(o.FirmedUnder); p.(*model.Intention).AutoFirm == nil {
					v.add(SeverityError, "firmed_under_target", id, "firmed_under names %s, which carries no auto_firm condition", o.FirmedUnder)
				}
			}
			if o.Window != nil && o.Window.Relative != nil {
				check("window.relative.target", o.Window.Relative.Target, model.TypeIntention, model.TypeCommitment)
			}
			if o.Retired != nil {
				check("retired.superseded_by", o.Retired.SupersededBy, model.TypeIntention)
			}
		case *model.Availability:
			if o.Window != nil && o.Window.Relative != nil {
				check("window.relative.target", o.Window.Relative.Target, model.TypeIntention, model.TypeCommitment)
			}
			if o.Retired != nil {
				check("retired.superseded_by", o.Retired.SupersededBy, model.TypeAvailability)
			}
		case *model.Commitment:
			check("intention", o.Intention, model.TypeIntention)
			check("origin.resolution", o.Origin.Resolution, model.TypeResolution)
		case *model.Resolution:
			check("intention", o.Intention, model.TypeIntention)
			for i, d := range o.Displaced {
				check(fmt.Sprintf("displaced[%d]", i), d, model.TypeIntention, model.TypeCommitment)
			}
		}
	}
}

// graph reports serves cycles (one error per strongly connected component)
// and the terminus rule.
func (v *validator) graph() {
	for _, scc := range StronglyConnected(v.g) {
		sort.Strings(scc)
		for _, id := range scc {
			v.add(SeverityError, "cycle", id, "in a serves cycle with %s; every member is unresolvable", strings.Join(scc, ", "))
		}
	}
	for _, in := range v.g.Intentions() {
		for _, r := range in.Serves {
			if r.Role != model.RoleForTheSakeOf {
				continue
			}
			target, _ := v.g.Get(r.ID)
			if t, ok := target.(*model.Intention); ok && len(t.Serves) > 0 {
				v.add(SeverityError, "terminus", t.ID, "targeted for-the-sake-of by %s but carries serves entries; a terminus is a sink", in.ID)
			}
		}
	}
}

// StronglyConnected returns every strongly connected component of size > 1
// (or with a self-loop) in the serves graph, using Tarjan's algorithm.
func StronglyConnected(g *store.Graph) [][]string {
	index := 0
	indices := map[string]int{}
	low := map[string]int{}
	onStack := map[string]bool{}
	var stack []string
	var out [][]string
	var strong func(id string)
	strong = func(id string) {
		indices[id], low[id] = index, index
		index++
		stack = append(stack, id)
		onStack[id] = true
		in, _ := g.Get(id)
		if intent, ok := in.(*model.Intention); ok {
			for _, r := range intent.Serves {
				if _, exists := g.Objects[r.ID]; !exists {
					continue
				}
				if _, seen := indices[r.ID]; !seen {
					strong(r.ID)
					low[id] = min(low[id], low[r.ID])
				} else if onStack[r.ID] {
					low[id] = min(low[id], indices[r.ID])
				}
			}
		}
		if low[id] == indices[id] {
			var comp []string
			for {
				n := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				onStack[n] = false
				comp = append(comp, n)
				if n == id {
					break
				}
			}
			selfLoop := false
			if intent, ok := in.(*model.Intention); ok {
				for _, r := range intent.Serves {
					if r.ID == id {
						selfLoop = true
					}
				}
			}
			if len(comp) > 1 || selfLoop {
				out = append(out, comp)
			}
		}
	}
	for _, id := range g.Order {
		if _, ok := g.Objects[id].(*model.Intention); !ok {
			continue
		}
		if _, seen := indices[id]; !seen {
			strong(id)
		}
	}
	return out
}

func (v *validator) instances() {
	ctx := v.ws.Config.Context(time.Now())
	seen := map[string][]string{}
	for _, in := range v.g.Intentions() {
		// A placement must lie within its own window's calendar bounds.
		if in.Placement != nil && in.Window != nil && in.Window.Calendar != nil {
			if ivs, err := temporal.Bounds(temporal.Window{Calendar: in.Window.Calendar}, ctx); err == nil && len(ivs) == 1 && !ivs[0].Contains(in.Placement.Bounds(ctx)) {
				v.add(SeverityError, "placement_outside_window", in.ID, "placement %s lies outside the window %s", in.Placement.Start.Raw, in.Window.Calendar)
			}
		}
		if in.Retired != nil || in.Occurrence == "" {
			continue
		}
		standing := in.InstanceOf()
		if standing == "" {
			continue
		}
		// The occurrence must lie within the recurring intention's calendar anchor.
		if recObj, ok := v.g.Get(standing); ok {
			if rec, ok := recObj.(*model.Intention); ok && rec.Window != nil && rec.Window.Calendar != nil {
				if day, err := temporal.ParseGranule(in.Occurrence); err == nil {
					d := day
					occ := temporal.Calendar{Start: &d, End: &d}
					recIvs, e1 := temporal.Bounds(temporal.Window{Calendar: rec.Window.Calendar}, ctx)
					occIvs, e2 := temporal.Bounds(temporal.Window{Calendar: &occ}, ctx)
					if e1 == nil && e2 == nil && len(recIvs) == 1 && len(occIvs) == 1 && !recIvs[0].Contains(occIvs[0]) {
						v.add(SeverityError, "occurrence_outside_window", in.ID, "occurrence %s lies outside the recurring intention's window %s", in.Occurrence, rec.Window.Calendar)
					}
				}
			}
		}
		key := standing + "@" + in.Occurrence
		seen[key] = append(seen[key], in.ID)
	}
	for _, r := range v.g.Resolutions() {
		if r.Selector == "" || r.Selector == "person" {
			continue
		}
		subject := ""
		if obj, ok := v.g.Get(r.Intention); ok {
			if in, ok := obj.(*model.Intention); ok {
				subject = in.Subject
			}
		}
		p, err := model.LookupPolicy(v.g, r.Selector, subject)
		if err != nil {
			v.add(SeverityError, "selector_not_policy", r.ID, "selector: %v", err)
		} else if p.AutoSelect == nil {
			v.add(SeverityError, "selector_not_policy", r.ID, "selector %s carries no auto_select condition", r.Selector)
		}
	}
	for key, ids := range seen {
		if len(ids) > 1 {
			sort.Strings(ids)
			for _, id := range ids {
				v.add(SeverityError, "duplicate_instance", id, "duplicate active instance for %s (also %s)", key, strings.Join(ids, ", "))
			}
		}
	}
}

func (v *validator) versions() {
	for _, id := range v.g.Order {
		obj := v.g.Objects[id]
		if len(v.g.Loaded[id].Problems) > 0 {
			continue
		}
		cached := obj.CachedVersion()
		if cached == "" {
			v.add(SeverityWarning, "missing_version", id, "no version in the file; every writer stamps one immediately after id, and the next edit will add it")
			continue
		}
		computed, err := projection.Version(obj)
		if err != nil {
			continue
		}
		if cached != computed {
			v.add(SeverityWarning, "stale_version", id, "cached version %s disagrees with computed %s; the computed value is authoritative", cached, computed)
		}
	}
}

func (v *validator) index() {
	committed, err := v.ws.ReadIndex()
	if err != nil {
		v.addPath(SeverityWarning, "index_unreadable", store.IndexFile, "%v; run `intentions index` to regenerate", err)
		return
	}
	d := store.Compare(committed, store.Rebuild(v.g))
	for _, id := range d.Missing {
		v.addPath(SeverityWarning, "index_drift", store.IndexFile, "%s is not in the index; run `intentions index`", id)
	}
	for _, id := range d.Extra {
		v.addPath(SeverityWarning, "index_drift", store.IndexFile, "%s is in the index but has no file; run `intentions index`", id)
	}
	for _, id := range d.Changed {
		v.addPath(SeverityWarning, "index_drift", store.IndexFile, "index entry for %s disagrees with the file; the file wins; run `intentions index`", id)
	}
	for _, e := range committed.Entries {
		known := false
		for _, t := range model.Types {
			if string(t) == e.Type {
				known = true
			}
		}
		if !known {
			v.addPath(SeverityInfo, "index_unknown_type", store.IndexFile, "entry %s has type %q, which this implementation does not know", e.ID, e.Type)
		}
	}
}

func (v *validator) terms() {
	users := map[string][]string{}
	for _, in := range v.g.Intentions() {
		if in.Activity != "" {
			users[in.Activity] = append(users[in.Activity], in.ID)
		}
	}
	for _, av := range v.g.Availabilities() {
		for _, t := range av.Conditional {
			users[t] = append(users[t], av.ID)
		}
	}
	var terms []string
	for t := range users {
		terms = append(terms, t)
	}
	sort.Strings(terms)
	for _, t := range terms {
		if len(users[t]) == 1 {
			v.add(SeverityInfo, "lonely_term", users[t][0], "activity term %q is used by this object only", t)
		}
	}
}

func (v *validator) conventions() {
	if _, err := os.Stat(filepath.Join(v.ws.Root, store.ConventionsFile)); err != nil {
		v.addPath(SeverityInfo, "no_conventions", store.ConventionsFile, "no %s; a prose conventions file helps agents and people agree on activity terms", store.ConventionsFile)
	}
	for _, k := range v.ws.Config.UnknownResolverKeys {
		if k == "week_start" {
			v.addPath(SeverityInfo, "resolver_unknown_key", store.ConfigFile, "resolver.week_start is no longer a configuration key and is ignored; weeks are ISO weeks, Monday to Sunday, in every context")
			continue
		}
		v.addPath(SeverityInfo, "resolver_unknown_key", store.ConfigFile, "resolver.%s is not a configuration key this implementation knows and is ignored", k)
	}
}
