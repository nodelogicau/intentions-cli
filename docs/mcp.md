# Using intentions from Claude Desktop and other MCP clients

`intentions serve --mcp` speaks the Model Context Protocol over stdio. One
server is bound to one workspace; a second workspace is a second server entry
in your client. There is one tool per CLI operation, and every result is
exactly what the corresponding verb prints with `--json`.

## Which should I use: the MCP server or the skill and CLI?

Both reach the same core and write the same files, so this is not a lock-in
decision. It comes down to whether your harness already has a shell.

| | Always in context | When planning work happens |
|---|---|---|
| MCP server | ~10,700 tokens (19 tool schemas ≈ 7,100, instructions ≈ 3,600) | the same |
| Skill + CLI | ~160 tokens (the skill's frontmatter description) | ~3,400 tokens (the body loads when triggered) |

**Use the skill and the CLI in harnesses that have a shell**: Claude Code,
Cursor's agent, Codex.

- The tool definitions and instructions are loaded in *every* session, whether
  or not it touches the person's plans; the skill stays collapsed to its
  description until something triggers it.
- The shell composes: `intentions resolve <id> --json | jq …`, a loop over
  ids, `check --fail-on-flags` in CI. MCP tools are atomic calls.
- The CLI resolves the workspace on **every invocation**, so it follows you
  across worktrees, branches and repositories within one session. The server
  binds its workspace root and `intentions.yaml` once at startup (object files
  are still read fresh per call).
- Review is shell work anyway: branch, write, commit, open the pull request.

**Use the MCP server where there is no shell**, or where you would rather not
grant one: Claude Desktop chat, mobile, and similar clients. There, typed
arguments earn their cost. A window is `{calendar, clock, relative}` rather
than three flags the model has to remember, `serves` is a list of `{id, role}`
pairs, and a ranged duration is `{nominal, min, max}`. It is also the right
choice if you want writes confined to an explicit allowlist of operations,
since the CLI is reachable through any shell command.

**Avoid running both in the same harness.** Two surfaces for the same
operations duplicate the context cost and make it a coin flip which one the
model reaches for. Mixing across harnesses is expected: Claude Desktop through
the bundle and Claude Code through the CLI, against one git repository. Every
object records the harness that wrote it.

## Connecting

### Claude Desktop: extension bundle

Download `intentions-<version>.mcpb` from the
[releases page](https://github.com/nodelogicau/intentions-cli/releases) and
open it; Claude Desktop asks for:

| Setting | What it is |
|---|---|
| Intentions workspace | the folder containing `intentions.yaml` (create one with `intentions init <folder>`) |
| Your URI or name | recorded as `source.author`; optional when `intentions.yaml` sets `defaults.source.author` |

The bundle contains the server binary (macOS universal, Windows x64); nothing
else needs installing. The server starts when a conversation uses its tools.

**Enable the extension after installing it.** Claude Desktop installs
extensions switched off; until you toggle `intentions` on in Settings →
Extensions, no intentions tools appear and no server is launched, so there is
nothing in `~/Library/Logs/Claude/` to explain the absence either. If tools
are still missing once it is enabled,
`~/Library/Logs/Claude/mcp-server-intentions.log` will exist and say why.

### Claude Desktop: manual configuration

If you already have `intentions` installed:

```json
{
  "mcpServers": {
    "intentions": {
      "command": "/opt/homebrew/bin/intentions",
      "args": ["serve", "--mcp", "--workspace", "/Users/you/planning", "--author", "https://example.com/people/you"]
    }
  }
}
```

### Claude Code

`.mcp.json` in the repository. The server is spawned in the project, so it
finds the workspace the same way the CLI does, `intentions.yaml` in an
ancestor or a `.intentions` pointer, and no `--workspace` is needed:

```json
{
  "mcpServers": {
    "intentions": { "command": "intentions", "args": ["serve", "--mcp"] }
  }
}
```

Prefer the skill in Claude Code (see above); this is here for completeness.

### Cursor and others

Any client that launches stdio servers: command `intentions`, args
`serve --mcp [--workspace <dir>]`.

`--workspace` and `$INTENTIONS_WORKSPACE` accept either the workspace root or
a directory holding a `.intentions` pointer to it, so a host may pass its
project directory. A server that exits before answering `initialize` (clients
report `EOF` or "connection closed") almost always could not resolve a
workspace; run the same command by hand from the same directory to see the
message.

## What the model is told

At `initialize` the server sends `instructions`: which workspace it is bound
to and its default subject, that writes land as files for a person to review,
that the tool names are this implementation's (the format names none), and the
full intentions discipline, the same text as the agent skill: list before you
add, a duration and a window never a slot, draft tentative, resolve then ask,
a policy is the only way a harness firms or selects alone. The same text is
available as the prompt `intentions-discipline` for clients that surface
prompts.

### Workspace conventions

A workspace's own conventions, the activity terms in use and what the
workspace is for, live in `intentions.md` at the workspace root, the prose
sibling of `intentions.yaml`. The server appends it to the instructions under
a heading naming the file. At least the first 16 KiB is always delivered, cut
only on a character boundary, with a note naming the file when longer. For
clients that never surface `instructions`, the same document is listed as an
MCP resource (`file://…/intentions.md`, `text/markdown`), read from disk on
each request, so an edit after startup is visible there while the
instructions stay a startup snapshot. Keep it short: it rides in every
session's context.

## Tools

The Intentions Format defines no tool set; these names are this
implementation's, one per CLI operation. Every optional field may be omitted;
`source` is `{author?, harness?, model?}`; a `window` is
`{calendar?, clock?, relative?: {target, relation, gap?: {min?, max?}}}`; a
`duration` is an ISO 8601 string or `{nominal, min?, max?}`.

| Tool | Parameters | Returns (= CLI `--json`) |
|---|---|---|
| `intention_add` | `title`, `subject?`, `duration?`, `window?`, `activity?`, `location[]?`, `parties[]?`, `serves[{id, role}]?`, `cadence?`, `preference?`, `auto_select?`, `auto_firm?`, `stability?`, `policy?`, `reference?`, `description?`, `timestamp?`, `source?` | `{id, type, path, version, object, created}` |
| `intention_edit` | `id`, any of the above, `clear[]?` | `{…, previous_version, projection_changed, flags}` |
| `intention_firm` | `id`, `policy?`, `source?` | `{…, previous_version, policy?}` |
| `intention_retire` | `id`, `kind`, `superseded_by?`, `reason?` | `{…, retired}` |
| `intention_show` | `id`, `now?` | `{…, serves_resolved, flags}` |
| `intention_list` | `subject?`, `activity?`, `stability?`, `recurring?`, `placed?`, `unplaced?`, `instances_of?`, `retired?` | `{intentions, count}` |
| `availability_add` | `subject`, `duration`, `window`, `title?`, `conditional[]?`, `location[]?`, `cadence?`, `valid_until?`, `scope?`, `description?`, `timestamp?`, `source?` | `{…, effective_valid_until, effective_valid_until_from, flags, created}` |
| `availability_renew` | `id`, `valid_until` | `{…, previous_version}` |
| `availability_supersede` | `id`, changed terms, `reason?` | `{…, created, superseded: {id, version}}` |
| `availability_retire` | `id`, `kind`, `superseded_by?`, `reason?` | `{…, retired}` |
| `availability_list` | `subject?`, `conditional?`, `scope?`, `retired?`, `now?` | `{availability, count}` |
| `generate` | `horizon?`, `recurring[]?`, `now?` | `{created, skipped, count, range}` |
| `resolve` | `id`, `limit?`, `step?`, `scope?`, `now?` | `{intention, candidates, candidates_considered, generated, range?, reason?, blocked_on?, no_supply?, excluded?}` |
| `select` | `id`, `candidate` **or** `policy`, `replace?`, `scope?`, `now?` | `{resolution, intention, candidate, commitment?, replaced?, flags, policy?}` |
| `check` | `ids[]?`, `scope?`, `now?` | `{flags, count, counts}` |
| `acknowledge` | `id`, `kind`, `counterpart?`, `reason?`, `now?` | `{…, acknowledgement, flags}` |
| `bounds` | `id?`, `calendar?`, `clock?`, `timezone?`, `hemisphere?`, `now?` | `{window, timezone, hemisphere, intervals, count}` |
| `validate` | | `{findings, counts, ok}` |
| `workspace_status` | | root, subject, author, timezone, counts, `validate` summary, flag count, and inside a git checkout `git.uncommitted` (read-only) |

Errors are tool results with `isError: true`, a text line `<code>: <message>`,
and `{"error": {"code", "message"}}` using the CLI's codes (`usage`,
`refused`, `invalid`, `not_found`, `runtime`). Every refusal the verbs make,
the tools make: a harness firming without a policy, a subject change, a cycle
in `serves`, a narrowed scope, selecting a placed intention without `replace`.

## Attribution

`source.harness` defaults to the connected client's name from the MCP
handshake (`claude-ai`, `claude-code`, `cursor`, …) when neither the call, the
server flags, `INTENTIONS_HARNESS`, nor `intentions.yaml` supplies one. So an
act through this server is always a harness's act: `intention_firm` needs
`policy`, and `select` needs either the person's `candidate` relayed by you or
a `policy`. `source.author` comes from the call, `--author`,
`INTENTIONS_AUTHOR`, or `defaults.source.author` in `intentions.yaml`; an
intention or availability with no author is refused.

**Leaving "Your URI or name" blank in the Desktop extension is safe.** Claude
Desktop substitutes `${user_config.author}` only when the field has a value;
left empty, the literal text reaches the server. An unsubstituted `${…}` in
any attribution value, whether flag, `INTENTIONS_*`, or config, is treated as
absent, so it falls through to the workspace default rather than being
recorded as the author.

## Reviewing what a Desktop conversation wrote

The server never runs git. Files accumulate in the workspace folder; if that
folder is a git checkout, `workspace_status` lists what is uncommitted so the
assistant can tell you "three intentions and a resolution await review".
Committing, branching, and opening the pull request remain yours; see
[review-workflow.md](review-workflow.md).
