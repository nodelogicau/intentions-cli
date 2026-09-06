package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/nodelogicau/intentions-cli/internal/store"
	"github.com/nodelogicau/intentions-cli/internal/temporal"
)

func (a *app) initCmd() *cobra.Command {
	var author, subject, timezone, hemisphere, availHorizon, genHorizon string
	var pointer bool
	cmd := &cobra.Command{
		Use:   "init [dir] [--pointer]",
		Short: "Create a workspace: intentions.yaml, the type directories, index.yaml, intentions.md",
		Args:  cobra.MaximumNArgs(1),
		RunE: a.run(func(cmd *cobra.Command, args []string) error {
			dir := "."
			if len(args) == 1 {
				dir = args[0]
			}
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			absDir, err := filepath.Abs(dir)
			if err != nil {
				return err
			}
			if pointer && absDir == cwd {
				return usageErr("--pointer needs a workspace directory other than the current directory, e.g. `intentions init ./planning --pointer`")
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
			out := map[string]any{
				"workspace": map[string]any{"root": ws.Root, "format": ws.Config.Format},
				"config":    ws.Config,
				"created":   created,
			}
			var pointerPath string
			if pointer {
				rel, err := filepath.Rel(cwd, ws.Root)
				if err != nil {
					return err
				}
				if pointerPath, err = store.WritePointer(cwd, filepath.ToSlash(rel)); err != nil {
					return err
				}
				out["pointer"] = pointerPath
			}
			return a.emit(out, func(w io.Writer) {
				fmt.Fprintf(w, "Initialised Intentions workspace at %s\n", ws.Root)
				for _, c := range created {
					fmt.Fprintf(w, "  %s\n", c)
				}
				if pointerPath != "" {
					fmt.Fprintf(w, "Wrote %s pointing at the workspace\n", pointerPath)
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
	cmd.Flags().BoolVar(&pointer, "pointer", false, "also write ./"+store.PointerFile+" pointing at dir")
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
	cmd := &cobra.Command{
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
	cmd.AddCommand(a.workspacePointerCmd())
	return cmd
}

// workspacePointerCmd writes the pointer that `init --pointer` writes at
// creation time, for a workspace that already exists.
func (a *app) workspacePointerCmd() *cobra.Command {
	var at string
	var force bool
	cmd := &cobra.Command{
		Use:   "pointer [workspace-dir]",
		Short: "Write a " + store.PointerFile + " pointer so this directory resolves to the workspace",
		Long: `Writes <dir>/` + store.PointerFile + ` naming the workspace, so ` + "`intentions`" + ` run anywhere at
or below <dir> finds it without --workspace or $INTENTIONS_WORKSPACE. This is
what ` + "`init --pointer`" + ` writes when the workspace is created; use this verb when
the workspace already exists.

With no argument the pointer names the workspace that would be used now
(--workspace, then $INTENTIONS_WORKSPACE, then discovery). The path is written
relative when the workspace lies inside <dir>, so the file survives being cloned
elsewhere, and absolute when it does not, in which case it is machine-specific
and should not be committed.`,
		Args: cobra.MaximumNArgs(1),
		RunE: a.run(func(cmd *cobra.Command, args []string) error {
			dir := at
			if dir == "" {
				cwd, err := os.Getwd()
				if err != nil {
					return err
				}
				dir = cwd
			}
			dir, err := filepath.Abs(dir)
			if err != nil {
				return err
			}
			if fi, err := os.Stat(dir); err != nil || !fi.IsDir() {
				return usageErr("%s is not a directory", dir)
			}
			var ws *store.Workspace
			if len(args) == 1 {
				ws, err = store.Open(args[0])
			} else {
				ws, _, err = store.Discover(a.workspace)
			}
			if err != nil {
				return err
			}
			if ws.Root == dir {
				return usageErr("%s is the workspace itself; its %s is already found from here", dir, store.ConfigFile)
			}
			if _, err := os.Stat(filepath.Join(dir, store.ConfigFile)); err == nil {
				return usageErr("%s already holds a %s, which wins over a pointer at the same level", dir, store.ConfigFile)
			}
			target, relative := ws.Root, false
			if rel, err := filepath.Rel(dir, ws.Root); err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
				target, relative = filepath.ToSlash(rel), true
			}
			path := filepath.Join(dir, store.PointerFile)
			if force {
				if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
					return err
				}
			}
			path, err = store.WritePointer(dir, target)
			if err != nil {
				return runtimeErr(fmt.Errorf("%v; pass --force to replace it", err))
			}
			return a.emit(map[string]any{"pointer": path, "root": ws.Root, "target": target, "relative": relative}, func(w io.Writer) {
				fmt.Fprintf(w, "Wrote %s -> %s\n", path, target)
				if !relative {
					fmt.Fprintf(w, "  absolute: the workspace is outside %s, so this pointer is machine-specific; do not commit it\n", dir)
				}
			})
		}),
	}
	cmd.Flags().StringVar(&at, "at", "", "directory to write "+store.PointerFile+" in (default: the current directory)")
	cmd.Flags().BoolVar(&force, "force", false, "replace an existing pointer that names a different workspace")
	return cmd
}
