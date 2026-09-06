package cli

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/nodelogicau/intentions-cli/internal/store"
	"github.com/nodelogicau/intentions-cli/internal/temporal"
)

func (a *app) initCmd() *cobra.Command {
	var author, subject, timezone, hemisphere, availHorizon, genHorizon string
	cmd := &cobra.Command{
		Use:   "init [dir]",
		Short: "Create a workspace: intentions.yaml, the type directories, index.yaml, intentions.md",
		Args:  cobra.MaximumNArgs(1),
		RunE: a.run(func(cmd *cobra.Command, args []string) error {
			dir := "."
			if len(args) == 1 {
				dir = args[0]
			}
			cfg := store.NewConfig()
			cfg.Defaults.Source.Author = firstNonEmpty(author, os.Getenv(EnvAuthor))
			if cfg.Defaults.Source.Author == "" {
				return usageErr("--author is required (or set %s)", EnvAuthor)
			}
			cfg.Defaults.Subject = subject
			cfg.Resolver.Timezone = firstNonEmpty(timezone, localZoneName())
			if _, err := time.LoadLocation(cfg.Resolver.Timezone); err != nil {
				return usageErr("--timezone %q is not a known IANA zone", cfg.Resolver.Timezone)
			}
			if hemisphere != "" {
				h, err := temporal.ParseHemisphere(hemisphere)
				if err != nil {
					return usageErr("--hemisphere: %v", err)
				}
				cfg.Resolver.Hemisphere = string(h)
			}
			for name, v := range map[string]string{"--availability-horizon": availHorizon, "--generation-horizon": genHorizon} {
				if _, err := temporal.ParseDuration(v); err != nil {
					return usageErr("%s: %v", name, err)
				}
			}
			cfg.Availability.DefaultHorizon = availHorizon
			cfg.Generation.Horizon = genHorizon
			ws, created, err := store.Init(dir, cfg)
			if err != nil {
				return err
			}
			return a.emit(map[string]any{
				"workspace": map[string]any{"root": ws.Root, "format": ws.Config.Format},
				"config":    ws.Config,
				"created":   created,
			}, func(w io.Writer) {
				fmt.Fprintf(w, "Initialised Intentions workspace at %s\n", ws.Root)
				for _, c := range created {
					fmt.Fprintf(w, "  %s\n", c)
				}
			})
		}),
	}
	cmd.Flags().StringVar(&author, "author", "", "defaults.source.author: the person the workspace is for (URI or name)")
	cmd.Flags().StringVar(&subject, "subject", "", "defaults.subject: applied to intentions written without one (omit for an organisation workspace)")
	cmd.Flags().StringVar(&timezone, "timezone", "", "resolver.timezone (default: the local zone, else UTC)")
	cmd.Flags().StringVar(&hemisphere, "hemisphere", "", "resolver.hemisphere for season codes: north or south (default north)")
	cmd.Flags().StringVar(&availHorizon, "availability-horizon", "P13W", "availability.default_horizon")
	cmd.Flags().StringVar(&genHorizon, "generation-horizon", "P4W", "generation.horizon")
	return cmd
}

// localZoneName returns the process's IANA zone name when the platform
// exposes one, else UTC.
func localZoneName() string {
	if tz := os.Getenv("TZ"); tz != "" {
		if _, err := time.LoadLocation(tz); err == nil {
			return tz
		}
	}
	name := time.Local.String()
	if name == "" || name == "Local" {
		return "UTC"
	}
	if _, err := time.LoadLocation(name); err != nil {
		return "UTC"
	}
	return name
}

func (a *app) workspaceCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "workspace",
		Short: "Print the resolved workspace root, how it was found, and its configuration",
		Args:  cobra.NoArgs,
		RunE: a.run(func(cmd *cobra.Command, args []string) error {
			ws, res, err := store.Discover(a.workspace)
			if err != nil {
				return err
			}
			out := map[string]any{"root": ws.Root, "found_by": res.FoundBy, "config": ws.Config}
			if res.Pointer != "" {
				out["pointer"] = res.Pointer
			}
			return a.emit(out, func(w io.Writer) {
				fmt.Fprintf(w, "%s (found by %s)\n", ws.Root, res.FoundBy)
				fmt.Fprintf(w, "  format %s, hash %s\n", ws.Config.Format, ws.Config.Hash)
				fmt.Fprintf(w, "  resolver %s, %s hemisphere; weeks are ISO weeks\n", ws.Config.Resolver.Timezone, firstNonEmpty(ws.Config.Resolver.Hemisphere, "north"))
				fmt.Fprintf(w, "  author %s\n", ws.Config.Defaults.Source.Author)
				if ws.Config.Defaults.Subject != "" {
					fmt.Fprintf(w, "  subject %s\n", ws.Config.Defaults.Subject)
				}
			})
		}),
	}
}
