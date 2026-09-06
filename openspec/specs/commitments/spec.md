# Commitments

## Purpose

Answering a commitment and cancelling one: party statuses, which are recorded and never decided; the terminal `cancelled` retirement and the placement it frees; and reading commitments.

## Requirements

### Requirement: Answer a commitment
`intentions commitment accept <id> [--party <uri>]` and `intentions commitment decline <id> [--party <uri>]` SHALL set that party's `status` to `accepted` or `declined` on an unretired commitment, changing no other party and no other field. The party SHALL default to `defaults.subject` and SHALL be required when the workspace declares none. No policy SHALL authorise either act and no check, flag or resolution SHALL set a status. The result SHALL carry the commitment, its previous version, and the current flags.

#### Scenario: Owner accepts
- **WHEN** `commitment accept cmt_A` is run in a workspace whose default subject is a party at `tentative`
- **THEN** that entry becomes `accepted`, every other party is unchanged, and the version changes

#### Scenario: Party not on the commitment
- **WHEN** `--party` names a URI the commitment does not list
- **THEN** the command exits with code 2 naming the parties it does list

#### Scenario: Already that status
- **WHEN** the party is already `accepted` and `commitment accept` is run
- **THEN** the command exits with code 2 and the file is unchanged

#### Scenario: Retired commitment refuses
- **WHEN** the commitment carries a `retired` record
- **THEN** `accept` and `decline` exit with code 2

### Requirement: Cancel a commitment
`intentions commitment cancel <id> [--reason <text>]` SHALL append a `retired` record with `kind: cancelled`, the only kind a commitment admits, and SHALL clear the `placement` of the intention it names when that intention is unretired and placed. The intention SHALL keep its window, duration and stability, SHALL NOT be retired, and SHALL become eligible for resolution again. The RESOLUTION record SHALL NOT be modified. The result SHALL carry the commitment and the intention it freed.

#### Scenario: Cancel and re-resolve
- **WHEN** a commitment created from int_A is cancelled
- **THEN** the commitment carries `retired.kind: cancelled` with the reason, source and timestamp, int_A has no placement, and `unresolved` lists int_A again

#### Scenario: Cancellation is terminal
- **WHEN** a cancelled commitment is cancelled again
- **THEN** the command exits with code 2 saying it is already retired

#### Scenario: The intention is not retired
- **WHEN** a commitment is cancelled
- **THEN** the intention it named carries no `retired` record

### Requirement: Read commitments
`intentions commitment show <id>` SHALL print the commitment with its computed version, the intention it fulfils, and the current flags. `intentions commitment list [--party <uri>] [--status <s>] [--intention <id>] [--cancelled]` SHALL list active commitments, or cancelled ones with `--cancelled`, filtered by the flags given.

#### Scenario: List by party and status
- **WHEN** `commitment list --party <uri> --status tentative` is run
- **THEN** only unretired commitments listing that party at `tentative` are returned
