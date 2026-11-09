## MODIFIED Requirements

### Requirement: Warnings and info
Validation SHALL report a warning for a cached `version` that disagrees with the computed value, for an object file with no `version`, for any drift between `index.yaml` and the files, for an intention that reaches no firm terminus of its own subject (code `unserved`), for an unretired intention with an `in-order-to` or `instance-of` link to an intention retired as `abandoned` (code `abandoned_end`), naming the dependent and the abandoned end and suggesting it be retired or re-pointed, while a link to one retired as `fulfilled` raises nothing, for a RESOLUTION record whose `selector` names a policy that has since been set tentative or retired (code `selector_withdrawn`), as an act authorised when it happened under a policy since withdrawn; the placement it produced stands and is the person's to keep or re-resolve, and for an intention, retired or not, whose `external` `system` and `uid` match an unretired commitment's `external` (code `hold_imported`), reported on the intention once per matching commitment, naming the commitment, the link and the fixes: if the commitment is the intention's own hold imported before it was recognised, cancel the commitment; if it is a real event that shares the UID, clear the intention's `external` (offered only for an unretired intention). It SHALL report at info level every activity or conditional term used by exactly one object, the absence of `intentions.md`, every tentative terminus as a draft (code `draft_terminus`), and any unknown key under `resolver` or `generation` in `intentions.yaml`, naming `week_start` as no longer a key and `generation.horizon` as replaced by `resolver.horizon` when that is what it finds.

#### Scenario: Stale version
- **WHEN** a file's `version` differs from the computed value
- **THEN** `validate` reports a warning naming the file and both values

#### Scenario: Missing version
- **WHEN** a file carries no `version`
- **THEN** `validate` reports a warning and exits 0 if nothing else is wrong

#### Scenario: Selected under a policy since withdrawn
- **WHEN** a resolution record's `selector` names a policy that has since been set tentative or retired
- **THEN** `validate` reports a warning with code `selector_withdrawn` on the record and exits 0 for it

#### Scenario: Stale week_start
- **WHEN** `intentions.yaml` carries `resolver.week_start`
- **THEN** `validate` reports it at info level

#### Scenario: Stale generation horizon
- **WHEN** `intentions.yaml` carries `generation.horizon`
- **THEN** `validate` reports it at info level saying `resolver.horizon` is the planning horizon, and the value is ignored

#### Scenario: Lonely term
- **WHEN** only one object uses the term `piano-practice`
- **THEN** `validate` reports it at info level and exits 0

#### Scenario: Draft terminus
- **WHEN** a terminus is `tentative`
- **THEN** `validate` reports it at info level with code `draft_terminus` and exits 0

#### Scenario: Abandoned end
- **WHEN** an unretired int_A serves int_B `in-order-to` and int_B is retired as `abandoned`
- **THEN** `validate` reports a warning with code `abandoned_end` on int_A naming int_B, and exits 0 for it

#### Scenario: Ended arrangement
- **WHEN** a recurring intention serving a firm terminus is retired as `abandoned` and two of its instances are unretired
- **THEN** `validate` reports each instance with `abandoned_end` naming the recurring intention, and neither as `unserved`

#### Scenario: Fulfilled end is silent
- **WHEN** an unretired int_A serves int_B `in-order-to` and int_B is retired as `fulfilled`
- **THEN** `validate` reports nothing about the link

#### Scenario: Hold imported before it was recognised
- **WHEN** int_A carries `external: {system: icalendar, uid: U}` and unretired cmt_X with `origin: import` carries `external: {system: icalendar, uid: U}`
- **THEN** `validate` reports a warning with code `hold_imported` on int_A naming cmt_X, `icalendar:U`, `commitment cancel cmt_X` and `intention edit int_A --clear-external`, and exits 0 for it

#### Scenario: Retired intention offers only the cancel
- **WHEN** int_A, retired as `fulfilled`, carries the link an unretired commitment also carries
- **THEN** `validate` reports `hold_imported` on int_A naming `commitment cancel` and not `--clear-external`

#### Scenario: Recurring import
- **WHEN** three unretired commitments carry the link int_A carries
- **THEN** `validate` reports three `hold_imported` warnings on int_A, one naming each commitment

#### Scenario: Cancelled duplicate is quiet
- **WHEN** the commitment sharing int_A's `external` is retired as `cancelled`
- **THEN** `validate` reports no `hold_imported` for the pair

#### Scenario: Same UID under another system
- **WHEN** int_A carries `{system: icalendar, uid: U}` and an unretired commitment carries `{system: jscalendar, uid: U}`
- **THEN** `validate` reports no `hold_imported`
