// Package resolve matches an intention's demand against the availability of
// every required particular and produces ranked candidate placements. It
// never chooses: Select is a recorded act the caller performs on the person's
// word or under a policy they hold. Generate materialises instances of
// recurring intentions. Everything here is pure over a loaded graph except
// Generate and Select, which write through the workspace they are given.
package resolve

import (
	"fmt"
	"sort"
	"time"

	"github.com/nodelogicau/intentions-cli/internal/model"
	"github.com/nodelogicau/intentions-cli/internal/store"
	"github.com/nodelogicau/intentions-cli/internal/temporal"
)

// Env is everything resolution needs beyond the intention.
type Env struct {
	G       *store.Graph
	WS      *store.Workspace
	Ctx     temporal.Context // Now must be set
	Step    temporal.Duration
	Horizon temporal.Duration
	Scope   string
}

// NewEnv builds an Env from a workspace and graph at now, with the
// configuration's defaults.
func NewEnv(ws *store.Workspace, g *store.Graph, now time.Time) Env {
	return Env{G: g, WS: ws, Ctx: ws.Config.Context(now), Step: ws.Config.Step(), Horizon: ws.Config.ResolverHorizon(), Scope: ws.Config.ResolverScope()}
}

func (e Env) now() time.Time {
	if e.Ctx.Now.IsZero() {
		return time.Now()
	}
	return e.Ctx.Now
}

func (e Env) stepDuration() time.Duration {
	d := e.Step
	if d.IsZero() {
		d = temporal.Duration{Minutes: 15}
	}
	return time.Duration(d.Hours)*time.Hour + time.Duration(d.Minutes)*time.Minute + time.Duration(d.Seconds)*time.Second
}

// goDuration converts a nominal duration to a time.Duration for placement
// arithmetic, using calendar arithmetic from a reference instant.
func goDuration(from time.Time, d temporal.Duration) time.Duration {
	return temporal.AddDuration(from, d).Sub(from)
}

func scopeRank(s string) int {
	for i, x := range model.Scopes {
		if x == s {
			return i
		}
	}
	return -1
}

// Placed is a placed object as resolution and consistency see it: an
// interval, the particulars whose time it occupies, and how costly it is to
// reconsider.
type Placed struct {
	ID          string
	Type        model.Type
	Interval    temporal.Interval
	Particulars []string
	Firm        bool // firm intention or accepted commitment
	Activity    string
	Location    string
	Object      model.Object
}

// PlacedObjects returns every unretired placed intention and every unretired
// opaque commitment, with their intervals in the environment's context.
// Transparent commitments occupy no time and are never included.
func (e Env) PlacedObjects() []Placed {
	var out []Placed
	// A commitment fulfilling an intention is the same occupancy as the
	// intention it fulfils: keep the commitment, whose party status carries
	// the reconsideration cost, and drop the intention.
	fulfilled := map[string]bool{}
	for _, c := range e.G.Commitments() {
		if c.Retired == nil && c.Placement != nil && c.Intention != "" {
			fulfilled[c.Intention] = true
		}
	}
	for _, in := range e.G.Intentions() {
		if in.Retired != nil || in.Placement == nil || fulfilled[in.ID] {
			continue
		}
		out = append(out, Placed{ID: in.ID, Type: model.TypeIntention, Interval: in.Placement.Bounds(e.Ctx),
			Particulars: append([]string{in.Subject}, in.Parties...), Firm: in.Stability == "firm", Activity: in.Activity, Location: in.Placement.Location, Object: in})
	}
	for _, c := range e.G.Commitments() {
		if c.Retired != nil || c.Placement == nil || c.IsTransparent() {
			continue
		}
		var parts []string
		firm := false
		for _, p := range c.Parties {
			parts = append(parts, p.URI)
			if p.Status == "accepted" {
				firm = true
			}
		}
		activity := ""
		if c.Intention != "" {
			if in, ok := e.G.Get(c.Intention); ok {
				if ii, ok := in.(*model.Intention); ok {
					activity = ii.Activity
				}
			}
		}
		out = append(out, Placed{ID: c.ID, Type: model.TypeCommitment, Interval: c.Placement.Bounds(e.Ctx), Particulars: parts, Firm: firm, Activity: activity, Location: c.Placement.Location, Object: c})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// Involves reports whether the placed object occupies the particular's time.
func (p Placed) Involves(uri string) bool {
	for _, x := range p.Particulars {
		if x == uri {
			return true
		}
	}
	return false
}

// PlacedInterval returns the interval of a placed object by id, when it is
// unretired and placed. Retired and cancelled objects count as unplaced.
func (e Env) PlacedInterval(id string) (temporal.Interval, bool) {
	obj, ok := e.G.Get(id)
	if !ok || obj.GetRetired() != nil {
		return temporal.Interval{}, false
	}
	switch o := obj.(type) {
	case *model.Intention:
		if o.Placement != nil {
			return o.Placement.Bounds(e.Ctx), true
		}
	case *model.Commitment:
		if o.Placement != nil {
			return o.Placement.Bounds(e.Ctx), true
		}
	}
	return temporal.Interval{}, false
}

// InCycle reports whether the intention is in a serves cycle.
func (e Env) InCycle(id string) []string {
	for _, scc := range stronglyConnected(e.G) {
		for _, m := range scc {
			if m == id {
				sort.Strings(scc)
				return scc
			}
		}
	}
	return nil
}

// stronglyConnected is Tarjan over the serves graph, duplicated from the
// query package to keep resolve free of a dependency on validate.
func stronglyConnected(g *store.Graph) [][]string {
	index := 0
	indices, low := map[string]int{}, map[string]int{}
	onStack := map[string]bool{}
	var stack []string
	var out [][]string
	var strong func(id string)
	strong = func(id string) {
		indices[id], low[id] = index, index
		index++
		stack = append(stack, id)
		onStack[id] = true
		obj, _ := g.Get(id)
		in, _ := obj.(*model.Intention)
		if in != nil {
			for _, r := range in.Serves {
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
			self := false
			if in != nil {
				for _, r := range in.Serves {
					if r.ID == id {
						self = true
					}
				}
			}
			if len(comp) > 1 || self {
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

func fmtTime(t time.Time) string { return t.Format(time.RFC3339) }

func describe(iv temporal.Interval) string {
	return fmt.Sprintf("%s to %s", fmtTime(iv.Start), fmtTime(iv.End))
}
