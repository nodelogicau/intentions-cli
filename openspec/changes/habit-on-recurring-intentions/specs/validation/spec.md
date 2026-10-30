## ADDED Requirements

### Requirement: Habit errors
Validation SHALL report an error with code `habit_not_recurring` for an intention carrying `habit: true` without `cadence`, which covers instances and termini, and an error with code `habit_format` for an intention carrying `habit: true` in a workspace whose `format` is `intentions/0.1`, naming `intentions migrate` as the fix. Each finding SHALL name the intention and the field.

#### Scenario: Habit on an instance
- **WHEN** a generated instance carries `habit: true`
- **THEN** `validate` reports `habit_not_recurring` naming the instance and exits 4

#### Scenario: Habit in a 0.1 workspace
- **WHEN** a recurring intention in a 0.1 workspace carries `habit: true` by merge
- **THEN** `validate` reports `habit_format` naming the intention and `migrate`, exits 4, and reports no `stale_version` for a cached version computed without `habit`
