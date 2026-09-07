package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/nodelogicau/intentions-cli/internal/consistency"
	"github.com/nodelogicau/intentions-cli/internal/model"
	"github.com/nodelogicau/intentions-cli/internal/projection"
	"github.com/nodelogicau/intentions-cli/internal/render"
	"github.com/nodelogicau/intentions-cli/internal/resolve"
	"github.com/nodelogicau/intentions-cli/internal/store"
	"github.com/nodelogicau/intentions-cli/internal/temporal"
)

// resolverFlags are the context overrides shared by resolve, select, check.
type resolverFlags struct {
	step, scope string
}

func addResolverFlags(cmd *cobra.Command, f *resolverFlags) {
	cmd.Flags().StringVar(&f.step, "step", "", "candidate grid as an ISO 8601 duration (default resolver.step, PT15M)")
	cmd.Flags().StringVar(&f.scope, "scope", "", "the resolver's own scope: personal, organisation, public (default resolver.scope)")
}

func (a *app) env(ws *store.Workspace, g *store.Graph, act actFlags, f resolverFlags) (resolve.Env, error) {
	now, err := resolveNow(act)
	if err != nil {
		return resolve.Env{}, err
	}
	e := resolve.NewEnv(ws, g, now)
	if f.step != "" {
		d, err := temporal.ParseDuration(f.step)
		if err != nil || d.IsZero() || !d.HasTimePart() {
			return e, usageErr("--step must be a positive time-of-day duration such as PT30M")
		}
		e.Step = d
	}
	if f.scope != "" {
		ok := false
		for _, s := range model.Scopes {
			if s == f.scope {
				ok = true
			}
		}
		if !ok {
			return e, usageErr("--scope must be personal, organisation, or public")
		}
		e.Scope = f.scope
	}
	return e, nil
}

// flagsFor runs the check and keeps the flags touching the given ids, as
// subject or counterpart.
func flagsFor(e resolve.Env, ids ...string) ([]consistency.Flag, error) {
	return consistency.Check(e, ids)
}

// flagsOn runs the check and keeps only the flags reported on the object
// itself: what show, edit and acknowledge carry.
func flagsOn(e resolve.Env, id string) ([]consistency.Flag, error) {
	all, err := consistency.Check(e, []string{id})
	if err != nil {
		return nil, err
	}
	out := []consistency.Flag{}
	for _, f := range all {
		if f.Subject == id {
			out = append(out, f)
		}
	}
	return out, nil
}

func (a *app) generateCmd() *cobra.Command {
	var act actFlags
	var horizon string
	var only []string
	cmd := &cobra.Command{
		Use:   "generate [--horizon <ISO duration>] [--recurring <id>]...",
		Short: "Materialise instances of recurring intentions over a horizon; idempotent on (recurring, occurrence)",
		Args:  cobra.NoArgs,
		RunE: a.run(func(cmd *cobra.Command, args []string) error {
			ws, err := a.openWorkspace()
			if err != nil {
				return err
			}
			g, err := loadGraph(ws)
			if err != nil {
				return err
			}
			e, err := a.env(ws, g, act, resolverFlags{})
			if err != nil {
				return err
			}
			h := ws.Config.GenerationHorizon()
			if horizon != "" {
				if h, err = temporal.ParseDuration(horizon); err != nil {
					return usageErr("--horizon: %v", err)
				}
			}
			for _, id := range only {
				if _, err := getIntention(g, id); err != nil {
					return err
				}
			}
			ts, err := resolveTimestamp(act, e.Ctx.Now)
			if err != nil {
				return err
			}
			r := temporal.Interval{Start: e.Ctx.Now, End: temporal.AddDuration(e.Ctx.Now, h)}
			created, skipped, err := resolve.Generate(e, resolve.GenerateOptions{Range: r, Only: only, Source: resolveSource(ws, act), Timestamp: ts})
			if err != nil {
				return err
			}
			list := make([]map[string]any, 0, len(created))
			for _, in := range created {
				m, err := model.ToMap(in)
				if err != nil {
					return err
				}
				list = append(list, m)
			}
			if skipped == nil {
				skipped = []resolve.Skip{}
			}
			out := map[string]any{"created": list, "skipped": skipped, "count": len(created), "range": map[string]any{"start": r.Start.Format("2006-01-02T15:04:05Z07:00"), "end": r.End.Format("2006-01-02T15:04:05Z07:00")}}
			return a.emit(out, func(w io.Writer) {
				for _, in := range created {
					fmt.Fprintf(w, "Created %s  %s  %s\n", in.ID, in.Occurrence, oneLine(in.Title, 60))
				}
				for _, s := range skipped {
					fmt.Fprintf(w, "Skipped %s %s: %s\n", s.Recurring, s.Occurrence, s.Reason)
				}
				fmt.Fprintf(w, "%d created, %d skipped\n", len(created), len(skipped))
			})
		}),
	}
	cmd.Flags().StringVar(&horizon, "horizon", "", "how far ahead to generate (default resolver.horizon)")
	cmd.Flags().StringArrayVar(&only, "recurring", nil, "only this recurring intention (repeatable)")
	addActFlags(cmd, &act)
	return cmd
}

func (a *app) resolveCmd() *cobra.Command {
	var act actFlags
	var rf resolverFlags
	var limit int
	cmd := &cobra.Command{
		Use:   "resolve <id>",
		Short: "Rank candidate placements for an intention against the availability of every required particular; chooses nothing",
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
			in, err := getIntention(g, args[0])
			if err != nil {
				return err
			}
			e, err := a.env(ws, g, act, rf)
			if err != nil {
				return err
			}
			generated, err := a.generateForRange(e, in, act)
			if err != nil {
				return err
			}
			res, err := resolve.Resolve(e, in, resolve.Options{Limit: limit})
			if err != nil {
				return err
			}
			res.Generated = generated
			out := resultMap(res, in.ID)
			return a.emit(out, func(w io.Writer) {
				a.printResult(w, in, res)
			})
		}),
	}
	cmd.Flags().IntVar(&limit, "limit", 20, "candidates to return; candidates_considered keeps the full count")
	addResolverFlags(cmd, &rf)
	addActFlags(cmd, &act)
	return cmd
}

// generateForRange runs generation over the intention's resolution range so
// instances exist before ranking, as the spec requires.
func (a *app) generateForRange(e resolve.Env, in *model.Intention, act actFlags) ([]string, error) {
	generated := []string{}
	if in.Window == nil {
		return generated, nil
	}
	r, reason := temporal.ResolutionRange(*in.Window, e.Ctx, e.Horizon)
	if reason != "" {
		return generated, nil
	}
	created, _, err := resolve.Generate(e, resolve.GenerateOptions{Range: r, Source: resolveSource(e.WS, act), Timestamp: e.Ctx.Now})
	if err != nil {
		return nil, err
	}
	for _, c := range created {
		generated = append(generated, c.ID)
	}
	return generated, nil
}

func resultMap(res resolve.Result, id string) map[string]any { return render.Resolution(res, id) }

func (a *app) printResult(w io.Writer, in *model.Intention, res resolve.Result) {
	fmt.Fprintf(w, "%s  %s\n", in.ID, oneLine(in.Title, 60))
	if res.RangeText != "" {
		fmt.Fprintf(w, "range %s\n", res.RangeText)
	}
	for _, g := range res.Generated {
		fmt.Fprintf(w, "generated %s\n", g)
	}
	if res.Reason != "" {
		fmt.Fprintf(w, "no candidates: %s\n", res.Reason)
	}
	if len(res.Presumed) > 0 {
		fmt.Fprintf(w, "presumed %s: the workspace holds no availability for them, so their time is assumed\n", strings.Join(res.Presumed, ", "))
	}
	for _, ex := range res.Excluded {
		fmt.Fprintf(w, "  excluded %s: %s\n", ex.Availability, ex.Reason)
	}
	for i, c := range res.Candidates {
		disp := ""
		if len(c.Displaces) > 0 {
			disp = "  displaces " + strings.Join(c.Displaces, ", ")
		}
		loc := ""
		if c.Location != "" {
			loc = "  at " + c.Location
		}
		fmt.Fprintf(w, "%3d  rank %d  %s / %s%s%s\n", i+1, c.Rank, c.Start, c.End, loc, disp)
	}
	if res.Considered > len(res.Candidates) {
		fmt.Fprintf(w, "(%d of %d candidates shown)\n", len(res.Candidates), res.Considered)
	}
}

func (a *app) selectCmd() *cobra.Command {
	var act actFlags
	var rf resolverFlags
	var candidate, limit int
	var policy string
	var replace bool
	cmd := &cobra.Command{
		Use:   "select <id> (--candidate N | --policy <int_id>) [--replace]",
		Short: "Select a candidate: the recorded act that writes the RESOLUTION record, the placement, and a commitment when parties are involved",
		Args:  cobra.ExactArgs(1),
		RunE: a.run(func(cmd *cobra.Command, args []string) error {
			if (candidate == 0) == (policy == "") {
				return usageErr("pass exactly one of --candidate N (a person's choice) or --policy <int_id> (a harness under a policy)")
			}
			ws, err := a.openWorkspace()
			if err != nil {
				return err
			}
			g, err := loadGraph(ws)
			if err != nil {
				return err
			}
			in, err := getIntention(g, args[0])
			if err != nil {
				return err
			}
			e, err := a.env(ws, g, act, rf)
			if err != nil {
				return err
			}
			ts, err := resolveTimestamp(act, e.Ctx.Now)
			if err != nil {
				return err
			}
			src := resolveSource(ws, act)
			if _, err := a.generateForRange(e, in, act); err != nil {
				return err
			}
			sel, res, err := resolve.Select(e, in, resolve.SelectOptions{Candidate: candidate, Policy: policy, Source: src, Timestamp: ts, Replace: replace, Limit: limit})
			if err != nil {
				return err
			}
			flags, err := flagsFor(e, sel.Intention.ID, sel.Resolution.ID, commitmentID(sel))
			if err != nil {
				return err
			}
			out := render.Selection(sel, res, flags, policy)
			return a.emit(out, func(w io.Writer) {
				fmt.Fprintln(w, sel.Describe())
				printFlags(w, flags)
			})
		}),
	}
	cmd.Flags().IntVar(&candidate, "candidate", 0, "1-based index into the candidates resolve lists (a person's choice)")
	cmd.Flags().StringVar(&policy, "policy", "", "id of the terminus carrying auto_select that authorises a harness to select the top rank-1 candidate")
	cmd.Flags().BoolVar(&replace, "replace", false, "re-resolve a placed intention, clearing its placement")
	cmd.Flags().IntVar(&limit, "limit", 0, "unused; candidates are indexed over the full ranked set")
	addResolverFlags(cmd, &rf)
	addActFlags(cmd, &act)
	return cmd
}

func commitmentID(sel resolve.Selection) string {
	if sel.Commitment == nil {
		return ""
	}
	return sel.Commitment.ID
}

func printFlags(w io.Writer, flags []consistency.Flag) {
	for _, f := range flags {
		cp := ""
		if f.Counterpart != "" {
			cp = " ↔ " + f.Counterpart
		}
		fmt.Fprintf(w, "flag %-24s %s%s: %s\n", f.Kind, f.Subject, cp, f.Detail)
	}
}

func (a *app) checkCmd() *cobra.Command {
	var act actFlags
	var rf resolverFlags
	var failOnFlags bool
	cmd := &cobra.Command{
		Use:   "check [<id>]...",
		Short: "The consistency check: every way the prospective graph fails to hang together, as flags; writes nothing",
		RunE: a.run(func(cmd *cobra.Command, args []string) error {
			ws, err := a.openWorkspace()
			if err != nil {
				return err
			}
			g, err := loadGraph(ws)
			if err != nil {
				return err
			}
			for _, id := range args {
				if _, err := getObject(g, id); err != nil {
					return err
				}
			}
			e, err := a.env(ws, g, act, rf)
			if err != nil {
				return err
			}
			flags, err := consistency.Check(e, args)
			if err != nil {
				return err
			}
			counts := map[string]int{}
			for _, f := range flags {
				counts[f.Kind]++
			}
			if err := a.emit(map[string]any{"flags": flags, "count": len(flags), "counts": counts}, func(w io.Writer) {
				printFlags(w, flags)
				fmt.Fprintf(w, "%d flags\n", len(flags))
			}); err != nil {
				return err
			}
			if failOnFlags && len(flags) > 0 {
				return checkFailed()
			}
			return nil
		}),
	}
	cmd.Flags().BoolVar(&failOnFlags, "fail-on-flags", false, "exit 4 when any flag is reported")
	addResolverFlags(cmd, &rf)
	addActFlags(cmd, &act)
	return cmd
}

func (a *app) acknowledgeCmd() *cobra.Command {
	var act actFlags
	var rf resolverFlags
	var kind, counterpart, reason string
	cmd := &cobra.Command{
		Use:   "acknowledge <id> --kind <flag kind> [--counterpart <id>] [--reason <text>]",
		Short: "Record that the person has seen a flag against a specific state of its counterpart and is proceeding anyway",
		Args:  cobra.ExactArgs(1),
		RunE: a.run(func(cmd *cobra.Command, args []string) error {
			if !consistency.ValidKind(kind) {
				return usageErr("--kind must be one of %s", strings.Join(consistency.Kinds, ", "))
			}
			ws, err := a.openWorkspace()
			if err != nil {
				return err
			}
			g, err := loadGraph(ws)
			if err != nil {
				return err
			}
			obj, err := getObject(g, args[0])
			if err != nil {
				return err
			}
			e, err := a.env(ws, g, act, rf)
			if err != nil {
				return err
			}
			ts, err := resolveTimestamp(act, e.Ctx.Now)
			if err != nil {
				return err
			}
			ack := model.Acknowledgement{Kind: kind, Counterpart: counterpart, Reason: reason, Source: resolveSource(ws, act), Timestamp: ts}
			if counterpart != "" {
				cp, err := getObject(g, counterpart)
				if err != nil {
					return err
				}
				v, err := projectionVersion(cp)
				if err != nil {
					return err
				}
				ack.CounterpartVersion = v
			} else if kind != consistency.Cycle {
				// A window-clash against absent supply has no counterpart; any other kind names one.
				flags, _ := consistency.Check(e, []string{obj.GetID()})
				bare := false
				for _, f := range flags {
					if f.Subject == obj.GetID() && f.Kind == kind && f.Counterpart == "" {
						bare = true
					}
				}
				if !bare {
					return usageErr("--kind %s names a counterpart; pass --counterpart <id>", kind)
				}
			}
			var updated model.Object
			switch o := obj.(type) {
			case *model.Intention:
				c := *o
				c.Acknowledgements = append(append([]model.Acknowledgement(nil), o.Acknowledgements...), ack)
				updated = &c
			case *model.Commitment:
				c := *o
				c.Acknowledgements = append(append([]model.Acknowledgement(nil), o.Acknowledgements...), ack)
				updated = &c
			default:
				return usageErr("%s is a %s; only intentions and commitments carry acknowledgements", obj.GetID(), obj.GetType())
			}
			if err := ws.WriteObject(updated); err != nil {
				return err
			}
			g.Add(updated)
			flags, err := flagsOn(e, updated.GetID())
			if err != nil {
				return err
			}
			out, err := objectResult(updated)
			if err != nil {
				return err
			}
			out["acknowledgement"] = render.Acknowledgement(ack)
			out["flags"] = flags
			return a.emit(out, func(w io.Writer) {
				fmt.Fprintf(w, "Acknowledged %s on %s", kind, updated.GetID())
				if counterpart != "" {
					fmt.Fprintf(w, " against %s@%s", counterpart, ack.CounterpartVersion)
				}
				fmt.Fprintln(w)
				printFlags(w, flags)
			})
		}),
	}
	cmd.Flags().StringVar(&kind, "kind", "", "the flag kind acknowledged: "+strings.Join(consistency.Kinds, ", "))
	cmd.Flags().StringVar(&counterpart, "counterpart", "", "the other object the flag named")
	cmd.Flags().StringVar(&reason, "reason", "", "why the person is proceeding anyway")
	addResolverFlags(cmd, &rf)
	addActFlags(cmd, &act)
	return cmd
}

func (a *app) unresolvedCmd() *cobra.Command {
	var act actFlags
	var rf resolverFlags
	var subject string
	cmd := &cobra.Command{
		Use:   "unresolved [--subject <uri>]",
		Short: "Every active intention still without a placement, and what stands in its way; writes nothing",
		Long: `Lists each unretired intention that has no placement and could take one: not a
terminus, not a recurring intention (its instances are what get placed, and
they appear as resolution or generate materialises them). Each entry carries a status from a dry
resolution: ready (with the candidate count and best rank), blocked (on a
relational target with no placement), no_candidates (with the resolver's
reason), incomplete (duration or window missing), or unresolvable (a serves
cycle). Soonest deadline first.`,
		Args: cobra.NoArgs,
		RunE: a.run(func(cmd *cobra.Command, args []string) error {
			ws, err := a.openWorkspace()
			if err != nil {
				return err
			}
			g, err := loadGraph(ws)
			if err != nil {
				return err
			}
			e, err := a.env(ws, g, act, rf)
			if err != nil {
				return err
			}
			entries, err := resolve.Unresolved(e, resolve.UnresolvedOptions{Subject: subject})
			if err != nil {
				return err
			}
			return a.emit(resolve.UnresolvedResult(entries), func(w io.Writer) {
				printUnresolved(w, entries)
			})
		}),
	}
	cmd.Flags().StringVar(&subject, "subject", "", "only intentions of this subject")
	addResolverFlags(cmd, &rf)
	addActFlags(cmd, &act)
	return cmd
}

func printUnresolved(w io.Writer, entries []resolve.UnresolvedEntry) {
	for _, e := range entries {
		fmt.Fprintf(w, "%s  %s  %s\n", e.ID, e.Title, e.Status)
		switch e.Status {
		case resolve.StatusReady:
			fmt.Fprintf(w, "  %d candidates, best rank %d", e.Candidates, e.BestRank)
			if e.Deadline != "" {
				fmt.Fprintf(w, ", by %s", e.Deadline)
			}
			fmt.Fprintln(w)
		default:
			fmt.Fprintf(w, "  %s\n", e.Reason)
		}
	}
	fmt.Fprintf(w, "%d unresolved\n", len(entries))
}

func (a *app) commitmentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "commitment",
		Short: "Answer, cancel, show and list commitments",
	}
	cmd.AddCommand(a.commitmentAnswerCmd("accept"), a.commitmentAnswerCmd("decline"), a.commitmentCancelCmd(), a.commitmentShowCmd(), a.commitmentListCmd())
	return cmd
}

// commitmentAnswerCmd builds accept and decline, which differ only in the
// status they record.
func (a *app) commitmentAnswerCmd(verb string) *cobra.Command {
	status := map[string]string{"accept": "accepted", "decline": "declined"}[verb]
	var act actFlags
	var rf resolverFlags
	var party string
	cmd := &cobra.Command{
		Use:   verb + " <id> [--party <uri>]",
		Short: "Record that a party has " + status + " this commitment",
		Long: `Sets one party's status to ` + status + ` and changes nothing else. The party
defaults to defaults.subject, the workspace owner, the only entry a local act
may speak for; a workspace with no default subject must pass --party.

A party's status is a fact about their will. No policy authorises this act and
nothing infers it: record it only on the person's word. Answering for another
party is an iTIP reply, which arrives by import.`,
		Args: cobra.ExactArgs(1),
		RunE: a.run(func(cmd *cobra.Command, args []string) error {
			ws, err := a.openWorkspace()
			if err != nil {
				return err
			}
			g, err := loadGraph(ws)
			if err != nil {
				return err
			}
			c, err := getCommitment(g, args[0])
			if err != nil {
				return err
			}
			src := resolveSource(ws, act)
			if err := requireAuthor(src); err != nil {
				return err
			}
			i, err := model.CheckAnswer(c, party, status, ws.Config.Defaults.Subject)
			if err != nil {
				return err
			}
			prev, err := projection.Version(c)
			if err != nil {
				return err
			}
			o := *c
			o.Parties = append([]model.Party(nil), c.Parties...)
			o.Parties[i].Status = status
			if err := writeObject(ws, &o); err != nil {
				return err
			}
			g.Add(&o)
			out, err := objectResult(&o)
			if err != nil {
				return err
			}
			out["previous_version"] = prev
			out["party"] = o.Parties[i].URI
			out["status"] = status
			out["source"] = src
			e, err := a.env(ws, g, act, rf)
			if err != nil {
				return err
			}
			flags, err := flagsOn(e, o.ID)
			if err != nil {
				return err
			}
			out["flags"] = flags
			return a.emit(out, func(w io.Writer) {
				fmt.Fprintf(w, "%s is %s on %s (%s -> %s)\n", o.Parties[i].URI, status, o.ID, prev, o.Version)
				printFlags(w, flags)
			})
		}),
	}
	cmd.Flags().StringVar(&party, "party", "", "the party answering (default: defaults.subject)")
	addResolverFlags(cmd, &rf)
	addActFlags(cmd, &act)
	return cmd
}

func (a *app) commitmentCancelCmd() *cobra.Command {
	var act actFlags
	var reason string
	cmd := &cobra.Command{
		Use:   "cancel <id> [--reason <text>]",
		Short: "Cancel a commitment and free the intention it was for",
		Long: `Appends a retired record with kind cancelled, the only kind a commitment
admits and a terminal one. When the commitment names an intention that is
still placed, that placement is cleared in the same act: the intention keeps
its window, duration and stability, is not retired, and is eligible for
resolution again. The RESOLUTION record is left as history.`,
		Args: cobra.ExactArgs(1),
		RunE: a.run(func(cmd *cobra.Command, args []string) error {
			ws, err := a.openWorkspace()
			if err != nil {
				return err
			}
			g, err := loadGraph(ws)
			if err != nil {
				return err
			}
			c, err := getCommitment(g, args[0])
			if err != nil {
				return err
			}
			src := resolveSource(ws, act)
			if err := requireAuthor(src); err != nil {
				return err
			}
			at, err := resolveNow(act)
			if err != nil {
				return err
			}
			ts, err := resolveTimestamp(act, at)
			if err != nil {
				return err
			}
			r := model.Retired{Kind: "cancelled", Reason: reason, Source: src, Timestamp: ts}
			if err := model.CheckRetirement(g, c, r); err != nil {
				return err
			}
			prev, err := projection.Version(c)
			if err != nil {
				return err
			}
			o := *c
			o.Retired = &r
			if err := writeObject(ws, &o); err != nil {
				return err
			}
			g.Add(&o)
			out, err := objectResult(&o)
			if err != nil {
				return err
			}
			out["previous_version"] = prev
			out["cancelled"] = map[string]any{"reason": reason, "timestamp": temporal.FormatTimestamp(ts)}
			// Free the intention it was for: the placement goes, nothing else.
			if o.Intention != "" {
				if in, ok := g.Get(o.Intention); ok {
					if in, ok := in.(*model.Intention); ok && in.Retired == nil && in.Placement != nil {
						freed := *in
						freed.Placement = nil
						if err := writeObject(ws, &freed); err != nil {
							return err
						}
						g.Add(&freed)
						fm, err := objectResult(&freed)
						if err != nil {
							return err
						}
						out["freed"] = fm
					}
				}
			}
			return a.emit(out, func(w io.Writer) {
				fmt.Fprintf(w, "Cancelled %s\n", o.ID)
				if _, ok := out["freed"]; ok {
					fmt.Fprintf(w, "  %s is unplaced again and may be resolved\n", o.Intention)
				}
			})
		}),
	}
	cmd.Flags().StringVar(&reason, "reason", "", "why it was cancelled")
	addActFlags(cmd, &act)
	return cmd
}

func (a *app) commitmentShowCmd() *cobra.Command {
	var act actFlags
	var rf resolverFlags
	cmd := &cobra.Command{
		Use:   "show <id>",
		Short: "Show one commitment with its version, its intention and its flags",
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
			c, err := getCommitment(g, args[0])
			if err != nil {
				return err
			}
			out, err := objectResult(c)
			if err != nil {
				return err
			}
			if c.Intention != "" {
				entry := map[string]any{"id": c.Intention}
				if in, ok := g.Get(c.Intention); ok {
					if in, ok := in.(*model.Intention); ok {
						entry["title"] = in.Title
						entry["placed"] = in.Placement != nil
						if in.Retired != nil {
							entry["retired"] = in.Retired.Kind
						}
					}
				} else {
					entry["missing"] = true
				}
				out["intention_resolved"] = entry
			}
			e, err := a.env(ws, g, act, rf)
			if err != nil {
				return err
			}
			flags, err := flagsOn(e, c.ID)
			if err != nil {
				return err
			}
			out["flags"] = flags
			return a.emit(out, func(w io.Writer) {
				fmt.Fprintf(w, "%s  %s\n", c.ID, firstNonEmpty(c.Title, c.Intention))
				if c.Placement != nil {
					fmt.Fprintf(w, "  placement %s for %s\n", c.Placement.Start.Raw, c.Placement.Duration.String())
				}
				for _, p := range c.Parties {
					fmt.Fprintf(w, "  %-10s %s\n", p.Status, p.URI)
				}
				if c.Retired != nil {
					fmt.Fprintf(w, "  retired %s\n", c.Retired.Kind)
				}
				printFlags(w, flags)
			})
		}),
	}
	addResolverFlags(cmd, &rf)
	addActFlags(cmd, &act)
	return cmd
}

func (a *app) commitmentListCmd() *cobra.Command {
	var party, status, intention string
	var cancelled bool
	cmd := &cobra.Command{
		Use:   "list [--party <uri>] [--status <s>] [--intention <id>] [--cancelled]",
		Short: "List commitments, active by default",
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
			if status != "" && !model.ValidPartyStatus(status) {
				return usageErr("--status must be one of %s", strings.Join(model.PartyStatuses, ", "))
			}
			var items []*model.Commitment
			for _, o := range g.Commitments() {
				if (o.Retired != nil) != cancelled || (intention != "" && o.Intention != intention) {
					continue
				}
				if party != "" || status != "" {
					match := false
					for _, p := range o.Parties {
						if (party == "" || p.URI == party) && (status == "" || p.Status == status) {
							match = true
						}
					}
					if !match {
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
			return a.emit(map[string]any{"commitments": list, "count": len(list)}, func(w io.Writer) {
				for _, o := range items {
					start := ""
					if o.Placement != nil {
						start = o.Placement.Start.Raw
					}
					statuses := make([]string, 0, len(o.Parties))
					for _, p := range o.Parties {
						statuses = append(statuses, p.Status)
					}
					fmt.Fprintf(w, "%s  %-25s %-30s %s\n", o.ID, start, oneLine(firstNonEmpty(o.Title, o.Intention), 30), strings.Join(statuses, "/"))
				}
				if len(items) == 0 {
					fmt.Fprintln(w, "(no commitments)")
				}
			})
		}),
	}
	cmd.Flags().StringVar(&party, "party", "", "filter by a party URI")
	cmd.Flags().StringVar(&status, "status", "", "filter by a party status: tentative, accepted, declined")
	cmd.Flags().StringVar(&intention, "intention", "", "filter by the intention fulfilled")
	cmd.Flags().BoolVar(&cancelled, "cancelled", false, "list cancelled commitments instead of active")
	return cmd
}

// getCommitment resolves an id that must name a commitment.
func getCommitment(g *store.Graph, id string) (*model.Commitment, error) {
	obj, err := getObject(g, id)
	if err != nil {
		return nil, err
	}
	c, ok := obj.(*model.Commitment)
	if !ok {
		return nil, usageErr("%s is a %s, not a commitment", id, obj.GetType())
	}
	return c, nil
}
