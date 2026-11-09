## ADDED Requirements

### Requirement: A commitment and the intention it fulfils are one occupation
A commitment and the intention named by its `intention` SHALL NOT be reported as a `window-clash` with each other, whatever the commitment's `origin`. Where the two carry the same interval, the subject's capacity SHALL be consumed once and resolution SHALL count the interval once as occupied. Where their intervals differ, each SHALL occupy its own. Each SHALL be checked against every other object as usual.

#### Scenario: Resolution's commitment
- **WHEN** a multi-party intention is resolved and its commitment carries the same placement
- **THEN** no `window-clash` is reported between them and the subject's capacity for that interval is consumed once

#### Scenario: Adopted import
- **WHEN** an imported commitment has been adopted for an intention at the same placement
- **THEN** no `window-clash` is reported between them, and another intention's candidates treat the interval as occupied once

#### Scenario: Moved adopted event
- **WHEN** an adopted commitment was re-imported one day later and the intention keeps its old placement
- **THEN** no `window-clash` is reported between them, and both intervals are occupied for the subject
