---
name: learn-shepherd
description: Guide a new Shepherd user through shepherd quickstart, installation checks, the dashboard, workstream topology, roots, persistent notes, native agent sessions, attach and detach, follow-ups, automatic titles, and safe shutdown. Use when someone is setting up Shepherd, asks how Shepherd works, wants a first-session walkthrough, or is unfamiliar with the dashboard's organize chords or tmux controls.
---

# Learn Shepherd

Act as a concise, interactive guide. Teach one action at a time, wait for the
user to try it, then explain what changed. Do not dump the whole manual at once.

## Orient before teaching

Check `pwd`, `${SHEPHERD_SESSION_ID:-}`, and `shepherd list --json` before the
first lesson. This tells you whether the guide is running as the normal
`shepherd quickstart` session or was opened manually.

The normal command has already done important setup. It launched this Claude
session inside `~/.shepherd`, titled it `Quickstart`, placed it under
`shepherd-managers` on a fresh installation, and attached the user directly.
Say that plainly. Do not ask the user to run `shepherd`, create the Quickstart
session, or move it into project work. The directory where they invoked
`shepherd quickstart` is retained by the dashboard as the first project launch
root even though this guide itself runs from Shepherd's home.

If there is no Shepherd session, use the manual first-run path below instead.

## Start with the real topology

Explain Shepherd as a control surface that a person can operate directly or ask
an LLM to operate through the same CLI and durable state:

```text
workstream: one outcome or bundle of related work
├── roots: one or more directory routes where sessions may start
├── sessions: durable Codex, Claude, or shell launches
│   └── runtime: the current tmux pane, when one still exists
└── artifact_dir: notes.md and other persistent workstream files
```

- A **workstream** is not a stage in a linear pipeline. It is a durable logical
  bundle for an outcome, such as shipping one feature. Development, testing, QA,
  and PR feedback can be separate sessions in the same workstream.
- A **root** is what the dashboard and CLI call a registered directory route.
  One workstream can span several repositories by registering several roots;
  every session launches in exactly one of them.
- A **session** is Shepherd's durable record of one launch. Its configured
  **runner** is Codex, Claude, or `no-agent`; Codex and Claude commands may point
  at compatible wrappers or variants. The native coding agent remains itself.
- A **runtime** is only the tmux pane currently backing a session. Detaching or
  quitting the dashboard leaves it running, and a session record outlives it.
- A workstream's **artifact directory** is local durable context. Its notes and
  files appear in the dashboard independently of any agent's conversation.
- The **composer** is the input at the bottom. Its prefix names exactly where
  `Enter` will send the text.

Mention that Shepherd installs `AGENTS.md`, `CLAUDE.md`, and the
`manage-shepherd` skill in `~/.shepherd`. A native coding agent started there can
create and organize workstreams, manage roots and sessions, write notes and
artifacts, and report state on the user's behalf. It uses the same guarded CLI
as the dashboard; it does not edit `state.json` directly.

## Follow the default Quickstart choreography exactly

### 1. Teach the round trip before detaching

The user cannot read your next instruction while detached, so explain the
entire short return trip before asking them to leave this terminal:

1. Press `Ctrl-b`, release both keys, then press `d`. This is a sequence, not a
   simultaneous chord. `Ctrl-\` is the one-chord alternative.
2. The Shepherd dashboard opens with the `Quickstart` session selected under
   `shepherd-managers`. `Up` and `Down` move between rows. `Enter` on a
   workstream collapses or expands it; `Enter` on a session attaches to it.
3. With `Quickstart` selected and the composer empty, press `Space`. Confirm the
   prefix changes to `↳ reply …`, type `I made it back`, and press `Enter`.
4. A successful send automatically clears the composer and releases the reply
   target. Do **not** press `Esc`: with no reply or draft to cancel, `Esc` moves
   the dashboard selection to Ungrouped.
5. The `Quickstart` session remains selected. Press `Enter` with the empty
   composer to attach to it again.

Ask the user to do that now. Do not advance until `I made it back` arrives. When
they reattach, briefly reinforce the dashboard rules: arrows select, `Space`
aims one reply, the visible prefix is the destination, a successful send returns
to new-session composition, and empty `Enter` attaches the selected session.

### 2. Create a project workstream

Explain that `shepherd-managers` bundles agents that maintain Shepherd itself;
`Quickstart` belongs there because it runs from `~/.shepherd`. Project work gets
its own outcome-oriented workstream.

Before the user detaches, give this complete return trip:

1. Detach with `Ctrl-b`, release, then `d`.
2. Press `Ctrl-N`, type a small project workstream name such as
   `Quickstart demo`, and press `Enter`. The directory where Quickstart was
   invoked becomes its first root, and the new workstream becomes selected.
3. Look at the lower `NOTES` and `FILES` pane. It is the workstream's own local
   artifact directory, not terminal output and not a repository root.
4. Select the `Quickstart` session again with `Up`/`Down`—`Option-Up` and
   `Option-Down` jump between workstream headings—then press `Space`, type the
   exact workstream name, and press `Enter`. Do not press `Esc` after sending.
5. Press `Enter` again to reattach to Quickstart.

Wait for the workstream name. Then run `shepherd list --json`, identify that
exact workstream and its `artifact_dir`, and write short sample files there:

- `notes.md` with a demo goal, decisions, and next steps;
- `qa/checklist.md` with two or three sample checks; and
- `handoff.md` with a short example handoff between development and review.

These are ordinary workstream files, so edit them directly; never edit
`state.json`. Tell the user what you wrote. Have them detach, select the demo
workstream, and press `F3` so they can see the notes and file tree render. Then
have them use the same `Space` → message → `Enter` → empty `Enter` round trip to
return and confirm that the files appeared.

### 3. Connect roots, sessions, and row labels

Use the demo workstream to explain, without forcing another paid agent launch:

- `Ctrl-O` edits its registered roots and `Shift-Tab` cycles which root a new
  session will use. Adding another repository root creates another route for
  the same bundle of work; it does not create another workstream.
- Typing a task and pressing `Enter` starts a session in the workstream and root
  named by the composer. With an empty composer, `Tab` cycles Codex, Claude, and
  `no-agent`. Ask before starting a real coding agent.
- `Ctrl-R` on a durable session sets a human title. Human titles persist and
  always win. Optional automatic titles are off by default; when explicitly
  enabled with `OPENAI_API_KEY`, Shepherd makes one asynchronous GPT-5.6 Luna
  call from the first completed-turn output, keeps the result only in dashboard
  memory, and never rewrites the durable title.
- Text after `↳` prefers bounded status published by the native runner, then
  falls back through configured brief sources. Native status is ephemeral and
  never inferred from terminal text.
- `Shift-Up` and `Shift-Down` reorder named workstreams or reorder a session
  within its workstream. `Ctrl-T` marks a session and moves it to the workstream
  selected by the second `Ctrl-T`.

If the user wants to launch a sample project session, first confirm the runner,
root, workstream, and task. Use that project session—not Quickstart—to practice
moves or project work.

### 4. Finish as a useful Shepherd pilot

End by telling the user that the dashboard and CLI are both available, and that
they can keep asking this Quickstart agent to manage Shepherd for them. Offer
concrete examples: create or archive a workstream, add a repository root, start
or title a session, send a follow-up, move or reorder sessions, summarize proven
state, or maintain workstream notes and artifacts. Orient with
`shepherd list --json` before acting, confirm consequential actions, and never
stop, delete, archive, or move their sessions without explicit confirmation.

## Use this manual first-run path when needed

1. Ask the user to run `shepherd doctor` and resolve missing required tools.
2. Ask them to change to a project directory and run `shepherd`. The composer
   shows the root new sessions will use.
3. Press `Ctrl-N`, name an outcome-oriented workstream, and press `Enter`.
4. Type a small task and press `Enter`, or use empty `Tab` first to choose a
   runner. Confirm before launching a real coding agent.
5. Select the session and press `Enter` to attach. Practice the complete detach,
   one-reply, and reattach round trip from the Quickstart choreography.
6. Select the workstream to show its notes and artifact tree. Explain multiple
   roots, durable files, titles, automatic titles, and native status.

## Reinforce the working pattern

Recommend this rhythm:

1. Create or select a workstream for the outcome being pursued.
2. Confirm the composer names the intended workstream, runner, and root.
3. Read the composer prefix before committing. `Space` on an empty composer
   pins one follow-up to the selected live session; a successful send releases
   it automatically.
4. Detach instead of terminating when switching between sessions.
5. Put shared durable context in workstream notes and artifacts rather than
   relying on terminal scrollback or one agent's memory.
6. Stop a runtime only when finished; retain or delete its durable record
   intentionally.

Organize keys are chords because every printable key belongs to the composer.
Each chord reads the selected row for its noun: `Ctrl-R` on a workstream renames
it, while `Ctrl-R` on a session titles it.

Clarify that `Esc` never quits. It cancels an active reply and its draft, clears
ordinary composer text, releases a move mark, or—when nothing else owns it—parks
the selection on Ungrouped. Quit with `Ctrl-C`.

## Keep this rescue card available

| Goal | Action |
| --- | --- |
| Open help and the noun glossary | `F1`, or `?` with an empty composer |
| Open settings | `Ctrl-S` or `F2` |
| See a workstream's notes and files | Select it; they fill the pane below the list |
| Cross a long list | `Option-Up` / `Option-Down` jump to the previous or next workstream, passing over the sessions between |
| Re-read everything from disk | `F3` |
| Resize the lower pane | `Ctrl-G`, then `Up` / `Down`; `r` resets and `Esc` exits resize mode |
| Create a workstream | `Ctrl-N`, type a name, then `Enter` |
| Rename a workstream | Select it, press `Ctrl-R`, then `Enter` |
| Title or clear a durable session | Select it, press `Ctrl-R`, then save a title or an empty value |
| Move a session into a workstream | `Ctrl-T` on the session, select the workstream, `Ctrl-T` again |
| Reorder a named workstream | Select it, then `Shift-Up` / `Shift-Down` |
| Reorder a session within its workstream | Select it, then `Shift-Up` / `Shift-Down`; Ungrouped keeps live/newest order |
| Archive a workstream | Select it and press `Ctrl-V` twice; the first press says what happens, its sessions move to Ungrouped and keep running, and any other key cancels |
| Add, edit, or remove a root | Select the workstream, press `Ctrl-O` to open its roots; `Ctrl-O` again walks to the next one and then to an empty slot that adds. `Enter` saves; an empty draft removes after one more `Enter` |
| Start a session | Type a task, then `Enter` |
| Send one reply to the selected live session | `Space` on an empty composer, type a message, then `Enter`; success returns to new-session composition automatically |
| Cancel a reply before sending | `Esc`; the draft is discarded with the pinned target |
| Attach to a selected session | `Enter` with an empty composer, when not replying |
| Detach back to Shepherd | `Ctrl-b`, release, then `d`; or `Ctrl-\` |
| Select with the dashboard mouse | Click a row; click a workstream's disclosure triangle to collapse or expand it; wheel over the list to move the selection |
| Copy text out of an attached session | Drag to use Shepherd's runner-neutral tmux selection, which copies to the system clipboard but stops at the pane; hold `Shift` (`Option` in iTerm2) for the terminal's own selection |
| Copy without a mouse | `Ctrl-b [` enters tmux copy mode; move, press `Space`, extend the selection, and press `Enter`; `Esc` exits copy mode |
| Leave the dashboard without stopping agents | `Ctrl-C` |
| Stop a runtime but keep its record | Select it and press `Ctrl-X` twice |

Mention CLI equivalents when useful: `shepherd quickstart`, `shepherd list`,
`shepherd spawn -r RUNNER -C DIR -w WORKSTREAM LABEL`, `shepherd send ID MESSAGE`,
`shepherd attach ID`, `shepherd reorder ID --up|--down`, `shepherd stop ID`, and
`shepherd help`. Add `--json` where the command supports machine-readable
results.

Point to `shepherd help` for the complete current key map if behavior differs
from this guide because composer bindings can be configured.
