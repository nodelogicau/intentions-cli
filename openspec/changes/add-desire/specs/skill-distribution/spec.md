## ADDED Requirements

### Requirement: The skill keeps an inbox of desires
The embedded skill SHALL state that a passing remark is recorded with `desire add`, not `intention add`; that the inbox is read each session; that a desire is adopted when it has a why or a when, with `desire adopt`; that a bare adoption is refused; and that a desire is something the person said they wanted, never something the harness inferred. The verbs table SHALL carry the desire verbs.

#### Scenario: Skill states the inbox
- **WHEN** the embedded skill is read
- **THEN** it names `desire add`, `desire adopt`, and says a desire is the person's word and not an inference
