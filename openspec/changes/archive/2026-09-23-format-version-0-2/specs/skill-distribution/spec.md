## ADDED Requirements

### Requirement: The skill names the 0.2 format and migration
The embedded skill SHALL use the 0.2 names (`--capacity`, `--activities`), SHALL state that a person's own act is the absence of `firmed_under` on an object and `selector: person` on a record, and SHALL say how a workspace moves to 0.2: run `migrate --check`, walk up any unserved intention, then `migrate` on a clean checkout and review the diff; the harness reports and never runs the migration unasked.

#### Scenario: Skill states migration
- **WHEN** the embedded skill is read
- **THEN** it names `migrate --check`, says the person runs the migration, and uses `--capacity` and `--activities`
