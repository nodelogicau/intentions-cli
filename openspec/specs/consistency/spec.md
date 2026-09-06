# Consistency

## Purpose

`check`: the six flag kinds, flag shape, transparent commitments, suppression by acknowledgement and lapse, when writes run the check, and the rule that it changes nothing.

## Requirements

### Requirement: Check on demand
`intentions check [<id>]... [--now <RFC 3339>] [--fail-on-flags]` SHALL compute every flag over the workspace, or over the named objects and their counterparts, and report them without writing any file. With `--fail-on-flags` it SHALL exit 4 when any flag is reported. A second run on an unchanged workspace SHALL report the same flags.

#### Scenario: No flag file
- **WHEN** `check` reports flags
- **THEN** no object file is modified and a second run reports the same flags

#### Scenario: Scoped check
- **WHEN** `check int_A` is run
- **THEN** only flags whose subject or counterpart is int_A are reported

### Requirement: Flag kinds
A flag SHALL have exactly one kind: `window-clash` (a placement overlaps another opaque placement of an involved particular, or no eligible occasion contains it), `condition-mismatch` (an availability whose window contains the placement has a `conditional` that excludes the intention's `activity`), `location-mismatch` (the placement's location is outside such an availability's `location` list, or outside the intention's), `expired-ground` (every availability whose window contains the placement has expired or been retired), `intention-inconsistency` (two active unplaced intentions of one subject each have candidates alone but no non-overlapping pair within shared capacity), `cycle` (the intention is in a serves cycle). Flags SHALL carry no severity.

#### Scenario: Window clash on both
- **WHEN** commitment cmt_X overlaps intention int_Y
- **THEN** a `window-clash` is reported on cmt_X with counterpart int_Y and on int_Y with counterpart cmt_X

#### Scenario: No supply
- **WHEN** a placed intention falls where the subject has no eligible occasion
- **THEN** a `window-clash` with no counterpart is reported on it

#### Scenario: Condition mismatch
- **WHEN** an availability with `conditional: [deep-work]` contains a placement for an intention with `activity: meeting`
- **THEN** a `condition-mismatch` is reported on the intention with the availability as counterpart

#### Scenario: Availability retracted after placement
- **WHEN** the only availability a placement rests on is retired
- **THEN** the next check reports `expired-ground` on the placed object

#### Scenario: Inconsistent intentions
- **WHEN** two unplaced intentions of one subject each have candidates but every pair overlaps
- **THEN** both carry an `intention-inconsistency` flag naming the other and neither gains a `retired` record

#### Scenario: Cycle
- **WHEN** the serves graph has a cycle
- **THEN** every member carries a `cycle` flag with no counterpart

### Requirement: Flag shape
A flag SHALL carry `kind`, `subject`, `counterpart` (absent for `cycle` and for `window-clash` against absent supply), `counterpart_version` (the counterpart's computed version at check time), and a human-readable `detail`.

#### Scenario: Counterpart version present
- **WHEN** a flag has a counterpart
- **THEN** `counterpart_version` equals `version-of <counterpart>`

### Requirement: Transparent commitments occupy no time and rest on no supply
A commitment with `transparent: true` SHALL be neither subject nor counterpart of `window-clash`, and never subject of `condition-mismatch`, `location-mismatch` or `expired-ground`.

#### Scenario: Conference week does not clash
- **WHEN** an imported commitment with `transparent: true` spans 21 to 25 September and a firm intention is placed on 23 September
- **THEN** no `window-clash` is reported on either

#### Scenario: Opaque import still clashes
- **WHEN** an imported commitment with no `transparent` overlaps a placed intention
- **THEN** `window-clash` is reported on both

### Requirement: Suppression and lapse
A flag SHALL be suppressed when its subject carries an acknowledgement with the same `kind`, the same `counterpart`, and `counterpart_version` equal to the counterpart's current version. When the counterpart's version has moved, the flag SHALL be reported again and the acknowledgement SHALL remain in the list.

#### Scenario: Suppressed
- **WHEN** an acknowledged clash is rechecked and the counterpart is unchanged
- **THEN** the flag is not reported

#### Scenario: Lapsed
- **WHEN** the counterpart's window is edited after acknowledgement
- **THEN** the flag is reported again and the old acknowledgement is retained

#### Scenario: Prose edit does not lapse
- **WHEN** only the counterpart's description is edited after acknowledgement
- **THEN** the flag remains suppressed

### Requirement: Writes run the check
`select`, `intention edit`, `availability add|edit|renew|supersede|retire` and `acknowledge` SHALL run the check for the objects they touched and carry `flags` in their result. The check SHALL never retire an object or change stability, placement or party status.

#### Scenario: Selection reports its clash
- **WHEN** `select` places an intention over a tentative one
- **THEN** the result's `flags` carries the `window-clash` on both and both files are otherwise unchanged
