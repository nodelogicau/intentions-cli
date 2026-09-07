## 1. Untracked parties

- [ ] 1.1 `Env.Tracks(uri)`: the workspace holds any availability with that subject, retired and expired included
- [ ] 1.2 Resolution exempts untracked parties from supply and from `no_supply`, never the subject, and returns the presumed set
- [ ] 1.3 Confirm an untracked party's placements still constrain: displacement and ranking, and a declined entry frees them

## 2. The record

- [ ] 2.1 `presumed` on the RESOLUTION model, written after `supply`, omitted when empty, outside the projection; `select` fills it from the result

## 3. Tests

- [ ] 3.1 Untracked party resolves, tracked party with nothing eligible does not, a retired record still counts as tracked, the subject is never exempt
- [ ] 3.2 No double-booking an untracked party; a declined untracked party frees the hour
- [ ] 3.3 `presumed` on the record and absent from the projection; the MCP result carries it

## 4. Docs and release

- [ ] 4.1 Reverse the skill rule and state that declaring capacity narrows it; README and `docs/mcp.md` where they describe supply
- [ ] 4.2 SPEC-FEEDBACK item 25 records the settlement; CHANGELOG 0.11.0; reply on intentions-cli#1
- [ ] 4.3 Tag `v0.11.0` and verify the release
