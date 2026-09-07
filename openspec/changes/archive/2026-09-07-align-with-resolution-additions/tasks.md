## 1. One planning horizon (item 14)

- [x] 1.1 `resolver.horizon` answers generation as well as resolution; `Config.Generation` is gone and `init` takes `--horizon`
- [x] 1.2 A stale `generation.horizon` loads, is ignored, and `validate` reports it at info level

## 2. Ranged durations (item 17)

- [x] 2.1 Enumeration retries on the same grid down to `min`, offering the longest that yields any; the candidate and the placement carry the offered duration

## 3. Personal availability (item 20)

- [x] 3.1 `Eligible` admits a `personal` availability only for the intention's subject or a party, with an exclusion reason that says so

## 4. Replacement carries the commitment (item 22)

- [x] 4.1 `select --replace` cancels a live commitment on the old placement with the new resolution's id as reason, and writes a fresh one with every party tentative

## 5. Tests, docs and release

- [x] 5.1 Tests for all four, including a shorter-duration placement, an invisible personal availability, and the cancelled-and-recreated commitment
- [x] 5.2 README, skill and `docs/mcp.md` where they mention the horizon or scope; SPEC-FEEDBACK items 14, 17, 20, 22 record the settlement; CHANGELOG 0.10.0
- [x] 5.3 Tag `v0.10.0` and verify the release
