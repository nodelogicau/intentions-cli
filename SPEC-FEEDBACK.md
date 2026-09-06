# Feedback on the Intentions Format draft from the `intentions` reference implementation

The [Intentions Format](https://github.com/nodelogicau/intentions) README says
that no implementation exists yet and that the first will shape the text as the
DKF reference implementation shaped its specification. This file is that
record: the points where implementing the draft forced a decision the text did
not make, what this implementation decided, and how each was resolved upstream
once it was. "We" below means this implementation as it was when the item was
raised.

Items 1 to 13 were raised on 2026-09-06 from the `bootstrap-intentions-cli`
change (v0.1.0), which implements the object model, the temporal engine,
validation and the index and stops before resolution. **All thirteen were
settled the same day** in
[nodelogicau/intentions@b3bb420](https://github.com/nodelogicau/intentions/commit/b3bb420)
(the `settle-first-reader-feedback` change); the resolution is recorded under
each item. Nine were adopted as proposed; bold entries are where the draft
decided differently and v0.2.0 changed to follow it.

Items 14 to 22 were raised on 2026-09-06 from the `add-resolution` change
(v0.4.0), which implements generation, resolution, selection, the consistency
check and acknowledgement. Each is a rule the text did not state and the
implementation had to invent; "we decided" is what v0.4.0 does.

| # | Topic | Status |
|---|---|---|
| 1 | Duration normalisation in the projection | adopted, [#1](https://github.com/nodelogicau/intentions/issues/1) |
| 2 | Set-valued list ordering in the projection | adopted, [#2](https://github.com/nodelogicau/intentions/issues/2) |
| 3 | Position of the cached `version` in canonical order | adopted, [#3](https://github.com/nodelogicau/intentions/issues/3) |
| 4 | The seed an RRULE cadence expands from | adopted, [#4](https://github.com/nodelogicau/intentions/issues/4) |
| 5 | Week granules under a non-Monday `week_start` | **decided differently**, [#5](https://github.com/nodelogicau/intentions/issues/5) |
| 6 | Season codes need a hemisphere | **decided differently**, [#6](https://github.com/nodelogicau/intentions/issues/6) |
| 7 | Explicit `transparent: false` in the projection | adopted, [#7](https://github.com/nodelogicau/intentions/issues/7) |
| 8 | Where a policy-authorised firming is recorded | **decided differently**, [#8](https://github.com/nodelogicau/intentions/issues/8) |
| 9 | "Standing intention" names two things | **decided differently**, [#9](https://github.com/nodelogicau/intentions/issues/9) |
| 10 | Deictic vocabulary a writer accepts | adopted, [#10](https://github.com/nodelogicau/intentions/issues/10) |
| 11 | `validate` cannot tell a policy-authorised harness firming from an unauthorised one | adopted, [#11](https://github.com/nodelogicau/intentions/issues/11) |
| 12 | The canonical form of a zero duration | adopted, [#12](https://github.com/nodelogicau/intentions/issues/12) |
| 13 | The example objects disagree on list style | adopted, [#13](https://github.com/nodelogicau/intentions/issues/13) |
| 14 | Resolution needs a bounded range: horizon and now | open, [#14](https://github.com/nodelogicau/intentions/issues/14) |
| 15 | Capacity depletes per occasion | open, [#15](https://github.com/nodelogicau/intentions/issues/15) |
| 16 | What a placement rests on is recomputed; propose `supply` on the record | open, [#16](https://github.com/nodelogicau/intentions/issues/16) |
| 17 | Candidate enumeration on a step grid | open, [#17](https://github.com/nodelogicau/intentions/issues/17) |
| 18 | Relational anchor bounds per relation | open, [#18](https://github.com/nodelogicau/intentions/issues/18) |
| 19 | `intention-inconsistency` is pairwise | open, [#19](https://github.com/nodelogicau/intentions/issues/19) |
| 20 | The resolver's scope comes from `resolver.scope` | open, [#20](https://github.com/nodelogicau/intentions/issues/20) |
| 21 | Instances copy `location` and the recurring intention's clock | open, [#21](https://github.com/nodelogicau/intentions/issues/21) |
| 22 | Re-resolving a placed intention | open, [#22](https://github.com/nodelogicau/intentions/issues/22) |

---

## 1. Duration normalisation in the projection ([#1](https://github.com/nodelogicau/intentions/issues/1))

**The draft says:** projection values are normalised with "ISO 8601 durations
with no zero components", and the scenario *Normalisation before hashing*
requires `PT1H` and `PT60M` to hash identically.

**The problem:** dropping zero components does not make `PT1H` and `PT60M`
agree. Some conversion rule is needed, and the obvious one (total seconds)
would also make `P1D` and `PT24H` agree, which is wrong on a day with a
timezone transition.

**We decided:** drop zero components; fold weeks into days (`P1W` becomes
`P7D`, since a week is exactly seven days); re-express the time part from total
seconds as hours, minutes and seconds (`PT90M` becomes `PT1H30M`, `PT60M`
becomes `PT1H`); never convert between the date part and the time part, so
`P1D` and `PT24H` stay distinct, and never convert months or years.

**Proposed text:** state these rules in the Versioning section.

**Resolution (b3bb420):** Settled as proposed: zero components dropped, weeks folded into days, the time part re-expressed from total seconds as hours, minutes and seconds, no conversion between the date and time parts, months and years untouched. Stated as a list under Versioning.

## 2. Set-valued list ordering in the projection ([#2](https://github.com/nodelogicau/intentions/issues/2))

**The draft says:** "reference lists sorted by id".

**The problem:** `location`, `conditional`, `parties` on an intention, and
`displaced` are sets too. Two writers that order them differently would hash
differently for identical state.

**We decided:** sort every set-valued list in the projection: references by id
then role, commitment parties by uri, strings lexically. The on-disk writer
sorts them too, so files are byte-identical across writers.

**Proposed text:** "set-valued lists sorted" in place of "reference lists
sorted by id", naming the lists.

**Resolution (b3bb420):** Settled as proposed. Every set-valued list is sorted and the lists are named: references by id then role, commitment parties by uri, and location, conditional, intention parties and displaced lexically. The on-disk writer sorts them too.

## 3. Position of the cached `version` in canonical order ([#3](https://github.com/nodelogicau/intentions/issues/3))

**The draft says:** "A tool may cache the version in the file under `version`;
the computed value is authoritative", and "fields an implementation adds beyond
this specification are written after all specified fields".

**The problem:** `version` is mentioned by the specification but absent from
every canonical field list, so by the letter it goes last, after `retired`.
Two writers that both cache it and place it differently are not byte-identical.

**We decided:** always write it, because a diff then shows whether an edit
changed the projection, which is what acknowledgement lapse hinges on; and
place it last, per the letter.

**Proposed text:** give `version` a canonical position. We suggest immediately
after `id`, where a reviewer reading a diff sees it first.

**Resolution (b3bb420):** Settled: `version` is now always written, immediately after `id`, on every object and on RESOLUTION records, as suggested. Optional caching defeated byte-identical files, and the second line of a diff is where a reviewer looks. v0.2.0 writes it second; a missing `version` is a validation warning.

## 4. The seed an RRULE cadence expands from ([#4](https://github.com/nodelogicau/intentions/issues/4))

**The draft says:** a cadence is an RRULE using only date-level parts.

**The problem:** an RRULE expands from a DTSTART, which the format never
names. `FREQ=WEEKLY;BYDAY=TU` is unambiguous, but `FREQ=WEEKLY` alone takes its
weekday from DTSTART, and `FREQ=MONTHLY` without `BYMONTHDAY` takes its day.

**We decided:** the seed is the first local day of the window's calendar
anchor; when the lower bound is open (`../X`) the seed is the start of the
expansion horizon the caller supplies. `WKST` defaults to the resolver's
`week_start` unless the rule states one.

**Proposed text:** state the seed in the Cadence section.

**Resolution (b3bb420):** Settled as proposed: the seed is the first local day of the calendar anchor's lower bound, or the horizon start when that bound is open. Added: an object with `cadence` must have a calendar anchor, which validation checks. `WKST` defaults to `MO` since `week_start` is gone (see 5).

## 5. Week granules under a non-Monday `week_start` ([#5](https://github.com/nodelogicau/intentions/issues/5))

**The draft says:** `resolver.week_start` is part of the context a window's
bounds are computed from, and `2026-W36` is an ISO week.

**The problem:** ISO week identifiers are Monday-based by definition. It is
not clear what `2026-W36` means to a resolver whose week starts on Sunday.

**We decided:** `YYYY-Www` spans the seven local days starting on the
`week_start` day on or before that ISO week's Monday. With `monday` it is the
ISO week exactly; with `sunday` it is the Sunday-to-Saturday week containing
that Monday. `this-week` resolves to the ISO week of today either way.

**Proposed text:** either state this rule, or drop `week_start` and let weeks
be ISO weeks.

**Resolution (b3bb420):** **Decided differently:** `week_start` is removed and weeks are ISO weeks in every context. The proposed rule contradicted its own deixis rule on a Sunday, where `this-week` resolves to the ISO week of today but the shifted W36 bounds ended the day before. A week identifier that lies about the ISO week it names seemed worse than no preference. A stale `week_start` key is ignored and reported at info level. v0.2.0 drops the flag and the key.

## 6. Season codes need a hemisphere ([#6](https://github.com/nodelogicau/intentions/issues/6))

**The draft says:** `2026-21` to `2026-24` are spring, summer, autumn, winter,
and the example resolver is in Melbourne.

**The problem:** EDTF's season codes name seasons without months. Spring is
March in Europe and September in Melbourne.

**We decided:** an optional `resolver.hemisphere` in `intentions.yaml`, `north`
(the default) or `south`. North maps spring to March–May; south maps it to
September–November. A winter or summer that begins in December starts in the
granule's year and runs into the next.

**Proposed text:** add `resolver.hemisphere` to the workspace configuration,
and state the month mapping.

**Resolution (b3bb420):** **Partly differently:** EDTF's own hemisphere-specific season codes `25` to `32` are now admitted, from the same extended set the quarter codes come from, so a season can be unambiguous in the file. The neutral `21` to `24` stay admitted and resolve through `resolver.hemisphere`, default north, with the proposed month mapping and December seasons running into the next year. v0.2.0 admits all twelve codes.

## 7. Explicit `transparent: false` in the projection ([#7](https://github.com/nodelogicau/intentions/issues/7))

**The draft says:** `transparent` is optional and "absent means false", and it
is in the commitment's projection.

**The problem:** a writer that emits `transparent: false` explicitly (the
draft's own example does) would hash differently from one that omits it, for
identical state.

**We decided:** `transparent` enters the projection only when true.

**Proposed text:** say so under Versioning.

**Resolution (b3bb420):** Settled as proposed: a boolean equal to its documented default leaves the projection, and the writer omits it. The example that wrote `transparent: false` is fixed upstream; v0.2.0 omits it on write.

## 8. Where a policy-authorised firming is recorded ([#8](https://github.com/nodelogicau/intentions/issues/8))

**The draft says:** `firm` may be set by a harness "acting under a standing
intention of the subject whose `auto_firm` condition the intention satisfies,
naming that policy on the act."

**The problem:** a resolution record has `selector` to name the policy. A
firming is an edit to `stability` and the intention has no field to carry the
policy. The act names it only in the command's result and in git history.

**We decided:** the command result carries `policy: <int_id>`; nothing is
written to the file.

**Proposed text:** either a field on the intention (`firmed_under`) or an
embedded record, so the file itself shows the policy.

**Resolution (b3bb420):** **Decided with a field rather than a record:** `firmed_under` on the intention names the policy, sits after `stability` and outside the projection like `source`, and is required whenever a firm intention's source carries a harness. A firming is an edit to stability, not a record, so a field is the format's own pattern for it. v0.2.0 writes it on every harness firming and clears it on a person's own act.

## 9. "Standing intention" names two things ([#9](https://github.com/nodelogicau/intentions/issues/9))

**The draft says:** "An intention with a `cadence` generates instances" is a
standing intention, and a policy is "a standing intention (a terminus or
policy)" carrying `auto_select` or `auto_firm`.

**The problem:** a terminus has no window and a cadence needs one to expand
within; a policy holder need not recur. The two uses of "standing" pick out
different objects.

**We decided:** `--policy` may name any active intention of the subject that
carries `auto_firm`, cadence or not. Instance generation applies to intentions
with a cadence only.

**Proposed text:** distinct terms, for example "recurring intention" for the
cadenced kind and "policy" for the condition holder.

**Resolution (b3bb420):** **Decided with distinct terms, narrower than this implementation's reading:** an intention with `cadence` is a recurring intention; a terminus is an intention with no window, duration or outbound references; a policy is a terminus carrying `auto_select` or `auto_firm`. Conditions are admitted on termini only, because a policy is a for-the-sake-of sink and a scheduled intention should not be able to authorise a firming. v0.2.0 accepts only termini as `--policy`, refuses a condition elsewhere, and renames `list --standing` to `--recurring`.

## 10. Deictic vocabulary a writer accepts ([#10](https://github.com/nodelogicau/intentions/issues/10))

**The draft says:** "Deixis is resolved at write time".

**We decided:** the writer accepts `today`, `tomorrow`, `this-week`,
`next-week`, `this-month`, `next-month`, `this-quarter`, `next-quarter`,
`this-year`, resolved at the current instant in the resolver's timezone.

**Proposed text:** informative; a list of terms writers should accept helps
harnesses behave alike.

**Resolution (b3bb420):** Settled: the deixis paragraph now lists `today`, `tomorrow`, `this-week`, `next-week`, `this-month`, `next-month`, `this-quarter`, `next-quarter` and `this-year`, resolved in the resolver's timezone, plus a note on `this-week` on a Sunday.

## 11. `validate` cannot tell a policy-authorised harness firming from an unauthorised one ([#11](https://github.com/nodelogicau/intentions/issues/11))

**The draft says:** validation reports as an error "`firm` set by a harness
without a policy".

**The problem:** because of item 8 the file does not carry the policy, so
validation cannot distinguish the authorised case. The check can only be
write-time.

**We decided:** `validate` reports a warning, not an error, for an intention
with `stability: firm` whose `source` carries a harness. The write path
refuses the unauthorised case.

**Proposed text:** resolve item 8, after which this becomes an error again.

**Resolution (b3bb420):** Settled via 8: with `firmed_under` in the file, `firm` with a harness source and no `firmed_under` is an error again, and `firmed_under` naming anything but an active policy of the subject is also an error. v0.2.0 reports both.

## 12. The canonical form of a zero duration ([#12](https://github.com/nodelogicau/intentions/issues/12))

**The draft says:** nothing; its example writes `gap: {min: P0D, max: P3D}`.

**The problem:** ISO 8601 admits `P0D`, `PT0S` and others for zero. A writer
must pick one to be byte-stable, and the projection must pick one to hash
stably.

**We decided:** `P0D`, as the example writes it, on disk and in the projection.

**Proposed text:** state it under Values.

**Resolution (b3bb420):** Settled as proposed: `P0D` is the one form of zero, on disk and in the projection, stated under Values and in the normalisation list.

## 13. The example objects disagree on list style ([#13](https://github.com/nodelogicau/intentions/issues/13))

**The draft says:** the availability example writes
`conditional: [deep-work, writing]` in flow style and `location:` as a block
sequence; the intention example writes `serves` entries in flow style and
`location` as a block sequence.

**The problem:** canonical order "is what makes two implementations produce
byte-identical files", but the examples do not fix a style for lists.

**We decided:** string lists (`location`, `conditional`, `parties` on an
intention, `displaced`) are block sequences; small records (`serves` entries,
ranged durations, `gap`, policy conditions) are flow mappings; everything else
is block style. Multi-line prose is a literal block scalar.

**Proposed text:** fix a list style in the Field order section and align the
examples.

**Resolution (b3bb420):** Settled as proposed: block sequences for string lists, flow mappings for small records, literal block scalars for multi-line prose. Stated under Field order; the availability example's `conditional` is now a block sequence.

## 14. Resolution needs a bounded range: horizon and now ([#14](https://github.com/nodelogicau/intentions/issues/14))

**The draft says:** resolution takes an intention with a window and produces
candidate placements; the deixis rule says deixis is resolved at write time
and bounds at resolution time.

**The problem:** a window such as `../2026-12` or `2026-09/..` has an open
side, and even a closed window that runs for months would enumerate
candidates nobody wants. Nothing says how far ahead resolution looks or
whether a candidate in the past may be offered.

**We decided:** the resolution range starts at the later of the window's
start and now, and ends at the earlier of the window's end and now plus
`resolver.horizon` (a new optional key, default `P4W`). An open side
contributes nothing. A window that has passed, or starts beyond the horizon,
yields no candidates and a reason. Candidates before now are never offered.

**Proposed text:** state the range rule under Resolution and add
`resolver.horizon` to the workspace configuration.

## 15. Capacity depletes per occasion ([#15](https://github.com/nodelogicau/intentions/issues/15))

**The draft says:** an availability's `duration` is "capacity offered per
occasion" and "may be shorter than the window's clock interval".

**The problem:** whether that capacity is consumed as placements land on the
occasion is not stated. Read as a length limit only, two two-hour intentions
may both be placed on one three-hour morning.

**We decided:** an occasion offers its nominal duration (the max when
ranged); a candidate may not exceed it, and the opaque unretired placements
already resting on the occasion plus the candidate may not exceed it. Two
two-hour intentions cannot both land on one three-hour Tuesday morning.

**Proposed text:** say under Availability that capacity is consumed by the
placements resting on an occasion.

## 16. What a placement rests on is recomputed; propose `supply` on the record ([#16](https://github.com/nodelogicau/intentions/issues/16))

**The draft says:** `expired-ground` is "an availability the placement rests
on has expired or been retired", and `condition-mismatch` and
`location-mismatch` speak of "the supplying availability".

**The problem:** nothing records which availability a placement rests on.
The RESOLUTION record carries the placement and what it displaced, not what
supplied it.

**We decided:** at check time a placement rests on every occasion of each
involved particular that contains it. No containing occasion is a
`window-clash` without counterpart; containing occasions that all fail are
reported by their failure (retired or expired as `expired-ground`,
conditional as `condition-mismatch`, location as `location-mismatch`). The
RESOLUTION record gains an implementation-added `supply` list naming the
availabilities the placement was chosen against, written after `timestamp`
and outside the projection.

**Proposed text:** add `supply` to the RESOLUTION record, and state the
"rests on" rule under Consistency.

## 17. Candidate enumeration on a step grid ([#17](https://github.com/nodelogicau/intentions/issues/17))

**The draft says:** resolution "produces a ranked set of candidate
placements".

**The problem:** a window of a week with a three-hour clock interval admits
infinitely many starts. Something must make the set finite, and two
implementations should produce the same set.

**We decided:** candidate starts lie on a grid of `resolver.step` (a new
optional key, default `PT15M`) aligned to each supply interval's start, with
the candidate spanning the intention's nominal duration. A ranged duration
tries the nominal only. The full ranked set is computed;
`candidates_considered` records its size and a caller limits what it sees.

**Proposed text:** state the grid under Resolution and add `resolver.step`.

## 18. Relational anchor bounds per relation ([#18](https://github.com/nodelogicau/intentions/issues/18))

**The draft says:** a relational anchor names a target, one of the four RFC
9253 relations, and an optional `gap` of `{min, max}`; the dependent window
holds the reference.

**The problem:** what the relation constrains on the candidate, and what an
absent `gap` or an absent `max` means, is not stated.

**We decided:** `FINISHTOSTART` puts the candidate's start in
`[target end + min, target end + max]`; `STARTTOSTART` the start in
`[target start + min, target start + max]`; `FINISHTOFINISH` the end in
`[target end + min, target end + max]`; `STARTTOFINISH` the end in
`[target start + min, target start + max]`. An absent gap is a minimum of
zero and no maximum; an absent `max` alone is unbounded above. A retired or
cancelled target counts as unplaced, so the intention is blocked.

**Proposed text:** a table under WINDOW's relational anchor.

## 19. `intention-inconsistency` is pairwise ([#19](https://github.com/nodelogicau/intentions/issues/19))

**The draft says:** the flag means "two active intentions cannot both be
placed within their windows given available supply".

**The problem:** taken globally this is a feasibility problem; taken
pairwise it is tractable. The text does not say which.

**We decided:** pairwise, over active unplaced intentions of one subject
with both duration and window whose ranges intersect: each has candidates
alone, but no pair of candidates exists that do not overlap. The flag's
detail says "pairwise". Anything global is out of scope.

**Proposed text:** say "pairwise" in the flag's definition.

## 20. The resolver's scope comes from `resolver.scope` ([#20](https://github.com/nodelogicau/intentions/issues/20))

**The draft says:** "only availability with scope at or wider than a
resolver's own scope shall be visible to it".

**The problem:** nothing in `intentions.yaml` or on the resolver says what
its own scope is.

**We decided:** a new optional key `resolver.scope`, default `personal`,
overridable per call with `--scope`. In a personal workspace the default sees
everything, which is what the draft's example implies.

**Proposed text:** add `resolver.scope` to the workspace configuration.

## 21. Instances copy `location` and the recurring intention's clock ([#21](https://github.com/nodelogicau/intentions/issues/21))

**The draft says:** an instance carries "a window derived from that
occurrence, and the recurring intention's subject, duration, activity, and
parties unless overridden".

**The problem:** `location` is not in the copied list, and "derived from
that occurrence" does not say whether the recurring intention's clock anchor
survives. A place constraint that vanished on generation would surprise, and
a Tuesday-mornings arrangement whose instances lost their mornings would too.

**We decided:** an instance's window is `calendar: <occurrence day>` plus
the recurring intention's `clock` when it has one, and `location` is copied
with the rest.

**Proposed text:** add `location` and the clock to the instance rule.

## 22. Re-resolving a placed intention ([#22](https://github.com/nodelogicau/intentions/issues/22))

**The draft says:** resolution takes an "unplaced" intention, and a
displaced intention "carries a window-clash flag until acknowledged or
re-resolved".

**The problem:** "re-resolved" has no rule: a placed intention cannot be
resolved, and nothing clears a placement except cancelling a commitment.

**We decided:** `select --replace` clears the previous placement and writes
the new one in the same act, with a new RESOLUTION record; the previous
record is left as it is. Without `--replace`, selecting a placed intention is
refused.

**Proposed text:** say under Resolution that a new selection may replace a
placement, and that each selection is its own record.

## 23. A reference tool set for harnesses ([#23](https://github.com/nodelogicau/intentions/issues/23))

**The draft says:** nothing about how a harness reaches an implementation
programmatically. The verbs are described as operations without naming a
tool set, and every harness that is not a shell has to invent one.

**The problem:** DKF names its tools so two implementations expose the same
surface to a model and the discipline transfers between them. The Intentions
Format does not, so another MCP server for intentions would name and shape its
tools differently, and a skill written against one would not work against the
other.

**We decided:** one MCP tool per operation, named after the verb. As of
v0.8.0 the set is twenty-five: six `intention_*`, five `availability_*`, five
`commitment_*`, `generate`, `resolve`, `select`, `unresolved`, `check`,
`acknowledge`, `bounds`, `validate` and `workspace_status`. Inputs are typed
after the format's own structures: a window is `{calendar, clock, relative}`,
a duration a string or `{nominal, min, max}`, `serves` a list of `{id, role}`,
and `select` takes a person's `candidate` or a harness's `policy`. Results
equal the CLI's `--json`. The server tells the model these names are the
implementation's, not the format's. See [docs/mcp.md](docs/mcp.md#tools).

Three carry semantics a second implementation would have to keep for a skill
to transfer: `select` (a candidate or a policy, never both, while `resolve`
chooses nothing), `intention_firm` (a harness needs a policy), and the two
answering tools (no policy exists at all, because a party's status is a fact
about their will). `unresolved` is the one tool with no verb in the draft.

**Proposed text:** a non-normative "Reference tool set" appendix listing
these names and parameters, with the status DKF gives its tool names: an
implementation MAY expose them under other names, but one that exposes these
SHOULD keep their semantics, so that skills and prompts transfer.

## 24. What a declined party means ([#24](https://github.com/nodelogicau/intentions/issues/24))

**The draft says:** a party's status is one of `tentative`, `accepted`,
`declined`, changed only by that party's own act; cancellation is a separate
`retired` record that clears the intention's placement.

**The problem:** nothing says what `declined` means for the rest of the
model. An unretired commitment occupies its parties' time, so a declined one
keeps blocking the person's calendar and costing later resolutions a
displacement, for a thing they have refused. Its intention also stays placed,
so it is no longer resolvable, though the person has decided this placement
will not happen. And the subject declining is not the same situation as a
counterparty declining, though the field records them identically.

**We decided:** v0.8.0 records the answer and changes nothing else. A declined
commitment still occupies its parties' time and its intention stays placed;
only `cancel` frees either. That never frees time the other parties still
believe is booked, at the cost of an explicit cancellation.

**Proposed text:** state under Commitment what `declined` entails: either a
declined party's own time is free while the commitment stands, said also where
Resolution defines occupancy and naming whose decline frees whose time, or
declining is purely a record and cancellation is the only act that frees
anything, in which case say a decline by the subject SHOULD prompt one.
