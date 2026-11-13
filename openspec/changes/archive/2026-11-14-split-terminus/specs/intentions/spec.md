## MODIFIED Requirements

### Requirement: Retire an intention
`intentions intention retire <id> --kind fulfilled|abandoned|superseded [--superseded-by id]... [--reason text]` SHALL append a `retired` record with `source` and `timestamp`. `superseded_by` SHALL be required when and only when the kind is `superseded`, and every target SHALL exist and SHALL NOT be the intention itself. `--superseded-by` MAY be repeated only on a terminus: one value SHALL be written as an id, and two or more SHALL be written as a list sorted by id. A repeated value SHALL be refused with exit code 2, and so SHALL two or more values on an intention that is not a terminus (rule `superseded_by`); nothing SHALL be written. The result's `retired.superseded_by` SHALL be a string for one successor and an array for a split.

When the intention is a terminus the command SHALL also refuse with exit code 2, and nothing SHALL be written:
- the kind `fulfilled` (rule `terminus_fulfilled`), since a self-understanding is never discharged; the message SHALL name `abandoned` and `superseded`;
- a `superseded_by`, or any entry of a list, that is not a terminus of the same subject (rule `terminus_successor`);
- an act whose source carries a harness (rule `harness_retire`), unless the terminus is `tentative` and its own `source` carries a harness; the message SHALL name the act as the person's;
- any retirement after which some unretired intention that reaches a firm terminus before the act would reach none (rule `ungrounds`), computed by the walk in Every intention reaches a terminus. The message SHALL name those intentions and the alternatives: set the terminus tentative with `intention edit <id> --stability tentative`, or retire it as `superseded` by one or more termini of the subject, at least one of them firm, with `--superseded-by` repeated for each.

An intention with another firm ground, the dependents of a tentative terminus, desires serving the terminus, and intentions whose `firmed_under` names it SHALL NOT block the act. These refusals SHALL apply under `intentions/0.1` and `intentions/0.2` alike.

A split's result SHALL carry, under `findings`, the `split_terminus` finding for each unretired intention left resting on the terminus, as `validate` would report it, and text output SHALL print them.

#### Scenario: Superseded
- **WHEN** `intention retire int_A --kind superseded --superseded-by int_C` is run and int_C exists
- **THEN** the file carries `retired: {kind: superseded, superseded_by: int_C, source: …, timestamp: …}` and its version changes

#### Scenario: Superseded without target refused
- **WHEN** `intention retire int_A --kind superseded` is run
- **THEN** the command exits with code 2

#### Scenario: Superseded-by on another kind refused
- **WHEN** `intention retire int_A --kind fulfilled --superseded-by int_C` is run
- **THEN** the command exits with code 2

#### Scenario: Unknown kind refused
- **WHEN** `intention retire int_A --kind cancelled` is run
- **THEN** the command exits with code 2 naming the three intention kinds

#### Scenario: Fulfilled terminus refused
- **WHEN** `intention retire int_T --kind fulfilled` is run on a terminus
- **THEN** the command exits with code 2 naming `abandoned` and `superseded`, and the file is unchanged

#### Scenario: Terminus superseded by a scheduled intention
- **WHEN** `intention retire int_T --kind superseded --superseded-by int_B` is run on a terminus and int_B has a window
- **THEN** the command exits with code 2 with rule `terminus_successor`

#### Scenario: Sole ground refused
- **WHEN** the person runs `intention retire int_T --kind abandoned` on the only firm terminus that unretired int_A and int_B reach
- **THEN** the command exits with code 2 naming int_A, int_B, `--stability tentative` and supersession, and the file is unchanged

#### Scenario: Another ground remains
- **WHEN** int_A serves firm termini int_T and int_U `for-the-sake-of` and the person retires int_T as `abandoned`
- **THEN** the command exits 0 and `validate` reports nothing for int_A

#### Scenario: Nothing rests on it
- **WHEN** the person retires as `abandoned` a firm terminus that no unretired intention reaches
- **THEN** the command exits 0

#### Scenario: Superseded by a firm terminus
- **WHEN** the person retires the sole ground of int_A as `superseded` with `--superseded-by` naming a firm terminus int_U of the subject
- **THEN** the command exits 0 and int_A is served through int_U

#### Scenario: Superseded by a draft
- **WHEN** the person retires the sole ground of int_A as `superseded` with `--superseded-by` naming a tentative terminus of the subject
- **THEN** the command exits with code 2 naming int_A

#### Scenario: Harness withdraws its own draft
- **WHEN** `intention retire int_T --kind abandoned --harness claude` is run on a tentative terminus whose `source` carries a harness and on which nothing rests
- **THEN** the command exits 0

#### Scenario: Harness retires a firm terminus
- **WHEN** `intention retire int_T --kind abandoned --harness claude` is run on a firm terminus
- **THEN** the command exits with code 2 with rule `harness_retire`, naming the act as the person's

#### Scenario: Harness retires the person's draft
- **WHEN** `intention retire int_T --kind abandoned --harness claude` is run on a tentative terminus whose `source` carries no harness
- **THEN** the command exits with code 2 with rule `harness_retire`

#### Scenario: Refused under 0.1 too
- **WHEN** the person retires as `abandoned` the sole firm terminus of int_A in a workspace whose `format` is `intentions/0.1`
- **THEN** the command exits with code 2 naming int_A

#### Scenario: Split into two
- **WHEN** the person runs `intention retire int_T --kind superseded --superseded-by int_V --superseded-by int_U` on a terminus, int_U and int_V termini of its subject, int_U firm
- **THEN** the file carries `superseded_by` as a block sequence `[int_U, int_V]`, and the result's `findings` name each unretired dependent of int_T with code `split_terminus`

#### Scenario: Split with only drafts
- **WHEN** the person splits the sole ground of int_A into two tentative termini
- **THEN** the command exits with code 2 with rule `ungrounds` naming int_A

#### Scenario: Repeated successor
- **WHEN** `intention retire int_T --kind superseded --superseded-by int_U --superseded-by int_U` is run on a terminus
- **THEN** the command exits with code 2 and the file is unchanged

#### Scenario: Split of a scheduled intention
- **WHEN** `intention retire int_B --kind superseded --superseded-by int_C --superseded-by int_D` is run and int_B has a window
- **THEN** the command exits with code 2 with rule `superseded_by`

#### Scenario: One entry of another subject
- **WHEN** a terminus of Ada's is split into one of Ada's termini and one of Priya's
- **THEN** the command exits with code 2 with rule `terminus_successor` naming Priya's terminus

#### Scenario: Other verbs take one successor
- **WHEN** `desire retire` or `availability retire` is run with `--superseded-by` given twice
- **THEN** the command exits with code 2 and nothing is written

