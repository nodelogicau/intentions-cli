## MODIFIED Requirements

### Requirement: Version verb
`intentions version` SHALL print the binary version, the format version it writes (`intentions/0.2`), and the format versions it reads (`intentions/0.1`, `intentions/0.2`); in JSON mode `version`, `format`, and `reads` as a list.

#### Scenario: Version in JSON
- **WHEN** `intentions version --json` is run
- **THEN** stdout is `{"version": <string>, "format": "intentions/0.2", "reads": ["intentions/0.1", "intentions/0.2"]}`
