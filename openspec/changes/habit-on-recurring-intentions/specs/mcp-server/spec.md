## ADDED Requirements

### Requirement: intention_add and intention_edit carry habit
`intention_add` and `intention_edit` SHALL accept an optional boolean `habit`, and `intention_edit` SHALL accept `habit` in `clear`. `habit: true` that sets the field SHALL be an error result with code `refused` when the call's source carries a harness, whatever `policy` it names, naming the act as the person's; since every call through this server carries the client as harness, habituation is never done through it, and the tools' descriptions SHALL say the harness may suggest habituation and the person sets it in the CLI. `habit: false` or `clear: [habit]` SHALL clear it from any source. In a 0.1 workspace setting it SHALL be refused naming `migrate`. Results SHALL equal the CLI's.

#### Scenario: Client habituates
- **WHEN** a client identifying as `claude-ai` calls `intention_edit{id, habit: true}` on a recurring intention
- **THEN** the result is an error with code `refused` naming the act as the person's, and the file is unchanged

#### Scenario: Client reverts
- **WHEN** the same client calls `intention_edit{id, habit: false}` on a habit
- **THEN** the file no longer carries `habit` and its version changes

#### Scenario: Unresolved omits dormant instances
- **WHEN** the only unplaced intentions are instances of a habit and the client calls `unresolved{}`
- **THEN** the result has no entries
