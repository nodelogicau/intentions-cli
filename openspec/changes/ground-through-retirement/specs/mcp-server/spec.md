## ADDED Requirements

### Requirement: intention_retire guards termini
`intention_retire` on a terminus SHALL apply the same refusals as `intentions intention retire`, as error results with code `refused` and nothing written: the kind `fulfilled`; a `superseded_by` that is not a terminus of the same subject; any retirement that would leave a served intention with no firm terminus, naming those intentions and the alternatives, `intention_edit` with `stability: tentative` or supersession by a firm terminus; and a call whose source carries a harness, unless the terminus is tentative and its own `source` carries a harness. Since every call through this server carries the client as harness unless it names an author with no harness, a firm terminus is retired through it only by such a call. The tool's description SHALL say that a terminus admits `abandoned` and `superseded` only and that a firm terminus is the person's to retire.

#### Scenario: Refusal names what rests on it
- **WHEN** a call naming an author and no harness invokes `intention_retire{id, kind: abandoned}` on the sole firm terminus of two unretired intentions
- **THEN** the result is an error with code `refused` naming both intentions, `intention_edit` to set the terminus tentative, and supersession

#### Scenario: Client retires a firm terminus
- **WHEN** a client identifying as `claude-ai` calls `intention_retire{id, kind: abandoned}` on a firm terminus nothing rests on
- **THEN** the result is an error with code `refused` naming the act as the person's, and the file is unchanged

#### Scenario: Client withdraws its own draft
- **WHEN** the same client calls `intention_retire{id, kind: abandoned}` on a tentative terminus whose `source` carries a harness and nothing rests on
- **THEN** the file carries the `retired` record

#### Scenario: Client and a person's draft
- **WHEN** the same client calls `intention_retire` on a tentative terminus whose own `source` carries no harness
- **THEN** the result is an error with code `refused`
