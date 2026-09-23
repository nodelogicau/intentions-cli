## 1. Model

- [x] 1.1 `TypeDesire`, prefix `des`, directory `desires`, in `Types` first; the id patterns gain `des`
- [x] 1.2 `Desire` struct with the admitted fields and the `Object` methods; `Refs` from `serves` and `retired.superseded_by` and `retired.adopted_as`
- [x] 1.3 `Retired.AdoptedAs` decoded, encoded after `superseded_by`, and carried in every type's `Refs`
- [x] 1.4 Decoder for a desire that records a `forbidden_field` problem for each of the intention's temporal and deontic keys; encoder in canonical order
- [x] 1.5 Structural check: subject, title, term, URIs, serves roles `for-the-sake-of` only, source with author, timestamp, retirement with the `adopted_as` rules
- [x] 1.6 `DesireRetirementKinds`; `CheckRetirement` refuses `adopted` from a retirement act on any type and checks `adopted_as` shape
- [x] 1.7 `CheckDesireServes`: role, target an active terminus of the same subject, tentative or firm; the sink rule covers a terminus targeted by a desire
- [x] 1.8 `Adopt`: builds the tentative intention, refuses a retired desire and a bare adoption naming what is missing, returns the intention and the retirement
- [x] 1.9 Projection `Fields[TypeDesire] = {subject, serves, retired.kind}` and the desire case in `Project`

## 2. Store, validation

- [x] 2.1 `Graph.Desires()`; the loader and index need nothing beyond `Types`; `init` creates `desires/`
- [x] 2.2 Validation: referential checks for a desire's serves targets, `superseded_by` type, `adopted_as` presence and type; `forbidden_field` problems surface as structural errors; terms count desire activities

## 3. CLI

- [x] 3.1 `cmd_desire.go`: `desire add | edit | adopt | retire | show | list` with the intention verbs' flag shapes; `adopt` takes `--duration` and the window flags and returns `{intention, desire, findings?}`
- [x] 3.2 The root registers `desire`; `show <id>` reads a `des_` id

## 4. MCP

- [x] 4.1 `tools_desire.go`: the six tools with typed inputs, required parameters (`title` on add; `id` on edit, adopt, retire, show; `kind` on retire), descriptions that say what a desire is and when to adopt
- [x] 4.2 The instructions name the `desire_*` tools; the tool-list and required-parameter tests grow to thirty-one

## 5. Tests

- [x] 5.1 Model: decode and encode round trip, forbidden fields, serves rules, adoption rules including bare adoption, retirement rules
- [x] 5.2 Validation: the desire scenarios; conflicting desires report nothing; a desire is not in `unresolved` and gets no flag
- [x] 5.3 CLI: add, edit (prose leaves version, serves moves it), adopt with findings, bare adopt refused, retire kinds, show and list, `init` creates the directory
- [x] 5.4 MCP: adopt and bare adopt, retire refuses `adopted`, list

## 6. Skill, docs and release

- [x] 6.1 Skill: the inbox section, the loop line, the verbs table rows; regenerate the installed copy
- [x] 6.2 README: verbs table, object types, index and directories; `docs/mcp.md`: six rows, token estimate; SPEC-FEEDBACK line on the reference set count
- [x] 6.3 CHANGELOG 0.14.0; reply on intentions-cli#7
- [x] 6.4 Tag `v0.14.0` and verify the release
