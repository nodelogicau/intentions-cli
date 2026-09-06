## 1. Shared rendering

- [x] 1.1 Move `objectResult`, the resolve result map and the selection map from `internal/cli` into `internal/render`, and point the CLI at them; no behaviour change

## 2. Server

- [x] 2.1 `internal/mcp/server.go`: `Options`, `New`, `Run`, instructions (header, skill body, `intentions.md` to 16 KiB on a rune boundary), the `intentions-discipline` prompt, the conventions resource, attribution resolution with the client name fallback and placeholder handling, `errResult`/`okResult`, annotations, the write mutex
- [x] 2.2 `internal/mcp/tools_intention.go`: `intention_add`, `intention_edit`, `intention_firm`, `intention_retire`, `intention_show`, `intention_list` with typed inputs, the same checks and writes as the verbs, flags in results
- [x] 2.3 `internal/mcp/tools_availability.go`: `availability_add`, `availability_renew`, `availability_supersede`, `availability_retire`, `availability_list`
- [x] 2.4 `internal/mcp/tools_resolve.go`: `generate`, `resolve`, `select`, `check`, `acknowledge`, `bounds`; `tools_status.go`: `validate`, `workspace_status` with read-only git status
- [x] 2.5 `internal/cli/cmd_serve.go`: `serve --mcp` with attribution flags; register on the root command
- [x] 2.6 Tests over in-memory transports for every `mcp-server` scenario, including the same-file comparison with the CLI, the harness boundary, handshake attribution, truncation, the resource, and parallel adds; a stdio smoke test through the built binary

## 3. Bundle and release

- [x] 3.1 `bundle/manifest.json.tmpl` and `bundle/icon.png`; `scripts/build-bundle.sh` ported; `make bundle`
- [x] 3.2 GoReleaser: universal darwin binary with the bundle script as post hook, `.mcpb` as an extra release file in the checksums
- [x] 3.3 CI bundle job on macOS asserting a universal binary and a complete manifest

## 4. Docs, feedback and release

- [x] 4.1 `docs/mcp.md`: skill-versus-server guidance, client configurations, the tool table, attribution, reviewing what a Desktop conversation wrote
- [x] 4.2 README section and verb row; skill mentions `serve --mcp`; CHANGELOG 0.5.0; SPEC-FEEDBACK item 23 raised as an issue and linked
- [x] 4.3 Tag `v0.5.0`, verify the release including the `.mcpb`
