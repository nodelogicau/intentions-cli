package cli

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/nodelogicau/intentions-cli/internal/consistency"
	"github.com/nodelogicau/intentions-cli/internal/model"
	"github.com/nodelogicau/intentions-cli/internal/projection"
	"github.com/nodelogicau/intentions-cli/internal/resolve"
	"github.com/nodelogicau/intentions-cli/internal/store"
	"github.com/nodelogicau/intentions-cli/internal/temporal"
)

type availabilityFields struct {
	subject, title, description, descriptionFile string
	duration, cadence, validUntil, scope         string
	conditional, location                        []string
	window                                       windowFlags
}

func addAvailabilityFieldFlags(cmd *cobra.Command, f *availabilityFields) {
	cmd.Flags().StringVar(&f.subject, "subject", "", "URI of the particular whose availability this is: a person, a room, anything (required)")
	cmd.Flags().StringVar(&f.title, "title", "", "prose title")
	cmd.Flags().StringVar(&f.description, "description", "", "prose description")
	cmd.Flags().StringVar(&f.descriptionFile, "description-file", "", "read the description from a file, or - for piped stdin")
	cmd.Flags().StringVar(&f.duration, "duration", "", "capacity offered per occasion: ISO 8601 duration or nominal:min:max (required)")
	cmd.Flags().StringArrayVar(&f.conditional, "conditional", nil, "activity term this supply is good for (repeatable; absent means anything)")
	cmd.Flags().StringArrayVar(&f.location, "location", nil, "URI at which this capacity holds (repeatable; absent means anywhere)")
	cmd.Flags().StringVar(&f.cadence, "cadence", "", "RRULE with date-level parts only; makes this recurring")
	cmd.Flags().StringVar(&f.validUntil, "valid-until", "", "validity horizon: an EDTF expression or RFC 3339 datetime")
	cmd.Flags().StringVar(&f.scope, "scope", "", "personal, organisation, or public (default personal)")
	addWindowFlags(cmd, &f.window, false)
}

func (a *app) availabilityCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "availability",
		Short: "Add, edit, renew, supersede, retire, show and list availability",
	}
	cmd.AddCommand(a.availabilityAddCmd(), a.availabilityEditCmd(), a.availabilityRenewCmd(), a.availabilitySupersedeCmd(), a.availabilityRetireCmd(), a.availabilityShowCmd(), a.availabilityListCmd())
	return cmd
}

// applyAvailabilityFields merges flags into o.
func (a *app) applyAvailabilityFields(cmd *cobra.Command, f availabilityFields, o *model.Availability, g *store.Graph, ctx temporal.Context) error {
	changed := cmd.Flags().Changed
	if changed("subject") {
		o.Subject = f.subject
	}
	if changed("title") {
		o.Title = f.title
	}
	if changed("description") || changed("description-file") {
		d, err := a.readDescription(f.description, f.descriptionFile)
		if err != nil {
			return err
		}
		o.Description = d
	}
	if changed("duration") {
		d, err := parseDurationFlag(f.duration)
		if err != nil {
			return err
		}
		o.Duration = d
	}
	w, err := applyWindow(cmd, f.window, o.Window, ctx, g)
	if err != nil {
		return err
	}
	if w != nil || changed("calendar") || changed("clock") || changed("relative") {
		o.Window = w
	}
	if changed("conditional") {
		o.Conditional = model.SortStrings(f.conditional)
	}
	if changed("location") {
		if err := checkURIs("location", f.location); err != nil {
			return err
		}
		o.Location = model.SortStrings(f.location)
	}
	if changed("cadence") {
		c, err := parseCadenceFlag(f.cadence)
		if err != nil {
			return err
		}
		o.Cadence = c
	}
	if changed("valid-until") {
		v, err := temporal.ParseValidUntil(f.validUntil)
		if err != nil {
			return invalidErr("--valid-until: %v", err)
		}
		o.ValidUntil = v
	}
	if changed("scope") {
		o.Scope = f.scope
	}
	if o.Scope == "" {
		o.Scope = "personal"
	}
	return nil
}

// effectiveValidUntil is the explicit horizon, else timestamp plus the
// workspace default for a recurring availability, else the end of the
// window's calendar bounds. Zero when unbounded.
func effectiveValidUntil(ws *store.Workspace, o *model.Availability, ctx temporal.Context) (time.Time, string) {
	if !o.ValidUntil.IsZero() {
		return o.ValidUntil.Instant(ctx), "explicit"
	}
	if o.Cadence != nil {
		return temporal.AddDuration(o.Timestamp, ws.Config.DefaultHorizon()), "default_horizon"
	}
	if o.Window != nil {
		ivs, err := temporal.Bounds(temporal.Window{Calendar: o.Window.Calendar}, ctx)
		if err == nil && len(ivs) > 0 && !ivs[len(ivs)-1].OpenEnd {
			return ivs[len(ivs)-1].End, "window"
		}
	}
	return time.Time{}, "unbounded"
}

// availabilityFlags runs the check for flags naming the availability as a
// counterpart, after a write, when a graph is at hand.
func (a *app) availabilityFlags(ws *store.Workspace, g *store.Graph, o *model.Availability, now time.Time) []consistency.Flag {
	if g == nil {
		return nil
	}
	g.Add(o)
	e := resolve.NewEnv(ws, g, now)
	flags, err := consistency.Check(e, []string{o.ID})
	if err != nil {
		return nil
	}
	return flags
}

func (a *app) availabilityResult(ws *store.Workspace, o *model.Availability, ctx temporal.Context) (map[string]any, error) {
	out, err := objectResult(o)
	if err != nil {
		return nil, err
	}
	t, how := effectiveValidUntil(ws, o, ctx)
	if !t.IsZero() {
		out["effective_valid_until"] = t.UTC().Format(temporal.TimeLayout)
	} else {
		out["effective_valid_until"] = nil
	}
	out["effective_valid_until_from"] = how
	return out, nil
}

func (a *app) availabilityAddCmd() *cobra.Command {
	var f availabilityFields
	var act actFlags
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Create an availability: a standing statement that a particular has capacity within a window",
		Args:  cobra.NoArgs,
		RunE: a.run(func(cmd *cobra.Command, args []string) error {
			if f.subject == "" {
				return usageErr("--subject is required: availability names its particular explicitly")
			}
			if f.duration == "" {
				return usageErr("--duration is required: the capacity offered per occasion")
			}
			if f.window.calendar == "" && f.window.clock == "" && f.window.relative == "" {
				return usageErr("a window is required: pass at least one of --calendar, --clock, --relative")
			}
			ws, err := a.openWorkspace()
			if err != nil {
				return err
			}
			g, err := loadGraph(ws)
			if err != nil {
				return err
			}
			now, err := resolveNow(act)
			if err != nil {
				return err
			}
			ts, err := resolveTimestamp(act, now)
			if err != nil {
				return err
			}
			src := resolveSource(ws, act)
			if err := requireAuthor(src); err != nil {
				return err
			}
			o := &model.Availability{ID: model.MintID(model.TypeAvailability), Source: src, Timestamp: ts}
			ctx := ws.Config.Context(now)
			if err := a.applyAvailabilityFields(cmd, f, o, g, ctx); err != nil {
				return err
			}
			if err := writeObject(ws, o); err != nil {
				return err
			}
			out, err := a.availabilityResult(ws, o, ctx)
			if err != nil {
				return err
			}
			out["created"] = true
			out["flags"] = a.availabilityFlags(ws, g, o, ctx.Now)
			return a.emit(out, func(w io.Writer) {
				fmt.Fprintf(w, "Created %s (%s)\n  %s\n", o.ID, o.Version, out["path"])
			})
		}),
	}
	addAvailabilityFieldFlags(cmd, &f)
	addActFlags(cmd, &act)
	return cmd
}

func (a *app) availabilityEditCmd() *cobra.Command {
	var title, description, descriptionFile, scope string
	var act actFlags
	cmd := &cobra.Command{
		Use:   "edit <id>",
		Short: "Edit prose or widen scope; a change of terms is refused (use supersede)",
		Args:  cobra.ExactArgs(1),
		RunE: a.run(func(cmd *cobra.Command, args []string) error {
			ws, err := a.openWorkspace()
			if err != nil {
				return err
			}
			g, err := loadGraph(ws)
			if err != nil {
				return err
			}
			before, err := getAvailability(g, args[0])
			if err != nil {
				return err
			}
			if err := model.CheckNotRetired(before); err != nil {
				return err
			}
			o := *before
			if cmd.Flags().Changed("title") {
				o.Title = title
			}
			if cmd.Flags().Changed("description") || cmd.Flags().Changed("description-file") {
				d, err := a.readDescription(description, descriptionFile)
				if err != nil {
					return err
				}
				o.Description = d
			}
			if cmd.Flags().Changed("scope") {
				if err := model.CheckScope(before.Scope, scope); err != nil {
					return err
				}
				o.Scope = scope
			}
			if err := model.CheckAvailabilityTerms(before, &o); err != nil {
				return err
			}
			prev, _ := projection.Version(before)
			if err := writeObject(ws, &o); err != nil {
				return err
			}
			now, _ := resolveNow(act)
			out, err := a.availabilityResult(ws, &o, ws.Config.Context(now))
			if err != nil {
				return err
			}
			out["previous_version"] = prev
			out["flags"] = a.availabilityFlags(ws, g, &o, now)
			return a.emit(out, func(w io.Writer) {
				fmt.Fprintf(w, "Edited %s (%s)\n", o.ID, o.Version)
			})
		}),
	}
	cmd.Flags().StringVar(&title, "title", "", "prose title")
	cmd.Flags().StringVar(&description, "description", "", "prose description")
	cmd.Flags().StringVar(&descriptionFile, "description-file", "", "read the description from a file, or - for piped stdin")
	cmd.Flags().StringVar(&scope, "scope", "", "widen to organisation or public; narrowing is refused")
	// Terms flags are accepted so the refusal can name supersede rather than
	// cobra reporting an unknown flag.
	for _, name := range []string{"calendar", "clock", "relative", "duration", "cadence", "valid-until"} {
		cmd.Flags().String(name, "", "not admitted on edit: a change of terms is a supersession")
	}
	cmd.Flags().StringArray("conditional", nil, "not admitted on edit: a change of terms is a supersession")
	cmd.Flags().StringArray("location", nil, "not admitted on edit: a change of terms is a supersession")
	pre := cmd.RunE
	cmd.RunE = func(c *cobra.Command, args []string) error {
		for _, name := range []string{"calendar", "clock", "relative", "duration", "cadence", "conditional", "location"} {
			if c.Flags().Changed(name) {
				return refusedErr("--%s changes the availability's terms and is not admitted on edit; use `intentions availability supersede %s --%s ...` to create a new availability and retire this one as superseded", name, args[0], name)
			}
		}
		if c.Flags().Changed("valid-until") {
			return refusedErr("--valid-until is renewal; use `intentions availability renew %s --valid-until ...`", args[0])
		}
		return pre(c, args)
	}
	addActFlags(cmd, &act)
	return cmd
}

func (a *app) availabilityRenewCmd() *cobra.Command {
	var validUntil string
	var act actFlags
	cmd := &cobra.Command{
		Use:   "renew <id>",
		Short: "Advance valid_until on the same object",
		Args:  cobra.ExactArgs(1),
		RunE: a.run(func(cmd *cobra.Command, args []string) error {
			if validUntil == "" {
				return usageErr("--valid-until is required")
			}
			ws, err := a.openWorkspace()
			if err != nil {
				return err
			}
			g, err := loadGraph(ws)
			if err != nil {
				return err
			}
			before, err := getAvailability(g, args[0])
			if err != nil {
				return err
			}
			if err := model.CheckNotRetired(before); err != nil {
				return err
			}
			v, err := temporal.ParseValidUntil(validUntil)
			if err != nil {
				return invalidErr("--valid-until: %v", err)
			}
			now, err := resolveNow(act)
			if err != nil {
				return err
			}
			ctx := ws.Config.Context(now)
			current, how := effectiveValidUntil(ws, before, ctx)
			if !current.IsZero() && v.Instant(ctx).Before(current) {
				return refusedErr("renewal moves valid_until earlier than the current horizon %s (%s); renewal only advances it", current.UTC().Format(temporal.TimeLayout), how)
			}
			o := *before
			o.ValidUntil = v
			prev, _ := projection.Version(before)
			if err := writeObject(ws, &o); err != nil {
				return err
			}
			out, err := a.availabilityResult(ws, &o, ctx)
			if err != nil {
				return err
			}
			out["previous_version"] = prev
			out["flags"] = a.availabilityFlags(ws, g, &o, ctx.Now)
			return a.emit(out, func(w io.Writer) {
				fmt.Fprintf(w, "Renewed %s until %s (%s -> %s)\n", o.ID, o.ValidUntil, prev, o.Version)
			})
		}),
	}
	cmd.Flags().StringVar(&validUntil, "valid-until", "", "new validity horizon: an EDTF expression or RFC 3339 datetime")
	addActFlags(cmd, &act)
	return cmd
}

func (a *app) availabilitySupersedeCmd() *cobra.Command {
	var f availabilityFields
	var act actFlags
	var reason string
	cmd := &cobra.Command{
		Use:   "supersede <id>",
		Short: "Create a new availability from this one with changed terms, and retire this one as superseded",
		Args:  cobra.ExactArgs(1),
		RunE: a.run(func(cmd *cobra.Command, args []string) error {
			ws, err := a.openWorkspace()
			if err != nil {
				return err
			}
			g, err := loadGraph(ws)
			if err != nil {
				return err
			}
			old, err := getAvailability(g, args[0])
			if err != nil {
				return err
			}
			if err := model.CheckNotRetired(old); err != nil {
				return err
			}
			now, err := resolveNow(act)
			if err != nil {
				return err
			}
			ts, err := resolveTimestamp(act, now)
			if err != nil {
				return err
			}
			src := resolveSource(ws, act)
			if err := requireAuthor(src); err != nil {
				return err
			}
			ctx := ws.Config.Context(now)
			n := *old
			n.ID = model.MintID(model.TypeAvailability)
			n.Source = src
			n.Timestamp = ts
			n.Retired = nil
			n.Version = ""
			n.Extras = nil
			n.Conditional = append([]string(nil), old.Conditional...)
			n.Location = append([]string(nil), old.Location...)
			if old.Window != nil {
				w := *old.Window
				n.Window = &w
			}
			if err := a.applyAvailabilityFields(cmd, f, &n, g, ctx); err != nil {
				return err
			}
			if err := model.CheckScope(old.Scope, n.Scope); err != nil {
				return err
			}
			if ps := model.Check(&n); len(ps) > 0 {
				return ps
			}
			if err := writeObject(ws, &n); err != nil {
				return err
			}
			g.Add(&n)
			r := model.Retired{Kind: "superseded", Reason: reason, SupersededBy: n.ID, Source: src, Timestamp: ts}
			if err := model.CheckRetirement(g, old, r); err != nil {
				return err
			}
			retired := *old
			retired.Retired = &r
			if err := writeObject(ws, &retired); err != nil {
				return err
			}
			out, err := a.availabilityResult(ws, &n, ctx)
			if err != nil {
				return err
			}
			out["created"] = true
			out["superseded"] = map[string]any{"id": old.ID, "version": retired.Version}
			g.Add(&retired)
			out["flags"] = a.availabilityFlags(ws, g, &n, ctx.Now)
			return a.emit(out, func(w io.Writer) {
				fmt.Fprintf(w, "Created %s (%s)\nRetired %s as superseded by it\n", n.ID, n.Version, old.ID)
			})
		}),
	}
	addAvailabilityFieldFlags(cmd, &f)
	cmd.Flags().StringVar(&reason, "reason", "", "prose reason recorded on the retirement")
	addActFlags(cmd, &act)
	return cmd
}

func (a *app) availabilityRetireCmd() *cobra.Command {
	var act actFlags
	var kind, supersededBy, reason string
	cmd := &cobra.Command{
		Use:   "retire <id>",
		Short: "Append a retirement record: retracted, or superseded (with --superseded-by)",
		Args:  cobra.ExactArgs(1),
		RunE: a.run(func(cmd *cobra.Command, args []string) error {
			if kind == "" {
				return usageErr("--kind is required: %s", strings.Join(model.AvailabilityRetirementKinds, ", "))
			}
			ws, err := a.openWorkspace()
			if err != nil {
				return err
			}
			g, err := loadGraph(ws)
			if err != nil {
				return err
			}
			before, err := getAvailability(g, args[0])
			if err != nil {
				return err
			}
			now, err := resolveNow(act)
			if err != nil {
				return err
			}
			ts, err := resolveTimestamp(act, now)
			if err != nil {
				return err
			}
			r := model.Retired{Kind: kind, Reason: reason, SupersededBy: supersededBy, Source: resolveSource(ws, act), Timestamp: ts}
			if err := model.CheckRetirement(g, before, r); err != nil {
				return err
			}
			o := *before
			o.Retired = &r
			prev, _ := projection.Version(before)
			if err := writeObject(ws, &o); err != nil {
				return err
			}
			out, err := objectResult(&o)
			if err != nil {
				return err
			}
			out["previous_version"] = prev
			out["retired"] = map[string]any{"kind": kind, "superseded_by": supersededBy, "reason": reason}
			out["flags"] = a.availabilityFlags(ws, g, &o, now)
			return a.emit(out, func(w io.Writer) {
				fmt.Fprintf(w, "Retired %s as %s (%s -> %s)\n", o.ID, kind, prev, o.Version)
				printFlags(w, out["flags"].([]consistency.Flag))
			})
		}),
	}
	cmd.Flags().StringVar(&kind, "kind", "", "retracted or superseded")
	cmd.Flags().StringVar(&supersededBy, "superseded-by", "", "the replacing availability (required with kind superseded)")
	cmd.Flags().StringVar(&reason, "reason", "", "prose reason")
	addActFlags(cmd, &act)
	return cmd
}

func (a *app) availabilityShowCmd() *cobra.Command {
	var nowFlag string
	cmd := &cobra.Command{
		Use:   "show <id>",
		Short: "Show an availability with its computed version and effective validity horizon",
		Args:  cobra.ExactArgs(1),
		RunE: a.run(func(cmd *cobra.Command, args []string) error {
			ws, err := a.openWorkspace()
			if err != nil {
				return err
			}
			g, err := ws.Load()
			if err != nil {
				return err
			}
			o, err := getAvailability(g, args[0])
			if err != nil {
				return err
			}
			now, err := resolveNow(actFlags{now: nowFlag})
			if err != nil {
				return err
			}
			out, err := a.availabilityResult(ws, o, ws.Config.Context(now))
			if err != nil {
				return err
			}
			if ld := g.Loaded[o.ID]; ld != nil && len(ld.Problems) > 0 {
				out["problems"] = ld.Problems
			}
			return a.emit(out, func(w io.Writer) {
				data, _ := model.Encode(o)
				fmt.Fprint(w, string(data))
				fmt.Fprintf(w, "# effective_valid_until: %v (%s)\n", out["effective_valid_until"], out["effective_valid_until_from"])
			})
		}),
	}
	cmd.Flags().StringVar(&nowFlag, "now", "", "the current time as RFC 3339, for tests")
	return cmd
}

func (a *app) availabilityListCmd() *cobra.Command {
	var subject, conditional, scope string
	var retired bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List availability, active by default",
		Args:  cobra.NoArgs,
		RunE: a.run(func(cmd *cobra.Command, args []string) error {
			ws, err := a.openWorkspace()
			if err != nil {
				return err
			}
			g, err := ws.Load()
			if err != nil {
				return err
			}
			var items []*model.Availability
			for _, o := range g.Availabilities() {
				if (o.Retired != nil) != retired {
					continue
				}
				if subject != "" && o.Subject != subject {
					continue
				}
				if scope != "" && o.Scope != scope {
					continue
				}
				if conditional != "" {
					found := false
					for _, c := range o.Conditional {
						if c == conditional {
							found = true
						}
					}
					if !found {
						continue
					}
				}
				items = append(items, o)
			}
			list := make([]map[string]any, 0, len(items))
			for _, o := range items {
				m, err := model.ToMap(o)
				if err != nil {
					return err
				}
				list = append(list, m)
			}
			return a.emit(map[string]any{"availability": list, "count": len(list)}, func(w io.Writer) {
				for _, o := range items {
					flags := ""
					if o.Cadence != nil {
						flags += " " + o.Cadence.Raw
					}
					if o.Retired != nil {
						flags += " retired:" + o.Retired.Kind
					}
					fmt.Fprintf(w, "%s  %-6s %-30s %s%s\n", o.ID, durationText(o.Duration), oneLine(windowText(o.Window), 30), oneLine(firstNonEmpty(o.Title, o.Subject), 50), flags)
				}
				if len(items) == 0 {
					fmt.Fprintln(w, "(no availability)")
				}
			})
		}),
	}
	cmd.Flags().StringVar(&subject, "subject", "", "filter by subject URI")
	cmd.Flags().StringVar(&conditional, "conditional", "", "filter by a conditional term")
	cmd.Flags().StringVar(&scope, "scope", "", "filter by scope")
	cmd.Flags().BoolVar(&retired, "retired", false, "list retired availability instead of active")
	return cmd
}
