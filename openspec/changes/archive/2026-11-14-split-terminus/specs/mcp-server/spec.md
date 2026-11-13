## MODIFIED Requirements

### Requirement: intention_retire guards termini
`intention_retire` on a terminus SHALL apply the same refusals as `intentions intention retire`, as error results with code `refused` and nothing written: the kind `fulfilled`; a `superseded_by`, or any entry of a list, that is not a terminus of the same subject; a list with a repeated entry; a call whose source carries a harness, unless the terminus is tentative and its own `source` carries a harness; and any retirement that would leave a served intention with no firm terminus, naming those intentions and the alternatives, `intention_edit` with `stability: tentative` or supersession by one or more termini, at least one of them firm. `superseded_by` SHALL accept a string or an array of strings; an array of one SHALL be written as an id, an array of two or more SHALL be written sorted by id, and an array on anything but a terminus SHALL be refused. A result that leaves dependents resting on a split SHALL carry the `split_terminus` finding for each of them, as the CLI's does. `desire_retire` and `availability_supersede` SHALL take a single id only. Since every call through this server carries the client as harness, a firm terminus and a person's draft are never retired through it, and the tool's description SHALL say that a terminus admits `abandoned` and `superseded` only that a firm terminus is the person's to retire in the CLI, and that a terminus may be superseded by several.

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

#### Scenario: Client splits its own draft
- **WHEN** the same client calls `intention_retire{id, kind: superseded, superseded_by: [int_V, int_U]}` on a tentative terminus it drafted, int_U and int_V termini of the subject
- **THEN** the file carries `superseded_by: [int_U, int_V]` and the result's `retired.superseded_by` is that array

#### Scenario: Array on a non-terminus
- **WHEN** the same client calls `intention_retire` with `superseded_by` an array on an intention with a window
- **THEN** the result is an error with code `refused` and the file is unchanged

