## Context

particulars-cli's MCP server is a thin front-end: an SDK server bound at startup to one workspace, tool handlers that call the same store and query APIs the CLI does, results that are the CLI's `--json` maps, errors mapped to the CLI's codes, the skill body as `initialize` instructions, and a Claude Desktop bundle assembled from the cross-compiled binaries. This change ports that shape. The differences are the tool surface, which the Intentions spec does not define, and the write operations, which here run through a write policy and the consistency check.

## Goals / Non-Goals

**Goals:**

- Same files whichever front-end wrote them. A tool and its verb share every check and write.
- The harness boundary holds over MCP: `intention_firm` and `select --policy` need the same policy a CLI harness needs, and a client that identifies itself is a harness.
- Results a client can act on: candidate lists, flags with counterpart versions, refusals naming the rule.
- One bundle install for Claude Desktop, no other setup.

**Non-Goals:**

- Refactoring the CLI into a shared operations layer. The handlers are written against the core packages, as particulars' are; the duplication is the flag-to-value step, which differs by nature between a shell and a typed schema.
- Transports other than stdio.
- Any operation the CLI does not have.

## Decisions

### D1. Tool names follow the verbs (feedback 23)

The spec names no tools. Names are `<noun>_<verb>` for object verbs (`intention_add`, `availability_renew`) and the bare verb where the CLI has one (`generate`, `resolve`, `select`, `check`, `acknowledge`, `bounds`, `validate`), plus `workspace_status`. Every description says whether the tool writes. Proposed upstream as the reference tool set, so a second implementation can match it.

### D2. Inputs are typed; the schema teaches the register

Each tool's input struct carries `jsonschema` descriptions that say what the CLI's help says, and more where a typed field can enforce it: `stability` is an enum; `serves` is a list of `{id, role}`; `window` is `{calendar, clock, relative}`; `duration` is a string or `{nominal, min, max}` (typed `any`, as particulars types its document union, because a bare string must stay valid). Descriptions of `intention_add`, `intention_firm` and `select` carry the register: draft tentative, never invent a slot, a policy is the only way a harness firms or selects alone.

### D3. Attribution

Per call: `source{author, harness, model}` on the input; then `--author/--harness/--model` on `serve`; then `INTENTIONS_*`; then `intentions.yaml` defaults; then the client's `clientInfo.name` as `harness`. `author` is required on intentions and availability as it is on the CLI. `--now` has no MCP form; `INTENTIONS_NOW` still applies to the server process for tests, and `check`, `resolve`, `select`, `generate` take an optional `now`.

### D4. Results equal `--json`

Handlers build the same maps the CLI builds (`objectResult`, `resultMap`, the selection map) by calling shared helpers moved from `internal/cli` into a small `internal/render` package that both import. This is the one refactor: three functions and no behaviour change, so the CLI tests keep guarding them.

### D5. Instructions, prompt, resource

`initialize` instructions: a header naming the workspace root and the person it is for, a line saying tool names are this implementation's and results equal the CLI's `--json`, the skill body, then `intentions.md` under a heading naming it, delivered to at least 16 KiB cut on a rune boundary with a truncation note. Prompt `intentions-discipline` returns the same text. `intentions.md` is a `file://` resource read at request time.

### D6. Write serialisation

One mutex around every mutating tool: load, check, write, index. Read-only tools load without the lock.

### D7. The bundle

`bundle/manifest.json.tmpl` ports particulars' with the intentions tool list, `serve --mcp --workspace ${user_config.workspace} --author ${user_config.author}`, and the same user configuration form. `scripts/build-bundle.sh` ports unchanged apart from names. GoReleaser builds a universal darwin binary and runs the script as its post hook; the `.mcpb` is an extra release file and lands in `checksums.txt`. CI's bundle job on macOS asserts the binary is universal.

### D8. Unsubstituted placeholders

Claude Desktop passes `${user_config.author}` literally when the field is blank. Any attribution value or workspace path matching `^\$\{[^}]*\}$` is treated as absent, as particulars does since its 0.5.2.

## Risks / Trade-offs

- [Context cost of nineteen tool schemas in every Desktop session] → Documented in `docs/mcp.md` with the same guidance as particulars: the skill and CLI where there is a shell, the server where there is not.
- [Handler logic drifts from the CLI's] → Both call the same checks and `WriteObject`; the MCP tests run the same scenarios the CLI tests run and compare file bytes for one object written both ways.
- [A blank author from the bundle recorded as `${user_config.author}`] → D8.
- [GoReleaser universal binary hook ordering] → The script exits 0 until both binaries exist, as particulars' does.

## Migration Plan

Additive. A new verb, a new release asset. Nothing existing changes.

## Open Questions

- Whether `bounds` earns a tool at all; kept because a Desktop client has no other way to see what a window means.

## Spec Feedback (to raise upstream)

23. **A reference tool set.** The specification names no MCP tools; propose the nineteen names and input shapes this implementation uses, so implementations agree the way DKF's do.
