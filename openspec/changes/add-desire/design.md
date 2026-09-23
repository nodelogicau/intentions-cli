## Context

Every type-specific place in this binary either switches on the object's Go type or iterates `model.Types`: the prefix and directory, the decoder and encoder, the structural checker, the projection field sets, the loader, the index, validation's referential pass, and the show verb. Availability is the closest template: one CLI file, one MCP file, a struct, and a case in each switch. Two things DESIRE needs that no existing type has: a decoder that refuses named fields instead of keeping them as extras, and a write that produces two objects in one act.

## Goals / Non-Goals

**Goals:**
- A desire is a first-class object everywhere an object is read, written, validated, indexed and shown, and is invisible everywhere an intention is planned.
- Adoption is one verb with one result carrying both ids, and every rule it needs is stated in the model so the CLI and the server cannot drift.
- Nothing about intentions changes.

**Non-Goals:**
- Ranking, ageing or aggregating desires.
- A transaction across the two files adoption writes.
- Site or bundle changes beyond the tool count.

## Decisions

### D1. `Desire` is its own struct, not an `Intention` with fields forbidden
A separate struct with only the admitted fields makes the exemptions free: every planner iterates `g.Intentions()`, so a desire is never resolved, checked, listed as unresolved or walked for grounding without any code saying so. The cost is one more case in each switch, which is the cost of a type.

### D2. The decoder names the forbidden fields
The generic decoder keeps unknown keys as extras and round-trips them silently, which is right for a key the format has not defined and wrong for `window` on a desire, which the spec makes an error. The desire decoder lists each forbidden key (`duration`, `window`, `stability`, `firmed_under`, `parties`, `cadence`, `occurrence`, `placement`, `preference`, `auto_select`, `auto_firm`, `acknowledgements`) and records a `forbidden_field` problem for it, so validation reports it and a write carrying it is refused as invalid. Truly unknown keys stay extras, as on every type.

### D3. `Retired` gains `AdoptedAs`, and the checks branch on the object's type
`adopted_as` sits in the retired block after `superseded_by`, is decoded and encoded on every type, and is checked as `superseded_by` is: required when and only when the kind is `adopted`, an identifier, naming an existing intention. `CheckRetirement` refuses `adopted` from a retirement act on any type, since only adoption writes it, and requires a desire's `superseded_by` to name a desire, which the existing same-type rule already does. Validation's referential pass checks `adopted_as` resolves to an intention.

### D4. The serves rules for a desire live in one function
`CheckDesireServes(g, o)` requires every entry's role to be `for-the-sake-of` and every target to be an active terminus of the desire's own subject, tentative or firm; it does not walk for cycles, since a desire has no inbound references and cannot be on one. The existing sink rule needs one addition: an intention targeted `for-the-sake-of` by a desire is a terminus and may not gain serves, which follows once desires contribute to the graph's inbound index through `Refs`.

### D5. Adoption is a model function that returns the intention and the retirement
`model.Adopt(g, d, duration, window, source, timestamp) (*Intention, Retired, error)` builds the intention (tentative, the desire's fields, the act's source, fresh id and timestamp), refuses a retired desire, refuses a bare adoption when the result would be a terminus with a message naming what is missing, and returns the retirement record for the caller to append. The CLI and the server each write the intention, then write the desire; the result carries `intention`, `desire`, and the intention's `findings` (the unserved warning when its terminus is a draft, which is the moment to ask). The two writes are not a transaction: if the second fails the intention exists and the desire is still active, which is visible, and the retry lists before it adds.

### D6. Projection and version
`Fields[TypeDesire] = {subject, serves, retired.kind}`. Since `Fields` is frozen per format version and the upstream text keeps `intentions/0.1`, this is the one addition to a frozen table: a new key, no change to an existing set, which is what the format version guards.

### D7. Show and list mirror intention's, smaller
`desire show` returns the object with `serves_resolved` (the terminus's title, stability and retirement) and no flags, since a desire has none. `desire list` filters by `--subject`, `--activity`, `--retired`. The generic `show <id>` already dispatches on the prefix through `TypeOfID`.

### D8. Ids
The lenient and strict patterns gain `des`. `TypeOfID` iterates `Types`, so adding the constant is enough there.

## Risks / Trade-offs

- [A frozen projection table gains a key] → Additive; no existing object's version moves. Stated in the changelog.
- [The tool count in docs, tests and SPEC-FEEDBACK] → All named in tasks; the required-parameter test grows six rows.
- [Adoption's two writes] → D5; the intention is the record and a stale active desire is visible in `desire list`.
- [The skill grows] → One short section and two table rows; the loop gains one line.

## Migration Plan

Additive. `init` creates the directory for new workspaces; an existing workspace gains `desires/` on its first `desire add`, since writes create the directory. Release as 0.14.0.
