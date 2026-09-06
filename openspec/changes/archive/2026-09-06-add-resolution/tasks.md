## 1. Temporal Additions

- [x] 1.1 `Interval` arithmetic: `Overlaps`, `Intersect`, `Subtract`, `Clamp`, union of interval lists, sorted merge
- [x] 1.2 `RelativeBounds(rel, targetPlacement, ctx)`: the start or end constraint per relation with the gap rules of D7
- [x] 1.3 `Occasions(window, cadence, ctx, range)`: one interval per cadence day clipped to the clock, or the plain bounds when not recurring
- [x] 1.4 `ResolutionRange(window, ctx)`: closed bounds as they are, open sides from now and `resolver.horizon`, and the past-window case
- [x] 1.5 Tests for each, including a midnight-crossing occasion and a `FINISHTOFINISH` constraint

## 2. Configuration and Store

- [x] 2.1 `resolver.step`, `resolver.horizon`, `resolver.scope` in `store.Config` with defaults, validation, `init` writing `step` and `horizon`
- [x] 2.2 `store.WriteObject` for `Commitment` and `Resolution`; `Resolution.Supply` as an implementation-added field written after `timestamp` and preserved through decode
- [x] 2.3 Graph helpers: placed objects per particular, instances per recurring intention, occasions cache per availability over a range

## 3. Generation (`internal/resolve/generate.go`)

- [x] 3.1 `Generate(g, ws, ctx, horizon, only []id, act)`: expand each active recurring intention, build instances per D14, skip existing active or retired occurrences, return created and skipped
- [x] 3.2 Tests for every `specs/generation` scenario

## 4. Resolution (`internal/resolve`)

- [x] 4.1 `supply.go`: eligibility (retired, horizon, conditional, location, scope), occasions per particular, intersection across subject and parties, intention clock intersection, the no-supply reason naming the particular
- [x] 4.2 `capacity.go`: placements resting on an occasion, remaining capacity, candidate length check
- [x] 4.3 `candidates.go`: grid enumeration per supply interval, relational anchor constraint, displacement detection against opaque placed objects of every involved particular, rank assignment, preference ordering including `adjacent` and `spread`, `--limit` with full count
- [x] 4.4 Preconditions: duration, window, retired, cycle (reuse `query.StronglyConnected`), already placed, blocked target
- [x] 4.5 Tests for every `specs/resolution` scenario

## 5. Selection (`internal/resolve/select.go`)

- [x] 5.1 Person selection by candidate index; policy selection under `auto_select` with the rank-1 rule and the harness requirement
- [x] 5.2 Build the RESOLUTION record (with `supply`), the placement with location choice, the COMMITMENT when parties exist; `--replace` semantics
- [x] 5.3 Tests for every `specs/selection` scenario

## 6. Consistency (`internal/consistency`)

- [x] 6.1 `flags.go`: window-clash (overlap and no-supply), condition-mismatch, location-mismatch, expired-ground, cycle, with counterpart versions and detail text; clashes reported on both objects; transparent commitments excluded per the spec
- [x] 6.2 `inconsistency.go`: pairwise check over unplaced intentions of one subject within the range, using the candidate engine
- [x] 6.3 `suppress.go`: acknowledgement matching and lapse
- [x] 6.4 Tests for every `specs/consistency` scenario

## 7. CLI

- [x] 7.1 `generate` with `--horizon`, `--recurring`, `--now`, attribution flags; text and JSON results
- [x] 7.2 `resolve` with `--limit`, `--step`, `--scope`, `--now`; refusals with exit 2, empty sets with reasons at exit 0; candidates numbered from 1
- [x] 7.3 `select` with `--candidate`, `--policy`, `--replace`, `--scope`, `--now`, attribution; writes record, intention, commitment; result with `flags`
- [x] 7.4 `check` with ids, `--now`, `--fail-on-flags`; `acknowledge` with `--kind`, `--counterpart`, `--reason`
- [x] 7.5 Run the check after `select`, `intention edit`, availability writes and `acknowledge`; add `flags` to `intention show`; `list --placed|--unplaced|--instances-of`
- [x] 7.6 Validation additions: occurrence outside the recurring window, placement outside its window, selector not a policy
- [x] 7.7 End-to-end tests: the full loop from `init` through `generate`, `resolve`, `select` with parties, `check`, `acknowledge`, lapse after an edit, `--replace`, and policy selection by a harness

## 8. Feedback, Docs and Release

- [x] 8.1 SPEC-FEEDBACK.md items 14 to 22 in the established form; raise them as issues on nodelogicau/intentions
- [x] 8.2 Update `skills/intentions/SKILL.md` with the resolution loop (resolve, put candidates to the person, select on their word, check, acknowledge on their word) and regenerate the committed copy
- [x] 8.3 README: status table, verb table, a resolution walkthrough; CHANGELOG 0.4.0
- [x] 8.4 Tag `v0.4.0` and verify the release
