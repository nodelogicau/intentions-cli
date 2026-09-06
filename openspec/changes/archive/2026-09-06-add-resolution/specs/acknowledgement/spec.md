## ADDED Requirements

### Requirement: Acknowledge a flag
`intentions acknowledge <id> --kind <flag kind> [--counterpart <id>] [--reason <text>]` SHALL append to the intention's or commitment's `acknowledgements` an entry with `kind`, `counterpart`, `counterpart_version` equal to the counterpart's current computed version, `reason`, the act's `source`, and `timestamp`. It SHALL refuse an unknown kind, a counterpart that does not exist, and a kind that takes a counterpart given without one. The list SHALL be append-only and the object's version SHALL be unchanged. The result SHALL carry the entry and the check's remaining `flags` for the object.

#### Scenario: Acknowledge a clash
- **WHEN** `acknowledge cmt_X --kind window-clash --counterpart int_Y --reason "The review matters more"` is run
- **THEN** cmt_X gains an acknowledgement with kind `window-clash`, counterpart int_Y, int_Y's current version, the reason, the person as `source.author`, and a timestamp, and cmt_X's version is unchanged

#### Scenario: Acknowledgement on a retired object
- **WHEN** `acknowledge` targets an intention that carries a `retired` record
- **THEN** the append is accepted and no other field changes

#### Scenario: Unknown kind refused
- **WHEN** `acknowledge int_A --kind overlap` is run
- **THEN** the command exits with code 2 naming the six kinds

#### Scenario: Missing counterpart refused
- **WHEN** `acknowledge int_A --kind window-clash` is run with no `--counterpart` and int_A has supply
- **THEN** the command exits with code 2 saying the kind names a counterpart

#### Scenario: Cycle acknowledgement has no counterpart
- **WHEN** `acknowledge int_A --kind cycle --reason "known, being untangled"` is run
- **THEN** the entry carries no `counterpart` and no `counterpart_version`

### Requirement: Acknowledgements are outside the projection
Appending an acknowledgement SHALL NOT change the object's version.

#### Scenario: Version unchanged
- **WHEN** an acknowledgement is appended
- **THEN** `version-of` returns the same value as before
