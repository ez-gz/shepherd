# Code-quality and reliability audit

This backlog captures the August 2026 audit of tmux/process execution,
configuration and filesystem safety, runtime health, CLI packaging, and release
hygiene. The full test suite, vet, race detector, and real tmux integration
tests passed during the audit.

## Priorities

### P0 · release gate

- [x] Automate the release check that source version, README install command, and
  tag agree. The README installs `@latest`, so the install command no longer
  carries a version to go stale. `scripts/check-version.sh` checks the source on
  every pull request and under `make check`, and checks it against the tag when
  one is pushed.
- [x] Add CI for gofmt, vet, normal tests, race tests, a build, and real tmux
  integration.
- [x] Make durable deletion fail closed when a matching tmux pane has malformed
  metadata, refuse deletion from a socket other than its durable binding, and
  retain an outcome-less pending launch whose socket was never recorded.
- [x] Fix organizer-context invalidation and make snapshot/preview polling
  generation-tagged and single-flight.

### P1 · operational reliability

- [x] Decision: do not add a diagnostic log or support bundle. Deep checks print
  bounded health results on demand without creating another sensitive artifact
  lifecycle.
- [x] Surface every marker-bearing pane whose required metadata cannot be parsed
  as a degraded observation instead of silently dropping it. Degraded panes do
  not rewrite durable lifecycle state or accept injected input, while attach,
  capture, and stop remain available.
- [ ] Preserve typed tmux command failures internally if future recovery logic
  needs to distinguish more than the current bounded user-facing error.
- [ ] Make the terminal preview honest about what `capture-pane` can return. A
  full-screen runner on the alternate screen has no scrollback, so `-S -120`
  yields its visible frame plus whatever normal-screen history preceded it.
  Measured against a live `claude` pane: 85 lines returned, 60 of them shell
  output produced before the runner started. Shepherd's own sessions `exec`
  directly and so return the bare frame. Either bound the request to the frame,
  label the preview as a frame rather than a transcript, or both; today the
  request implies a depth the runtime cannot supply.
- [x] Treat a retained dead pane with missing `pane_dead_status` as “outcome
  unknown,” especially on tmux 3.3/3.4. Never default an empty status to zero or
  persist `OutcomeExited` until an exit code is actually observed.
- [x] Replace parallel screen booleans and string edit/action modes with typed
  primary-screen, overlay, and organizer-edit state. Dashboard and organizer now
  share one indexed overview relationship model. Independent lifecycle
  confirmation fields remain intentionally narrow and explicit.
- [x] Add an explicit ordered v1-to-v2 state migration and versioned fixture
  tests before persisting durable session titles. Schema-only migration does not
  advance the domain revision.
- [x] Route current human mutations through a closed typed actor/scope command
  plane. The active authorizer remains local-human-only; session actors stay
  denied until manager grants are designed.
- [x] Resolve native runner argv through a trusted config-backed controller
  resolver instead of accepting executable argv in command actions.
- [x] Add machine-readable `shepherd list --json`, `shepherd spawn --json`, and
  `shepherd send --json` local CLI surfaces.
- [ ] Consolidate the remaining lifecycle confirmation fields into typed state
  only if that makes their transitions materially clearer.

### P2 · hardening

- [x] Extend `shepherd doctor --deep` with bounded state validation,
  lifecycle-lock, tmux socket, permissions, pane-health, mouse, and clipboard
  checks. It writes no diagnostic log.
- [x] Bound prompt and follow-up payload sizes before durable storage or tmux
  transport; use a non-argv transport if large prompts become a requirement.
- [x] Remove a newly created empty artifact directory when workstream state
  creation fails.
- [x] Smoke the actual local install shape, a fresh private home, and a schema-v4
  state from 0.7.9 in both `make check` and CI.
- [x] Route and test `shepherd --version` before dashboard flag parsing.
- [x] Make selected organizer rows valid UTF-8 and width-safe down to one column.
- [x] Add deterministic tests for abandoned artifact reads, coalesced polling,
  stale snapshot/preview completion, and unavailable-session previews.

## Strengths to preserve

- Runner launch and follow-up delivery use argv, `exec`, and tmux buffers rather
  than shell interpolation; integration tests exercise shell-looking input.
- Session identity is durable before launch, reconciliation is conservative,
  and deletion refuses any live or retained tmux pane.
- State writes are validated, locked, private, atomic, file-synced, and
  directory-synced. Ordered schema migrations preserve domain revisions and
  reject invalid, future, or version-inaccurate JSON.
- Settings use a strict JSON schema and argv arrays with private atomic creation.
- Human mutations share a closed typed command boundary, and configured runner
  argv is supplied only by the trusted resolver.
- Artifact context reads are byte/entry/depth bounded, symlink-aware, and
  isolated from registered repository roots.
- Typed UI screen/edit state and one shared overview projection keep dashboard
  and organizer navigation consistent.
- The long-lived tmux server refreshes removed credentials, with a regression
  test proving stale values do not leak into later sessions.
- The package graph is acyclic with domain types in a leaf, and
  `internal/architecture` fails the build on an import that contradicts it.
- Shared presentation helpers and environment variable names each have one
  declaring package, enforced by the same tests.
- CLI verbs take their writers and controller through an `app` struct, so a
  refusal is checked without dialling tmux and every verb is drivable in-process.

## Ordered cleanup

1. Preserve typed tmux errors only when a concrete recovery path needs them.
2. Replace the remaining independent lifecycle confirmation fields only if a
   typed state makes those transitions materially clearer.
3. Split the large UI file further by screen without building a generic
   framework.
4. Decide whether the terminal-frame preview should be cropped further; keep it
   explicitly labeled as a frame rather than history.
