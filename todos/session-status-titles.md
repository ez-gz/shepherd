# Session status, titles, and recency

Status: durable titles, optional ephemeral automatic titles, and authoritative
native status are shipped. Returned/seen attention state remains deferred until
stable completed-turn identities are part of the projection.

Current implementation: state schema v4 retains the optional user-owned session
title introduced in v2. Rows render a **brief** whose lead is that title, then
an opted-in process-local automatic title, falling back to the initial prompt. Its detail
prefers bounded runner-published native status, then transcript activity and the
latest user message successfully sent through Shepherd. Native status and
automatic titles never enter `state.json`.

The brief's source interface is where the observer below reaches the screen: a
`BriefSource` reads a cache and never blocks, so an observer that owns its own
cadence can fill either slot without the row learning about it. See the brief
section of `docs/DESIGN.md`, and `todos/brief-sources.md` for the configurable
and external sources that remain unbuilt.

The shipped signal is fixed at launch. Claude receives a session-local status
line plus lifecycle hooks; Codex receives a session-local terminal-title layout.
Tmux metadata projects it into `Session.NativeStatus`, with a marker required
before Codex pane titles are trusted. Transcript activity remains only a
fallback and never becomes current native status.

## Product goal

Make the dashboard answer three different questions without conflating them:

1. Is the native process alive?
2. What is the agent doing in its current turn?
3. Is there a returned result the user has not opened yet?

At the same time, let users give sessions durable titles and show the most
recent message routed through Shepherd instead of treating the immutable launch
prompt as the session's forever-label.

## Honest status model

Keep three independent axes.

**Runtime lifecycle observation** remains process truth owned by `Supervisor`;
the controller owns the joined lifecycle projection shown by the UI:

- `starting`
- `live`
- `exited`
- `stopped`
- `start_failed`
- `unavailable`
- `degraded` (the pane exists, but required Shepherd metadata is unreadable)

**Agent turn state** comes only from a fresh, backend-specific observation:

- `working`: a turn is active;
- `ready`: the CLI can accept another message;
- `needs_input`: an explicit approval, question, or input request; and
- `unknown`: no trustworthy fresh observation.

**User attention state** distinguishes a newly completed result:

- `returned`: a completed turn has not been opened; and
- `seen`: that completion has been acknowledged.

The UI precedence is `needs you`, `working`, `returned`, `ready`, then
`live · unknown`. Terminal runtime states always win. Stale observations fall
back to `unknown`, but do not erase a previously recorded unread completion.

Tmux silence is not semantic evidence. Until a runner observer exists, render
facts such as `active 12s ago`, `quiet 3m`, or `attached`; never translate those
signals into `working`, `ready`, `returned`, or `needs input`.

## Slice A · durable session titles

Shipped in V0.3.4. Optional user-owned `SessionRecord.Title` is durable display
identity, not process state, and never renames the stable tmux session or native
provider conversation.

Completed:

1. [x] Add an explicit v1-to-v2 state migration with versioned JSON fixture
   tests. Schema-only migration preserves the domain revision.
2. [x] Add optional `SessionRecord.Title` and a controller
   `SetSessionTitle` action; an empty value clears the title.
3. [x] Make `r` contextual in F3: rename a workstream header or edit/clear a
   session title.
4. [x] Render title first, falling back to a one-line initial prompt. Keep the
   latest Shepherd-routed message as secondary detail when space permits.
5. [x] Keep title metadata out of tmux names, native provider identity, and
   `Supervisor` process truth.

## Deferred presentation activity

Keep recency and acknowledgement metadata separate from `SessionRecord`, keyed
by session ID:

```text
SessionActivity
  last_shepherd_user_preview
  sent_at
  truncated
  acknowledged_turn_id
```

This is bounded presentation metadata, not chat history or an event log. The
preview should be normalized to one line and capped near 280 characters. Write
it only after `Supervisor.Send` succeeds. If the send succeeds but persistence
fails, report the metadata failure and never retry the message automatically.

Call the field **latest via Shepherd** in the UI. Messages typed directly inside
an attached Claude or Codex TUI bypass Shepherd and cannot be claimed as known.
The initial prompt remains the fallback when no later preview exists.

## Dashboard and organizer UX

The title-first layout and contextual organizer rename behavior below shipped
in V0.3.4. The semantic state labels in the example remain the future target:

A dashboard row should prioritize attention state and the user-owned title:

```text
● working    Fix flaky OAuth tests
              latest via Shepherd · “also update the release notes”

◆ returned   Release Linux build
              returned 42s ago
```

Use the explicit title when present; otherwise use the optional process-local
automatic title, then the initial prompt. At narrow widths, keep status and
title before runner details or the message preview.

The details pane should show title, agent state, runtime activity, latest via
Shepherd, initial task, cwd, and the existing terminal preview.

In the F3 organizer, `r` now means **rename the selected noun**:

- on a workstream header, rename the workstream;
- on a session row, edit its title; and
- saving an empty session title clears it and restores the prompt-derived label.

Returned/seen acknowledgement is not implemented. When it is, selecting a row
must not acknowledge a returned result. Attaching/opening it with Enter, or
successfully sending the next message, can use the acknowledgement path.

The title slice did not add this table. The retained tmux preview already
provides the cheap latest-message shim; durable recency and acknowledgement can
wait until their lifecycle is needed.

## Slice B · real agent status — shipped

The retained-pane prerequisite still applies: process outcome and native status
remain separate. The implemented status source is runner-owned and ephemeral.
Claude publishes metrics and lifecycle edges through session-local settings;
Codex publishes its documented terminal-title fields. Inputs are sanitized,
single-line and capped at 240 runes, and old sessions are never rewritten.

### Deferred richer observer seam

Real turn status must come from reliable Claude- or Codex-specific signals, not
terminal scraping heuristics. Keep the observer above `Supervisor` and make it
batch-oriented:

```go
type AgentObservation struct {
	State           AgentState
	TurnID          string
	CompletedTurnID string
	StateSince      time.Time
	ObservedAt      time.Time
	FreshUntil      time.Time
	Source          string
}
```

A future backend-selected `AgentObserver.Observe` could return observations for
session references. The controller projection would combine those with runtime
and durable presentation metadata. The shipped native status intentionally does
not claim stable turn IDs or returned/seen state.
`CompletedTurnID` is the stable token used to determine whether a return is
unread.

Potential sources include Claude hooks or structured agent output and Codex
notifications or app-server events. Choose the first source only after proving
that its turn-start, turn-complete, and input-request signals are reliable.

## Acceptance order

1. [x] Ship Slice A: migration, title set/clear, contextual F3 rename,
   rows/details, and downgrade/invalid-state tests.
2. [x] Represent dead panes with unknown outcomes honestly and stop persisting
   guessed success.
3. [x] Probe authoritative Claude and Codex signals and choose launch-time
   publishers rather than transcript or terminal inference.
4. [x] Project runner-published `Ready`, `Working`, `Error`, and `Done` status
   separately from terminal outcomes.
5. [x] Add `Needs input` from explicit Claude permission, elicitation, and
   input-request hook events.
6. [ ] Add durable `returned`/`seen` acknowledgement only when a backend
   supplies stable completed-turn IDs.

Native labels and both title forms are shipped. The `returned` badge remains
deferred until Shepherd has stable completion identities it can trust.
