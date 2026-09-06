package model

import (
	"fmt"
	"strings"

	"github.com/nodelogicau/intentions-cli/internal/temporal"
)

// Write policy: the rules that decide whether a proposed edit is admissible
// before the codec runs. Anything here that needs the rest of the workspace
// takes a Graph. Every refusal is also reimplemented as a validate finding,
// because files arrive by merge without passing through a writer.

// Inbound is a reference pointing at an object: who references it and how.
type Inbound struct {
	From string
	Role string
}

// Graph is the loaded workspace as the policy needs it.
type Graph interface {
	Get(id string) (Object, bool)
	Inbound(id string) []Inbound
}

// Refusal is a write the format rules do not admit.
type Refusal struct {
	Rule    string
	Message string
}

func (r *Refusal) Error() string { return r.Message }

func refuse(rule, format string, args ...any) error {
	return &Refusal{Rule: rule, Message: fmt.Sprintf(format, args...)}
}

// CheckNotRetired refuses any edit to a retired object.
func CheckNotRetired(obj Object) error {
	if r := obj.GetRetired(); r != nil {
		return refuse("retired", "%s is retired (%s) and may not be edited; only an acknowledgement may still be appended", obj.GetID(), r.Kind)
	}
	return nil
}

// CheckRetirement refuses a second retirement and a malformed record.
func CheckRetirement(g Graph, obj Object, r Retired) error {
	if obj.GetRetired() != nil {
		return refuse("retired", "%s is already retired (%s)", obj.GetID(), obj.GetRetired().Kind)
	}
	kinds := RetirementKinds(obj.GetType())
	if !oneOf(r.Kind, kinds) {
		return refuse("retirement_kind", "%q is not a retirement kind for %s; admitted kinds are %s", r.Kind, obj.GetType(), strings.Join(kinds, ", "))
	}
	switch {
	case r.Kind == "superseded" && r.SupersededBy == "":
		return refuse("superseded_by", "kind superseded requires --superseded-by")
	case r.Kind != "superseded" && r.SupersededBy != "":
		return refuse("superseded_by", "--superseded-by is admitted only with kind superseded")
	}
	if r.SupersededBy != "" {
		target, ok := g.Get(r.SupersededBy)
		if !ok {
			return refuse("dangling", "superseded_by %s does not exist", r.SupersededBy)
		}
		if target.GetType() != obj.GetType() {
			return refuse("superseded_by", "superseded_by %s is a %s, not a %s", r.SupersededBy, target.GetType(), obj.GetType())
		}
		if r.SupersededBy == obj.GetID() {
			return refuse("superseded_by", "an object cannot supersede itself")
		}
	}
	return nil
}

// CheckServes refuses serves entries that dangle, close a cycle, or break the
// terminus rule, for intention self with the proposed list.
func CheckServes(g Graph, self string, serves []Ref) error {
	for _, r := range serves {
		if !oneOf(r.Role, Roles) {
			return refuse("serves_role", "serves role %q is not admitted; use %s", r.Role, strings.Join(Roles, ", "))
		}
		target, ok := g.Get(r.ID)
		if !ok {
			return refuse("dangling", "serves target %s does not exist", r.ID)
		}
		if target.GetType() != TypeIntention {
			return refuse("serves_target", "serves target %s is a %s; only intentions are served", r.ID, target.GetType())
		}
		if r.ID == self {
			return refuse("cycle", "%s cannot serve itself", self)
		}
		if r.Role == RoleForTheSakeOf {
			if t := target.(*Intention); len(t.Serves) > 0 {
				return refuse("terminus", "%s is targeted for-the-sake-of but carries serves entries of its own; a terminus is a sink", r.ID)
			}
		}
	}
	if len(serves) > 0 {
		for _, in := range g.Inbound(self) {
			if in.Role == RoleForTheSakeOf {
				return refuse("terminus", "%s is a terminus (targeted for-the-sake-of by %s) and cannot carry serves entries", self, in.From)
			}
		}
	}
	if path := findCycle(g, self, serves); path != nil {
		return refuse("cycle", "serves would close a cycle: %s", strings.Join(path, " -> "))
	}
	return nil
}

// findCycle walks outbound serves edges from each proposed target looking for
// self. It returns the path when found.
func findCycle(g Graph, self string, serves []Ref) []string {
	visited := map[string]bool{}
	var path []string
	var walk func(id string) bool
	walk = func(id string) bool {
		if id == self {
			return true
		}
		if visited[id] {
			return false
		}
		visited[id] = true
		obj, ok := g.Get(id)
		if !ok {
			return false
		}
		in, ok := obj.(*Intention)
		if !ok {
			return false
		}
		for _, r := range in.Serves {
			path = append(path, r.ID)
			if walk(r.ID) {
				return true
			}
			path = path[:len(path)-1]
		}
		return false
	}
	for _, r := range serves {
		path = []string{self, r.ID}
		if walk(r.ID) {
			return path
		}
	}
	return nil
}

// CheckAvailabilityTerms refuses an in-place change to the fields that make
// an availability a different disposition.
func CheckAvailabilityTerms(old, edited *Availability) error {
	changed := func(name string, a, b string) error {
		if a != b {
			return refuse("terms", "%s is a term of the availability and cannot change in place; use `intentions availability supersede %s` to create a new one and retire this", name, old.ID)
		}
		return nil
	}
	oldW, newW := windowString(old.Window), windowString(edited.Window)
	if err := changed("window", oldW, newW); err != nil {
		return err
	}
	if err := changed("duration", durationString(old.Duration), durationString(edited.Duration)); err != nil {
		return err
	}
	if err := changed("conditional", strings.Join(SortStrings(old.Conditional), ","), strings.Join(SortStrings(edited.Conditional), ",")); err != nil {
		return err
	}
	return changed("location", strings.Join(SortStrings(old.Location), ","), strings.Join(SortStrings(edited.Location), ","))
}

// CheckScope refuses a narrowing.
func CheckScope(old, edited string) error {
	if old == edited {
		return nil
	}
	rank := func(s string) int {
		for i, x := range Scopes {
			if x == s {
				return i
			}
		}
		return -1
	}
	if rank(edited) < rank(old) {
		return refuse("scope", "scope is only ever widened: %s cannot become %s", old, edited)
	}
	return nil
}

// CheckSubjectUnchanged refuses a subject change on edit.
func CheckSubjectUnchanged(old, edited string) error {
	if old != edited {
		return refuse("subject", "subject cannot change: an intention is the subject's own; write a new one")
	}
	return nil
}

// Satisfies reports whether an intention meets every term a policy states.
func Satisfies(p *Policy, target *Intention) error {
	if p == nil {
		return fmt.Errorf("no condition")
	}
	if p.MaxDuration != nil {
		if target.Duration == nil {
			return fmt.Errorf("max_duration %s: the intention has no duration", *p.MaxDuration)
		}
		if target.Duration.Nominal.ApproxSeconds() > p.MaxDuration.ApproxSeconds() {
			return fmt.Errorf("max_duration %s: the intention's duration is %s", *p.MaxDuration, target.Duration.Nominal)
		}
	}
	if p.Stability != "" && target.Stability != p.Stability {
		return fmt.Errorf("stability %s: the intention is %s", p.Stability, target.Stability)
	}
	return nil
}

// CheckFirm applies the harness boundary: firm may be set by an act with no
// harness, or by a harness under a policy of the subject that the intention
// satisfies. target is the intention as it stands before the act.
func CheckFirm(g Graph, target *Intention, act Source, policyID string) error {
	if act.Harness == "" {
		return nil
	}
	if policyID == "" {
		return refuse("harness_firm", "a harness may draft but may not make an intention firm without a policy the person holds; pass --policy <int_id> naming an intention of the subject carrying auto_firm")
	}
	obj, ok := g.Get(policyID)
	if !ok {
		return refuse("dangling", "policy %s does not exist", policyID)
	}
	policy, ok := obj.(*Intention)
	if !ok {
		return refuse("policy", "policy %s is not an intention", policyID)
	}
	if policy.Retired != nil {
		return refuse("policy", "policy %s is retired", policyID)
	}
	if policy.Subject != target.Subject {
		return refuse("policy", "policy %s belongs to %s, not to %s", policyID, policy.Subject, target.Subject)
	}
	if policy.AutoFirm == nil {
		return refuse("policy", "policy %s carries no auto_firm condition", policyID)
	}
	if err := Satisfies(policy.AutoFirm, target); err != nil {
		return refuse("policy", "policy %s does not cover %s: %v", policyID, target.ID, err)
	}
	return nil
}

func windowString(w *temporal.Window) string {
	if w == nil {
		return ""
	}
	return w.String()
}

func durationString(d *temporal.DurationSpec) string {
	if d == nil {
		return ""
	}
	return d.String()
}
