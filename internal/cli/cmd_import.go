package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/nodelogicau/intentions-cli/internal/importer"
	"github.com/nodelogicau/intentions-cli/internal/query"
	"github.com/nodelogicau/intentions-cli/internal/render"
	"github.com/nodelogicau/intentions-cli/internal/resolve"
)

func (a *app) commitmentImportCmd() *cobra.Command {
	var act actFlags
	var event, subject, commitment string
	cmd := &cobra.Command{
		Use:   "import --event <file|-> [--subject <uri>] [--commitment <id>]",
		Short: "Import one calendar occurrence, parsed by the harness, as a commitment",
		Long: `Reads one occurrence as JSON: system, uid, start, duration, and optionally
recurrence_id (its original start, for every occurrence of a recurring event),
title, location (a URI), transparent, organizer {address, status} and
attendees [{address, status, non_participant}]. status takes iCalendar PARTSTAT
or JSCalendar participationStatus.

Parties are mapped by rule: calendar.addresses names the subject's own
addresses, the subject is always a party, and nothing else of an attendee
crosses. An event that is an intention's own hold creates nothing. A re-import
updates the same commitment, keeping the intention it was adopted for, and a
retired commitment is never brought back.`,
		Args: cobra.NoArgs,
		RunE: a.run(func(cmd *cobra.Command, args []string) error {
			if event == "" {
				return usageErr("--event is required: a file holding the occurrence as JSON, or - for stdin")
			}
			var data []byte
			var err error
			if event == "-" {
				data, err = io.ReadAll(a.stdin)
			} else {
				data, err = os.ReadFile(event)
			}
			if err != nil {
				return usageErr("--event: %v", err)
			}
			ev, err := importer.Decode(data)
			if err != nil {
				return err
			}
			ws, err := a.openWorkspace()
			if err != nil {
				return err
			}
			g, err := loadGraph(ws)
			if err != nil {
				return err
			}
			if subject == "" {
				subject = ws.Config.Defaults.Subject
			}
			if subject == "" {
				return usageErr("an import is made for one particular: pass --subject, or set defaults.subject")
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
			r, err := importer.Import(ws, g, ev, importer.Options{Subject: subject, Commitment: commitment, Source: src, Timestamp: ts})
			if err != nil {
				return err
			}
			out := render.Import(r, nil, nil)
			var findings []query.Finding
			if r.Commitment != nil && r.Commitment.Retired == nil {
				e, err := a.env(ws, g, act, resolverFlags{})
				if err != nil {
					return err
				}
				fs, err := flagsOn(e, r.Commitment.ID)
				if err != nil {
					return err
				}
				findings = query.CommitmentFindings(g, r.Commitment)
				out = render.Import(r, fs, findings)
			}
			return a.emit(out, func(w io.Writer) {
				fmt.Fprintln(w, render.ImportText(r))
				printFindings(w, findings)
			})
		}),
	}
	cmd.Flags().StringVar(&event, "event", "", "file holding the occurrence as JSON, or - for stdin")
	cmd.Flags().StringVar(&subject, "subject", "", "the particular the import is made for (default: defaults.subject)")
	cmd.Flags().StringVar(&commitment, "commitment", "", "the commitment this occurrence updates, when the match is ambiguous")
	addActFlags(cmd, &act)
	return cmd
}

func (a *app) commitmentAdoptCmd() *cobra.Command {
	var act actFlags
	var rf resolverFlags
	var replace bool
	cmd := &cobra.Command{
		Use:   "adopt <cmt_id> <int_id> [--replace]",
		Short: "Record that an imported event is an intention's occasion: a selection by the person",
		Long: `Writes a RESOLUTION record with selector person and the commitment's
placement, gives the intention that placement, and names the intention on the
commitment, which stays an import. Only the person decides that an event is
their intention's occasion: run it on their word, never under a policy.

The intention must be the subject's own, with no parties and no hold link, and
no other commitment may name it. A placed intention needs --replace, unless it
is the same commitment adopted again after the event moved.`,
		Args: cobra.ExactArgs(2),
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
			in, err := getIntention(g, args[1])
			if err != nil {
				return err
			}
			e, err := a.env(ws, g, act, rf)
			if err != nil {
				return err
			}
			src := resolveSource(ws, act)
			if err := requireAuthor(src); err != nil {
				return err
			}
			ts, err := resolveTimestamp(act, e.Ctx.Now)
			if err != nil {
				return err
			}
			sel, err := resolve.Adopt(e, c, in, resolve.AdoptOptions{Source: src, Timestamp: ts, Replace: replace})
			if err != nil {
				return err
			}
			flags, err := flagsFor(e, sel.Intention.ID, sel.Resolution.ID, sel.Adopted.ID)
			if err != nil {
				return err
			}
			out := render.Selection(sel, resolve.Result{Considered: 1}, flags, "")
			return a.emit(out, func(w io.Writer) {
				fmt.Fprintln(w, sel.Describe())
				printFlags(w, flags)
			})
		}),
	}
	cmd.Flags().BoolVar(&replace, "replace", false, "place an intention that is already placed at the event's time instead")
	addResolverFlags(cmd, &rf)
	addActFlags(cmd, &act)
	return cmd
}
