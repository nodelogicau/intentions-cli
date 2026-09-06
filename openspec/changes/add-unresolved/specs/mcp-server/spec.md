## ADDED Requirements

### Requirement: Unresolved tool
The server SHALL expose `unresolved{subject?, scope?, now?}` as a read-only tool whose structured result equals `intentions unresolved --json`.

#### Scenario: Unresolved over MCP
- **WHEN** a client adds an intention with a duration and a window, and one with a window only, then calls `unresolved{}`
- **THEN** the result has two entries, one `ready` and one `incomplete`, and `counts` reflects both
