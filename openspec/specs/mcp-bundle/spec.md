# MCP Bundle

## Purpose

The Claude Desktop extension bundle (`.mcpb`): its contents and manifest, the user configuration it asks for, its attachment to every release, and the client configurations the documentation must show.

## Requirements

### Requirement: Desktop extension bundle
Each release SHALL attach `intentions-<version>.mcpb`, a zip containing `manifest.json` (`manifest_version` `0.3`, `name` `intentions`, `version` equal to the release), an icon, and the server binaries for darwin (universal) and win32 (x64) under `server/`. The manifest's `server.type` SHALL be `binary`, `mcp_config.command` SHALL point at the bundled binary via `${__dirname}` with `platform_overrides`, and `args` SHALL be `serve --mcp --workspace ${user_config.workspace} --author ${user_config.author}`. `compatibility.platforms` SHALL list `darwin` and `win32`. A bundle whose macOS binary is not universal SHALL NOT be produced.

#### Scenario: Bundle contents
- **WHEN** `make bundle` runs after a cross-compile
- **THEN** the `.mcpb` contains `manifest.json` with the build's version and every binary the manifest references, and the manifest parses as JSON

#### Scenario: Release asset
- **WHEN** a tag is released
- **THEN** the release includes `intentions-<version>.mcpb` alongside the archives and `checksums.txt` lists it

### Requirement: User configuration
The manifest SHALL declare `user_config.workspace` (type `directory`, required, described as the folder containing `intentions.yaml`) and `user_config.author` (type `string`, optional). A blank `author` SHALL be treated by the server as unset.

#### Scenario: Folder without intentions.yaml
- **WHEN** the user picks a folder that is not a workspace
- **THEN** the server exits with code 5 and a message on stderr naming the folder and `intentions init`

### Requirement: Documented client configuration
`docs/mcp.md` SHALL show how to connect from Claude Desktop (bundle and manual configuration), Claude Code (`.mcp.json` with the server spawned in the project so discovery finds the workspace), and any stdio client, SHALL list every tool with its parameters and result, and SHALL say when to prefer the skill and CLI over the server.

#### Scenario: Claude Code config
- **WHEN** a user follows the `.mcp.json` example in a repository with a `.intentions` pointer
- **THEN** the server starts without `--workspace` and `workspace_status` reports the pointed-to workspace
