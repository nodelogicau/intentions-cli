## Why

The specification defines a fourth object type, DESIRE (nodelogicau/intentions changes `add-desire` and `adoption-makes-a-plan`, capability `desire`), and [#7](https://github.com/nodelogicau/intentions-cli/issues/7) asks this CLI to implement it. It is the inbox the grounding rule needed: since v0.12.0 an intention without a why is a warning and will one day be a refusal, so a passing remark ("call the accountant") has had nowhere to go but an intention it does not deserve or the harness's memory. A desire is a want the person expressed and has not committed to: a world-to-mind pro-attitude, not a commissive act, so it is free to conflict, is never resolved, is checked against nothing, and need not say what it is for. Adoption turns it into an intention, which costs a why or a when, and leaves the trail.

## What Changes

- **A new object type.** Prefix `des_`, directory `desires/`, canonical order `id, version, subject, title, description, activity, location, serves, reference, source, timestamp, retired`. It carries none of an intention's temporal or deontic fields; any of them on a desire is a validation error, and the decoder names each rather than keeping it as an unknown extra. No strength or priority field. Projection `subject`, `serves`, `retired.kind`.
- **`serves` is `for-the-sake-of` only**, each target a terminus of the same subject, tentative or firm. Empty is fine. A desire is never unserved.
- **Exemptions.** `resolve`, `check`, `unresolved` and the grounding walk never see a desire. Two desires may conflict freely.
- **Six verbs and six tools.** `desire add | edit | adopt | retire | show | list`, and `desire_add`, `desire_edit`, `desire_adopt`, `desire_retire`, `desire_show`, `desire_list`, taking the reference tool set from twenty-five to thirty-one.
- **Adoption.** `desire adopt <id>` writes a tentative intention carrying the desire's subject, title, description, serves, reference, activity and location plus the supplied duration and window, then appends `retired: {kind: adopted, adopted_as: <int id>}` to the desire. The intention is the record. Refused when the desire is retired, and refused when the result would be a terminus, that is empty `serves` with neither duration nor window, naming what is missing, a why or a when. A harness may add and adopt desires without a policy.
- **Retirement.** Kinds `abandoned`, `superseded` (naming a desire), `adopted` (written only by adoption; `desire retire --kind adopted` is refused). `Retired` gains `adopted_as`, required when and only when the kind is `adopted`, naming an intention.
- **The index** gains the `desire` type; `init` creates `desires/`.
- **The skill** gains an inbox: a passing remark is `desire add`; read the inbox each session; adopt when it has a why or a when; a desire is something the person said they wanted, not something inferred.
- CHANGELOG 0.14.0; SPEC-FEEDBACK item 28 is already recorded. Format version stays `intentions/0.1`.

## Capabilities

### New Capabilities
- `desire`: the object, its rules, adoption, retirement, the six verbs and their results.

### Modified Capabilities
- `object-format`: the `des` prefix, the `desires/` directory, the canonical order, `adopted_as` in the retired block.
- `index`: the `desire` entry type; `adopted_as` in `refs`.
- `workspace`: `init` creates `desires/`.
- `validation`: the desire errors.
- `mcp-server`: the six tools on the surface.
- `skill-distribution`: the inbox guidance.

## Impact

Additive. No existing file changes shape; an old binary reading a new workspace sees an unknown directory it never opens and index entries of a type it preserves. The type wiring touches every per-type switch: constants, ids, decode, encode, structural check, projection, the loader, the index, validation, one new CLI file and one new MCP file. Tests that pin the tool count and the required-parameter table grow by six. The skill grows by a short section.
