package cli

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/nodelogicau/intentions-cli/internal/model"
	"github.com/nodelogicau/intentions-cli/internal/projection"
	"github.com/nodelogicau/intentions-cli/internal/store"
	"github.com/nodelogicau/intentions-cli/internal/temporal"
)

// Environment variables.
const (
	EnvAuthor  = "INTENTIONS_AUTHOR"
	EnvHarness = "INTENTIONS_HARNESS"
	EnvModel   = "INTENTIONS_MODEL"
	EnvNow     = "INTENTIONS_NOW"
)

// actFlags are the flags every writing verb shares: attribution and time.
type actFlags struct {
	author, harness, model string
	now, timestamp         string
}

func addActFlags(cmd *cobra.Command, f *actFlags) {
	cmd.Flags().StringVar(&f.author, "author", "", "who this is on behalf of (default: $"+EnvAuthor+", then defaults.source.author)")
	cmd.Flags().StringVar(&f.harness, "harness", "", "the agent harness performing the act (default: $"+EnvHarness+")")
	cmd.Flags().StringVar(&f.model, "model", "", "the model behind the harness (default: $"+EnvModel+")")
	cmd.Flags().StringVar(&f.now, "now", "", "the current time as RFC 3339, for tests (default: $"+EnvNow+", then the clock)")
	cmd.Flags().StringVar(&f.timestamp, "timestamp", "", "assertion time as RFC 3339 (default: now)")
}

func firstNonEmpty(vs ...string) string {
	for _, v := range vs {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

// resolveSource applies flag, then environment, then intentions.yaml defaults.
func resolveSource(ws *store.Workspace, f actFlags) model.Source {
	return model.Source{
		Author:  firstNonEmpty(f.author, os.Getenv(EnvAuthor), ws.Config.Defaults.Source.Author),
		Harness: firstNonEmpty(f.harness, os.Getenv(EnvHarness), ws.Config.Defaults.Source.Harness),
		Model:   firstNonEmpty(f.model, os.Getenv(EnvModel), ws.Config.Defaults.Source.Model),
	}
}

func requireAuthor(src model.Source) error {
	if src.Author == "" {
		return refusedErr("an author is required: pass --author, set %s, or set defaults.source.author in intentions.yaml; a speech act with no speaker is not one", EnvAuthor)
	}
	return nil
}

// resolveNow applies --now, then INTENTIONS_NOW, then the clock.
func resolveNow(f actFlags) (time.Time, error) {
	s := firstNonEmpty(f.now, os.Getenv(EnvNow))
	if s == "" {
		return time.Now().UTC().Truncate(time.Second), nil
	}
	t, err := temporal.ParseTimestamp(s)
	if err != nil {
		return time.Time{}, usageErr("--now: %v", err)
	}
	return t, nil
}

// resolveTimestamp applies --timestamp, else now.
func resolveTimestamp(f actFlags, now time.Time) (time.Time, error) {
	if f.timestamp == "" {
		return now, nil
	}
	t, err := temporal.ParseTimestamp(f.timestamp)
	if err != nil {
		return time.Time{}, usageErr("--timestamp: %v", err)
	}
	return t, nil
}

// readDescription returns prose from --description or --description-file.
func (a *app) readDescription(text, file string) (string, error) {
	switch {
	case text != "" && file != "":
		return "", usageErr("use either --description or --description-file, not both")
	case file == "-":
		if a.stdinIsTerminal != nil && a.stdinIsTerminal() {
			return "", usageErr("--description-file - requires piped stdin; refusing to read from a terminal")
		}
		b, err := io.ReadAll(a.stdin)
		if err != nil {
			return "", err
		}
		return string(b), nil
	case file != "":
		b, err := os.ReadFile(file)
		if err != nil {
			return "", usageErr("--description-file: %v", err)
		}
		return string(b), nil
	}
	return text, nil
}

// windowFlags are the anchor flags shared by add, edit and supersede.
type windowFlags struct {
	calendar, clock, relative              string
	clearWindow, clearCalendar, clearClock bool
	clearRelative                          bool
}

func addWindowFlags(cmd *cobra.Command, f *windowFlags, withClear bool) {
	cmd.Flags().StringVar(&f.calendar, "calendar", "", "calendar anchor: an EDTF expression (2026-W36, 2026-09, ../2026-09) or a deictic term (this-week, next-month)")
	cmd.Flags().StringVar(&f.clock, "clock", "", "clock anchor: a time-of-day interval HH:MM/HH:MM, which may cross midnight")
	cmd.Flags().StringVar(&f.relative, "relative", "", "relational anchor: target:RELATION[:min[:max]], e.g. int_A:FINISHTOSTART:P0D:P3D")
	if withClear {
		cmd.Flags().BoolVar(&f.clearWindow, "clear-window", false, "remove the window")
		cmd.Flags().BoolVar(&f.clearCalendar, "clear-calendar", false, "remove the calendar anchor")
		cmd.Flags().BoolVar(&f.clearClock, "clear-clock", false, "remove the clock anchor")
		cmd.Flags().BoolVar(&f.clearRelative, "clear-relative", false, "remove the relational anchor")
	}
}

// applyWindow merges the anchor flags into an existing window (nil for a new
// object). Each flag replaces its own anchor; clear flags remove one.
func applyWindow(cmd *cobra.Command, f windowFlags, existing *temporal.Window, ctx temporal.Context, g *store.Graph) (*temporal.Window, error) {
	w := &temporal.Window{}
	if existing != nil && !f.clearWindow {
		*w = *existing
	}
	if f.clearCalendar {
		w.Calendar = nil
	}
	if f.clearClock {
		w.Clock = nil
	}
	if f.clearRelative {
		w.Relative = nil
	}
	if cmd.Flags().Changed("calendar") {
		c, err := temporal.ParseCalendarOrDeixis(f.calendar, ctx)
		if err != nil {
			return nil, invalidErr("--calendar: %v", err)
		}
		w.Calendar = &c
	}
	if cmd.Flags().Changed("clock") {
		c, err := temporal.ParseClock(f.clock)
		if err != nil {
			return nil, invalidErr("--clock: %v", err)
		}
		w.Clock = &c
	}
	if cmd.Flags().Changed("relative") {
		r, err := temporal.ParseRelative(f.relative)
		if err != nil {
			return nil, invalidErr("--relative: %v", err)
		}
		target, ok := g.Get(r.Target)
		if !ok {
			return nil, refusedErr("--relative: target %s does not exist", r.Target)
		}
		if t := target.GetType(); t != model.TypeIntention && t != model.TypeCommitment {
			return nil, refusedErr("--relative: target %s is a %s; a relational anchor targets an intention or a commitment", r.Target, t)
		}
		w.Relative = &r
	}
	if w.IsZero() {
		return nil, nil
	}
	return w, nil
}

func parseDurationFlag(s string) (*temporal.DurationSpec, error) {
	if s == "" {
		return nil, nil
	}
	d, err := temporal.ParseDurationSpec(s)
	if err != nil {
		return nil, invalidErr("--duration: %v", err)
	}
	return &d, nil
}

func parseCadenceFlag(s string) (*temporal.Cadence, error) {
	if s == "" {
		return nil, nil
	}
	c, err := temporal.ParseCadence(s)
	if err != nil {
		return nil, invalidErr("--cadence: %v", err)
	}
	return &c, nil
}

func parsePolicyFlag(name, s string) (*model.Policy, error) {
	if s == "" {
		return nil, nil
	}
	p := &model.Policy{}
	for _, term := range strings.Split(s, ",") {
		kv := strings.SplitN(strings.TrimSpace(term), "=", 2)
		if len(kv) != 2 {
			return nil, usageErr("--%s: use term=value pairs such as max_duration=PT30M,stability=tentative", name)
		}
		switch kv[0] {
		case "max_duration":
			d, err := temporal.ParseDuration(kv[1])
			if err != nil {
				return nil, invalidErr("--%s: %v", name, err)
			}
			p.MaxDuration = &d
		case "stability":
			p.Stability = kv[1]
		default:
			return nil, invalidErr("--%s: unknown policy term %q; admitted terms are max_duration and stability", name, kv[0])
		}
	}
	return p, nil
}

func parseServesFlags(vs []string) ([]model.Ref, error) {
	out := make([]model.Ref, 0, len(vs))
	for _, v := range vs {
		parts := strings.SplitN(v, ":", 2)
		if len(parts) != 2 {
			return nil, usageErr("--serves: use id:role, e.g. int_A:in-order-to")
		}
		out = append(out, model.Ref{ID: parts[0], Role: parts[1]})
	}
	return out, nil
}

// checkURIs validates a repeatable URI flag.
func checkURIs(name string, vs []string) error {
	for _, v := range vs {
		if !model.ValidURI(v) {
			return invalidErr("--%s: %q is not an absolute URI (a scheme followed by a colon)", name, v)
		}
	}
	return nil
}

// loadGraph loads the workspace, refusing to proceed on unreadable files.
func loadGraph(ws *store.Workspace) (*store.Graph, error) {
	g, err := ws.Load()
	if err != nil {
		return nil, err
	}
	if len(g.Unreadable) > 0 {
		return nil, fmt.Errorf("%s cannot be read: %s; fix the file or run `intentions validate`", g.Unreadable[0].Path, g.Unreadable[0].Err)
	}
	return g, nil
}

// getObject fetches one object from the graph by id, typed by prefix.
func getObject(g *store.Graph, id string) (model.Object, error) {
	if _, ok := model.TypeOfID(id); !ok {
		return nil, notFoundErr("%q is not an identifier", id)
	}
	obj, ok := g.Get(id)
	if !ok {
		return nil, notFoundErr("%s does not exist", id)
	}
	return obj, nil
}

func getIntention(g *store.Graph, id string) (*model.Intention, error) {
	obj, err := getObject(g, id)
	if err != nil {
		return nil, err
	}
	in, ok := obj.(*model.Intention)
	if !ok {
		return nil, usageErr("%s is a %s, not an intention", id, obj.GetType())
	}
	return in, nil
}

func getAvailability(g *store.Graph, id string) (*model.Availability, error) {
	obj, err := getObject(g, id)
	if err != nil {
		return nil, err
	}
	av, ok := obj.(*model.Availability)
	if !ok {
		return nil, usageErr("%s is a %s, not an availability", id, obj.GetType())
	}
	return av, nil
}

// writeObject checks the single-object rules, then writes.
func writeObject(ws *store.Workspace, obj model.Object) error {
	if ps := model.Check(obj); len(ps) > 0 {
		return ps
	}
	return ws.WriteObject(obj)
}

// objectResult is the common JSON shape for a written or shown object.
func objectResult(obj model.Object) (map[string]any, error) {
	m, err := model.ToMap(obj)
	if err != nil {
		return nil, err
	}
	v, err := projection.Version(obj)
	if err != nil {
		return nil, err
	}
	out := map[string]any{
		"id":      obj.GetID(),
		"type":    string(obj.GetType()),
		"path":    store.RelPath(obj.GetType(), obj.GetID()),
		"version": v,
		"object":  m,
	}
	if cached := obj.CachedVersion(); cached != "" && cached != v {
		out["version_cached"] = cached
	}
	return out, nil
}

func oneLine(s string, max int) string {
	s = strings.TrimSpace(strings.ReplaceAll(s, "\n", " "))
	if r := []rune(s); max > 0 && len(r) > max {
		return string(r[:max-1]) + "…"
	}
	return s
}

func windowText(w *temporal.Window) string {
	if w == nil {
		return "-"
	}
	return w.String()
}

func durationText(d *temporal.DurationSpec) string {
	if d == nil {
		return "-"
	}
	return d.String()
}
