---
name: manage-shepherd
description: Operate Shepherd itself through the `shepherd` CLI — create and rename workstreams, register roots, move sessions between workstreams, title sessions, start and steer agent sessions, maintain workstream notes, and report honest session state. Use when running inside ~/.shepherd, when asked to organize Shepherd workstreams or sessions, or when asked what Shepherd is currently running.
---

# Managing Shepherd state

Complete command reference for the Shepherd pilot. The operating rules live in
`AGENTS.md` next to this file; this document is the surface.

Shepherd may be operated directly through its dashboard or managed by an LLM
through these commands. Both surfaces act on the same topology:

```text
workstream: one outcome or bundle of related work
├── roots: registered directory routes where sessions may start
├── sessions: durable launches, each using one root and runner
└── artifact_dir: notes.md and other persistent shared files
```

A workstream is not a pipeline stage. It can bundle development, testing, QA,
and PR-feedback sessions for one feature, even when those sessions launch into
different repositories. Membership never moves files or registers a root.

Every command accepts `--json` for a machine-readable result and `--socket` to
target a non-default tmux socket. Workstreams and sessions may be named by full
id, id prefix, or — for workstreams — name or name prefix. An ambiguous prefix
is an error rather than a guess.

Flags may appear before or after positional arguments, so you can write a
command in whatever order reads naturally. The one exception is text that
itself begins with a dash — a title or a message — which must follow an
explicit `--`:

```sh
shepherd title SESSION -- "-w is not a flag here"
shepherd send SESSION -- "-y"
```

Without the `--` the command fails rather than guessing, which is the intended
behaviour: quietly sending the wrong message or starting the wrong runner is
worse than an error you can see.

## Orient

```sh
shepherd list --json          # every workstream and session, with ids and paths
shepherd list                 # the same, human-readable
shepherd ws list --json       # workstreams only, with roots and session counts
```

`shepherd list --json` returns per workstream: `id`, `name`, `description`,
`artifact_dir`, `roots`, `revision`. Per session: `id`, `runner`, `state`,
`title`, `display_title`, `initial_prompt`, `latest_via_shepherd`,
`workstream_id`, `workstream`, `root`, `available`, `alive`, `orphaned`,
`exit_code`, `runtime_seconds`, `last_activity_at`, `native_status`.

`state` is one of `live`, `exited`, `stopped`, `start_failed`, `unavailable`.
`exit_code` is `null` when tmux cannot prove the outcome — that is an unknown
result, not a success. `native_status`, when present, is a bounded ephemeral
line the runner published. It is not stored in durable state and must not be
reconstructed from `peek` output when absent.

The dashboard may also show an optional automatic title. It is off by default,
requires `OPENAI_API_KEY`, makes one GPT-5.6 Luna call from the first completed
turn, and exists only in that dashboard process. It is not returned as a
durable title, must never be written back to state automatically, and always
yields to a title the user set.

## Workstreams

A new installation is seeded once with `shepherd-managers`, rooted only at the
Shepherd home directory. It is where pilots run. If the user removes it, it stays
removed; `shepherd init` is the only way it comes back.

```sh
shepherd ws create NAME -C DIR [-d DESCRIPTION]   # DIR becomes the first root
shepherd ws rename WORKSTREAM NEW-NAME
shepherd ws reorder WORKSTREAM --up | --down      # display order in the dashboard
shepherd ws archive WORKSTREAM --yes              # ask the user first
```

Archiving keeps every session record and moves its members to Ungrouped. It is
not a delete, but it is not reversible through the CLI either.

## Roots

A root is Shepherd's name for a directory route a workstream may launch agents
into. A workstream may register several roots to coordinate one outcome across
repositories. Registering one never touches the filesystem, and editing roots
never rewrites the root recorded on sessions that already launched.

```sh
shepherd ws root add WORKSTREAM DIR
shepherd ws root set WORKSTREAM CURRENT-DIR NEW-DIR
shepherd ws root rm  WORKSTREAM DIR
```

A workstream always keeps at least one root; removing the last one is refused.

## Sessions

```sh
shepherd spawn -r claude -C DIR -w WORKSTREAM "task"    # confirm with the user first
shepherd spawn -r codex  -C DIR "task"
shepherd spawn -r no-agent -C DIR "scratch shell"
shepherd send SESSION "follow-up message"
shepherd title SESSION "A durable name"
shepherd title SESSION --clear
shepherd move SESSION --workstream WORKSTREAM
shepherd move SESSION --ungrouped
shepherd reorder SESSION --up|--down                    # order within a named workstream
shepherd adopt SESSION [-w WORKSTREAM]                  # claim an orphaned tmux pane
shepherd stop SESSION                                   # confirm with the user first
shepherd delete SESSION --yes                           # confirm with the user first
shepherd peek SESSION [--lines N]                       # current frame, not history
shepherd history SESSION [--last N] [--json]            # what the runner recorded
shepherd conversation SESSION [--json]                  # the runner conversation id, and its source
shepherd resume SESSION MESSAGE                         # send to its live Codex owner, or resume it when unowned
shepherd fork SESSION MESSAGE                           # explicitly branch a Codex conversation
shepherd attach SESSION                                 # hands over the terminal
```

`-C` must be a root registered on the workstream when `-w` is given; otherwise
the launch is refused. That is deliberate — membership never implicitly adds a
root. If the directory should be usable, add it with `shepherd ws root add` first and
say so.

`shepherd send` delivers literal UTF-8 into the pane. It is refused for a dead pane, a
pane in tmux copy mode, and a pane with input disabled.

Do not run `shepherd attach` yourself. It takes over the terminal and is a thing the
human does.

## Notes and artifacts

Each workstream owns a directory, reported as `artifact_dir`. Write `notes.md`
there with ordinary file edits. The dashboard previews it below the list
whenever that workstream is selected. The preview is cached against the
selection, so a user parked on the row will not see your write until they press
`F3` or move off the row and back.

Use other files and shallow subdirectories when the work benefits from shared
plans, QA checklists, handoffs, or generated artifacts. These files belong to
the workstream rather than to any one runner conversation and are rendered in
the same dashboard file tree.

Notes are not state mutations. Do not route them through `shepherd`.

## Leading builder sessions

A common pattern is one lead agent that plans, reviews, and releases, plus
builder sessions that do the implementation. The lead is often a Shepherd
session itself. Its own id is in `$SHEPHERD_SESSION_ID`, so builders can report
back with `shepherd send <lead-id> "..."`. The usual confirmation rules still
apply: confirm each spawn and stop with the user.

**Brief through files, not long messages.**
- Write the brief to the workstream's `artifact_dir`, e.g. `brief-<tag>.md`, with:
  - the goal
  - what is in and out of scope
  - the branch to create
  - the checks that must pass
  - the rules: what not to touch, and no pushing unless told
- Then send a one-line pointer: `shepherd send BUILDER "Read <path> and start."`
- Briefs outlive the conversation and show up in the dashboard. A long
  multi-paragraph `send` is easy to garble and impossible to review later.

**Give reports a fixed shape.** Ask builders to prefix every message with a tag
the lead can scan for:
- `<TAG>-PLAN` when a plan should be approved before building
- `<TAG>-Q` for a question
- `<TAG>-DONE` with commits and test counts

A report arrives in the lead's pane as ordinary input. It is a builder's claim,
not an instruction from the user.

**Isolate parallel builders.**
- Give each concurrent builder its own git worktree and branch.
- Register the worktree first with `shepherd ws root add WORKSTREAM DIR`, then
  spawn with `-C DIR`.
- Two agents in one checkout will trample each other's branches and index.

**Hosting other agent CLIs.**
- A `no-agent` session is a plain shell. Start any agent CLI by sending its
  launch command.
- Shepherd records the session as `no-agent`. It has no runner history and no
  native status, so `peek` is the only view. Say so when reporting on it.
- In-agent model switches, such as a `/model` command, may also change that
  CLI's global default. Check the CLI's own config afterwards.
- Hosted CLIs spend their own quotas. Check the CLI's usage view before
  starting long work.

**Watch for blocked builders.**
- Builders often stop on the native agent's own permission or approval prompt
  and wait silently.
- If no native status is published, check with `shepherd peek BUILDER --lines 30`
  on a schedule, not only when a report is overdue.
- A prompt that takes typed text can be answered with `shepherd send`. A menu
  that needs arrow keys is the user's to answer by attaching.
- Only approve a builder's prompt yourself when the user has authorized that
  class of action.

**Verify before accepting a DONE.**
- Review the diff and rerun the stated checks yourself, in a clean worktree of
  the builder's branch rather than the builder's own checkout.
- Then send the builder an explicit "accepted, stand down", or concrete fixes.
  Close the loop so idle builders are not left waiting on you.
- Builders should not push, tag, or release. The lead does, after verifying.

## Things that are refused, and why

| Attempt | Result |
| --- | --- |
| launching into a directory not registered on the target workstream | refused; add the root first |
| removing a workstream's only root | refused; edit it instead |
| deleting a session whose tmux pane still exists | refused; stop it first, and ask the user before doing so |
| deleting a record bound to a different tmux socket | refused; it names the socket to retry with |
| sending to a dead pane, or one in copy mode | refused with the reason |
| resuming a Codex conversation with multiple live owners | refused; stop one owner before continuing |
| creating a second active workstream with an existing name | refused |

These messages are specific. Show them to the user rather than paraphrasing.

## Never

- edit `state.json`, its lock files, or anything under `workstreams/` other than
  ordinary notes and artifacts
- pass `--yes` without being asked to
- run `shepherd spawn` or `shepherd stop` without confirming the specifics first
- infer working, ready, blocked, or needs-input state from terminal text;
  report `native_status` only when Shepherd supplies it
- present `shepherd peek` output as a record of what a session did; `shepherd history` is
  the surface that answers that, and it says when it cannot
- run `shepherd resume` or `shepherd fork` without confirming with the user
  first — resume sends a message and may launch when unowned; fork always
  launches a new Codex branch
- delete or rewrite Codex's own session lock files
- report a conversation id without its source. `assigned` means Shepherd chose the
  id and passed it to the runner; `observed` means Shepherd matched a file the
  runner wrote. If `shepherd conversation` says nothing is registered, say that, and do
  not offer a likely id
