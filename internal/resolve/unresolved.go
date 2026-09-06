package resolve

import (
	"sort"
	"time"

	"github.com/nodelogicau/intentions-cli/internal/temporal"
)

// Statuses an unresolved intention can be in.
const (
	StatusReady        = "ready"         // at least one candidate
	StatusBlocked      = "blocked"       // relational target has no placement
	StatusNoCandidates = "no_candidates" // resolvable, but the resolver found nothing
	StatusIncomplete   = "incomplete"    // duration or window missing
	StatusUnresolvable = "unresolvable"  // in a serves cycle
)

// UnresolvedEntry is one intention awaiting a placement and where it stands.
type UnresolvedEntry struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	Subject    string `json:"subject"`
	Activity   string `json:"activity,omitempty"`
	Duration   string `json:"duration,omitempty"`
	Window     string `json:"window,omitempty"`
	Status     string `json:"status"`
	Reason     string `json:"reason,omitempty"`
	BlockedOn  string `json:"blocked_on,omitempty"`
	Candidates int    `json:"candidates"`
	BestRank   int    `json:"best_rank,omitempty"`
	Range      string `json:"range,omitempty"`
	Deadline   string `json:"deadline,omitempty"`
	Timestamp  string `json:"timestamp"`
	deadline   time.Time
	timestamp  time.Time
}

// UnresolvedOptions narrow the listing.
type UnresolvedOptions struct {
	Subject string
}

// Unresolved lists every active intention with no placement that could take
// one (not a terminus, not a recurring parent), classified by a dry
// resolution that writes nothing, soonest deadline first.
func Unresolved(e Env, opts UnresolvedOptions) ([]UnresolvedEntry, error) {
	entries := []UnresolvedEntry{}
	for _, in := range e.G.Intentions() {
		if in.Retired != nil || in.Placement != nil || in.IsRecurring() {
			continue
		}
		if in.Duration == nil && in.Window == nil && len(in.Serves) == 0 {
			continue // a terminus is a sink, never placed
		}
		if opts.Subject != "" && in.Subject != opts.Subject {
			continue
		}
		ent := UnresolvedEntry{ID: in.ID, Title: in.Title, Subject: in.Subject, Activity: in.Activity, Timestamp: temporal.FormatTimestamp(in.Timestamp), timestamp: in.Timestamp}
		if in.Duration != nil {
			ent.Duration = in.Duration.Nominal.String()
		}
		if in.Window != nil {
			ent.Window = in.Window.String()
		}
		switch {
		case in.Duration == nil && in.Window == nil:
			ent.Status, ent.Reason = StatusIncomplete, "no duration and no window; both are required before resolution"
		case in.Duration == nil:
			ent.Status, ent.Reason = StatusIncomplete, "no duration; a duration is required before resolution"
		case in.Window == nil:
			ent.Status, ent.Reason = StatusIncomplete, "no window; a window is required before resolution"
		default:
			res, err := Resolve(e, in, Options{Limit: 1})
			if err != nil {
				ent.Status, ent.Reason = StatusUnresolvable, err.Error()
				break
			}
			if res.Range != nil {
				ent.Range, ent.Deadline, ent.deadline = res.RangeText, res.Range.End.Format(time.RFC3339), res.Range.End
			}
			ent.Candidates = res.Considered
			switch {
			case res.Considered > 0:
				ent.Status, ent.BestRank = StatusReady, res.Candidates[0].Rank
			case res.Blocked != "":
				ent.Status, ent.BlockedOn, ent.Reason = StatusBlocked, res.Blocked, res.Reason
			default:
				ent.Status, ent.Reason = StatusNoCandidates, res.Reason
			}
		}
		entries = append(entries, ent)
	}
	sort.SliceStable(entries, func(i, j int) bool {
		a, b := entries[i], entries[j]
		switch {
		case a.deadline.IsZero() != b.deadline.IsZero():
			return !a.deadline.IsZero()
		case !a.deadline.IsZero() && !a.deadline.Equal(b.deadline):
			return a.deadline.Before(b.deadline)
		case !a.timestamp.Equal(b.timestamp):
			return a.timestamp.Before(b.timestamp)
		}
		return a.ID < b.ID
	})
	return entries, nil
}

// UnresolvedResult is the map both front-ends emit.
func UnresolvedResult(entries []UnresolvedEntry) map[string]any {
	counts := map[string]int{}
	for _, e := range entries {
		counts[e.Status]++
	}
	return map[string]any{"entries": entries, "count": len(entries), "counts": counts}
}
