## ADDED Requirements

### Requirement: Write tools report what they accepted with a warning
`intention_add`, `intention_edit` and `intention_firm` SHALL carry `findings` in their structured result exactly as the CLI verbs do: a list of `{severity, code, id, message}` present when the written intention is unserved (code `unserved`, severity `warning`) or is a tentative terminus (code `draft_terminus`, severity `info`), and absent otherwise. The tool descriptions SHALL say that an intention that is not a terminus should serve one, and that a terminus is firmed only by the person.

#### Scenario: Unserved write over MCP
- **WHEN** a client identifying as `claude-ai` calls `intention_add{title, duration, window}` in a workspace holding no firm terminus of the subject
- **THEN** the file is written, the result is not an error, and its structured content carries `findings` with one `unserved` entry naming the new id

#### Scenario: Draft terminus over MCP
- **WHEN** the same client calls `intention_add{title}` with no window, duration or serves
- **THEN** the write is accepted tentative and the result carries a `draft_terminus` finding at info level

## MODIFIED Requirements

### Requirement: The harness boundary holds
`intention_firm` and `select` with `policy` SHALL require, for a source carrying a harness, a terminus of the subject carrying the matching condition that the intention satisfies, exactly as the CLI does; a client identifying itself in `initialize` is a harness unless the call names an author with no harness. `intention_firm` without `policy` from a harness SHALL be an error result. When the intention is a terminus, `intention_firm` SHALL refuse any call naming a `policy` and any call whose source carries a harness, so a terminus is firmed only by a call naming an author with no harness.

#### Scenario: Client firms without a policy
- **WHEN** a client identifying as `claude-ai` calls `intention_firm{id}` with no `policy`
- **THEN** the result is an error with code `refused` and the file is unchanged

#### Scenario: Client firms under a policy
- **WHEN** the same client calls `intention_firm{id, policy: <terminus id>}` and the condition holds
- **THEN** the file carries `stability: firm`, `firmed_under: <terminus id>`, and `source.harness: claude-ai`

#### Scenario: Client firms a terminus under a policy
- **WHEN** the same client calls `intention_firm{id: <terminus id>, policy: <policy id>}`
- **THEN** the result is an error with code `refused` saying no policy applies to a terminus, and the file is unchanged

#### Scenario: Person firms a terminus through the server
- **WHEN** a call `intention_firm{id: <terminus id>, source: {author: <uri>}}` names an author and no harness
- **THEN** the file carries `stability: firm`, no `firmed_under`, and no `source.harness`
