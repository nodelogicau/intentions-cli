## Context

Discovery reads `.intentions` already: the first non-blank, non-comment line is a path, resolved against the pointer's own directory. Only the writing side is missing.

## Decisions

### D1. A subcommand of `workspace`, not a top-level verb
`workspace pointer` keeps the workspace vocabulary in one place, and the parent verb already reports `found_by: pointer`, so the pair reads as one topic. This matches particulars.

### D2. Relative when it can be, absolute when it must
A pointer inside a repository should survive cloning to a different path, so the target is written relative whenever the workspace lies at or below the pointer's directory. When it does not, the pointer is machine-specific; the text output says so and the JSON carries `relative: false`, leaving the decision to commit it to the person.

### D3. Refusals, not surprises
Writing over a pointer that names a different workspace would silently redirect every verb run in that tree, so it fails with exit code 1 unless `--force`. Rewriting the identical pointer succeeds so the verb is idempotent. Three usage errors guard the obvious mistakes: the pointer directory is the workspace itself, it already holds `intentions.yaml` (which wins over a pointer at the same level), or `--at` is not a directory.

### D4. `init --pointer` needs an explicit directory
`init --pointer` with no `dir`, or with `dir` equal to the current directory, would write a pointer beside the `intentions.yaml` it points at, which discovery ignores. That is a usage error rather than a silent no-op.

## Risks / Trade-offs

- An absolute pointer committed by accident breaks for everyone else. The verb reports it rather than refusing, because a workspace outside the tree is a legitimate arrangement for a personal machine.
