## ADDED Requirements

### Requirement: A habit generates nothing
`generate`, and the generation resolution runs over its own range, SHALL create no instance for a recurring intention carrying `habit: true` in a `intentions/0.2` workspace. When `generate --recurring` names such an intention, the result SHALL list it under `skipped` with a reason saying it is held as habit; an unnamed run SHALL skip it without listing it. In a `intentions/0.1` workspace `habit` SHALL have no effect on generation.

#### Scenario: A habit generates nothing
- **WHEN** `generate` runs over a horizon containing occurrences of a recurring intention carrying `habit: true` in a 0.2 workspace
- **THEN** no instance is created for it and it is not listed under `skipped`

#### Scenario: A named habit is reported
- **WHEN** `generate --recurring int_R` runs and int_R carries `habit: true`
- **THEN** nothing is created and `skipped` lists int_R saying it is held as habit

#### Scenario: Merged habit under 0.1
- **WHEN** `generate` runs in a 0.1 workspace where a recurring intention carries `habit: true` by merge
- **THEN** instances are generated as if `habit` were absent
