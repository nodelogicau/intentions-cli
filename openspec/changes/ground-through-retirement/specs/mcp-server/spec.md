## ADDED Requirements

### Requirement: intention_retire guards termini
`intention_retire` on a terminus SHALL apply the same refusals as `intentions intention retire`, as error results with code `refused` and nothing written: the kind `fulfilled`; a `superseded_by` that is not a terminus of the same subject; a call whose source carries a harness, unless the terminus is tentative and its own `source` carries a harness; and any retirement that would leave a served intention with no firm terminus, naming those intentions and the alternatives, `intention_edit` with `stability: tentative` or supersession by a firm terminus. Since every call through this server carries the client as harness, a firm terminus and a person's draft are never retired through it, and the tool's description SHALL say that a terminus admits `abandoned` and `superseded` only and that a firm terminus is the person's to retire in the CLI.

#### Scenario: Client retires a firm terminus
- **WHEN** a client identifying as `claude-ai` calls `intention_retire{id, kind: abandoned}` on a firm terminus
- **THEN** the result is an error with code `refused` naming the act as the person's, and the file is unchanged

#### Scenario: Client withdraws its own draft
- **WHEN** the same client calls `intention_retire{id, kind: abandoned}` on a tentative terminus whose `source` carries a harness
- **THEN** the file carries the `retired` record

#### Scenario: Client and a person's draft
- **WHEN** the same client calls `intention_retire` on a tentative terminus whose own `source` carries no harness
- **THEN** the result is an error with code `refused`

#### Scenario: Client fulfils a terminus
- **WHEN** the same client calls `intention_retire{id, kind: fulfilled}` on a tentative terminus it drafted
- **THEN** the result is an error with code `refused` naming `abandoned` and `superseded`
