## Why

The specification replaced its terminus convention (nodelogicau/intentions, change `terminus-as-practice`). A terminus is no longer titled as who the person is (*being someone who follows through*): holding oneself as a fixed thing is what Sartre calls bad faith. It is titled as a practice the person chooses to keep doing (*following through*), and the test is whether the title could ever be done. This CLI still carries the old convention in the `init` stub, the embedded skill and several messages (#19).

## What Changes

- The `init` conventions stub's **Termini** section states the practice convention and the test.
- The embedded skill's first question becomes what the person means to keep doing, for its own sake. Its titling rule, split example, bare-adoption line and "Who are you trying to be?" bullet are reworded to match.
- Messages: `terminus_fulfilled` says a practice is never finished; `bare_adoption`, `serves_target` and the unserved fix no longer speak of who the person is or of a self.
- The README's split and bare-adoption examples are retitled.
- Test fixture titles are left alone, since wording is never validated. No rule, code, exit code or projection changes.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `skill-distribution`: the walk-up requirement's first question and the stub's Termini section state the practice convention; the terminus-change requirement speaks of a practice found to be several.
- `intentions`: Retire an intention gives "a practice is never finished" as the reason a terminus can't be `fulfilled`.

## Impact

`skills/intentions/SKILL.md` and its installed copy (`make skill`), `internal/store/workspace.go`, `internal/model/{policy,rules,desire,grounding}.go`, `internal/query/validate.go`, one assertion in `internal/cli/cli_test.go`, README, CHANGELOG.
