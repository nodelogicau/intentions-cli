## MODIFIED Requirements

### Requirement: Configuration file contents
`intentions.yaml` SHALL declare `format` (one of `intentions/0.1`, `intentions/0.2`; any other value SHALL be refused by every verb naming the file's version and the versions this binary implements), `hash`, `resolver.timezone`, `resolver.horizon` (the one planning horizon, shared by resolution and instance generation, absent means `P4W`), `availability.default_horizon`, and `defaults.source.author`. It MAY declare `resolver.hemisphere` (`north` or `south`, absent means `north`), `resolver.step` (an ISO 8601 duration, the candidate grid, absent means `PT15M`), `resolver.scope` (`personal`, `organisation` or `public`, absent means `personal`), `defaults.subject`, and `calendar.addresses` (a map from a subject URI to a list of calendar addresses, each an absolute URI such as `mailto:ben@example.org`, identifying that particular in imported events; no address SHALL appear under two subjects, and the key is outside every projection). It SHALL NOT declare `resolver.week_start` or `generation.horizon`. Keys this implementation does not know SHALL be preserved when the file is rewritten; an unknown key under `resolver` or `generation` SHALL be ignored and reported by `validate` at info level. Every command SHALL fail with a usage error if `format` is not `intentions/0.1` or `hash` is not `sha256`.

#### Scenario: Newer format refused
- **WHEN** `intentions.yaml` carries `format: intentions/0.3`
- **THEN** every workspace verb exits with code 2 naming `intentions/0.3` and the versions this binary reads

#### Scenario: Older format accepted
- **WHEN** `intentions.yaml` carries `format: intentions/0.1`
- **THEN** every verb runs under the 0.1 rules and shapes

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
- **THEN** `intentions.yaml` carries `resolver.step: PT15M` and `resolver.horizon: P4W`, and no `generation` section

#### Scenario: Stale generation horizon
- **WHEN** an existing `intentions.yaml` still carries `generation.horizon: P2W`
- **THEN** every verb runs, the planning horizon is `resolver.horizon`, and `validate` reports the key at info level

#### Scenario: Invalid step refused
- **WHEN** `intentions.yaml` carries `resolver.step: 15m`
- **THEN** every workspace verb exits with code 2 naming the key

#### Scenario: Calendar addresses
- **WHEN** `intentions.yaml` declares `calendar.addresses` mapping `https://example.org/people/ben` to `[mailto:ben@example.org, mailto:ben@work.example]`
- **THEN** every verb loads it, an import maps either address to that URI, and `config` rewrites keep the key

#### Scenario: Address under two subjects
- **WHEN** `calendar.addresses` lists `mailto:x@example.org` under two subjects
- **THEN** every workspace verb exits with code 2 naming the address
