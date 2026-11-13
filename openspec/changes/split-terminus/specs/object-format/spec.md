## ADDED Requirements

### Requirement: A split terminus's superseded_by
`retired.superseded_by` SHALL be read as either a scalar id or a YAML sequence of ids. Writers SHALL emit a single successor as a scalar id, never as a one-element sequence. They SHALL emit two or more successors as a block sequence sorted by id, in the `superseded_by` position of the `retired` block. Readers SHALL accept a sequence in any order and SHALL preserve the order read when re-emitting a decoded record. A scalar `superseded_by` SHALL be read and written byte for byte as before. `superseded_by` SHALL stay outside every projection, so its form SHALL move no version.

#### Scenario: Split written
- **WHEN** a terminus is retired as `superseded` by int_V and int_U
- **THEN** the file's `retired` block carries `superseded_by:` followed by `- int_U` and `- int_V` as block-sequence items

#### Scenario: Single successor unchanged
- **WHEN** an intention is retired as `superseded` by int_C alone
- **THEN** the file carries `superseded_by: int_C` as a scalar

#### Scenario: Sequence on any object decodes
- **WHEN** a desire's file carries `superseded_by` as a sequence
- **THEN** the workspace loads and `validate` reports the error instead of failing to parse
