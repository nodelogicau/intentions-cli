## ADDED Requirements

### Requirement: Commitment tools
The server SHALL expose `commitment_accept`, `commitment_decline`, `commitment_cancel`, `commitment_show` and `commitment_list`, whose structured results equal the corresponding CLI verbs' `--json` output. The answering tools SHALL be described as recording the person's own answer, never the model's inference.

#### Scenario: Accept over MCP
- **WHEN** a client calls `commitment_accept{id}` for a commitment listing the workspace's default subject
- **THEN** that party's status is `accepted`, the source records the client as harness, and the result equals `commitment accept --json`

#### Scenario: Cancel frees the intention
- **WHEN** a client calls `commitment_cancel{id, reason}` and then `unresolved{}`
- **THEN** the commitment is retired as cancelled and the intention it named is listed again
