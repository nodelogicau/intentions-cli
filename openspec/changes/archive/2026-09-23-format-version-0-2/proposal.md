## Why

The specification has declared v0.1 (tag `v0.1` at nodelogicau/intentions 54c9a83, the text this CLI implements as `intentions/0.1`) and its current text is the `intentions/0.2` draft (60796e8, change `format-version-0-2`), settling SPEC-FEEDBACK item 29: the `origin` change moved a projection under an unchanged version string, so upstream fixed the rule (a projection is frozen from the moment any implementation writes the version string into a file) and made 0.2 the one break carrying everything the text had deferred. [#9](https://github.com/nodelogicau/intentions-cli/issues/9) asks this CLI to write 0.2, read 0.1 as it is, migrate between them as the person's act, and refuse anything newer.

## What Changes

- **Two formats, one model.** The in-memory model carries the 0.2 names. Every object knows the format it was read under or will be written under. The decoder accepts both spellings and both `origin` shapes; the encoder writes the shape the workspace's `format` names; the projection is computed per format, so a 0.1 file's version never moves until migration. **BREAKING** for new workspaces only: `init` writes `format: intentions/0.2`.
- **The four breaks, under 0.2.** A commitment's `origin` is `resolution` or `import` with a sibling `resolution` field, both projected. Availability's `duration` is `capacity` and `conditional` is `activities`, both projected under the new names. An intention that reaches no firm terminus is a validation error, and a write that would leave one unserved is refused. Under 0.1 the old shapes, names and warning stand.
- **Read 0.1 as it is.** A 0.1 workspace is read and written under 0.1 rules and shapes until migrated. Loading and re-saving a 0.1 file produces identical bytes and an unchanged version.
- **Refuse newer.** A `format` this binary does not implement is refused for read and write, naming both versions. The existing check refused everything but one version; it now accepts 0.1 and 0.2.
- **`migrate`**, a verb and a reference tool. Refuses to rewrite `format` while any intention is unserved under the 0.2 rule, naming them. Otherwise rewrites `origin`, renames the two availability fields, recomputes every version under 0.2, rewrites `counterpart_version` on every acknowledgement whose counterpart changed only by the migration, regenerates the index, and rewrites `format` last. Nothing else moves. `--check` reports what would change and writes nothing. The tool set is thirty-two.
- **Flags and JSON keys.** `--capacity` and `--activities` on `availability add`, `supersede` and `list`; `--duration` and `--conditional` stay as hidden aliases, since the skill is one text serving both formats. The MCP schemas take `capacity` and `activities` and still accept the old keys.
- **The three refinements upstream confirmed apply under both formats.** `parties` on a desire, a hint carried onto the intention on adoption. `timestamp` on an object is the time of its last write, set by every verb that rewrites an object file, records keeping the time of the act. The person's act stated in the skill: absence of `firmed_under` on an object, `selector: person` on a record.
- `version` prints what the binary reads and what it writes. CHANGELOG 0.15.0; SPEC-FEEDBACK item 29 is already recorded.

## Capabilities

### Modified Capabilities
- `object-format`: format versions read and written, the newer-format refusal, migration, the 0.2 canonical field names.
- `projection-version`: per-format field sets and the freeze rule.
- `availability`: `capacity` and `activities`, the aliases, and the list filter.
- `selection`: the commitment's `origin` shape in the recorded act.
- `consistency`: `condition-mismatch` names `activities`.
- `index`: the workspace's format string.
- `workspace`: `init` writes 0.2; the config accepts both formats.
- `validation`: unserved as an error under 0.2.
- `intentions`: unserved write refused under 0.2; timestamp as last write.
- `desire`: `parties`.
- `cli-interface`: the version verb.
- `mcp-server`: the `migrate` tool.
- `skill-distribution`: the person's act, the 0.2 names, and migration guidance.

## Impact

Existing workspaces keep working unchanged until their owner runs `migrate`; their files, versions and acknowledgements do not move. New workspaces are 0.2 and refuse an unserved write from the first, which is the point of the walk-up the skill already carries. A 0.1 binary cannot read a 0.2 workspace and says so by the `format` key. Code touched: the object interface gains a format, and the loader, writer, encoder, projection and validator honour it; one new verb and tool; the renames across the model, resolve, consistency, CLI, MCP, skill and docs; the timestamp rule in every object-rewriting verb.
