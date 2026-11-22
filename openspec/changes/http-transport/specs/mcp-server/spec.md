## MODIFIED Requirements

### Requirement: Stdio MCP server bound to one workspace
`intentions serve --mcp [--workspace <dir>] [--author] [--harness] [--model] [--http <addr>] [--http-token-file <path>]` SHALL run a Model Context Protocol server bound at startup to exactly one workspace, resolved as every verb resolves it. Without `--http` the server SHALL speak stdio, and SHALL write nothing to stdout except JSON-RPC. With `--http` it SHALL speak Streamable HTTP as *HTTP transport* requires, and SHALL write nothing to stdout. Without a workspace it SHALL exit with code 5 and a message on stderr, before speaking either transport. `serve` without `--mcp` SHALL be a usage error, and so SHALL `--http-token-file` without `--http`.

#### Scenario: Started in a project
- **WHEN** `intentions serve --mcp` is started in a directory whose ancestor has a `.intentions` pointer at a valid workspace
- **THEN** the server initialises over stdio and every tool operates on that workspace

#### Scenario: No workspace
- **WHEN** `intentions serve --mcp` is started with no resolvable workspace
- **THEN** the process exits with code 5 and stdout is empty

#### Scenario: No workspace over HTTP
- **WHEN** `intentions serve --mcp --http 127.0.0.1:8080` is started with no resolvable workspace
- **THEN** the process exits with code 5 without listening

## ADDED Requirements

### Requirement: HTTP transport
With `--http <addr>`, where `addr` is `host:port` and a missing host means every interface, the server SHALL serve MCP Streamable HTTP at the path `/mcp`. Every session SHALL be served by the one server bound to the workspace. Tools, results, errors, instructions, prompts and resources SHALL be identical to those over stdio, and every write SHALL apply the same rules and refusals. Other paths SHALL return 404, except `/healthz`. On startup the server SHALL write to stderr the version, the workspace root, the address it listens on, and whether a token is required.

#### Scenario: The same result over both transports
- **WHEN** one client calls `intention_list` over stdio and another calls it over HTTP, on the same workspace
- **THEN** both results equal `intentions list --json`

#### Scenario: Unknown path
- **WHEN** a request is sent to `/` on a server started with `--http`
- **THEN** the response is 404

### Requirement: HTTP requires a token off loopback
The token SHALL be read from `INTENTIONS_HTTP_TOKEN`, or from the file named by `--http-token-file`, whose contents with surrounding whitespace trimmed are the token. The file SHALL win when both are set. An empty token SHALL count as none. When a token is configured, every request to `/mcp` SHALL carry `Authorization: Bearer <token>`. The comparison SHALL be in constant time. A request without the token, or with a different one, SHALL get 401 and SHALL not reach the server. When no token is configured, the server SHALL listen only if `addr`'s host is `127.0.0.1`, `::1` or `localhost`. Otherwise it SHALL exit with a usage error before listening.

#### Scenario: Missing token
- **WHEN** a server started with `INTENTIONS_HTTP_TOKEN=s3cret --http :8080` receives an `initialize` request with no `Authorization` header
- **THEN** the response is 401 and no session is created

#### Scenario: Correct token
- **WHEN** the same server receives `initialize` with `Authorization: Bearer s3cret`
- **THEN** a session is created and the tools are listed as over stdio

#### Scenario: Tokenless on all interfaces
- **WHEN** `intentions serve --mcp --http :8080` is started with no token configured
- **THEN** it exits with a usage error saying a token is required off loopback, and does not listen

#### Scenario: Tokenless on loopback
- **WHEN** `intentions serve --mcp --http 127.0.0.1:8080` is started with no token configured
- **THEN** it listens, and serves requests without an `Authorization` header

### Requirement: Health endpoint
`GET /healthz` SHALL return 200 with a JSON body `{"status": "ok", "version": <version>}` once the workspace is loaded and the listener is up. It SHALL not require the token, and SHALL reveal no workspace content, path or configuration.

#### Scenario: Readiness probe
- **WHEN** a container platform sends `GET /healthz` without a token to a server started with a token
- **THEN** the response is 200 with `status: ok` and the binary's version, and nothing else

### Requirement: HTTP sessions share one workspace and one write lock
Concurrent HTTP sessions SHALL share the server-wide lock that serialises mutating tools, so that writes from different sessions never interleave load, write and index update. Each session SHALL take its attribution defaults from its own `initialize`, as *Attribution from the handshake* requires.

#### Scenario: Parallel adds from two sessions
- **WHEN** two HTTP sessions each issue ten `intention_add` calls concurrently
- **THEN** twenty intention files exist and `intentions index --check` passes

#### Scenario: Each session's own harness
- **WHEN** a session whose client is named `executive` and a session whose client is named `backend` each add an intention with no `source`
- **THEN** the intentions carry `source.harness: executive` and `source.harness: backend` respectively

### Requirement: A remote session is never the person
Over HTTP, when no harness is supplied by the call, server flags, environment or `intentions.yaml`, and the session's `initialize` carries no `clientInfo.name`, `source.harness` SHALL default to `http`. Every call through such a session SHALL therefore carry a harness for *The harness boundary holds*. Over stdio the existing default SHALL be unchanged.

#### Scenario: Nameless remote client adds
- **WHEN** an HTTP session whose `initialize` has an empty `clientInfo.name` calls `intention_add` with no `source`, in a workspace with no default harness
- **THEN** the intention is written with `source.harness: http`

#### Scenario: Nameless remote client cannot firm a terminus
- **WHEN** the same session calls `intention_firm{id: <terminus id>, source: {author: <uri>}}`
- **THEN** the result is an error with code `refused`, because the session carries the harness `http`, and the file is unchanged

### Requirement: Clean shutdown over HTTP
On SIGTERM or SIGINT, a server speaking HTTP SHALL stop accepting new connections. It SHALL let requests in flight finish for up to 30 seconds, then exit 0. A mutating call that has started SHALL either complete, or leave the workspace as it was before the call.

#### Scenario: Terminated during a write
- **WHEN** SIGTERM arrives while an `intention_add` call is in flight
- **THEN** the call completes, its intention file and index entry are written, and the process exits 0
