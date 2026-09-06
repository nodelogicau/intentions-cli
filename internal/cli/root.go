// Package cli wires the core to a cobra command tree. It owns the
// agent-facing contract: no prompts, --json everywhere, documented exit codes.
package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/nodelogicau/intentions-cli/internal/apperr"
	"github.com/nodelogicau/intentions-cli/internal/model"
	"github.com/nodelogicau/intentions-cli/internal/store"
)

// version is set at build time via -ldflags "-X .../internal/cli.version=...".
var version = "dev"

// ExitError carries an exit code and a machine-readable error code.
type ExitError = apperr.Error

var (
	usageErr    = apperr.Usage
	notFoundErr = apperr.NotFound
	refusedErr  = apperr.Refused
	invalidErr  = apperr.Invalid
	runtimeErr  = apperr.Runtime
	classify    = apperr.Classify
)

// app holds per-invocation state shared by all commands.
type app struct {
	jsonOut         bool
	workspace       string
	stdin           io.Reader
	stdout          io.Writer
	stderr          io.Writer
	stdinIsTerminal func() bool
	// warnings collects advisory notes raised while a verb runs. emit carries
	// them in the JSON result and prints them to stderr in text mode.
	warnings []string
}

// Execute runs the CLI with the given arguments and streams, returning the
// process exit code. It never calls os.Exit, so tests can drive it in-process.
func Execute(args []string, stdin io.Reader, stdout, stderr io.Writer, stdinIsTerminal func() bool) int {
	a := &app{stdin: stdin, stdout: stdout, stderr: stderr, stdinIsTerminal: stdinIsTerminal}
	root := a.rootCmd()
	root.SetArgs(args)
	root.SetIn(stdin)
	root.SetOut(stdout)
	root.SetErr(stderr)

	err := root.Execute()
	if err == nil {
		return apperr.ExitOK
	}
	ee := classify(err)
	if !isTyped(err) {
		ee = &ExitError{Code: apperr.ExitUsage, ErrCode: "usage", Err: err}
	}
	if !errors.Is(ee.Err, errQuiet) {
		a.printError(ee)
	}
	return ee.Code
}

func isTyped(err error) bool {
	return apperr.IsDomain(err) || errors.Is(err, errRuntime) || errors.Is(err, errQuiet) || errors.Is(err, apperr.ErrNoWorkspace) || errors.Is(err, apperr.ErrNotFound)
}

// errRuntime wraps arbitrary core errors so Execute can tell them apart from
// cobra's own usage errors.
var errRuntime = errors.New("runtime")

// errQuiet marks a check verdict whose report has already been emitted: the
// exit code carries the verdict and no error envelope follows it.
var errQuiet = errors.New("check failed")

func checkFailed() error {
	return &ExitError{Code: apperr.ExitCheckFailed, ErrCode: "check_failed", Err: errQuiet}
}

// run wraps a command body so that every error leaving it is typed.
func (a *app) run(fn func(cmd *cobra.Command, args []string) error) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		err := fn(cmd, args)
		if err == nil || isTyped(err) {
			return err
		}
		var refusal *model.Refusal
		if errors.As(err, &refusal) {
			return refusedErr("%s", refusal.Message)
		}
		var probs model.Problems
		if errors.As(err, &probs) {
			return invalidErr("%s", probs.Error())
		}
		return fmt.Errorf("%w: %v", errRuntime, err)
	}
}

func (a *app) printError(ee *ExitError) {
	msg := strings.TrimPrefix(ee.Err.Error(), errRuntime.Error()+": ")
	if a.jsonOut {
		enc := json.NewEncoder(a.stderr)
		_ = enc.Encode(map[string]any{"error": map[string]any{"code": ee.ErrCode, "message": msg}})
		return
	}
	fmt.Fprintf(a.stderr, "error: %s\n", msg)
}

// emit writes the command result: JSON when --json, otherwise via text.
func (a *app) emit(v map[string]any, text func(w io.Writer)) error {
	if len(a.warnings) > 0 {
		v["warnings"] = a.warnings
	}
	if a.jsonOut {
		enc := json.NewEncoder(a.stdout)
		enc.SetIndent("", "  ")
		enc.SetEscapeHTML(false)
		return enc.Encode(v)
	}
	text(a.stdout)
	for _, w := range a.warnings {
		fmt.Fprintf(a.stderr, "warning: %s\n", w)
	}
	return nil
}

func (a *app) openWorkspace() (*store.Workspace, error) {
	ws, _, err := store.Discover(a.workspace)
	return ws, err
}

func (a *app) rootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "intentions",
		Short: "Write and check Intentions Format workspaces: what a person means to do with their time",
		Long: `intentions reads and writes Intentions Format workspaces: directories of
YAML files holding intentions (a duration and a window, not a slot),
availability (the supply intentions draw on), and commitments (where another
party is involved).

It is built to be driven by an LLM harness and reviewed by people through
git: every verb is non-interactive, supports --json, and uses these exit codes:

  0 success   1 runtime error   2 usage error or refused write
  3 not found 4 check failed    5 no workspace

Attribution: --author/--harness/--model flags, then INTENTIONS_AUTHOR,
INTENTIONS_HARNESS, INTENTIONS_MODEL, then defaults.source.author in
intentions.yaml. A harness may draft; it may not make an intention firm
without a policy the person holds.`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.PersistentFlags().BoolVar(&a.jsonOut, "json", false, "emit a single JSON object on stdout (errors as JSON on stderr)")
	root.PersistentFlags().StringVar(&a.workspace, "workspace", "", "workspace directory (default: $INTENTIONS_WORKSPACE, else nearest ancestor with intentions.yaml or a .intentions pointer)")
	root.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error { return usageErr("%v", err) })
	root.CompletionOptions.DisableDefaultCmd = true

	root.AddCommand(
		a.initCmd(),
		a.workspaceCmd(),
		a.intentionCmd(),
		a.availabilityCmd(),
		a.generateCmd(),
		a.resolveCmd(),
		a.selectCmd(),
		a.checkCmd(),
		a.unresolvedCmd(),
		a.acknowledgeCmd(),
		a.showCmd(),
		a.versionOfCmd(),
		a.boundsCmd(),
		a.validateCmd(),
		a.indexCmd(),
		a.skillCmd(),
		a.serveCmd(),
		a.versionCmd(),
	)
	return root
}

func (a *app) versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the binary version and the format version it implements",
		Args:  cobra.NoArgs,
		RunE: a.run(func(cmd *cobra.Command, args []string) error {
			return a.emit(map[string]any{"version": version, "format": model.Format}, func(w io.Writer) {
				fmt.Fprintf(w, "intentions %s (%s)\n", version, model.Format)
			})
		}),
	}
}
