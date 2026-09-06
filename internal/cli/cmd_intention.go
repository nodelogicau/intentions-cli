package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/nodelogicau/intentions-cli/internal/model"
	"github.com/nodelogicau/intentions-cli/internal/projection"
	"github.com/nodelogicau/intentions-cli/internal/store"
	"github.com/nodelogicau/intentions-cli/internal/temporal"
)

// intentionFields are the field flags shared by add and edit.
type intentionFields struct {
	subject, title, description, descriptionFile string
	duration, stability, activity, cadence       string
	preference, autoSelect, autoFirm, reference  string
	location, parties, serves                    []string
	policy                                       string
	window                                       windowFlags
	clear                                        map[string]*bool
}

var clearableIntentionFields = []string{"description", "duration", "activity", "location", "parties", "serves", "cadence", "preference", "auto-select", "auto-firm", "reference"}

func addIntentionFieldFlags(cmd *cobra.Command, f *intentionFields, edit bool) {
	cmd.Flags().StringVar(&f.title, "title", "", "prose title (required on add)")
	cmd.Flags().StringVar(&f.description, "description", "", "prose description")
	cmd.Flags().StringVar(&f.descriptionFile, "description-file", "", "read the description from a file, or - for piped stdin")
	cmd.Flags().StringVar(&f.duration, "duration", "", "ISO 8601 duration (PT90M) or a range nominal:min:max (PT1H:PT30M:PT2H)")
	cmd.Flags().StringVar(&f.stability, "stability", "", "tentative or firm (default tentative; firm by a harness needs --policy)")
	cmd.Flags().StringVar(&f.activity, "activity", "", "activity term, lowercase kebab-case, matched against availability conditional")
	cmd.Flags().StringArrayVar(&f.location, "location", nil, "URI at one of which this must happen (repeatable)")
	cmd.Flags().StringArrayVar(&f.parties, "party", nil, "URI of another particular whose availability must be satisfied (repeatable)")
	cmd.Flags().StringArrayVar(&f.serves, "serves", nil, "outbound reference id:role, role in-order-to, for-the-sake-of, instance-of (repeatable)")
	cmd.Flags().StringVar(&f.cadence, "cadence", "", "RRULE with date-level parts only; makes this a recurring intention, whose window needs a calendar anchor")
	cmd.Flags().StringVar(&f.preference, "preference", "", "earliest, latest, adjacent, or spread")
	cmd.Flags().StringVar(&f.autoSelect, "auto-select", "", "policy condition on a terminus, e.g. max_duration=PT30M,stability=tentative")
	cmd.Flags().StringVar(&f.autoFirm, "auto-firm", "", "policy condition on a terminus, e.g. max_duration=PT30M")
	cmd.Flags().StringVar(&f.reference, "reference", "", "informal pointer to a DKF claim; never validated")
	cmd.Flags().StringVar(&f.policy, "policy", "", "id of the terminus carrying auto_firm that authorises a harness to set firm; written to firmed_under")
	addWindowFlags(cmd, &f.window, edit)
	if !edit {
		cmd.Flags().StringVar(&f.subject, "subject", "", "URI of the particular whose intention this is (default: defaults.subject)")
	}
	f.clear = map[string]*bool{}
	if edit {
		for _, name := range clearableIntentionFields {
			b := new(bool)
			f.clear[name] = b
			cmd.Flags().BoolVar(b, "clear-"+name, false, "remove "+name)
		}
	}
}

func (a *app) intentionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "intention",
		Short: "Add, edit, firm, retire, show and list intentions",
	}
	cmd.AddCommand(a.intentionAddCmd(), a.intentionEditCmd(), a.intentionFirmCmd(), a.intentionRetireCmd(), a.intentionShowCmd(), a.intentionListCmd())
	return cmd
}

// applyIntentionFields merges flags into o. When edit is false, o is new.
func (a *app) applyIntentionFields(cmd *cobra.Command, f intentionFields, o *model.Intention, ws *store.Workspace, g *store.Graph, ctx temporal.Context) error {
	changed := cmd.Flags().Changed
	cleared := func(name string) bool { b := f.clear[name]; return b != nil && *b }

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
	if cleared("description") {
		o.Description = ""
	}
	if changed("duration") {
		d, err := parseDurationFlag(f.duration)
		if err != nil {
			return err
		}
		o.Duration = d
	}
	if cleared("duration") {
		o.Duration = nil
	}
	w, err := applyWindow(cmd, f.window, o.Window, ctx, g)
	if err != nil {
		return err
	}
	o.Window = w
	if changed("stability") {
		o.Stability = f.stability
		if o.Stability != "firm" {
			// A person's act, or an unfirming: firmed_under belongs to firm only.
			o.FirmedUnder = ""
		}
	}
	if changed("activity") {
		o.Activity = f.activity
	}
	if cleared("activity") {
		o.Activity = ""
	}
	if changed("location") {
		if err := checkURIs("location", f.location); err != nil {
			return err
		}
		o.Location = model.SortStrings(f.location)
	}
	if cleared("location") {
		o.Location = nil
	}
	if changed("party") {
		if err := checkURIs("party", f.parties); err != nil {
			return err
		}
		o.Parties = model.SortStrings(f.parties)
	}
	if cleared("parties") {
		o.Parties = nil
	}
	if changed("serves") {
		refs, err := parseServesFlags(f.serves)
		if err != nil {
			return err
		}
		o.Serves = model.SortRefs(refs)
	}
	if cleared("serves") {
		o.Serves = []model.Ref{}
	}
	if o.Serves == nil {
		o.Serves = []model.Ref{}
	}
	if changed("cadence") {
		c, err := parseCadenceFlag(f.cadence)
		if err != nil {
			return err
		}
		o.Cadence = c
	}
	if cleared("cadence") {
		o.Cadence = nil
	}
	if changed("preference") {
		o.Preference = f.preference
	}
	if cleared("preference") {
		o.Preference = ""
	}
	if changed("auto-select") {
		p, err := parsePolicyFlag("auto-select", f.autoSelect)
		if err != nil {
			return err
		}
		o.AutoSelect = p
	}
	if cleared("auto-select") {
		o.AutoSelect = nil
	}
	if changed("auto-firm") {
		p, err := parsePolicyFlag("auto-firm", f.autoFirm)
		if err != nil {
			return err
		}
		o.AutoFirm = p
	}
	if cleared("auto-firm") {
		o.AutoFirm = nil
	}
	if changed("reference") {
		o.Reference = f.reference
	}
	if cleared("reference") {
		o.Reference = ""
	}
	return nil
}

// checkIntentionWrite applies the workspace-level write policy for a proposed
// state, given the state before the act (nil on add). When the act sets firm
// it also settles firmed_under on o: the policy id for a harness, absent for
// a person's own act.
func checkIntentionWrite(g *store.Graph, before, o *model.Intention, act model.Source, policy string) error {
	if before != nil {
		if err := model.CheckNotRetired(before); err != nil {
			return err
		}
		if err := model.CheckSubjectUnchanged(before.Subject, o.Subject); err != nil {
			return err
		}
	}
	if err := model.CheckServes(g, o.ID, o.Serves); err != nil {
		return err
	}
	if o.Stability == "firm" && (before == nil || before.Stability != "firm") {
		target := o
		if before != nil {
			target = before
		}
		fu, err := model.CheckFirm(g, target, act, policy)
		if err != nil {
			return err
		}
		o.FirmedUnder = fu
	}
	if o.Stability != "firm" {
		o.FirmedUnder = ""
	}
	return nil
}

func (a *app) intentionAddCmd() *cobra.Command {
	var f intentionFields
	var act actFlags
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Create an intention",
		Args:  cobra.NoArgs,
		RunE: a.run(func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(f.title) == "" {
				return usageErr("--title is required")
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
			o := &model.Intention{ID: model.MintID(model.TypeIntention), Stability: "tentative", Source: src, Timestamp: ts, Acknowledgements: []model.Acknowledgement{}, Serves: []model.Ref{}}
			o.Subject = firstNonEmpty(f.subject, ws.Config.Defaults.Subject)
			if o.Subject == "" {
				return refusedErr("subject is required: pass --subject, or set defaults.subject in intentions.yaml; this workspace has no default")
			}
			if err := a.applyIntentionFields(cmd, f, o, ws, g, ws.Config.Context(now)); err != nil {
				return err
			}
			if ps := model.Check(o); len(ps) > 0 {
				return ps
			}
			if err := checkIntentionWrite(g, nil, o, src, f.policy); err != nil {
				return err
			}
			if err := writeObject(ws, o); err != nil {
				return err
			}
			out, err := objectResult(o)
			if err != nil {
				return err
			}
			out["created"] = true
			if f.policy != "" && o.Stability == "firm" {
				out["policy"] = f.policy
			}
			return a.emit(out, func(w io.Writer) {
				fmt.Fprintf(w, "Created %s (%s)\n  %s\n", o.ID, o.Version, out["path"])
			})
		}),
	}
	addIntentionFieldFlags(cmd, &f, false)
	addActFlags(cmd, &act)
	return cmd
}

func (a *app) intentionEditCmd() *cobra.Command {
	var f intentionFields
	var act actFlags
	cmd := &cobra.Command{
		Use:   "edit <id>",
		Short: "Edit an intention in place; prose edits leave the version unchanged",
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
			before, err := getIntention(g, args[0])
			if err != nil {
				return err
			}
			now, err := resolveNow(act)
			if err != nil {
				return err
			}
			src := resolveSource(ws, act)
			o := *before
			o.Serves = append([]model.Ref(nil), before.Serves...)
			if err := a.applyIntentionFields(cmd, f, &o, ws, g, ws.Config.Context(now)); err != nil {
				return err
			}
			if ps := model.Check(&o); len(ps) > 0 {
				return ps
			}
			if err := checkIntentionWrite(g, before, &o, src, f.policy); err != nil {
				return err
			}
			prev, _ := projection.Version(before)
			if err := writeObject(ws, &o); err != nil {
				return err
			}
			g.Add(&o)
			out, err := objectResult(&o)
			if err != nil {
				return err
			}
			out["previous_version"] = prev
			out["projection_changed"] = prev != o.Version
			if e, err := a.env(ws, g, act, resolverFlags{}); err == nil {
				if flags, err := flagsOn(e, o.ID); err == nil {
					out["flags"] = flags
				}
			}
			if f.policy != "" && o.Stability == "firm" {
				out["policy"] = f.policy
			}
			return a.emit(out, func(w io.Writer) {
				if prev == o.Version {
					fmt.Fprintf(w, "Edited %s (version unchanged %s)\n", o.ID, o.Version)
				} else {
					fmt.Fprintf(w, "Edited %s (%s -> %s)\n", o.ID, prev, o.Version)
				}
			})
		}),
	}
	addIntentionFieldFlags(cmd, &f, true)
	addActFlags(cmd, &act)
	return cmd
}

func (a *app) intentionFirmCmd() *cobra.Command {
	var act actFlags
	var policy string
	cmd := &cobra.Command{
		Use:   "firm <id>",
		Short: "Set stability to firm; a harness needs --policy naming an auto_firm policy the subject holds",
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
			before, err := getIntention(g, args[0])
			if err != nil {
				return err
			}
			if err := model.CheckNotRetired(before); err != nil {
				return err
			}
			src := resolveSource(ws, act)
			if before.Stability == "firm" {
				return refusedErr("%s is already firm", before.ID)
			}
			fu, err := model.CheckFirm(g, before, src, policy)
			if err != nil {
				return err
			}
			o := *before
			o.Stability = "firm"
			o.FirmedUnder = fu
			prev, _ := projection.Version(before)
			if err := writeObject(ws, &o); err != nil {
				return err
			}
			out, err := objectResult(&o)
			if err != nil {
				return err
			}
			out["previous_version"] = prev
			out["source"] = src
			if policy != "" {
				out["policy"] = policy
			}
			return a.emit(out, func(w io.Writer) {
				fmt.Fprintf(w, "Firmed %s (%s -> %s)\n", o.ID, prev, o.Version)
			})
		}),
	}
	cmd.Flags().StringVar(&policy, "policy", "", "id of the terminus carrying auto_firm that authorises a harness to firm; written to firmed_under")
	addActFlags(cmd, &act)
	return cmd
}

func (a *app) intentionRetireCmd() *cobra.Command {
	var act actFlags
	var kind, supersededBy, reason string
	cmd := &cobra.Command{
		Use:   "retire <id>",
		Short: "Append a retirement record: fulfilled, abandoned, or superseded (with --superseded-by)",
		Args:  cobra.ExactArgs(1),
		RunE: a.run(func(cmd *cobra.Command, args []string) error {
			if kind == "" {
				return usageErr("--kind is required: %s", strings.Join(model.IntentionRetirementKinds, ", "))
			}
			ws, err := a.openWorkspace()
			if err != nil {
				return err
			}
			g, err := loadGraph(ws)
			if err != nil {
				return err
			}
			before, err := getIntention(g, args[0])
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
			return a.emit(out, func(w io.Writer) {
				fmt.Fprintf(w, "Retired %s as %s (%s -> %s)\n", o.ID, kind, prev, o.Version)
			})
		}),
	}
	cmd.Flags().StringVar(&kind, "kind", "", "fulfilled, abandoned, or superseded")
	cmd.Flags().StringVar(&supersededBy, "superseded-by", "", "the replacing intention (required with kind superseded)")
	cmd.Flags().StringVar(&reason, "reason", "", "prose reason")
	addActFlags(cmd, &act)
	return cmd
}

func (a *app) intentionShowCmd() *cobra.Command {
	var nowFlag string
	cmd := &cobra.Command{
		Use:   "show <id>",
		Short: "Show an intention with its computed version, resolved serves targets, and current flags",
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
			o, err := getIntention(g, args[0])
			if err != nil {
				return err
			}
			out, err := objectResult(o)
			if err != nil {
				return err
			}
			resolved := []map[string]any{}
			for _, r := range o.Serves {
				entry := map[string]any{"id": r.ID, "role": r.Role}
				if t, ok := g.Get(r.ID); ok {
					if ti, ok := t.(*model.Intention); ok {
						entry["title"] = ti.Title
						if ti.Retired != nil {
							entry["retired"] = ti.Retired.Kind
						}
					}
				} else {
					entry["missing"] = true
				}
				resolved = append(resolved, entry)
			}
			out["serves_resolved"] = resolved
			if ld := g.Loaded[o.ID]; ld != nil && len(ld.Problems) > 0 {
				out["problems"] = ld.Problems
			}
			e, err := a.env(ws, g, actFlags{now: nowFlag}, resolverFlags{})
			if err != nil {
				return err
			}
			flags, err := flagsOn(e, o.ID)
			if err != nil {
				return err
			}
			out["flags"] = flags
			return a.emit(out, func(w io.Writer) {
				data, _ := model.Encode(o)
				fmt.Fprint(w, string(data))
				for _, f := range flags {
					fmt.Fprintf(w, "# flag %s %s: %s\n", f.Kind, f.Counterpart, f.Detail)
				}
			})
		}),
	}
	cmd.Flags().StringVar(&nowFlag, "now", "", "the current time as RFC 3339, for the check")
	return cmd
}

func (a *app) intentionListCmd() *cobra.Command {
	var subject, activity, stability, instancesOf string
	var recurring, retired, placed, unplaced bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List intentions, active by default",
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
			var items []*model.Intention
			for _, o := range g.Intentions() {
				if (o.Retired != nil) != retired {
					continue
				}
				if subject != "" && o.Subject != subject {
					continue
				}
				if activity != "" && o.Activity != activity {
					continue
				}
				if stability != "" && o.Stability != stability {
					continue
				}
				if recurring && !o.IsRecurring() {
					continue
				}
				if placed && o.Placement == nil {
					continue
				}
				if unplaced && o.Placement != nil {
					continue
				}
				if instancesOf != "" && o.InstanceOf() != instancesOf {
					continue
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
			return a.emit(map[string]any{"intentions": list, "count": len(list)}, func(w io.Writer) {
				for _, o := range items {
					flags := ""
					if o.IsRecurring() {
						flags += " recurring"
					}
					if o.Retired != nil {
						flags += " retired:" + o.Retired.Kind
					}
					if o.Placement != nil {
						flags += " placed:" + o.Placement.Start.Raw
					}
					if o.Occurrence != "" {
						flags += " occurrence:" + o.Occurrence
					}
					fmt.Fprintf(w, "%s  %-9s %-6s %-22s %s%s\n", o.ID, o.Stability, durationText(o.Duration), oneLine(windowText(o.Window), 22), oneLine(o.Title, 60), flags)
				}
				if len(items) == 0 {
					fmt.Fprintln(w, "(no intentions)")
				}
			})
		}),
	}
	cmd.Flags().StringVar(&subject, "subject", "", "filter by subject URI")
	cmd.Flags().StringVar(&activity, "activity", "", "filter by activity term")
	cmd.Flags().StringVar(&stability, "stability", "", "filter by stability")
	cmd.Flags().BoolVar(&recurring, "recurring", false, "only recurring intentions (with a cadence)")
	cmd.Flags().BoolVar(&placed, "placed", false, "only intentions with a placement")
	cmd.Flags().BoolVar(&unplaced, "unplaced", false, "only intentions without a placement")
	cmd.Flags().StringVar(&instancesOf, "instances-of", "", "only instances of this recurring intention")
	cmd.Flags().BoolVar(&retired, "retired", false, "list retired intentions instead of active ones")
	return cmd
}
