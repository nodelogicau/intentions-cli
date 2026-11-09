## MODIFIED Requirements

### Requirement: Commitment tools
The server SHALL expose `commitment_accept`, `commitment_decline`, `commitment_cancel`, `commitment_show`, `commitment_list`, `commitment_import` (taking the event object as `event`, and optional `subject` and `commitment`) and `commitment_adopt` (taking `id`, `intention`, and optional `replace`), whose structured results equal the corresponding CLI verbs' `--json` output. The answering tools SHALL be described as recording the person's own answer, never the model's inference. `commitment_adopt` SHALL be described as the person's selection, made only on their instruction.

#### Scenario: Accept over MCP
- **WHEN** a client calls `commitment_accept{id}` for a commitment listing the workspace's default subject
- **THEN** that party's status is `accepted`, the source records the client as harness, and the result equals `commitment accept --json`

#### Scenario: Cancel frees the intention
- **WHEN** a client calls `commitment_cancel{id, reason}` and then `unresolved{}`
- **THEN** the commitment is retired as cancelled and the intention it named is listed again

#### Scenario: Import over MCP
- **WHEN** a client calls `commitment_import{event}` for a new event
- **THEN** the outcome is `created` and the result equals `commitment import --json` for the same event

#### Scenario: Adopt over MCP
- **WHEN** a client calls `commitment_adopt{id, intention}`
- **THEN** the source records the client as harness, the record's `selector` is `person`, and the result equals `commitment adopt --json`
