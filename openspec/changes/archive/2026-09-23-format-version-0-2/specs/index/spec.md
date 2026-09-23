## MODIFIED Requirements

### Requirement: Index entry shape
`index.yaml` SHALL carry `format` equal to the workspace's format version and `entries`, one per object file in every type directory, each with `id`, `type` (`desire`, `intention`, `availability`, `commitment`, `resolution`), `subject` (omitted for commitments and resolutions), `path` (relative to the workspace root), `version`, `retired` (the kind, when present), and `refs` (the sorted outbound ids from `serves`, `window.relative.target`, `retired.superseded_by`, `retired.adopted_as`, `intention`, `origin.resolution`, `displaced`). Entries SHALL be sorted by id. Nothing not derivable from the file SHALL appear.

#### Scenario: Intention entry
- **WHEN** an intention serving int_B with a relative anchor on int_C is indexed
- **THEN** its entry carries `type: intention`, its subject, `path: intentions/<id>.yaml`, its version, no `retired`, and `refs: [int_B, int_C]`

#### Scenario: Adopted desire entry
- **WHEN** a desire adopted as int_D is indexed
- **THEN** its entry carries `type: desire`, `path: desires/<id>.yaml`, `retired: adopted`, and `refs` including int_D
