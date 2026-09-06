## MODIFIED Requirements

### Requirement: Configuration file contents
`intentions.yaml` SHALL declare `format`, `hash`, `resolver.timezone`, `availability.default_horizon`, `generation.horizon`, and `defaults.source.author`. It MAY declare `resolver.hemisphere` (`north` or `south`, absent means `north`), `resolver.step` (an ISO 8601 duration, the candidate grid, absent means `PT15M`), `resolver.horizon` (an ISO 8601 duration, how far ahead resolution looks, absent means `P4W`), `resolver.scope` (`personal`, `organisation` or `public`, absent means `personal`), and `defaults.subject`. It SHALL NOT declare `resolver.week_start`. Keys this implementation does not know SHALL be preserved when the file is rewritten; an unknown key under `resolver` SHALL be ignored and reported by `validate` at info level. Every command SHALL fail with a usage error if `format` is not `intentions/0.1` or `hash` is not `sha256`.

#### Scenario: Unsupported hash
- **WHEN** `intentions.yaml` carries `hash: sha512`
- **THEN** every workspace verb exits with code 2 naming the admitted algorithm

#### Scenario: Unknown key preserved
- **WHEN** `intentions.yaml` carries a key `custom: 1` and a command rewrites the file
- **THEN** `custom: 1` is still present

#### Scenario: Stale week_start key
- **WHEN** an existing `intentions.yaml` still carries `resolver.week_start: sunday`
- **THEN** every verb runs, weeks are ISO weeks, and `validate` reports the key at info level

#### Scenario: Resolver keys written by init
- **WHEN** `init` runs
- **THEN** `intentions.yaml` carries `resolver.step: PT15M` and `resolver.horizon: P4W`

#### Scenario: Invalid step refused
- **WHEN** `intentions.yaml` carries `resolver.step: 15m`
- **THEN** every workspace verb exits with code 2 naming the key
