package cli

import (
	"fmt"
	"io"
	"time"

	"github.com/spf13/cobra"

	"github.com/nodelogicau/intentions-cli/internal/model"
	"github.com/nodelogicau/intentions-cli/internal/projection"
	"github.com/nodelogicau/intentions-cli/internal/temporal"
)

func (a *app) showCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <id>",
		Short: "Show any object by id, including commitments and resolutions written by other tools",
		Args:  cobra.ExactArgs(1),
		RunE: a.run(func(cmd *cobra.Command, args []string) error {
			ws, err := a.openWorkspace()
			if err != nil {
				return err
			}
			obj, probs, err := ws.ReadObject(args[0])
			if err != nil {
				return err
			}
			out, err := objectResult(obj)
			if err != nil {
				return err
			}
			if len(probs) > 0 {
				out["problems"] = probs
			}
			return a.emit(out, func(w io.Writer) {
				data, _ := model.Encode(obj)
				fmt.Fprint(w, string(data))
				for _, p := range probs {
					fmt.Fprintf(w, "# problem: %s\n", p.Error())
				}
			})
		}),
	}
}

func (a *app) versionOfCmd() *cobra.Command {
	var showProjection bool
	cmd := &cobra.Command{
		Use:   "version-of <id>",
		Short: "Print an object's computed version, and with --projection the canonical JSON it hashes",
		Args:  cobra.ExactArgs(1),
		RunE: a.run(func(cmd *cobra.Command, args []string) error {
			ws, err := a.openWorkspace()
			if err != nil {
				return err
			}
			obj, _, err := ws.ReadObject(args[0])
			if err != nil {
				return err
			}
			canon, err := projection.Canonical(obj)
			if err != nil {
				return err
			}
			v, _ := projection.Version(obj)
			out := map[string]any{"id": obj.GetID(), "version": v}
			if cached := obj.CachedVersion(); cached != "" && cached != v {
				out["version_cached"] = cached
			}
			if showProjection {
				out["projection"] = projection.Project(obj)
				out["canonical"] = string(canon)
			}
			return a.emit(out, func(w io.Writer) {
				if showProjection {
					fmt.Fprintln(w, string(canon))
				}
				fmt.Fprintln(w, v)
			})
		}),
	}
	cmd.Flags().BoolVar(&showProjection, "projection", false, "also print the canonical JSON projection")
	return cmd
}

func (a *app) boundsCmd() *cobra.Command {
	var calendar, clock, timezone, weekStart, hemisphere, now, horizonStart, horizonEnd string
	cmd := &cobra.Command{
		Use:   "bounds [<id>]",
		Short: "Compute the clock-time bounds of an object's window, or of an ad hoc --calendar/--clock window; writes nothing",
		Args:  cobra.MaximumNArgs(1),
		RunE: a.run(func(cmd *cobra.Command, args []string) error {
			ws, err := a.openWorkspace()
			if err != nil {
				return err
			}
			t, err := resolveNow(actFlags{now: now})
			if err != nil {
				return err
			}
			ctx := ws.Config.Context(t)
			if timezone != "" {
				loc, err := time.LoadLocation(timezone)
				if err != nil {
					return usageErr("--timezone %q is not a known IANA zone", timezone)
				}
				ctx.Location = loc
			}
			if weekStart != "" {
				d, err := temporal.ParseWeekday(weekStart)
				if err != nil {
					return usageErr("--week-start: %v", err)
				}
				ctx.WeekStart = d
			}
			if hemisphere != "" {
				h, err := temporal.ParseHemisphere(hemisphere)
				if err != nil {
					return usageErr("--hemisphere: %v", err)
				}
				ctx.Hemisphere = h
			}
			if horizonStart != "" || horizonEnd != "" {
				h := &temporal.Interval{OpenStart: horizonStart == "", OpenEnd: horizonEnd == ""}
				if horizonStart != "" {
					if h.Start, err = temporal.ParseTimestamp(horizonStart); err != nil {
						return usageErr("--horizon-start: %v", err)
					}
				}
				if horizonEnd != "" {
					if h.End, err = temporal.ParseTimestamp(horizonEnd); err != nil {
						return usageErr("--horizon-end: %v", err)
					}
				}
				ctx.Horizon = h
			}
			var win temporal.Window
			var id string
			if len(args) == 1 {
				obj, _, err := ws.ReadObject(args[0])
				if err != nil {
					return err
				}
				id = obj.GetID()
				switch o := obj.(type) {
				case *model.Intention:
					if o.Window != nil {
						win = *o.Window
					}
				case *model.Availability:
					if o.Window != nil {
						win = *o.Window
					}
				default:
					return usageErr("%s is a %s and has no window", id, obj.GetType())
				}
			}
			if calendar != "" {
				c, err := temporal.ParseCalendarOrDeixis(calendar, ctx)
				if err != nil {
					return invalidErr("--calendar: %v", err)
				}
				win.Calendar = &c
			}
			if clock != "" {
				c, err := temporal.ParseClock(clock)
				if err != nil {
					return invalidErr("--clock: %v", err)
				}
				win.Clock = &c
			}
			if win.IsZero() {
				return usageErr("nothing to bound: pass an id with a window, or --calendar and/or --clock")
			}
			ivs, err := temporal.Bounds(win, ctx)
			if err != nil {
				return usageErr("%v", err)
			}
			list := make([]map[string]any, 0, len(ivs))
			for _, iv := range ivs {
				m := map[string]any{}
				if iv.OpenStart {
					m["start"] = nil
				} else {
					m["start"] = iv.Start.Format(time.RFC3339)
				}
				if iv.OpenEnd {
					m["end"] = nil
				} else {
					m["end"] = iv.End.Format(time.RFC3339)
				}
				list = append(list, m)
			}
			out := map[string]any{"window": win.String(), "timezone": ctx.Location.String(), "week_start": temporal.WeekdayName(ctx.WeekStart), "hemisphere": string(ctx.Hemisphere), "intervals": list, "count": len(list)}
			if id != "" {
				out["id"] = id
			}
			if win.Relative != nil {
				a.warnings = append(a.warnings, "the relational anchor contributes nothing here: it needs its target's placement, which a resolver supplies")
			}
			return a.emit(out, func(w io.Writer) {
				fmt.Fprintf(w, "%s in %s (week starts %s)\n", win, ctx.Location, temporal.WeekdayName(ctx.WeekStart))
				for _, m := range list {
					s, e := "..", ".."
					if m["start"] != nil {
						s = m["start"].(string)
					}
					if m["end"] != nil {
						e = m["end"].(string)
					}
					fmt.Fprintf(w, "  %s / %s\n", s, e)
				}
			})
		}),
	}
	cmd.Flags().StringVar(&calendar, "calendar", "", "calendar anchor to bound (EDTF or deictic)")
	cmd.Flags().StringVar(&clock, "clock", "", "clock anchor to apply")
	cmd.Flags().StringVar(&timezone, "timezone", "", "override resolver.timezone")
	cmd.Flags().StringVar(&weekStart, "week-start", "", "override resolver.week_start")
	cmd.Flags().StringVar(&hemisphere, "hemisphere", "", "override resolver.hemisphere")
	cmd.Flags().StringVar(&now, "now", "", "the current time as RFC 3339, for deictic terms")
	cmd.Flags().StringVar(&horizonStart, "horizon-start", "", "clamp an open lower bound to this RFC 3339 instant")
	cmd.Flags().StringVar(&horizonEnd, "horizon-end", "", "clamp an open upper bound to this RFC 3339 instant")
	return cmd
}
