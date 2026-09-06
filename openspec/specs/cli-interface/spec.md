# CLI Interface

## Purpose

The agent-facing contract every verb of `intentions` honours: non-interactive operation, `--json` output, stable exit codes, workspace selection, source attribution, and a deterministic clock for tests.

## Requirements

### Requirement: Non-interactive operation
The CLI SHALL never prompt for input or read from a terminal. Prose fields SHALL be accepted via `--title`, `--description`, or `--description-file <path>` where `-` means piped stdin. A command lacking required input SHALL fail with a usage error rather than prompting.

#### Scenario: Description supplied via stdin pipe
- **WHEN** `intentions intention add --title X --description-file -` is run with text piped to stdin
- **THEN** the piped bytes become the intention's `description` and the command completes without prompting

#### Scenario: Missing required flag
- **WHEN** `intentions intention add` is run without `--title`
- **THEN** the command exits with code 2 and prints a usage error to stderr

### Requirement: JSON output mode
Every verb SHALL accept `--json`. In JSON mode the command SHALL write exactly one JSON object to stdout on success, and on failure SHALL write `{"error": {"code": <string>, "message": <string>}}` to stderr and nothing to stdout. The JSON shape of each verb is the stable machine contract; text output is informational.

#### Scenario: Successful JSON output
- **WHEN** any verb is run with `--json` and succeeds
- **THEN** stdout contains a single parseable JSON object and stderr is empty

#### Scenario: Failure in JSON mode
- **WHEN** any verb is run with `--json` and fails
- **THEN** stdout is empty and stderr contains a single JSON object with `error.code` and `error.message`

### Requirement: Exit codes
The CLI SHALL use exit codes: `0` success; `1` runtime error; `2` usage error, including every write the format rules refuse; `3` not found (an id that resolves to no file); `4` check failed (`validate` with errors, `index --check` with drift); `5` no workspace found.

#### Scenario: Unknown id
- **WHEN** `intentions show int_does-not-exist` is run in a workspace
- **THEN** the command exits with code 3

#### Scenario: Refused write
- **WHEN** a write is refused by a format rule (for example a cycle in `serves`)
- **THEN** the command exits with code 2, names the rule in the message, and no file is modified

#### Scenario: No workspace
- **WHEN** any workspace verb is run outside a workspace with no `--workspace` and no `INTENTIONS_WORKSPACE`
- **THEN** the command exits with code 5

### Requirement: Workspace selection
Every workspace verb SHALL accept `--workspace <dir>`. Selection precedence SHALL be `--workspace`, then `INTENTIONS_WORKSPACE`, then discovery from the current directory as defined by the `workspace` capability.

#### Scenario: Flag wins over environment
- **WHEN** both `--workspace A` and `INTENTIONS_WORKSPACE=B` are set
- **THEN** workspace A is used

### Requirement: Source attribution on every writing verb
Every verb that writes an object or record SHALL accept `--author`, `--harness`, and `--model`, falling back to `INTENTIONS_AUTHOR`, `INTENTIONS_HARNESS`, `INTENTIONS_MODEL`, then to `defaults.source.author` in `intentions.yaml`. The resolved values SHALL be written to the `source` block of whatever the verb writes and SHALL appear in the JSON result.

#### Scenario: Harness drafts on the default author's behalf
- **WHEN** `intentions intention add --title X --harness claude --model claude-fable-5-1` is run in a workspace whose `defaults.source.author` is `https://example.com/people/ada`
- **THEN** the file carries `source: {author: https://example.com/people/ada, harness: claude, model: claude-fable-5-1}`

#### Scenario: No author anywhere
- **WHEN** `intentions intention add --title X --harness claude` is run in a workspace with no `defaults.source.author` and no `INTENTIONS_AUTHOR`
- **THEN** the write is refused with exit code 2 and a message naming `author`

### Requirement: Deterministic time for tests
Writing verbs SHALL accept `--now <RFC 3339>` and honour `INTENTIONS_NOW`, used for the default `timestamp` and for deictic resolution. When neither is set the process clock is used. The flag SHALL be documented as test-facing.

#### Scenario: Timestamp from --now
- **WHEN** an intention is added with `--now 2026-09-04T09:12:00Z`
- **THEN** the file carries `timestamp: 2026-09-04T09:12:00Z`

### Requirement: Version verb
`intentions version` SHALL print the binary version, the format version it implements (`intentions/0.1`), and in JSON mode both as fields.

#### Scenario: Version in JSON
- **WHEN** `intentions version --json` is run
- **THEN** stdout is `{"version": <string>, "format": "intentions/0.1"}`

### Requirement: No git operations
The CLI SHALL NOT run git or modify any git state. Review is the responsibility of the surrounding pull-request workflow.

#### Scenario: Write inside a repository
- **WHEN** any verb writes files inside a git repository
- **THEN** the working tree changes and no commit, branch, or staging operation occurs
