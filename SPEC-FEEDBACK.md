# Feedback on the Intentions Format draft from the `intentions` reference implementation

The [Intentions Format](https://github.com/nodelogicau/intentions) README says
that no implementation exists yet and that the first will shape the text as the
DKF reference implementation shaped its specification. This file is that
record: the points where implementing the draft forced a decision the text did
not make, what this implementation decided, and how each was resolved upstream
once it was. "We" below means this implementation as it was when the item was
raised.

Items 1 to 13 were raised from the `bootstrap-intentions-cli` change, which
implements the object model, the temporal engine, validation and the index and
stops before resolution.

| # | Topic | Status |
|---|---|---|
| 1 | Duration normalisation in the projection | open |
| 2 | Set-valued list ordering in the projection | open |
| 3 | Position of the cached `version` in canonical order | open |
| 4 | The seed an RRULE cadence expands from | open |
| 5 | Week granules under a non-Monday `week_start` | open |
| 6 | Season codes need a hemisphere | open |
| 7 | Explicit `transparent: false` in the projection | open |
| 8 | Where a policy-authorised firming is recorded | open |
| 9 | "Standing intention" names two things | open |
| 10 | Deictic vocabulary a writer accepts | open |
| 11 | `validate` cannot tell a policy-authorised harness firming from an unauthorised one | open |
| 12 | The canonical form of a zero duration | open |
| 13 | The example objects disagree on list style | open |

---

## 1. Duration normalisation in the projection

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

## 2. Set-valued list ordering in the projection

**The draft says:** "reference lists sorted by id".

**The problem:** `location`, `conditional`, `parties` on an intention, and
`displaced` are sets too. Two writers that order them differently would hash
differently for identical state.

**We decided:** sort every set-valued list in the projection: references by id
then role, commitment parties by uri, strings lexically. The on-disk writer
sorts them too, so files are byte-identical across writers.

**Proposed text:** "set-valued lists sorted" in place of "reference lists
sorted by id", naming the lists.

## 3. Position of the cached `version` in canonical order

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

## 4. The seed an RRULE cadence expands from

**The draft says:** a cadence is an RRULE using only date-level parts.

**The problem:** an RRULE expands from a DTSTART, which the format never
names. `FREQ=WEEKLY;BYDAY=TU` is unambiguous, but `FREQ=WEEKLY` alone takes its
weekday from DTSTART, and `FREQ=MONTHLY` without `BYMONTHDAY` takes its day.

**We decided:** the seed is the first local day of the window's calendar
anchor; when the lower bound is open (`../X`) the seed is the start of the
expansion horizon the caller supplies. `WKST` defaults to the resolver's
`week_start` unless the rule states one.

**Proposed text:** state the seed in the Cadence section.

## 5. Week granules under a non-Monday `week_start`

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

## 6. Season codes need a hemisphere

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

## 7. Explicit `transparent: false` in the projection

**The draft says:** `transparent` is optional and "absent means false", and it
is in the commitment's projection.

**The problem:** a writer that emits `transparent: false` explicitly (the
draft's own example does) would hash differently from one that omits it, for
identical state.

**We decided:** `transparent` enters the projection only when true.

**Proposed text:** say so under Versioning.

## 8. Where a policy-authorised firming is recorded

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

## 9. "Standing intention" names two things

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

## 10. Deictic vocabulary a writer accepts

**The draft says:** "Deixis is resolved at write time".

**We decided:** the writer accepts `today`, `tomorrow`, `this-week`,
`next-week`, `this-month`, `next-month`, `this-quarter`, `next-quarter`,
`this-year`, resolved at the current instant in the resolver's timezone.

**Proposed text:** informative; a list of terms writers should accept helps
harnesses behave alike.

## 11. `validate` cannot tell a policy-authorised harness firming from an unauthorised one

**The draft says:** validation reports as an error "`firm` set by a harness
without a policy".

**The problem:** because of item 8 the file does not carry the policy, so
validation cannot distinguish the authorised case. The check can only be
write-time.

**We decided:** `validate` reports a warning, not an error, for an intention
with `stability: firm` whose `source` carries a harness. The write path
refuses the unauthorised case.

**Proposed text:** resolve item 8, after which this becomes an error again.

## 12. The canonical form of a zero duration

**The draft says:** nothing; its example writes `gap: {min: P0D, max: P3D}`.

**The problem:** ISO 8601 admits `P0D`, `PT0S` and others for zero. A writer
must pick one to be byte-stable, and the projection must pick one to hash
stably.

**We decided:** `P0D`, as the example writes it, on disk and in the projection.

**Proposed text:** state it under Values.

## 13. The example objects disagree on list style

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
