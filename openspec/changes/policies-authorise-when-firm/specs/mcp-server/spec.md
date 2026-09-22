## MODIFIED Requirements

### Requirement: The harness boundary holds
`intention_firm` and `select` with `policy` SHALL require, for a source carrying a harness, an active, firm terminus of the subject carrying the matching condition that the intention satisfies, exactly as the CLI does; a tentative policy SHALL be an error result naming the draft and the command by which the person firms it. A client identifying itself in `initialize` is a harness unless the call names an author with no harness. `intention_firm` without `policy` from a harness SHALL be an error result. When the intention is a terminus, `intention_firm` SHALL refuse any call naming a `policy` and any call whose source carries a harness; since every call through this server carries the client as harness, a terminus is never firmed through it, and the tool's description SHALL say the person firms it in the CLI.

#### Scenario: Client firms without a policy
- **WHEN** a client identifying as `claude-ai` calls `intention_firm{id}` with no `policy`
- **THEN** the result is an error with code `refused` and the file is unchanged

#### Scenario: Client firms under a policy
- **WHEN** the same client calls `intention_firm{id, policy: <firm terminus id>}` and the condition holds
- **THEN** the file carries `stability: firm`, `firmed_under: <terminus id>`, and `source.harness: claude-ai`

#### Scenario: Client firms under a draft
- **WHEN** the same client calls `intention_firm{id, policy: <tentative terminus id>}`
- **THEN** the result is an error with code `refused` naming the draft and the person's command, and the file is unchanged

#### Scenario: Client selects under a draft
- **WHEN** the same client calls `select{id, policy: <tentative terminus id>}`
- **THEN** the result is an error with code `refused` and nothing is written

#### Scenario: Client firms a terminus under a policy
- **WHEN** the same client calls `intention_firm{id: <terminus id>, policy: <policy id>}`
- **THEN** the result is an error with code `refused` saying no policy applies to a terminus, and the file is unchanged

#### Scenario: Author-only call still carries the client as harness
- **WHEN** a call `intention_firm{id: <terminus id>, source: {author: <uri>}}` names an author and no harness
- **THEN** the result is an error with code `refused`, because the session's client is the harness, and the file is unchanged
