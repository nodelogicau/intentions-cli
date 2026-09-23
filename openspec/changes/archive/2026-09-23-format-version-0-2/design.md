## Context

The binary implements exactly one format: a constant, one decoder and encoder per type, one projection table, and a config check that refuses any other string. Twenty-three call sites compute a version, seventeen render an object, and none of them knows which workspace the object came from. Threading a format parameter through all of them is the obvious refactor and the wrong one: the format is a property of the file, and the object is the file in memory.

## Goals / Non-Goals

**Goals:**
- A 0.1 workspace is byte-stable under the new binary: load, re-save, identical files and versions.
- One model, one set of verbs, with the format deciding spelling, projection and the unserved severity.
- Migration is one act, reviewable as a diff, that changes shape and nothing a person wrote, and un-acknowledges nothing.
- A newer format is refused by name.

**Non-Goals:**
- A transaction across the files migration rewrites; the CLI does no git operations, and the person reviews the diff.
- Writing 0.1 into a new workspace; `init` writes the current format only.
- Any change to what the four breaks mean beyond what the text says.

## Decisions

### D1. The object carries its format
`Object` gains `Format() string` and `SetFormat(string)`. The loader sets every decoded object's format from the workspace's `format`; `WriteObject` sets it to the workspace's format before stamping, so a new object written into a 0.1 workspace is a 0.1 object. `model.Encode`, `projection.Project` and `projection.Version` read the object's format, defaulting to the current one when unset, so every existing call site stays as it is and every computed version is the version of the file as it stands. Migration is then: set the format to 0.2 on every object, stamp, write.

Alternative considered: a format parameter on every projection and render function. Rejected for sixty call sites and for making the wrong thing the unit of knowledge.

### D2. The struct fields take the 0.2 names; the decoder takes both spellings
`Availability.Capacity`, `Availability.Activities`, `Commitment.Origin` as a string and `Commitment.Resolution` as its own field. The decoder maps `duration` and `conditional` onto the new fields and accepts `origin` as either the 0.1 map or the 0.2 scalar, whatever the workspace's format, so a file in the other spelling reads correctly and the cached-version warning is what reports it. The encoder writes the spelling of the object's format. Callers that named `Duration` or `Conditional` on an availability change once.

### D3. Projection per format
`Fields` becomes `map[format]map[type][]string`, with 0.1 keeping `duration`, `conditional` and `origin` as a map, and 0.2 using `capacity`, `activities`, `origin` as a string plus `resolution`. `Project` switches on the object's format for the two types that differ. The golden vectors recorded by v0.1.0 stay green under 0.1.

### D4. The unserved severity keys on the format
`query.Validate` reports `unserved` at error under 0.2 and warning under 0.1. `WriteFindings` does the same, and the three intention writes, in both front ends, refuse when the object's workspace is 0.2 and the finding is an error, with the same message the warning carried. Adoption inherits it, since it writes an intention.

### D5. Migration
`intentions migrate [--check]` and the `migrate` tool. Steps, in order: load under the workspace's format, which must be 0.1 (0.2 is a no-op that says so; newer was already refused); compute the unserved set under the 0.2 rule and refuse naming it; for every object, record the old version, set the format to 0.2, stamp; for every acknowledgement on an intention or commitment whose `counterpart_version` equals its counterpart's old version, set it to the new one, leaving an already-lapsed one alone; write every object whose bytes changed; rebuild the index; rewrite `format` in `intentions.yaml` last, so an interruption leaves a 0.1 workspace with some files already in 0.2 spelling, which the lenient decoder reads and the next run finishes. `--check` prints the counts and the ids that would change and writes nothing. The result lists rewritten objects, rewritten acknowledgements, and the format before and after.

### D6. Timestamp as last write, in the writers
Every verb that rewrites an object file sets `Timestamp` to the act's time: the edits, `firm`, the retirements, `renew`, `supersede`, `select`'s placement write, cancel's freeing write, the commitment answers, and `acknowledge`. `WriteObject` is not the place, because records are written through it too and keep the time of the act. `--timestamp` still overrides where offered.

### D7. Aliases and schemas
`--duration` and `--conditional` on the availability verbs become hidden aliases of `--capacity` and `--activities`. The MCP availability input carries both keys, prefers the new, and documents the old as deprecated. The skill and docs use the new names only.

## Risks / Trade-offs

- [A 0.1 file's version moves by accident] → The byte-stability test loads the 0.1 fixture workspace, re-saves every object, and asserts identical bytes and versions; the golden vectors stay under 0.1.
- [Migration interrupted] → Ordered so the format flips last; re-running finishes; `--check` first.
- [An acknowledgement lapses through migration] → Only versions that matched before are rewritten; the test acknowledges a flag against a resolution-born commitment and migrates.
- [Old flags in scripts] → Aliases keep them working; only the file spelling is what changes.
- [Test fixtures in the old spelling] → The decoder reads them; tests that write raw 0.1 YAML into a 0.2 workspace get `stale_version` warnings they must expect or migrate around.

## Migration Plan

For users: nothing happens until they run `intentions migrate`. The changelog says so, and the skill says to run `--check` first, walk up any unserved intentions, then migrate on a clean checkout and review the diff. Release as 0.15.0.
