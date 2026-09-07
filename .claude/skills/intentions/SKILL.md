---
name: intentions
description: Record what a person means to do with their time as intentions, availability and commitments in an Intentions Format workspace using the `intentions` CLI. Use when the user says what they intend to do, when, or for how long; when they describe when they have capacity; when they ask you to plan, firm, retire or review their intentions; or before you draft anything about their time.
license: MIT
compatibility: Requires the `intentions` binary on PATH and an Intentions workspace (a directory containing intentions.yaml). Shell access required.
metadata:
  author: nodelogicau
  version: "dev"
  format: intentions/0.1
---
<!-- installed by intentions dev; regenerate with: intentions skill install -->

You are drafting a person's plans, which they review through git pull requests.
`intentions` stores what they mean to do as YAML files; you draft, it keeps the
format honest. An intention is **a duration and a window, not a slot**: ninety
minutes on the budget, sometime this week, before the board pack goes out. Never
invent a start time; the window is the person's, and clock time arrives only
when `resolve` ranks candidates and the person selects one.

Always pass `--json` and parse the result. Never prompt, never edit the YAML
files by hand, never delete a file, never touch `index.yaml`.

## The loop

Before you write anything about the person's time:

```
intentions workspace --json                  # which workspace, found how (exit 5 = none)
intentions intention list --json             # what they already mean to do
intentions availability list --json          # what capacity they have declared
intentions unresolved --json                 # what still needs a placement, and why
intentions validate --json                   # is the workspace sound (exit 4 = errors)
      │
      ▼  reason about it yourself
      │
intentions intention add …          # a new intention, tentative
intentions availability add …       # capacity the person told you about
intentions intention edit …         # fill the plan in as it becomes definite
intentions resolve <id> --json      # ranked candidates; nothing is chosen
      │
      ▼  put the candidates to the person
      │
intentions select <id> --candidate N   # their choice, recorded as the act
intentions check --json             # what fails to hang together, as flags
intentions validate --json          # check again before you hand over
```

List before you add. A second intention for the same thing is noise for the
reviewer; edit the one that exists. When the person's plan becomes more
definite, narrow the window on the same intention rather than writing a new one.

## Setup

Find the workspace. Precedence is **`--workspace <dir>`, then
`$INTENTIONS_WORKSPACE`, then the nearest ancestor directory containing
`intentions.yaml`** (or a `.intentions` pointer at the same level). Run
`intentions workspace --json` when it matters: `found_by` tells you which won. A
`$INTENTIONS_WORKSPACE` pointing at a directory with no `intentions.yaml` is an
error, not a fallback to the search.

If there is no workspace and the user wants one:

```sh
intentions init ./planning --author <user URI> --subject <user URI> \
  --timezone <IANA zone> [--hemisphere south] [--pointer] --json
```

`--pointer` writes `./.intentions` naming the new workspace, so every later
call finds it from anywhere in the repository. For a workspace that already
exists, `intentions workspace pointer ./planning --json` does the same.

Set attribution once per session rather than per call:

```sh
export INTENTIONS_HARNESS=claude INTENTIONS_MODEL=<your model id>
# INTENTIONS_AUTHOR defaults to intentions.yaml defaults.source.author — the human you work for
```

If this skill is not yet installed for your harness, `intentions skill install`
does it (`--harness copilot|agents|cursor|agents-md` for others; `skill --help`).
A harness with no shell uses `intentions serve --mcp` instead: the same
operations as MCP tools, the same files, the same refusals.

**In zsh, brace every id you follow with a colon:** write `--serves "${id}:in-order-to"`
and `--relative "${id}:FINISHTOSTART"`, not `"$id:…"`. zsh applies history
modifiers inside double quotes, so `"$id:in-order-to"` silently corrupts the id.

## Verbs

| Do | Command |
|---|---|
| Where am I | `intentions workspace --json` → `{root, found_by, config}` |
| What is intended | `intentions intention list [--subject <uri>] [--activity <term>] [--stability tentative\|firm] [--recurring] [--retired] --json` → `{intentions, count}` |
| What capacity exists | `intentions availability list [--subject <uri>] [--conditional <term>] [--scope <s>] [--retired] --json` |
| One object | `intentions show <id> --json` → `{object, version, path}`; `intention show` adds `serves_resolved` |
| Record an intention | `intentions intention add --title "<what>" [--duration PT90M] [--calendar <edtf\|deictic>] [--clock HH:MM/HH:MM] [--relative <id>:<RELATION>[:min[:max]]] [--activity <term>] [--location <uri>]... [--party <uri>]... [--serves <id>:in-order-to\|for-the-sake-of]... [--description "<prose>"] --json` |
| Multi-line prose | add `--description-file -` and pipe the text on stdin |
| Make it recurring | add `--cadence "FREQ=WEEKLY;BYDAY=TU"` with a `--calendar` anchor for it to expand within |
| Fill the plan in | `intentions intention edit <id> [same flags] [--clear-<field>] --json` → `{previous_version, version, projection_changed}` |
| Firm (person's act) | `intentions intention firm <id> --json` — only when the person says so |
| Firm (under a policy) | `intentions intention firm <id> --policy <terminus id> --json` — the only way a harness may firm; refused otherwise |
| Record capacity | `intentions availability add --subject <uri> --duration PT3H --calendar <edtf> [--clock HH:MM/HH:MM] [--conditional <term>]... [--location <uri>]... [--cadence <rrule>] [--valid-until <edtf>] [--scope personal\|organisation\|public] --json` |
| Extend capacity | `intentions availability renew <id> --valid-until <edtf> --json` |
| Change capacity terms | `intentions availability supersede <id> [changed flags] --json` — never `edit` for window, duration, conditional or location |
| Withdraw | `intentions intention retire <id> --kind fulfilled\|abandoned\|superseded [--superseded-by <id>] --reason "<why>" --json`; `intentions availability retire <id> --kind retracted\|superseded …` |
| What a window means | `intentions bounds <id> --json` or `intentions bounds --calendar this-week --clock 09:00/12:00 --json`; writes nothing |
| Generate instances | `intentions generate --horizon <P…> [--recurring <id>] --json` → `{created, skipped}`; idempotent. Only when the person asks to plan a named period, and narrow it to that period |
| Rank placements | `intentions resolve <id> [--limit N] --json` → `{candidates: [{rank, start, end, displaces, supply}], candidates_considered, reason}`; writes nothing but instances |
| Record the choice | `intentions select <id> --candidate N --json` → `{resolution, intention, commitment?, flags}`; `--replace` to move a placed one |
| Select under a policy | `intentions select <id> --policy <terminus id> --json` — harness only, top rank-1 candidate only; refused otherwise |
| What awaits placement | `intentions unresolved [--subject <uri>] --json` → `{entries: [{id, title, status, reason?, blocked_on?, candidates, best_rank?, deadline?}], count, counts}`; `ready` means candidates exist, `blocked` names the target to place first, `incomplete` needs an edit |
| Answer a commitment | `intentions commitment accept\|decline <id> [--party <uri>] --json` — only when the person has said so; no policy authorises it |
| Cancel a commitment | `intentions commitment cancel <id> [--reason "<why>"] --json` → the intention it was for loses its placement and may be resolved again |
| What is outstanding | `intentions commitment list [--party <uri>] [--status tentative] --json` |
| What clashes | `intentions check [<id>]... --json` → `{flags: [{kind, subject, counterpart, counterpart_version, detail}]}`; seven kinds, never decisions |
| Proceed anyway | `intentions acknowledge <id> --kind <kind> --counterpart <id> --reason "<why>" --json` — only on the person's word; lapses when the counterpart changes |
| Health check | `intentions validate --json`; `intentions index --check --json` |

Exit codes: `0` ok · `1` runtime · `2` usage or a write the format refuses · `3` not found · `4` check failed · `5` no workspace.
On failure stderr carries `{"error": {"code", "message"}}`; a refused write says which rule.

## Rules for intentions

- **Duration and window, never a slot.** Ask what the thing is for, how long
  it needs, and within what bounds. `--calendar` takes what the person said:
  `this-week`, `next-month`, `2026-W37`, `2026-09`, `../2026-09` (by the end
  of September), a quarter `2026-35`, a season `2026-21` (neutral) or
  `2026-29` (Southern). Deictic terms are resolved to the named granule when
  you write, so the file means the same week a month later. `--clock` is time
  of day and lives nowhere else. Never write a placement by hand; `select`
  writes it, and only after `resolve` has ranked the candidates.
- **Say what it is for.** `--serves <id>:in-order-to` links a means to an
  end; `--serves <id>:for-the-sake-of` links to a terminus, an intention with
  no window, no duration and no serves of its own, such as *being someone who
  follows through*. The graph is a DAG; a write that would close a cycle is
  refused. A terminus is a sink and cannot gain serves.
- **Draft tentative. Firm is the person's.** Every intention you add is
  `tentative`. Setting `firm` with a harness in the source is refused unless
  `--policy` names a terminus of the subject carrying `auto_firm` whose terms
  the intention satisfies; the tool then writes `firmed_under` so the file
  shows what authorised it. Never firm because the person sounds sure; firm
  because they said to, or because a policy they hold covers it.
- **Activity terms are lowercase kebab-case** and matched exactly against an
  availability's `conditional`. Reuse terms already in the workspace
  (`intentions.md` lists them; `validate` reports a term used once at info
  level). Do not invent near-synonyms.
- **Locations and parties are URIs.** A place is whatever URI the person
  uses for it, matched by string equality; a `geo:` URI is fine. A party is
  another particular whose availability must be satisfied. A room that must
  be booked and is also where the thing happens goes in both.
- **Subject defaults to the workspace's person.** Pass `--subject` only for
  someone or something else. An organisation workspace has no default and
  requires it every time.
- **Edit in place; never write a replacement.** A window narrowing, a duration
  settling, a title fix: `intention edit`. Prose edits leave the version
  unchanged; the result's `projection_changed` tells you whether the
  scheduling projection moved.

## Rules for resolution

- **Resolve, then ask.** `resolve` ranks; it does not place. Put the
  candidates to the person in their words (Tuesday at nine, for ninety
  minutes) and record what they chose with `select --candidate N`. The record
  says `selector: person` because it was.
- **A policy is the only way you select alone.** `select --policy` works
  when the person holds a terminus with `auto_select` covering the intention
  and a rank-1 candidate exists. It is refused otherwise, and refusal is not
  an invitation to pick a candidate yourself.
- **Displacement is a flag, not a fix.** A rank-2 or rank-3 candidate
  overlaps something. If the person chooses it anyway, the record lists what
  was displaced, `check` shows `window-clash` on both, and nothing else
  moves. Do not retire or re-place the displaced object to make the flag go
  away; tell the person.
- **Acknowledge only what the person has seen.** `acknowledge` records that
  they looked at a specific flag against a specific state of its counterpart.
  It lapses when the counterpart's projection changes, so it is never a way to
  silence a flag for good.
- **Do not materialise a recurrence in advance.** Creating a recurring
  intention writes one file and no instances, which is right: an instance is
  something intended on a particular day, and it should appear when that day
  is being planned. `resolve` materialises what its own range needs, so
  resolving is usually enough. Reach for `generate` only when the person asks
  to plan a named period, and pass `--horizon` and `--recurring` so you
  materialise that period and nothing more. A workspace full of instances
  nobody asked for is a worse review for the person.
- **No supply is an answer, not an obstacle.** When `resolve` reports no
  candidates, read the reason and tell the person: the window has passed, it
  lies beyond the horizon, a target it waits on is unplaced, or they have
  offered no capacity that fits. Never write availability so that a
  resolution succeeds, and never widen a window, drop an activity or extend a
  horizon to force a fit. A plan resting on capacity the person never offered
  is worse than no plan, and they cannot see that it was invented.
- **A party you do not track has no supply.** An intention naming someone
  outside the workspace has no candidates until that person's availability is
  recorded, and for an external party it never will be. Say so and ask how
  they want to proceed; do not invent capacity for other people, whose time
  is not the person's to declare.

## Rules for availability

- **Capacity is a fact about the person; only they can assert it.** Record
  availability from what they told you, with `--subject` explicit. Do not
  infer it from an empty calendar, do not write it to make a resolution
  succeed, and do not write it for anyone but the people whose capacity this
  workspace tracks. One availability per intention is a sign you are
  fabricating: real capacity is broader than the plans that draw on it.
- **Terms change by supersession.** Window (including clock), duration,
  conditional and location define the disposition. `availability edit`
  refuses to change them; `availability supersede` creates the new one and
  retires the old as superseded, keeping every reference valid.
- **Renewal advances `valid_until` only.** Recurring availability without one
  is treated as valid for the workspace default horizon from its timestamp;
  `availability show` reports `effective_valid_until`. Renew when the person
  reconfirms; do not renew on their behalf.
- **Scope is only ever widened.** Default `personal`; `organisation` when the
  person says others may plan against it; never `public` unless told.

## Rules for commitments

A commitment is the interpersonal object: `select` writes one whenever the
intention has parties, with everyone at `tentative`.

- **A party's status is theirs.** `accept` and `decline` record what the person
  told you and nothing else. No policy authorises them, no flag implies them,
  and a person sounding keen is not an acceptance. If you have not been told,
  ask; do not answer for them.
- **You answer for one party.** `--party` defaults to the workspace's subject,
  the person you work for. Answering for anyone else is an iTIP reply, which
  arrives by import, not by this verb.
- **Cancel frees the intention.** `commitment cancel` retires the commitment as
  `cancelled`, the only kind it admits, and clears the placement of the
  intention it was for. That intention keeps its window and returns to
  `unresolved`, so offer to resolve it again.
- **Declining frees the decliner, and only them.** A party who has declined is
  no longer occupied by the commitment; every other party still is, and their
  commitment still stands. Declining is not cancelling: the commitment remains
  until someone cancels it.
- **A decline against a standing plan is flagged.** When the intention a
  commitment fulfils is still placed, any decline raises `party-declined` on
  both objects. The subject's own decline frees nothing, because their
  placement still occupies the hour. Put the choice to the person: cancel,
  re-resolve with `select --replace`, or acknowledge and go ahead without the
  party who declined.

## Retirement

Nothing is deleted. An intention that is done is retired `fulfilled`; one the
person no longer holds, `abandoned`; one replaced by a specific other,
`superseded` with `--superseded-by`. Availability is `retracted` or
`superseded`. A retired object refuses further edits and keeps its file, so
every reference to it still resolves. Give a `--reason` a reviewer can read.

## Working with git

Work on a branch. The YAML you produce is the PR; the person's merge is the
acceptance. Commit the object files **and** `index.yaml` (the tool keeps it
current). If `index.yaml` ever conflicts on merge, run `intentions index`; do
not resolve it by hand. Run `intentions validate` after any merge: files that
arrive by merge never passed through the writer's refusals. Do not push or
open PRs unless the user's workflow says to.

When you write a branch up, say what each intention is for and what changed in
its window, not just which files moved. A reviewer reading `firmed_under` on a
diff should find the policy named in the PR body.

## Example session

```sh
# The user said: "I need about ninety minutes on the budget narrative this week,
# mornings, at home. It's for getting the board pack out."
intentions intention list --json
# → nothing on the budget yet; a "Board pack out" intention exists as int_B

intentions intention add --title "Draft the Q4 budget narrative" \
  --duration PT90M --calendar this-week --clock 09:00/12:00 \
  --activity deep-work --location https://example.com/places/home \
  --serves "${B}:in-order-to" \
  --description "Two focused sessions should be enough; the numbers are already in." \
  --json
# → int_A, stability tentative, calendar 2026-W37

# They also said Tuesday mornings are their deep-work time until December.
intentions availability add --subject https://example.com/people/ada \
  --title "Tuesday mornings for deep work" --duration PT3H \
  --calendar 2026-09/2026-12 --clock 09:00/12:00 --conditional deep-work \
  --cadence "FREQ=WEEKLY;BYDAY=TU" --valid-until 2026-12 --json

intentions validate --json
# → ok: true
```
