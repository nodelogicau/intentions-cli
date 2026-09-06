package cli

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"

	imcp "github.com/nodelogicau/intentions-cli/internal/mcp"
	"github.com/nodelogicau/intentions-cli/internal/store"
)

func (a *app) serveCmd() *cobra.Command {
	var useMCP bool
	var author, harness, model string
	cmd := &cobra.Command{
		Use:   "serve --mcp [--workspace <dir>] [--author] [--harness] [--model]",
		Short: "Serve this workspace to an MCP client over stdio",
		Long: `Runs a Model Context Protocol server bound to one workspace, resolved as every
verb does (--workspace, $INTENTIONS_WORKSPACE, then intentions.yaml or a
.intentions pointer from the working directory). One tool per CLI operation;
results equal the CLI's --json output. Stdout carries only the protocol;
diagnostics go to stderr. A second workspace is a second server entry in your
client.`,
		Args: cobra.NoArgs,
		RunE: a.run(func(cmd *cobra.Command, args []string) error {
			if !useMCP {
				return usageErr("serve requires --mcp (the only transport in this version)")
			}
			ws, res, err := store.Discover(imcp.Clean(a.workspace))
			if err != nil {
				return err
			}
			srv := imcp.New(imcp.Options{Workspace: ws, Version: version, Author: author, Harness: harness, Model: model})
			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer stop()
			_, _ = os.Stderr.WriteString("intentions " + version + " serving " + res.Root + " over stdio (found by " + res.FoundBy + ")\n")
			return srv.Run(ctx, &sdk.StdioTransport{})
		}),
	}
	cmd.Flags().BoolVar(&useMCP, "mcp", false, "speak MCP over stdio (required)")
	cmd.Flags().StringVar(&author, "author", "", "default source.author (else $INTENTIONS_AUTHOR, then intentions.yaml)")
	cmd.Flags().StringVar(&harness, "harness", "", "default source.harness (else $INTENTIONS_HARNESS, intentions.yaml, then the client's name)")
	cmd.Flags().StringVar(&model, "model", "", "default source.model")
	return cmd
}
