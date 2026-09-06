# Review workflow

`intentions` writes files and nothing else. Review happens through git, and the
tool is designed so that a pull-request diff tells the reviewer everything:

- files are written in canonical field order, so a diff shows only what changed;
- every object carries `source`, so the diff shows who drafted it and which
  harness held the pen;
- every object carries its `version`, so a two-line change to `version` tells
  the reviewer the scheduling projection moved, and a diff with no `version`
  change is prose or attribution only.

## The loop

```
harness branch ──▶ intentions intention add / edit / firm / retire ──▶ PR
                                                                       │
person reviews the diff ◀──────────────────────────────────────────────┘
      │
      ├── merge: the intention is the person's
      └── change request: the harness edits on the branch; nothing is deleted
```

A harness works on a branch. Everything it writes is a proposal until the
person merges. The harness may draft, edit its own drafts, and retire; it may
not set `firm` unless it names an `auto_firm` policy the person holds, and the
command result records the policy so the reviewer can check it.

Retirement never deletes. A file that is wrong is retired as `abandoned` or
`superseded`, and the reviewer sees the record and its reason.

## Merges

Two branches that both add objects never conflict on object files, because
every object is its own file with a time-ordered id. They both touch
`index.yaml`. The index is derived: resolve a conflict by regenerating it, never
by hand-merging.

```sh
git checkout --theirs index.yaml   # or ours; it does not matter
intentions index
git add index.yaml
```

Files that arrive by merge never passed through a writer, so the write-time
refusals (a cycle in `serves`, a change of terms on an availability) may not
have applied. `validate` is the invariant; run it after every merge.

## CI

A workflow that keeps `main` honest:

```yaml
name: intentions
on:
  pull_request:
  push:
    branches: [main]
jobs:
  check:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Install intentions
        env:
          INTENTIONS_INSTALL_DIR: ${{ runner.temp }}/bin
        run: curl -sSL https://raw.githubusercontent.com/nodelogicau/intentions-cli/main/install.sh | sh
      - name: Validate the workspace
        run: "$RUNNER_TEMP/bin/intentions" validate --workspace .
      - name: Index matches the files
        run: "$RUNNER_TEMP/bin/intentions" index --check --workspace .
```

`validate` exits 4 on any error and lists warnings and info without failing;
`index --check` exits 4 when the committed index disagrees with a rebuild.

## What a reviewer looks for

- **A new `firm`** with `harness` in the source: the file must carry
  `firmed_under` naming a terminus of the subject with `auto_firm`. Without
  one the write should have been refused; if it arrived by merge, `validate`
  reports an error.
- **A `retired` record** with a reason that makes sense, and a `superseded_by`
  that points at the replacement.
- **A `window` change** on an intention the person considered settled. The
  window is the person's, not the clock's; a harness narrowing it is a
  proposal.
- **A new availability** from a harness: capacity is a fact about the person
  and only they can assert it. Check the `author`.
