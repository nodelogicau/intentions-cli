## 1. Occupancy

- [x] 1.1 `PlacedObjects`: a commitment lists only parties at `tentative` or `accepted`, and one occupying nobody is dropped
- [x] 1.2 The intention/commitment merge holds only where the commitment still occupies the subject, so a declined subject keeps consuming through the placement
- [x] 1.3 `Firm` becomes `FirmFor(uri)`: an intention's stability, or the named party's own entry being `accepted`; ranking and `displaces` read it

## 2. The seventh flag

- [x] 2.1 `party-declined` in `consistency`: one flag per declining party, on both objects, detail naming the party and whether it is the subject
- [x] 2.2 Kinds list, `acknowledge` refusal text, and every "six kinds" in help, README, skill, docs and specs

## 3. Tests

- [x] 3.1 Per-party occupancy and capacity, including the subject-declines-their-own-plan case
- [x] 3.2 Ranking by the resolving subject's own entry, and a declined commitment neither displaced nor ranked
- [x] 3.3 The flag: both objects, both details, imports exempt, cancelled exempt, acknowledgement and lapse

## 4. Docs and release

- [x] 4.1 SPEC-FEEDBACK item 24 records the settlement; CHANGELOG 0.9.0; skill rules for commitments
- [x] 4.2 Tag `v0.9.0` and verify the release
