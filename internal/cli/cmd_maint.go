package cli

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/nodelogicau/intentions-cli/internal/apperr"
	"github.com/nodelogicau/intentions-cli/internal/query"
	"github.com/nodelogicau/intentions-cli/internal/store"
)

func (a *app) validateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Check the whole workspace; exits 4 on any error",
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
			report := query.Validate(ws, g)
			out := map[string]any{"findings": report.Findings, "counts": report.Counts, "ok": !report.HasErrors()}
			if err := a.emit(out, func(w io.Writer) {
				for _, f := range report.Findings {
					where := f.ID
					if where == "" {
						where = f.Path
					}
					fmt.Fprintf(w, "%-7s %-20s %s: %s\n", f.Severity, f.Code, where, f.Message)
				}
				fmt.Fprintf(w, "%d errors, %d warnings, %d info\n", report.Counts[query.SeverityError], report.Counts[query.SeverityWarning], report.Counts[query.SeverityInfo])
			}); err != nil {
				return err
			}
			if report.HasErrors() {
				return checkFailed()
			}
			return nil
		}),
	}
}

func (a *app) indexCmd() *cobra.Command {
	var check bool
	cmd := &cobra.Command{
		Use:   "index",
		Short: "Rebuild index.yaml from the object files, or with --check verify it and exit 4 on drift",
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
			rebuilt := store.Rebuild(g)
			if check {
				committed, err := ws.ReadIndex()
				if err != nil {
					return &ExitError{Code: apperr.ExitCheckFailed, ErrCode: "check_failed", Err: fmt.Errorf("%v; run `intentions index` to regenerate", err)}
				}
				d := store.Compare(committed, rebuilt)
				out := map[string]any{"ok": d.Empty(), "missing": d.Missing, "extra": d.Extra, "changed": d.Changed, "entries": len(rebuilt.Entries)}
				if err := a.emit(out, func(w io.Writer) {
					if d.Empty() {
						fmt.Fprintf(w, "index.yaml matches %d object files\n", len(rebuilt.Entries))
						return
					}
					for _, id := range d.Missing {
						fmt.Fprintf(w, "missing  %s\n", id)
					}
					for _, id := range d.Extra {
						fmt.Fprintf(w, "extra    %s\n", id)
					}
					for _, id := range d.Changed {
						fmt.Fprintf(w, "changed  %s\n", id)
					}
				}); err != nil {
					return err
				}
				if !d.Empty() {
					return checkFailed()
				}
				return nil
			}
			if err := ws.WriteIndex(rebuilt); err != nil {
				return err
			}
			out := map[string]any{"entries": len(rebuilt.Entries), "unreadable": g.Unreadable}
			if len(g.Unreadable) > 0 {
				for _, u := range g.Unreadable {
					a.warnings = append(a.warnings, fmt.Sprintf("%s could not be read and is not indexed: %s", u.Path, u.Err))
				}
			}
			return a.emit(out, func(w io.Writer) {
				fmt.Fprintf(w, "Wrote index.yaml with %d entries\n", len(rebuilt.Entries))
			})
		}),
	}
	cmd.Flags().BoolVar(&check, "check", false, "verify the committed index against a rebuild instead of writing")
	return cmd
}
