## MODIFIED Requirements

### Requirement: Instance and placement errors
Validation SHALL report an error for two active intentions carrying `instance-of` the same recurring intention and the same `occurrence`; for an instance whose `occurrence` lies outside its recurring intention's calendar anchor; for a placement whose `start` is a calendar day and whose `duration` is not whole days; for a placement that lies outside its own intention's window bounds; for a RESOLUTION whose `selector` names anything other than `person` or an active terminus of the intention's subject carrying `auto_select`; and for `transparent: true` on a commitment whose `origin` is a resolution. Validation SHALL NOT compute consistency flags.

#### Scenario: Duplicate active instance
- **WHEN** two active intentions each carry `serves: [{id: int_S, role: instance-of}]` and `occurrence: 2026-09-15`
- **THEN** `validate` reports an error naming both

#### Scenario: Retired duplicate tolerated
- **WHEN** one of the two instances carries a `retired` record
- **THEN** `validate` does not report a duplicate

#### Scenario: Occurrence outside the recurring window
- **WHEN** an instance carries `occurrence: 2027-03-02` and its recurring intention's window is `2026-09/2026-12`
- **THEN** `validate` reports an error

#### Scenario: Placement outside window
- **WHEN** an intention with `window.calendar: 2026-W37` carries a placement starting 2026-09-21T10:00:00+10:00
- **THEN** `validate` reports an error

#### Scenario: Selector is not a policy
- **WHEN** a resolution file carries `selector: int_X` and int_X has a window or no `auto_select`
- **THEN** `validate` reports an error

#### Scenario: Transparent resolution-born commitment
- **WHEN** a commitment file carries `origin: {resolution: res_A}` and `transparent: true`
- **THEN** `validate` reports an error

#### Scenario: Flags are not findings
- **WHEN** two placed intentions overlap
- **THEN** `validate` reports nothing about it and `check` does
