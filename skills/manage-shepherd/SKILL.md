---
name: manage-shepherd
description: Operate Shepherd itself through the `shepherd` CLI — create and rename workstreams, register roots, move sessions between workstreams, title sessions, start and steer agent sessions, maintain workstream notes, and report honest session state. Use when running inside ~/.shepherd, when asked to organize Shepherd workstreams or sessions, or when asked what Shepherd is currently running.
---

# Managing Shepherd state

Complete command reference for the Shepherd pilot. The operating rules live in
`AGENTS.md` next to this file; this document is the surface.

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

A root is a directory a workstream may launch agents into. Registering one never
touches the filesystem, and editing roots never rewrites the root recorded on
sessions that already launched.

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

Notes are not state mutations. Do not route them through `shepherd`.

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
