## 1. Formats in the model

- [ ] 1.1 `Format01`, `Format02`, `Format` (current) and `KnownFormats`; `Object.Format()` and `SetFormat()` on every type; the config accepts known formats and refuses others naming both
- [ ] 1.2 Rename `Availability.Duration` to `Capacity` and `Conditional` to `Activities`; `Commitment.Origin` string and `Resolution` field; every caller
- [ ] 1.3 Decoder accepts both spellings and both origin shapes; encoder writes by the object's format; retired block unchanged
- [ ] 1.4 Projection fields per format; `Project` switches on the object's format; golden vectors stay under 0.1
- [ ] 1.5 Loader sets the format on every object; `WriteObject` sets it from the workspace before stamping; the index carries the workspace's format

## 2. Rules by format

- [ ] 2.1 `validate`: `unserved` at error under 0.2, warning under 0.1; the structural check for `resolution` present iff `origin: resolution`
- [ ] 2.2 The intention writes and adoption refuse an unserved result under 0.2, in CLI and MCP, with the finding's message

## 3. Migrate

- [ ] 3.1 `model`/`store`: a migration that returns the plan (objects and acknowledgements that change, unserved blockers) and applies it in order, format last
- [ ] 3.2 `intentions migrate [--check]` and the `migrate` tool; results list what changed; a 0.2 workspace is a no-op

## 4. Refinements under both formats

- [ ] 4.1 `parties` on a desire: field, flag, schema, decoder, encoder, adoption copies it; removed from the forbidden list
- [ ] 4.2 Timestamp as last write in every object-rewriting verb, CLI and MCP; records unchanged
- [ ] 4.3 `--capacity`, `--activities` with hidden aliases; MCP keys with the old ones accepted; `availability list --activities`

## 5. Tests

- [ ] 5.1 A 0.1 fixture workspace: byte-stable re-save, versions unchanged, unserved is a warning, writes keep 0.1 spelling
- [ ] 5.2 Migration: refused while unserved; rewrites origin, names, versions, index, format; acknowledgement carried across; lapsed one left; `--check` writes nothing; no-op on 0.2
- [ ] 5.3 0.2 behaviour: `init` writes 0.2, unserved refused on write and error on validate, new spellings in files; newer format refused
- [ ] 5.4 Refinements: desire parties through adoption; timestamp moves on edit, firm, select, retire, answers; existing tests updated for the format string and the tool count (thirty-two)

## 6. Skill, docs and release

- [ ] 6.1 Skill: 0.2 names, migration guidance (`--check`, walk up, clean checkout), the person's act; regenerate the installed copy
- [ ] 6.2 README (verbs, availability, versions, migration), `docs/mcp.md` (tool row, keys, token estimate), `version` output
- [ ] 6.3 CHANGELOG 0.15.0; reply on intentions-cli#9
- [ ] 6.4 Tag `v0.15.0` and verify the release
