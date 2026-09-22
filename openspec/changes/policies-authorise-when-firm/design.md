## Context

Every policy check in this binary goes through `model.LookupPolicy`: `CheckFirm` for the firming boundary, `resolve.Select` for `--policy`, and validation twice, for `firmed_under` on an intention and for `selector` on a resolution record. It checks existence, retirement, subject, terminus-ness and the presence of a condition, and nothing about stability. Adding firmness there covers three of the four callers correctly. The fourth, the record check, must not inherit it: a record is history, and the spec now says a record naming a policy since withdrawn, tentative or retired, is a warning, where today a retired one is an error through the same lookup.

## Goals / Non-Goals

**Goals:**
- One firmness rule, applied where the policy is looked up, so every write path and the live-object validation agree.
- Refusals and errors that name the way out: the command that firms the draft, or the person's re-firm.
- Records keep their history: a withdrawn policy on a record warns and the placement stands.

**Non-Goals:**
- Any change to what a policy is, to its condition terms, or to the tool set.
- Re-checking a policy's condition after the fact; that stays write-time only, as the spec says.

## Decisions

### D1. Firmness goes into `LookupPolicy`, after the existing checks
`LookupPolicy` refuses a tentative policy with rule `policy_draft` and a message naming the draft and `intentions intention firm <id>` run by the person. Ordering after the terminus and condition checks means a scheduled or condition-less intention is still refused for what it is, not for being tentative. `CheckFirm`, `Select` and the `firmed_under` validation check inherit the refusal with no edits beyond messages.

Alternative considered: a check in each caller. Rejected as the drift the upstream review warned about, twice over.

### D2. The record check gets its own two-part lookup
`LookupPolicy` splits internally: `policyOf(g, id, subject)` finds a policy of the subject regardless of its state, and `LookupPolicy` adds the liveness checks (active, firm). The record check calls `policyOf` and then errors when the object is not a policy of the subject or carries no `auto_select`, and warns with code `selector_withdrawn` when it is retired or tentative. The message says the act was authorised under a policy since withdrawn and the placement is the person's to keep or re-resolve.

### D3. The `firmed_under` error names the two ways out
The message for a tentative or retired `firmed_under` target says: re-firm by your own act (`intentions intention firm <id>` with an author and no harness), or set the intention tentative. The code stays `firmed_under_target`; the message carries the fix, as the unserved warning does.

### D4. Suspend versus end is prose
Setting a policy tentative and retiring it already produce the same findings through the lookup. The skill and README state the two words and that either withdraws what rested on it. No code distinguishes them.

## Risks / Trade-offs

- [Existing tests use tentative policies to firm and select] → They fail on purpose and are fixed by the person firming the policy first, which documents the rule in every test that exercises it.
- [A record's retired-selector case drops from error to warning] → Intended by the spec: history is not re-litigated. CI that gated on that error keeps passing, since warnings exit zero.
- [A person un-firms a policy and every intention under it errors] → That is the withdrawal upstream chose, and each finding names the two ways out.

## Migration Plan

Code, tests, skill, docs, CHANGELOG 0.13.0, SPEC-FEEDBACK item 27 resolution, release. No file changes shape.
