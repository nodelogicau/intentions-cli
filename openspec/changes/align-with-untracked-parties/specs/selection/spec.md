## MODIFIED Requirements

### Requirement: Selection is a recorded act
`intentions select <id> (--candidate N | --policy <int_id>) [--replace] [--scope <s>] [--now <RFC 3339>]` SHALL recompute the candidates as `resolve` does, take the chosen one, and write in one act: a RESOLUTION record with `id`, `version`, `intention`, `placement`, `selector` (`person` or the policy id), `candidates_considered`, `displaced` (sorted), `source`, `timestamp`, an implementation-added `supply` list naming the availabilities the placement rests on, and, when the intention names parties the workspace does not track, `presumed` listing their URIs sorted, so a reader sees whose time the placement assumes without evidence; neither `supply` nor `presumed` enters the projection; the `placement` onto the intention, with `location` from the intersection of the intention's and the supply's location lists when either constrained it; and, when the intention has `parties`, a COMMITMENT with the subject and every party at `tentative`, `origin: {resolution: <res id>}`, `intention`, `title` copied, and the act's `source`. The result SHALL carry the record, the intention, the commitment when created, and `flags` from the check that follows.

#### Scenario: Person selects
- **WHEN** `select int_A --candidate 2` is run with an author and no harness
- **THEN** a resolution record exists with `selector: person`, `source.author` set to that person and `candidates_considered` equal to the full count; int_A carries the second candidate as `placement`; and the record lists anything displaced

#### Scenario: Placement location chosen
- **WHEN** an intention with `location: [https://h/, https://o/]` is selected against supply with `location: [https://o/]`
- **THEN** the placement carries `location: https://o/` and the intention's `location` list is unchanged

#### Scenario: Multi-party selection
- **WHEN** an intention with one counterparty is selected
- **THEN** a commitment is created with the subject and the counterparty both at `tentative`, `origin.resolution` naming the record, `intention` naming int_A, and no `transparent` or `external`

#### Scenario: Supply recorded
- **WHEN** a selection is made
- **THEN** the resolution file carries `supply:` after `timestamp` naming the availability ids, and the record's version is unaffected by it

#### Scenario: The presumption is on the record
- **WHEN** a placement is selected for an intention naming an untracked external party
- **THEN** the record carries `presumed` naming that party's URI, `supply` names only the availabilities actually used, and the record's version is unaffected by either

#### Scenario: Nothing presumed
- **WHEN** every party of the intention is tracked
- **THEN** the record carries no `presumed` key
