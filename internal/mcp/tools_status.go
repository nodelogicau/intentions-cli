package mcp

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/nodelogicau/intentions-cli/internal/consistency"
	"github.com/nodelogicau/intentions-cli/internal/query"
	"github.com/nodelogicau/intentions-cli/internal/resolve"
)

type emptyIn struct{}

func (s *Server) validateTool(ctx context.Context, req *sdk.CallToolRequest, _ emptyIn) (*sdk.CallToolResult, any, error) {
	g, err := s.ws.Load()
	if err != nil {
		return errResult(err), nil, nil
	}
	report := query.Validate(s.ws, g)
	out := map[string]any{"findings": report.Findings, "counts": report.Counts, "ok": !report.HasErrors()}
	return okResult(fmt.Sprintf("%d errors, %d warnings, %d info", report.Counts[query.SeverityError], report.Counts[query.SeverityWarning], report.Counts[query.SeverityInfo])), out, nil
}

func (s *Server) workspaceStatus(ctx context.Context, req *sdk.CallToolRequest, _ emptyIn) (*sdk.CallToolResult, any, error) {
	g, err := s.ws.Load()
	if err != nil {
		return errResult(err), nil, nil
	}
	report := query.Validate(s.ws, g)
	counts := map[string]int{}
	for _, id := range g.Order {
		counts[string(g.Objects[id].GetType())]++
	}
	at, _ := now("")
	flags, _ := consistency.Check(resolve.NewEnv(s.ws, g, at), nil)
	out := map[string]any{
		"root": s.ws.Root, "subject": s.ws.Config.Defaults.Subject, "author": s.ws.Config.Defaults.Source.Author, "timezone": s.ws.Config.Resolver.Timezone,
		"counts":   counts,
		"validate": map[string]any{"errors": report.Counts[query.SeverityError], "warnings": report.Counts[query.SeverityWarning], "unreadable": len(g.Unreadable)},
		"flags":    len(flags),
	}
	if gs := gitStatus(ctx, s.ws.Root); gs != nil {
		out["git"] = gs
	}
	summary := fmt.Sprintf("%s: %d intentions, %d availability, %d commitments, %d resolutions; validate %d errors/%d warnings; %d flags", s.ws.Root, counts["intention"], counts["availability"], counts["commitment"], counts["resolution"], report.Counts[query.SeverityError], report.Counts[query.SeverityWarning], len(flags))
	if gs, ok := out["git"].(map[string]any); ok {
		summary += fmt.Sprintf("; %d uncommitted file(s)", len(gs["uncommitted"].([]string)))
	}
	return okResult(summary), out, nil
}

// gitStatus returns {"checkout", "uncommitted"} when the workspace lies
// inside a git checkout and git is available; nil otherwise. Read-only.
func gitStatus(ctx context.Context, root string) map[string]any {
	if _, err := exec.LookPath("git"); err != nil {
		return nil
	}
	top, err := exec.CommandContext(ctx, "git", "-C", root, "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return nil
	}
	out, err := exec.CommandContext(ctx, "git", "-C", root, "status", "--porcelain", "--untracked-files=all", "--", ".").Output()
	if err != nil {
		return nil
	}
	files := []string{}
	for _, line := range strings.Split(strings.TrimRight(string(out), "\n"), "\n") {
		if len(line) < 4 {
			continue
		}
		p := strings.TrimSpace(line[3:])
		if i := strings.Index(p, " -> "); i >= 0 {
			p = p[i+4:]
		}
		files = append(files, filepath.ToSlash(p))
	}
	return map[string]any{"checkout": strings.TrimSpace(string(top)), "uncommitted": files}
}
