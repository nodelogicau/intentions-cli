## MODIFIED Requirements

### Requirement: Policy-authorised selection
`select --policy <id>` SHALL require a source carrying a harness, an active, firm terminus of the subject carrying `auto_select` whose terms the intention satisfies, and at least one rank-1 candidate; it SHALL select the top candidate and record the policy id as `selector`. A person passing `--policy`, a tentative policy whatever its condition, a policy that does not cover the intention, or no rank-1 candidate SHALL be refused with exit code 2 and the candidate set returned for the person. The refusal of a tentative policy SHALL name the draft and the command by which the person firms it.

#### Scenario: Policy authorises
- **WHEN** a firm terminus carries `auto_select: {max_duration: PT30M}` and `select int_A --policy <it> --harness claude` runs for a fifteen-minute intention with a rank-1 candidate
- **THEN** the top candidate is selected and the record carries the policy id as `selector` and the harness in `source.harness`

#### Scenario: Policy is a draft
- **WHEN** the terminus carrying `auto_select` is `tentative` and `select int_A --policy <it> --harness claude` runs
- **THEN** the command exits with code 2 naming the draft and `intentions intention firm <it>`, and no file changes

#### Scenario: Policy does not cover
- **WHEN** the same firm policy exists and the intention is two hours long
- **THEN** the command exits with code 2 and no file changes

#### Scenario: Only displacing candidates
- **WHEN** the policy covers the intention but every candidate displaces something
- **THEN** the command exits with code 2 saying a policy may select only a candidate that displaces nothing
