## ADDED Requirements

### Requirement: The skill says how a terminus changes
The embedded skill SHALL state that when a reason has changed the harness prefers superseding the terminus with a new one over rewording it in place, because the change stays visible and its dependents stay grounded; that a reason is withdrawn by setting its terminus tentative; that a terminus admits `abandoned` and `superseded` only; that retiring a terminus anything still rests on is refused; and that a harness retires only a tentative terminus it drafted itself. The verbs table's retire row SHALL note the terminus kinds.

#### Scenario: Skill states how a terminus changes
- **WHEN** the embedded skill is read
- **THEN** it says to supersede a changed reason, to set a terminus tentative to withdraw it, and never to retire one out from under what rests on it
