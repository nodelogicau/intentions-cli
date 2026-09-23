## MODIFIED Requirements

### Requirement: Tool surface
The server SHALL expose tools named `workspace_status`, `desire_add`, `desire_edit`, `desire_adopt`, `desire_retire`, `desire_show`, `desire_list`, `intention_add`, `intention_edit`, `intention_firm`, `intention_retire`, `intention_show`, `intention_list`, `availability_add`, `availability_renew`, `availability_supersede`, `availability_retire`, `availability_list`, `commitment_accept`, `commitment_decline`, `commitment_cancel`, `commitment_show`, `commitment_list`, `generate`, `resolve`, `select`, `unresolved`, `check`, `acknowledge`, `bounds`, `validate`, and `migrate`. Each tool's structured result SHALL equal the corresponding CLI verb's `--json` output. Every write SHALL apply the same rules and refusals as the verb. Query tools SHALL be annotated read-only and no tool destructive. Inputs that name a window SHALL take `{calendar, clock, relative}`; a duration SHALL be a string or `{nominal, min, max}`; `serves` a list of `{id, role}`. `desire_adopt` SHALL write the intention first and the desire's retirement second, return both, refuse a retired desire, and refuse a bare adoption naming what is missing; `desire_retire` SHALL accept `abandoned` and `superseded` and refuse `adopted`.

#### Scenario: Add, resolve, select over MCP
- **WHEN** a client calls `availability_add`, then `intention_add{title, duration: "PT90M", window: {calendar: "2026-W38"}}`, then `resolve{id}`, then `select{id, candidate: 1}`
- **THEN** the intention file carries the first candidate as its placement and the result equals `select --json`

#### Scenario: Same file either way
- **WHEN** an intention is added over MCP and an identical one through the CLI with the same timestamp
- **THEN** the two files differ only in `id` and `version`

#### Scenario: Refusal over MCP
- **WHEN** a client calls `intention_add` with `activity: "Deep Work"`
- **THEN** the result is an error with code `refused` or `invalid` naming the rule and nothing is written

#### Scenario: Adopt over MCP
- **WHEN** a client calls `desire_add{title}` then `desire_adopt{id, duration: "PT30M", window: {calendar: "2026-W40"}}`
- **THEN** the result carries `intention.id` and `desire.id`, the intention file is tentative with the title and duration, and the desire file is retired as `adopted` naming it

#### Scenario: Bare adopt over MCP
- **WHEN** a client calls `desire_adopt{id}` on a desire with an empty `serves`
- **THEN** the result is an error with code `refused` naming a why or a when, and nothing is written

## ADDED Requirements

### Requirement: Migrate tool
`migrate` SHALL take `check?` and SHALL do what `intentions migrate [--check]` does with the same result and refusals, so that a harness can report what a migration would change and the person can run it.

#### Scenario: Migrate check over MCP
- **WHEN** a client calls `migrate{check: true}` on a 0.1 workspace
- **THEN** the result lists what would change and no file is modified
