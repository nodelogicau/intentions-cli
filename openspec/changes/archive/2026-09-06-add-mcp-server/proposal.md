## Why

Every verb the CLI offers is reachable only through a shell. Claude Desktop, mobile clients and hosts that will not grant a shell reach tools through the Model Context Protocol, and particulars-cli settled how a sibling ships that: one stdio server bound to one workspace, tools whose results are exactly the CLI's `--json`, the skill delivered as `initialize` instructions, a Claude Desktop extension bundle on every release. This change is the last of the sibling pieces. Unlike DKF, the Intentions specification names no tool set, so this implementation's tool names are its own and go upstream as a feedback item.

## What Changes

- **`intentions serve --mcp`** runs an MCP server over stdio bound to the workspace resolved as every verb resolves it. Stdout carries only JSON-RPC; without a workspace it exits 5.
- **Nineteen tools**, one per CLI operation, with typed inputs and structured results equal to the corresponding verb's `--json`: `workspace_status`, `intention_add`, `intention_edit`, `intention_firm`, `intention_retire`, `intention_show`, `intention_list`, `availability_add`, `availability_renew`, `availability_supersede`, `availability_retire`, `availability_list`, `generate`, `resolve`, `select`, `check`, `acknowledge`, `bounds`, `validate`. Query tools are annotated read-only; writers are non-destructive; nothing deletes.
- **Attribution from the handshake.** `source.harness` defaults to the client's name from `initialize` when nothing else supplies it; `author` from the call, server flags, environment, then `intentions.yaml`. The harness boundary holds over MCP exactly as over the CLI: a harness cannot firm without a policy, cannot select without one.
- **The skill travels as instructions.** `initialize` carries a header naming the bound workspace, the skill body, and the workspace's `intentions.md` (at least the first 16 KiB, cut on a character boundary). The same text is the prompt `intentions-discipline`; `intentions.md` is also an MCP resource read on demand.
- **Errors are tool results** with `isError`, a `<code>: <message>` line, and `{"error": {"code", "message"}}` using the CLI's codes. Mutating tools are serialised by a server-wide lock.
- **A Claude Desktop extension bundle** `intentions-<version>.mcpb` on every release: a macOS universal binary and Windows x64, a manifest asking for the workspace folder and the person's name.
- **Docs**: `docs/mcp.md` with the tool table, client configurations, and the skill-versus-server guidance; README section; CHANGELOG 0.5.0; SPEC-FEEDBACK item 23 proposing a tool set.

## Capabilities

### New Capabilities
- `mcp-server`: `serve --mcp`, the tool surface and its results, attribution from the handshake, instructions and prompt, the conventions resource, error mapping, write serialisation.
- `mcp-bundle`: the `.mcpb` extension bundle, its manifest and user configuration, release attachment, and the documented client configurations.

### Modified Capabilities
<!-- none -->

## Impact

- New package `internal/mcp` on `github.com/modelcontextprotocol/go-sdk`; new verb in `internal/cli`; `bundle/` and `scripts/build-bundle.sh`; GoReleaser gains a universal darwin binary and the bundle as an extra release file; CI gains a bundle job on macOS.
- Tool handlers reuse the model, resolve and consistency packages directly; the CLI is not refactored. Both front-ends call the same checks and writes, so the files they produce are identical.
- Released as v0.5.0.
