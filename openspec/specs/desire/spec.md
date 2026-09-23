# Desire

## Purpose

Defines DESIRE, the rung below intention: a want the person has expressed and not yet committed to, its rules, adoption, retirement, and the six verbs.

## Requirements

### Requirement: The desire object
A DESIRE SHALL be stored as `desires/<id>.yaml` with an id prefixed `des_`, and SHALL carry, in canonical order: `id`; `version`; `subject` (a URI, applied from `defaults.subject` when the caller omits it, refused when neither supplies one); `title`; optional `description`; optional `activity` (a lowercase kebab-case term); optional `location` (absolute URIs, sorted); optional `parties` (bare URIs of other particulars the want involves, a hint outside the projection); `serves` (possibly empty); optional `reference`; `source` (with an author); `timestamp`; and `retired` when retired. A DESIRE SHALL NOT carry `duration`, `window`, `stability`, `firmed_under`, `cadence`, `occurrence`, `placement`, `preference`, `auto_select`, `auto_firm` or `acknowledgements`; a file carrying any of them SHALL be reported by validation as an error naming the field, and a write that would produce one SHALL be refused. A DESIRE SHALL carry no strength, priority or ranking field. Its scheduling projection SHALL be `subject`, `serves`, and `retired.kind`.

#### Scenario: A passing remark
- **WHEN** `intentions desire add --title "call the accountant" --json` is run in a workspace with a default subject
- **THEN** `desires/<id>.yaml` is written with `id`, `version`, the default `subject`, the title, `serves: []`, `source.author`, `timestamp`, and validation reports nothing about it

#### Scenario: A desire with a window
- **WHEN** a file in `desires/` carries `window`
- **THEN** `validate` reports an error naming `window`

#### Scenario: Prose edit leaves the version
- **WHEN** `desire edit <id> --description "…"` is run
- **THEN** the file changes and its version does not

#### Scenario: Terminus change moves the version
- **WHEN** `desire edit <id> --serves <terminus>:for-the-sake-of` is run
- **THEN** its version changes and the result reports `projection_changed: true`

### Requirement: A desire serves only a terminus, which may be a draft
`serves` on a DESIRE SHALL admit only entries whose role is `for-the-sake-of`, each naming an active terminus of the same subject, tentative or firm. A write giving a desire any other role, a target that is not a terminus, or a terminus of another subject SHALL be refused, and validation SHALL report the same as an error. An empty `serves` is valid, and the rule that every intention reaches a terminus SHALL NOT apply to desires. A terminus targeted `for-the-sake-of` by a desire SHALL be a sink as when targeted by an intention.

#### Scenario: Draft want, draft self
- **WHEN** a harness writes a tentative terminus and then a desire serving it `for-the-sake-of`
- **THEN** both writes are accepted, `validate` reports the terminus as a draft and nothing about the desire

#### Scenario: In-order-to on a desire
- **WHEN** `desire add --title X --serves <id>:in-order-to` is run
- **THEN** the command exits with code 2 saying a desire serves only for-the-sake-of

#### Scenario: Serving a scheduled intention
- **WHEN** `desire add --title X --serves <id>:for-the-sake-of` names an intention with a window
- **THEN** the command exits with code 2 saying the target is not a terminus

### Requirement: Desires are exempt from planning
`resolve`, `check`, `unresolved` and `generate` SHALL NOT read desires; a desire SHALL carry no flag and SHALL never be reported as unserved. Two desires MAY conflict in any way without any finding.

#### Scenario: Conflicting wants
- **WHEN** a person holds a desire to spend September writing and a desire to spend September travelling
- **THEN** `validate` and `check` report nothing

#### Scenario: Not in unresolved
- **WHEN** a workspace holds three desires and no unplaced intention
- **THEN** `unresolved` returns an empty list

### Requirement: Retirement of a desire
`intentions desire retire <id> --kind abandoned|superseded [--superseded-by <des id>] [--reason text]` SHALL append a `retired` record. `superseded_by` SHALL be required when and only when the kind is `superseded` and SHALL name a desire. `--kind adopted` SHALL be refused, since only adoption writes the intention the pointer must name. `adopted_as` SHALL be required when and only when the kind is `adopted` and SHALL name an intention; validation SHALL report an error when it is absent, does not resolve, or names anything but an intention. A retired desire SHALL NOT be edited or adopted.

#### Scenario: Abandoned
- **WHEN** `desire retire <id> --kind abandoned --reason "no longer wanted"` is run
- **THEN** the file gains the record, its version changes, and a later `desire edit` is refused

#### Scenario: Adopted by hand
- **WHEN** `desire retire <id> --kind adopted` is run
- **THEN** the command exits with code 2 and the desire stays active

#### Scenario: Superseded by an intention
- **WHEN** a desire file carries `retired: {kind: superseded, superseded_by: int_…}`
- **THEN** `validate` reports an error

#### Scenario: Adopted without pointer
- **WHEN** a desire file carries `retired: {kind: adopted}` with no `adopted_as`
- **THEN** `validate` reports an error

### Requirement: Adoption writes the intention and then retires the desire
`intentions desire adopt <id> [--duration] [--calendar|--clock|--relative] [--json]` SHALL create a new tentative INTENTION carrying the desire's `subject`, `title`, `description`, `serves`, `reference`, `activity`, `location` and `parties`, the supplied duration and window, and the act's `source` and timestamp; and SHALL then append `retired: {kind: adopted, adopted_as: <the new id>}` to the desire. The intention is the record; nothing else SHALL be written. Adoption SHALL be refused when the desire is retired, and when the intention it would write is a terminus, that is when the desire's `serves` is empty and the act supplies neither a duration nor a window; that refusal SHALL name what is missing, a why or a when, and SHALL stand in every revision. The result SHALL carry `intention` and `desire`, each with `id`, `version` and `path`, and the intention's `findings` when it warrants any. A harness MAY adopt.

#### Scenario: Harness adopts
- **WHEN** `desire adopt <id> --duration PT30M --calendar this-week --harness claude --json` is run
- **THEN** a tentative intention is written with the desire's title, serves, activity and location, `duration: PT30M` and the resolved week, the desire is retired as `adopted` naming it, and the result carries both ids

#### Scenario: Bare adoption refused
- **WHEN** `desire adopt <id>` is run on a desire with empty `serves` and no duration or window flags
- **THEN** the command exits with code 2 saying the want needs a why or a when, and nothing is written

#### Scenario: A why and no when
- **WHEN** `desire adopt <id>` is run on a desire serving a terminus, with no duration or window flags
- **THEN** a tentative intention is written with the `serves` and no window, and `unresolved` reports it as `incomplete`

#### Scenario: A when and no why
- **WHEN** `desire adopt <id> --calendar 2026-W40` is run on a desire with empty `serves`
- **THEN** a tentative intention is written and the result carries an `unserved` finding

#### Scenario: Adopting a draft-grounded want
- **WHEN** the adopted desire served a tentative terminus
- **THEN** the result carries an `unserved` finding naming the draft to firm

#### Scenario: Adopting twice
- **WHEN** `desire adopt` is run on a desire already carrying a `retired` record
- **THEN** the command exits with code 2 and nothing is written

### Requirement: Show and list desires
`intentions desire show <id>` SHALL return the object with its version, path, and `serves_resolved` naming each terminus's title, stability and retirement. `intentions desire list [--subject] [--activity] [--retired]` SHALL list desires, active by default, as `{desires, count}`. The generic `show <id>` SHALL read a `des_` id.

#### Scenario: Show resolves the terminus
- **WHEN** `desire show <id> --json` is run on a desire serving a tentative terminus
- **THEN** `serves_resolved` carries the terminus's id, title and `stability: tentative`

#### Scenario: List active by default
- **WHEN** a workspace holds two active desires and one adopted
- **THEN** `desire list --json` returns two and `desire list --retired --json` returns one
