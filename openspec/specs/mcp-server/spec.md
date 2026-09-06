# MCP Server

## Purpose

`serve --mcp`: one Model Context Protocol server over stdio bound to one workspace, the tool surface and its results, the harness boundary, attribution from the handshake, the instructions, prompt and conventions resource, error mapping to the CLI's codes, and write serialisation.

## Requirements

### Requirement: Stdio MCP server bound to one workspace
`intentions serve --mcp [--workspace <dir>] [--author] [--harness] [--model]` SHALL run a Model Context Protocol server over stdio, bound at startup to exactly one workspace resolved as every verb does. Without a workspace it SHALL exit with code 5 and a message on stderr. It SHALL write nothing to stdout except JSON-RPC. `serve` without `--mcp` SHALL be a usage error.

#### Scenario: Started in a project
- **WHEN** `intentions serve --mcp` is started in a directory whose ancestor has a `.intentions` pointer at a valid workspace
- **THEN** the server initialises and every tool operates on that workspace

#### Scenario: No workspace
- **WHEN** `intentions serve --mcp` is started with no resolvable workspace
- **THEN** the process exits with code 5 and stdout is empty

### Requirement: Tool surface
The server SHALL expose tools named `workspace_status`, `intention_add`, `intention_edit`, `intention_firm`, `intention_retire`, `intention_show`, `intention_list`, `availability_add`, `availability_renew`, `availability_supersede`, `availability_retire`, `availability_list`, `commitment_accept`, `commitment_decline`, `commitment_cancel`, `commitment_show`, `commitment_list`, `generate`, `resolve`, `select`, `unresolved`, `check`, `acknowledge`, `bounds`, and `validate`. Each tool's structured result SHALL equal the corresponding CLI verb's `--json` output. Every write SHALL apply the same rules and refusals as the verb. Query tools SHALL be annotated read-only and no tool destructive. Inputs that name a window SHALL take `{calendar, clock, relative}`; a duration SHALL be a string or `{nominal, min, max}`; `serves` a list of `{id, role}`.

#### Scenario: Add, resolve, select over MCP
- **WHEN** a client calls `availability_add`, then `intention_add{title, duration: "PT90M", window: {calendar: "2026-W38"}}`, then `resolve{id}`, then `select{id, candidate: 1}`
- **THEN** the intention file carries the first candidate as its placement and the result equals `select --json`

#### Scenario: Same file either way
- **WHEN** an intention is added over MCP and an identical one through the CLI with the same timestamp
- **THEN** the two files differ only in `id` and `version`

#### Scenario: Refusal over MCP
- **WHEN** a client calls `intention_add` with `activity: "Deep Work"`
- **THEN** the result is an error with code `refused` or `invalid` naming the rule and nothing is written

### Requirement: Unresolved tool
The server SHALL expose `unresolved{subject?, scope?, now?}` as a read-only tool whose structured result equals `intentions unresolved --json`.

#### Scenario: Unresolved over MCP
- **WHEN** a client adds an intention with a duration and a window, and one with a window only, then calls `unresolved{}`
- **THEN** the result has two entries, one `ready` and one `incomplete`, and `counts` reflects both

### Requirement: Commitment tools
The server SHALL expose `commitment_accept`, `commitment_decline`, `commitment_cancel`, `commitment_show` and `commitment_list`, whose structured results equal the corresponding CLI verbs' `--json` output. The answering tools SHALL be described as recording the person's own answer, never the model's inference.

#### Scenario: Accept over MCP
- **WHEN** a client calls `commitment_accept{id}` for a commitment listing the workspace's default subject
- **THEN** that party's status is `accepted`, the source records the client as harness, and the result equals `commitment accept --json`

#### Scenario: Cancel frees the intention
- **WHEN** a client calls `commitment_cancel{id, reason}` and then `unresolved{}`
- **THEN** the commitment is retired as cancelled and the intention it named is listed again

### Requirement: The harness boundary holds
`intention_firm` and `select` with `policy` SHALL require, for a source carrying a harness, a terminus of the subject carrying the matching condition that the intention satisfies, exactly as the CLI does; a client identifying itself in `initialize` is a harness unless the call names an author with no harness. `intention_firm` without `policy` from a harness SHALL be an error result.

#### Scenario: Client firms without a policy
- **WHEN** a client identifying as `claude-ai` calls `intention_firm{id}` with no `policy`
- **THEN** the result is an error with code `refused` and the file is unchanged

#### Scenario: Client firms under a policy
- **WHEN** the same client calls `intention_firm{id, policy: <terminus id>}` and the condition holds
- **THEN** the file carries `stability: firm`, `firmed_under: <terminus id>`, and `source.harness: claude-ai`

### Requirement: Attribution from the handshake
For each session the server SHALL default `source.harness` to the client's `clientInfo.name` when no harness is supplied by the call, server flags, environment, or `intentions.yaml`. `author` SHALL default from the call, `--author`, `INTENTIONS_AUTHOR`, then `intentions.yaml`. An attribution value or workspace path that is an unsubstituted `${…}` placeholder SHALL be treated as absent.

#### Scenario: Harness from the handshake
- **WHEN** a client identifying as `claude-ai` calls `intention_add` with no `source`
- **THEN** the intention carries `source.author` from the workspace default and `source.harness: claude-ai`

#### Scenario: Call overrides handshake
- **WHEN** the same client passes `source: {harness: "other"}`
- **THEN** the intention records `harness: other`

#### Scenario: Blank Desktop field
- **WHEN** the server is started with `--author '${user_config.author}'`
- **THEN** the author falls through to `intentions.yaml`

### Requirement: Errors are tool results with CLI codes
A domain failure SHALL be returned as a tool result with `isError: true`, a text block `<code>: <message>`, and structured `{"error": {"code", "message"}}` using the CLI's error codes (`usage`, `refused`, `invalid`, `not_found`, `check_failed`, `runtime`). Protocol-level failures only SHALL be JSON-RPC errors.

#### Scenario: Unknown id
- **WHEN** `intention_show{id: "int_nope"}` is called
- **THEN** the result has `isError: true` and `error.code: "not_found"`

### Requirement: Instructions, prompt, and conventions resource
The `initialize` response SHALL include `instructions`: a header naming the bound workspace root and its default subject, a line saying tool names are this implementation's and results equal the CLI's `--json`, the embedded skill's body, and, when `intentions.md` exists, its content under a heading naming the file, delivered to at least 16 KiB of UTF-8 cut only on a character boundary and followed by a note when truncated. A prompt named `intentions-discipline` SHALL return the same text. When `intentions.md` is readable at startup it SHALL be listed as an MCP resource with a `file://` URI, `name` `intentions.md`, `mimeType` `text/markdown`, read from disk at request time.

#### Scenario: Instructions present
- **WHEN** a client initialises
- **THEN** `instructions` contains the workspace root and the phrase "List before you add"

#### Scenario: Conventions delivered and readable
- **WHEN** the workspace root holds `intentions.md` and a client initialises, lists resources, and reads the one named `intentions.md`
- **THEN** `instructions` contains a heading naming `intentions.md` followed by its content after the skill body, the listing has exactly one resource, and its content equals the file's current bytes

#### Scenario: Truncation on a character boundary
- **WHEN** `intentions.md` is longer than 16 KiB and a multi-byte character straddles the mark
- **THEN** the delivered text is valid UTF-8, at least 16 KiB, contains the whole character, and ends with a note naming the file

### Requirement: Concurrent writes keep the index consistent
Mutating tools SHALL be serialised by a server-wide lock.

#### Scenario: Parallel adds
- **WHEN** twenty `intention_add` calls run concurrently
- **THEN** twenty intention files exist and `index --check` passes
