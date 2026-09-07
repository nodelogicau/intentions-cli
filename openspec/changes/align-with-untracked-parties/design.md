## Context

Supply is gathered per particular: `SupplyFor` collects the occasions of every availability whose subject is that URI, and `covers` then requires each involved particular to have an occasion containing the candidate. That second requirement is what makes an external party unplaceable. Occupancy is separate and never consults availability, so the occupancy half of the settled rule already holds.

## Decisions

### D1. Tracking is derived, not configured
A party is tracked when the graph holds any availability whose subject is that URI, whatever its state: retired, expired, wrong activity, wrong scope. The test is deliberately blunt, because it answers a question about the workspace rather than about the party's time. It needs no new key and no new file, and it makes the distinction the format wanted between "offered nothing that fits", which is an answer, and "we hold nothing about them", which is silence.

### D2. Untracked parties leave the supply calculation, not the resolution
`Resolve` drops an untracked party from the list it intersects supply over and from `covers`, and never adds them to `no_supply`. Everything else keeps them: they stay in `involved` for ranking and displacement, they are parties of the commitment, and they are carried on the record as `presumed`. The subject is filtered out of this exemption before it is applied, so a subject with no records still yields an empty set naming them, which is the failure the exemption would otherwise hide.

### D3. `presumed` is computed where the exemption is
The set of presumed URIs falls out of the same pass that decides which parties to exempt, so `Resolve` returns it on the result and `Select` copies it to the record. It is sorted, written after `supply`, and excluded from the projection with it. A resolution for an intention with no untracked parties omits the key entirely rather than writing an empty list, as every other optional list does.

### D4. The clash text changes, not the check
`clashes` already reports only when two placements share a particular that both occupy, and since v0.9.0 the particulars of a commitment exclude declined parties. The settled wording describes what the code does. The spec delta records that and adds the scenarios that pin it, including the one that would have been silently wrong: two commitments sharing only an untracked party still clash.

## Risks / Trade-offs

- **Declaring your first availability makes you harder to schedule with than declaring none.** That follows from supply being a positive assertion, and it is stated in the skill rather than left to be discovered, because a person meeting it by surprise would file it as a bug.
- A placement against an untracked party rests on an assumption about someone whose calendar the workspace cannot see. `presumed` is the only mitigation, and it is a record rather than a guard.
- The exemption is per resolution, so an intention whose only party becomes tracked mid-week changes behaviour between two runs with no edit to the intention. That is the rule working as intended, but it will read as surprising, so the exclusion output should keep saying which parties were presumed.
