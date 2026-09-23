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

// desireFields are the flags desire add and edit share.
type desireFields struct {
	title, description, descriptionFile string
	subject, activity, reference        string
	location, parties, serves           []string
	clear                               map[string]*bool
}

var clearableDesireFields = []string{"description", "activity", "location", "parties", "serves", "reference"}

func addDesireFieldFlags(cmd *cobra.Command, f *desireFields, edit bool) {
	cmd.Flags().StringVar(&f.title, "title", "", "prose title (required on add): what the person said they wanted")
	cmd.Flags().StringVar(&f.description, "description", "", "prose description")
	cmd.Flags().StringVar(&f.descriptionFile, "description-file", "", "read the description from a file, or - for piped stdin")
	cmd.Flags().StringVar(&f.activity, "activity", "", "activity term, a hint carried onto the intention on adoption")
	cmd.Flags().StringArrayVar(&f.location, "location", nil, "URI, a hint carried onto the intention on adoption (repeatable)")
	cmd.Flags().StringArrayVar(&f.parties, "party", nil, "URI of another particular the want involves, a hint carried onto the intention on adoption (repeatable)")
	cmd.Flags().StringArrayVar(&f.serves, "serves", nil, "<terminus id>:for-the-sake-of, the only role a desire admits; the terminus may be a draft (repeatable)")
	cmd.Flags().StringVar(&f.reference, "reference", "", "informal pointer to a DKF claim; never validated")
	if !edit {
		cmd.Flags().StringVar(&f.subject, "subject", "", "URI of the particular whose desire this is (default: defaults.subject)")
	}
	f.clear = map[string]*bool{}
	if edit {
		for _, name := range clearableDesireFields {
			b := new(bool)
			f.clear[name] = b
			cmd.Flags().BoolVar(b, "clear-"+name, false, "remove "+name)
		}
	}
}

func (a *app) desireCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "desire",
		Short: "Add, edit, adopt, retire, show and list desires: wants the person expressed and has not committed to",
	}
	cmd.AddCommand(a.desireAddCmd(), a.desireEditCmd(), a.desireAdoptCmd(), a.desireRetireCmd(), a.desireShowCmd(), a.desireListCmd())
	return cmd
}

func getDesire(g *store.Graph, id string) (*model.Desire, error) {
	obj, err := getObject(g, id)
	if err != nil {
		return nil, err
	}
	d, ok := obj.(*model.Desire)
	if !ok {
		return nil, usageErr("%s is a %s, not a desire", id, obj.GetType())
	}
	return d, nil
}

// applyDesireFields merges flags into o.
func (a *app) applyDesireFields(cmd *cobra.Command, f desireFields, o *model.Desire) error {
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
		o.Serves = refs
	}
	if cleared("serves") {
		o.Serves = []model.Ref{}
	}
	if changed("reference") {
		o.Reference = f.reference
	}
	if cleared("reference") {
		o.Reference = ""
	}
	return nil
}

func (a *app) desireAddCmd() *cobra.Command {
	var f desireFields
	var act actFlags
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Record a want the person expressed; no why or when required",
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
			o := &model.Desire{ID: model.MintID(model.TypeDesire), Source: src, Timestamp: ts, Serves: []model.Ref{}}
			o.Subject = firstNonEmpty(f.subject, ws.Config.Defaults.Subject)
			if o.Subject == "" {
				return refusedErr("subject is required: pass --subject, or set defaults.subject in intentions.yaml; this workspace has no default")
			}
			if err := a.applyDesireFields(cmd, f, o); err != nil {
				return err
			}
			if ps := model.Check(o); len(ps) > 0 {
				return ps
			}
			if err := model.CheckDesireServes(g, o); err != nil {
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
			return a.emit(out, func(w io.Writer) {
				fmt.Fprintf(w, "Created %s (%s)\n  %s\n", o.ID, o.Version, out["path"])
			})
		}),
	}
	addDesireFieldFlags(cmd, &f, false)
	addActFlags(cmd, &act)
	return cmd
}

func (a *app) desireEditCmd() *cobra.Command {
	var f desireFields
	var act actFlags
	cmd := &cobra.Command{
		Use:   "edit <id>",
		Short: "Edit a desire in place; only a change of terminus moves the version",
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
			before, err := getDesire(g, args[0])
			if err != nil {
				return err
			}
			if err := model.CheckNotRetired(before); err != nil {
				return err
			}
			o := *before
			o.Serves = append([]model.Ref{}, before.Serves...)
			if err := a.applyDesireFields(cmd, f, &o); err != nil {
				return err
			}
			if ps := model.Check(&o); len(ps) > 0 {
				return ps
			}
			if err := model.CheckDesireServes(g, &o); err != nil {
				return err
			}
			prev, _ := projection.Version(before)
			if ts, err := actTime(act); err != nil {
				return err
			} else {
				o.Timestamp = ts
			}
			if err := writeObject(ws, &o); err != nil {
				return err
			}
			out, err := objectResult(&o)
			if err != nil {
				return err
			}
			out["previous_version"] = prev
			out["projection_changed"] = prev != o.Version
			return a.emit(out, func(w io.Writer) {
				if prev == o.Version {
					fmt.Fprintf(w, "Edited %s (version unchanged %s)\n", o.ID, o.Version)
				} else {
					fmt.Fprintf(w, "Edited %s (%s -> %s)\n", o.ID, prev, o.Version)
				}
			})
		}),
	}
	addDesireFieldFlags(cmd, &f, true)
	addActFlags(cmd, &act)
	return cmd
}

func (a *app) desireAdoptCmd() *cobra.Command {
	var act actFlags
	var duration string
	var window windowFlags
	cmd := &cobra.Command{
		Use:   "adopt <id>",
		Short: "Turn a want into a plan: write a tentative intention from it, then retire the desire as adopted naming it",
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
			d, err := getDesire(g, args[0])
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
			var dur *temporal.DurationSpec
			if cmd.Flags().Changed("duration") {
				if dur, err = parseDurationFlag(duration); err != nil {
					return err
				}
			}
			win, err := applyWindow(cmd, window, nil, ws.Config.Context(now), g)
			if err != nil {
				return err
			}
			in, r, err := model.Adopt(d, dur, win, src, ts)
			if err != nil {
				return err
			}
			if ps := model.Check(in); len(ps) > 0 {
				return ps
			}
			if err := checkIntentionWrite(g, nil, in, src, ""); err != nil {
				return err
			}
			// The intention first: it is the record of the adoption. If the
			// retirement then fails, the desire stays visibly active.
			if err := writeObject(ws, in); err != nil {
				return err
			}
			g.Add(in)
			retired := *d
			retired.Retired = &r
			retired.Timestamp = ts
			if err := writeObject(ws, &retired); err != nil {
				return err
			}
			inOut, err := objectResult(in)
			if err != nil {
				return err
			}
			dOut, err := objectResult(&retired)
			if err != nil {
				return err
			}
			out := map[string]any{"intention": inOut, "desire": dOut, "id": in.ID}
			fs := attachFindings(out, g, in)
			return a.emit(out, func(w io.Writer) {
				fmt.Fprintf(w, "Adopted %s as %s (%s)\n  %s\n", d.ID, in.ID, in.Version, inOut["path"])
				printFindings(w, fs)
			})
		}),
	}
	cmd.Flags().StringVar(&duration, "duration", "", "ISO 8601 duration (PT90M) or a range nominal:min:max")
	addWindowFlags(cmd, &window, false)
	addActFlags(cmd, &act)
	return cmd
}

func (a *app) desireRetireCmd() *cobra.Command {
	var act actFlags
	var kind, supersededBy, reason string
	cmd := &cobra.Command{
		Use:   "retire <id>",
		Short: "Append a retirement record: abandoned, or superseded (with --superseded-by); adopted is written only by adopt",
		Args:  cobra.ExactArgs(1),
		RunE: a.run(func(cmd *cobra.Command, args []string) error {
			if kind == "" {
				return usageErr("--kind is required: abandoned or superseded")
			}
			ws, err := a.openWorkspace()
			if err != nil {
				return err
			}
			g, err := loadGraph(ws)
			if err != nil {
				return err
			}
			before, err := getDesire(g, args[0])
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
			o.Timestamp = ts
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
	cmd.Flags().StringVar(&kind, "kind", "", "abandoned or superseded")
	cmd.Flags().StringVar(&supersededBy, "superseded-by", "", "the replacing desire (required with kind superseded)")
	cmd.Flags().StringVar(&reason, "reason", "", "prose reason")
	addActFlags(cmd, &act)
	return cmd
}

// desireServesResolved names each terminus a desire serves.
func desireServesResolved(g *store.Graph, o *model.Desire) []map[string]any {
	resolved := []map[string]any{}
	for _, r := range o.Serves {
		entry := map[string]any{"id": r.ID, "role": r.Role}
		if t, ok := g.Get(r.ID); ok {
			if ti, ok := t.(*model.Intention); ok {
				entry["title"] = ti.Title
				entry["stability"] = ti.Stability
				if ti.Retired != nil {
					entry["retired"] = ti.Retired.Kind
				}
			}
		} else {
			entry["missing"] = true
		}
		resolved = append(resolved, entry)
	}
	return resolved
}

func (a *app) desireShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <id>",
		Short: "Show a desire with its computed version and the terminus it serves",
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
			o, err := getDesire(g, args[0])
			if err != nil {
				return err
			}
			out, err := objectResult(o)
			if err != nil {
				return err
			}
			out["serves_resolved"] = desireServesResolved(g, o)
			if ld := g.Loaded[o.ID]; ld != nil && len(ld.Problems) > 0 {
				out["problems"] = ld.Problems
			}
			return a.emit(out, func(w io.Writer) {
				data, _ := model.Encode(o)
				fmt.Fprint(w, string(data))
			})
		}),
	}
}

func (a *app) desireListCmd() *cobra.Command {
	var subject, activity string
	var retired bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List desires, active by default: the inbox",
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
			var items []*model.Desire
			for _, o := range g.Desires() {
				if (o.Retired != nil) != retired || (subject != "" && o.Subject != subject) || (activity != "" && o.Activity != activity) {
					continue
				}
				items = append(items, o)
			}
			list := make([]map[string]any, 0, len(items))
			for _, o := range items {
				m, err := objectResult(o)
				if err != nil {
					return err
				}
				list = append(list, m)
			}
			return a.emit(map[string]any{"desires": list, "count": len(list)}, func(w io.Writer) {
				for _, o := range items {
					fmt.Fprintf(w, "%s  %s\n", o.ID, oneLine(o.Title, 60))
				}
				fmt.Fprintf(w, "%d desires\n", len(items))
			})
		}),
	}
	cmd.Flags().StringVar(&subject, "subject", "", "filter by subject URI")
	cmd.Flags().StringVar(&activity, "activity", "", "filter by activity term")
	cmd.Flags().BoolVar(&retired, "retired", false, "list retired desires instead of active")
	return cmd
}
