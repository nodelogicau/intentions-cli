## Why

Discovery already resolves a `.intentions` pointer file, and the MCP documentation tells Claude Code users to rely on one, but nothing in the CLI writes it. A person who keeps their planning workspace in a subdirectory of a repository has to create the file by hand and guess its contents. particulars settled this with a pair: `init --pointer` at creation time and `workspace pointer` for a workspace that already exists.

## What Changes

- **`intentions workspace pointer [workspace-dir] [--at <dir>] [--force]`** writes `.intentions` in `--at` (default: the current directory) naming a workspace that already exists. With no argument it names the workspace that would be used now. The target is written relative when the workspace lies inside the pointer's directory, so the file survives being cloned elsewhere, and absolute otherwise, which the result reports as machine-specific.
- **`intentions init [dir] --pointer`** also writes `./.intentions` naming the new workspace, for the common case of creating one inside a repository.
- **`store.WritePointer` refuses to clobber**: a pointer naming the same target succeeds unchanged; one naming a different target fails with exit code 1 and leaves the file alone unless `--force` is given.
- README, the skill's setup section, and CHANGELOG 0.7.0.

## Capabilities

### New Capabilities
<!-- none -->

### Modified Capabilities
- `workspace`: the pointer-writing verb and the `init --pointer` flag.

## Impact

`internal/store` gains the refusal in `WritePointer`; `internal/cli` gains the subcommand and the flag. No file format change, no new dependencies, no change to discovery itself.
