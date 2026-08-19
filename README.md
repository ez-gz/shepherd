# shepherd

Shepherd is a fast terminal dashboard for parallel native coding agents. It starts
real Codex and Claude Code sessions inside a private tmux server, organizes them
into durable workstreams, lets you send follow-ups, and hands your terminal
directly to the native agent UI when you attach.

Tmux remains the runtime supervisor and the coding-agent CLIs remain the native
runners. Shepherd adds a small durable organization layer without introducing a
daemon, manager agent, task graph, or replacement execution engine.

## Nouns

- **Workstream** — a durable named grouping for roots, sessions, notes, and
  artifacts; it provides organization, not autonomy.
- **Session** — a durable launch identity with an optional user title, initial
  task, root, runner, and outcome; it persists beyond its runtime.
- **Runtime** — the tmux pane associated with a session and the source of its
  current process observations.
- **Root** — an explicit launch working directory registered on a workstream.
- **Runner** — the `codex`, `claude`, or `no-agent` command integration used to
  launch a native agent or shell.
- **Composer** — the dashboard input bar used to start sessions and send
  follow-up messages. Its prefix names the destination `Enter` will commit to.
- **Brief** — the one-line summary in the middle of a session row. Its lead is
  the session's title, initial task, or runner; the text after `↳` is what the
  runner last recorded the session doing, falling back to the latest message
  sent through Shepherd. A leading `~` marks text Shepherd derived rather than
  observed.
- **Ungrouped** — durable sessions with no active workstream membership.
- **Orphaned** — tmux panes carrying a Shepherd ID unknown to durable state; they
  are never silently adopted.

## Install

Shepherd currently targets macOS. Requirements: Go 1.25+, tmux 3.3+, and at
least one of `codex` or `claude`. Runner commands can be configured when they
are not on `PATH`; Shepherd also discovers Codex inside the macOS ChatGPT app
bundle.

```sh
go install github.com/ez-gz/shepherd/cmd/shepherd@latest
shepherd doctor
```

Run `shepherd doctor --deep` when setup or terminal interaction looks wrong. It
validates durable state, locks, private permissions, the tmux socket, pane
metadata, mouse selection, and clipboard integration without writing diagnostic
logs or reading prompts from terminal output.

`@latest` resolves to the newest release tag, so this command never goes stale.
Substitute an explicit tag when you need a particular release.

Go does not run package-defined post-install hooks. After successful checks,
`shepherd doctor` prints the next step: `shepherd quickstart`.

Ensure `$(go env GOPATH)/bin` is on your `PATH`. To install `shepherd` together
with the `s` and `S` aliases, build from source instead:

```sh
git clone https://github.com/ez-gz/shepherd.git
cd shepherd
make install
```

`make install` writes `shepherd` to `~/.local/bin` and adds `s` / `S` symlinks.
Override the destination with `make install PREFIX=/somewhere`.

For an agent-guided first run, run:

```sh
cd ~/code/my-project
shepherd quickstart
```

On first use Shepherd installs the [`learn-shepherd` guide](skills/learn-shepherd/SKILL.md)
as `~/.shepherd/QUICKSTART.md`. This command starts a real durable Claude session
inside `~/.shepherd`, gives it the title `Quickstart`, points it at that file,
and attaches immediately. Because the session starts in Shepherd's own home, it
also sees the installed agent contract and CLI skill instead of carrying either
one in an oversized launch prompt. After you detach, the dashboard keeps the
directory where you invoked `shepherd quickstart` as the launch root for your
first project workstream.

The guide's first lesson is how to detach. Press `Ctrl-b`, release both keys,
then press `d`; `shepherd quickstart` will open the dashboard with the guide selected
so it can walk you through sending a follow-up, reattaching, workstreams,
and persistent notes.

## Use it

Run `shepherd` (or its `s` / `S` aliases) from the directory that new agents
should use as their root:

```sh
cd ~/code/my-project
shepherd
```

The composer is always ready, and it picks its destination *before* you type
rather than when you commit. An empty composer starts a new session; `Space`
aims it at the selected live session and pins that target. Either way the
prefix names where the text is going and `Enter` sends it there, so the commit
key never depends on remembering which one you meant:

| Action | Result |
| --- | --- |
| Type a task or label, then `Enter` | Start the chosen Codex, Claude, or `no-agent` session |
| `Space` with an empty composer | Aim the composer at the selected live session; the prefix becomes `↳ reply …` |
| Type a message, then `Enter` | Send it to the pinned session. The selection is held there while you draft, so the pane below keeps showing the conversation you are answering. Once it lands, the composer returns to composing a new session — press `Space` again to follow up |
| `Esc` while replying | Return to composing a new session, discarding the draft with it so the next `Enter` cannot spawn a session from a follow-up |
| `Shift-Enter` | Insert a newline; `Ctrl-J` is the fallback for terminals that cannot distinguish shifted Enter |
| `Option-Left` / `Option-Right` | Move by word; `Option-Delete` deletes the previous word |
| `Command-Left` / `Command-Right` | Move to the start or end of the logical line |
| `Command-Up` / `Command-Down` | Move to the start or end of the whole draft |
| `Tab` | Cycle Codex → Claude → `no-agent`, with or without composer text |
| `Shift-Tab` | Cycle the selected workstream's explicit roots, with or without composer text |
| `F1`, or `?` with an empty composer | Open scrollable help, including the noun glossary and current composer keys |
| `Ctrl-S` or `F2` | Open settings; `e` edits JSON, `r` reloads, `Esc` returns |
| `F3` | Re-read sessions, the terminal preview, and the selected workstream's notes and files |
| `Ctrl-N` | Create a workstream, named through the composer |
| `Ctrl-R` | Rename the selected workstream, or edit/clear the selected session's durable title |
| `Ctrl-T` | Mark the selected session for a move; on a workstream, move the marked session there or adopt an orphan |
| `Ctrl-V` twice on a workstream | Archive it off the dashboard. The first press names what happens and the second does it; its sessions move to Ungrouped and keep running, and any other key cancels |
| `Shift-Up` / `Shift-Down` | Reorder a named workstream, or reorder a session within its named workstream; Ungrouped keeps live/newest order |
| `Up` / `Down` | Select a workstream or session, or move between multiline composer rows |
| `Option-Up` / `Option-Down` | Jump the selection to the previous or next workstream, passing over the sessions between |
| Click a dashboard row | Select it without attaching or performing an organizing action; click its disclosure triangle to collapse or expand a workstream |
| Wheel over the dashboard list, settings, or help | Move that viewport; mouse input never commits composer text or confirms a destructive action |
| `Ctrl-G` | Enter resize mode; `Up` grows the lower pane, `Down` shows more sessions, `r` resets, and `Esc` exits |
| `Enter` on a workstream | Collapse or expand its sessions |
| `Enter` on a session | Attach its native terminal; inactive while replying, so it cannot attach to a row other than the pinned target |
| `Ctrl-\` or `Ctrl-b d` while attached | Detach back to Shepherd |
| Drag while attached | Select and copy through tmux, independent of which runner owns the pane |
| `Shift-drag` while attached | Bypass tmux and select with the terminal, crossing panes and taking whole lines; iTerm2 uses `Option` for this |
| `Ctrl-b [` while attached | Enter keyboard copy mode; move to the start, press `Space`, extend the selection, and press `Enter` to copy; `Esc` exits |
| `Ctrl-X` twice | Stop/remove a present runtime; once no pane remains, press twice again to delete its durable record |
| `Esc` | Leave a reply and discard its draft, then clear the composer, then release a move mark, then select Ungrouped |
| `Ctrl-C` | Quit the dashboard; `Esc` never quits |

The terminal application decides whether macOS modifier chords reach a TUI.
Shepherd accepts enhanced Option/Command events plus common Alt, Home/End, and
Ctrl-key fallbacks. If a terminal reports `Shift-Enter` as ordinary Enter, use
`Ctrl-J` for a newline.

Inside an attached runner the same chords are tmux's business rather than
Shepherd's, and Shepherd settles them for you: the private server encodes modified
keys for every pane instead of letting each runner negotiate. Claude Code asks
for a scheme tmux implements and Codex asks for one it does not, and a pane that
falls back to the legacy encoding receives `Shift-Enter` as a plain Enter — so
the key meant to open a line sends the message instead. A pane reads this when
it starts, so a session launched before 0.7.0 keeps the old behaviour until you
restart it. A tmux too old to offer the encoding keeps the behaviour it had;
nothing else about the session changes.

The mouse has one Shepherd contract instead of a runner-specific one. On the
dashboard, click selects a row, its disclosure triangle collapses or expands a
workstream, and the wheel moves the list, settings, or help viewport. A click
never attaches, commits text, organizes, or confirms a destructive action.

Attached, tmux owns every drag even when the runner requested mouse events, so
the same gesture selects and copies in Codex, Claude, and a shell. Plain clicks
and wheel events remain available to a runner that implements them; otherwise
tmux handles them. The selection stops at the pane edge and includes whatever
borders the runner drew. On macOS the drag-end copy goes directly through
`pbcopy` as well as tmux's paste buffer and portable terminal-clipboard path, so
a terminal that blocks OSC 52 is not a silent failure. Hold `Shift` to hand a
dashboard or attached-session drag back to the terminal for native, cross-pane
selection. iTerm2 spells that bypass modifier `Option`. For a mouse-free path,
`Ctrl-b [` enters copy mode; move to the start, press `Space`, extend the
selection, and press `Enter`. Those keys are the same under tmux's vi and emacs
mode tables.

Every full-screen surface carries an unmistakable mode badge: **Dashboard**,
**Settings**, or **Help**. Organizing happens on the dashboard rather than in a
view of its own.

Each session row carries a **brief**: the durable user title when one is set,
then an optional ephemeral automatic title, otherwise a one-line initial task,
followed after `↳` by what the session is doing. The two halves get separate
width budgets, so a long title cannot crowd the second out and a narrow row
drops it rather than showing a fragment of it. Rows omit the label that names
the field — twenty columns the text itself can use — and the details pane below
still spells it out.

That second half is a real status line, not a restatement of what you typed:

```text
● claude  a1b2c3  live   Fix flaky OAuth tests   ↳ Working · Opus · high · 18% ctx
● codex   d4e5f6  live   Release the Linux build ↳ Working · gpt-5.6-codex · 42% ctx
● claude  9a8b7c  live   Rewrite the retry loop  ↳ ~editing retry.go
```

For sessions launched by this version, the runner publishes a bounded native
status. Claude receives a session-local status line and lifecycle hooks; Codex
receives a session-local terminal-title layout. Shepherd projects that data
through tmux without writing it to `state.json` or inferring it from terminal
text. Older live sessions keep the launch contract they started with and fall
through to transcript activity or the latest message sent through Shepherd.

The `~` is not decoration: transcript activity is a phrase derived from another
program's records, so it is marked as something Shepherd was told rather than
directly observed. Text entered directly in an attached native terminal remains
outside Shepherd's message history.

Which sources fill a brief is a single ordered layout you can change in
settings, so a row can show only your title, or drop the activity line, or carry
your own program's output. Anything a source cannot prove from durable state or
a tmux observation renders with a leading `~`, the same way an exit code tmux
cannot prove is reported as unknown rather than as zero.

Leaving the dashboard never stops an agent. Exited and failed panes remain
inspectable while tmux retains them. Stopping removes the runtime but preserves
the durable session record and its workstream history; deletion is offered only
after no tmux pane remains. Deletion fails closed: it checks stable tmux identity
without relying on rich pane metadata, refuses records bound to another socket,
and retains an interrupted pending launch when its original socket is unknown.

The same primitives are available without the TUI:

```sh
shepherd spawn -r claude -C ~/code/project "Investigate the flaky test"
shepherd spawn -r codex -C ~/code/project -w "Core" "Implement the fix"
shepherd spawn -r no-agent -C ~/code/project "scratch shell"
shepherd list
shepherd send a1b2c3 "Also check whether the retry hides the root cause"
shepherd list --json
shepherd spawn --json -r codex -C ~/code/project "Machine-readable launch"
shepherd send --json a1b2c3 "Machine-readable delivery result"
shepherd attach a1b2c3
shepherd stop a1b2c3
```

Every organizing action is also a command, so the whole durable model can be
driven without the TUI:

```sh
shepherd ws create "API work" -C ~/code/api -d "the public API"
shepherd ws list --json
shepherd ws root add "API work" ~/code/api-client
shepherd ws rename "API work" "Public API"
shepherd ws reorder "Public API" --up
shepherd title a1b2c3 "OAuth retry investigation"
shepherd move a1b2c3 --workstream "Public API"
shepherd reorder a1b2c3 --up
shepherd move a1b2c3 --ungrouped
shepherd adopt a1b2c3 -w "Public API"
shepherd peek a1b2c3
shepherd history a1b2c3 --last 10
shepherd conversation a1b2c3
shepherd resume a1b2c3 "Pick this back up and finish the retry work"
shepherd fork a1b2c3 "Try the alternate design in a new Codex branch"
shepherd ws archive "Public API" --yes
shepherd delete a1b2c3 --yes
```

Workstreams and sessions accept a full id, an id prefix, or a workstream name;
an ambiguous prefix is an error rather than a guess. Flags may appear before or
after positional arguments. `shepherd ws archive` and `shepherd delete` require an explicit
`--yes`.

`shepherd list --json` returns a machine-readable projection of workstreams and
sessions, including durable/display titles, latest-via-Shepherd text, bounded
`native_status`, runtime availability, a stable process-state enum, and an
`exit_code` that is `null` when tmux cannot prove the outcome. Every command
above accepts `--json` and returns a machine-readable result. These are local
human CLI surfaces; they do not enable manager authority.

## Resuming a conversation

A tmux pane is mortal. The conversation inside it is not: both runners write it
to disk and can continue it later by id. Shepherd registers that id on the session
automatically, so `shepherd resume` picks the work back up instead of restarting it
cold.

```sh
shepherd conversation a1b2c3    # the runner conversation id, and how Shepherd knows it
shepherd resume a1b2c3 "Pick this back up and finish the retry work"
```

For Codex, resume enforces one live writer per native conversation. If a live
Shepherd session already owns it, the message is sent to that pane. If it is
unowned, Shepherd starts `codex resume` in a new durable session. Multiple live
owners are refused rather than compounded. Creating a branch is a separate,
explicit operation:

```sh
shepherd fork a1b2c3 "Try the alternate design"
```

The unowned resume path leaves the original record exactly as it was, because
it is the durable account of what already happened. Shepherd never removes or
rewrites Codex's own lock files.

How the id is known differs by runner, and Shepherd reports which case it is
rather than presenting them as the same fact:

| | id chosen at launch | resume by id | Shepherd records it as |
| --- | --- | --- | --- |
| **Claude** | yes, `--session-id` | yes, `--resume` | `assigned` |
| **Codex** | no such flag | yes, `codex resume` | `observed` |

For Claude the id is a fact Shepherd caused: Shepherd already launched
`claude --session-id <durable id>`, so the conversation id *is* the session id
and nothing has to be looked up.

Codex mints its own id and offers no way to set it, so Shepherd learns it by
matching what Codex wrote. A rollout under `~/.codex/sessions` must agree on
three things before it is accepted: the launch directory, a start time inside
the match window, and the **verbatim initial prompt**. The prompt is what makes
this evidence rather than a guess — running several agents in one repository at
once is the point of Shepherd, so directory and time alone routinely describe more
than one session.

Anything other than exactly one match is refused. If no rollout matches, or if
two are genuinely indistinguishable, Shepherd records nothing and says so:

```text
$ shepherd conversation a1b2c3
no conversation registered for a1b2c3 (codex): more than one codex rollout
matches this session's launch directory, start time and initial prompt, so
Shepherd cannot tell which conversation is this one
```

That is a deliberate refusal, not a gap to be filled by picking the closest
match. A wrong id resumes someone else's work while looking exactly as
confident as a right one.

## The pilot

Because that command surface is complete, an ordinary agent can maintain
Shepherd's state. Shepherd writes the instructions for one into `~/.shepherd`:

```text
~/.shepherd/
  AGENTS.md                       operating contract, read by Codex and Claude
  CLAUDE.md                       pointer to AGENTS.md
  QUICKSTART.md                   interactive first-use guide read by Claude
  skills/manage-shepherd/SKILL.md   the full command reference
```

A new installation is also seeded with one workstream named `shepherd-managers`,
rooted only at `~/.shepherd`, so there is somewhere to launch pilots from the
dashboard without building it by hand. `shepherd quickstart` performs that
one-time provisioning before it records the Quickstart session, and places the
session in the workstream.

It is seeded only on an installation that has never written durable state, and
the state file is what marks that: reads never create it and no-op mutations
never write it. So deleting or archiving the workstream keeps it deleted, and an
installation you have already organized is never seeded behind your back. `shepherd init`
is the explicit way to create it, or to get it back.

Start a pilot by running an agent in that directory:

```sh
cd ~/.shepherd && claude
```

Codex works the same way. The instructions are deliberately vendor-neutral: the
contract lives in `AGENTS.md`, and `CLAUDE.md` only points at it, so there is
one source of truth rather than two that can drift.

Then ask for what you want in words — "make a workstream for the API work and
move those three sessions into it", "register ~/code/api-client as a root",
"what's running right now?" — instead of remembering which key does it.

Those files are installed on first run and are **never overwritten**, so house
rules you add to `AGENTS.md` survive upgrades. `shepherd init --force` refreshes them
from a newer binary.

The pilot is an ordinary agent with a shell, not a privileged one. It acts as
you, through the same CLI and the same command plane, and holds no grant of any
kind. Scope comes from what the instructions teach and which verbs require
`--yes`, which is a guardrail against mistakes rather than a security boundary.

## Workstreams

The main page is a workstream projection. Named workstreams appear first,
followed by two honest system groups:

- **Ungrouped** contains durable sessions with no membership and preserves the
  original raw-session workflow.
- **Orphaned tmux** contains panes carrying a Shepherd ID that is unknown to the
  durable store. They remain attachable and steerable but are never silently
  adopted into a workstream; `Ctrl-T` below makes adoption explicit.

Organizing is done in place. Each chord carries a verb and reads the selected
row for its noun, so one key covers both nouns it could apply to: `Ctrl-R`
renames a workstream or retitles a session, and `Shift-Up`/`Shift-Down`
reorders a named workstream or a member session within that workstream. `Ctrl-N`
creates a workstream. `Ctrl-T` marks a session with `◆` and moves it into the
next workstream you select, adopting an orphan explicitly when that is what it
is. The synthetic Ungrouped and Orphaned sections remain fixed after named
workstreams.

`Ctrl-O` edits the selected workstream's roots. Roots get one chord rather than
three because they are one list the dashboard already has a cursor for:
`Shift-Tab` picks which root a new session launches into, and that is the root
`Ctrl-O` opens. Press it again to walk to the next root, and once more to reach
an empty slot that adds one — so adding is editing the slot past the end rather
than a separate mode. `Enter` saves the path shown; an empty draft removes that
root and asks once more before doing it. A workstream always keeps its last
root. The composer prefix names the slot the whole time, which is what lets one
chord carry three outcomes honestly.

`Ctrl-V` archives the selected workstream, which is the one organize verb that
takes a row off the dashboard, so it asks: the first press names the workstream
and says what becomes of its sessions, the second press does it, and any other
key — or a paste — cancels. The question waits as long as you like, but the
answer cannot arrive in the fraction of a second that means the key is being
held down rather than pressed again. It is a bare control chord because those
arrive as a single byte and need none of the enhanced key reporting that
decides whether a modified arrow reaches Shepherd at all. `Ctrl-A` was the obvious letter and is not
available — the composer owns it as line start — and `v` is the only other
letter of "archive" that no chord had already claimed.

Because every printable key belongs to the composer, these are chords rather
than bare letters. That is the whole reason a separate organizer view existed;
folding the verbs into chords removed the view and the second set of keys with
it.

The read-only lower pane follows the selection: a workstream shows a bounded
`notes.md` preview and a shallow tree of its artifact directory, and a session
shows its terminal preview instead. A session resolves to its parent
workstream, so moving between a group and its members costs no extra read.
Press `Ctrl-G` to enter resize mode, then `Up` to grow the lower pane, `Down`
to expose more sessions, or `r` to restore automatic sizing.
Rendering context does not change domain state, inspect registered repository
roots, or modify files. The preview is cached against the selected workstream,
so it reads when the selection lands somewhere new and costs nothing while the
cursor sits still. Press `F3` after an agent or editor rewrites notes under a
stationary cursor; moving off the row and back does the same thing.

Roots are `Ctrl-O` on the dashboard and `shepherd ws root add|set|rm` on the CLI;
archiving is `Ctrl-V` on the dashboard and `shepherd ws archive` on the CLI. Every
workstream keeps at least one root, root edits never rewrite historical session
records or touch the filesystem, and archiving keeps all durable sessions,
stops no runtime, and moves their memberships to Ungrouped.

The composer always shows its exact workstream and launch root. A workstream may
contain sessions launched from several registered roots, but membership never
implicitly adds a root.

Workstream state is separate from settings. It remains a versioned, locked JSON
sidecar at `~/.shepherd/state.json`; ordinary workstream files live in
`~/.shepherd/workstreams/<id>/`. State updates are serialized with a
local advisory lock so CLI commands and the dashboard cannot overwrite one
another. State schema v4 stores durable titles, native conversation IDs, and a
dense position for each named-workstream membership. Valid v1-v3 files are
validated and atomically migrated without manufacturing a domain revision;
future or invalid versions are rejected. New and moved-in members append to the
bottom; Ungrouped sessions intentionally retain the live/newest projection.

## The Shepherd directory

Everything Shepherd owns lives in one place, `~/.shepherd`:

```text
~/.shepherd/
  config.json          settings
  state.json           durable workstream/session state
  workstreams/<id>/    notes.md and artifacts
```

Shepherd does not inspect or modify Heikou's home directory. Users who want to
carry selected durable records over can follow
[`docs/transfer-sessions.md`](docs/transfer-sessions.md); fresh installations
start here with no migration step.

## Configuration

Press `Ctrl-S` (or `F2`) in the dashboard to open the settings pane. Press `e`
there to create/open `~/.shepherd/config.json` in `$VISUAL`, `$EDITOR`, or
`vi`. Settings are deliberately one small JSON object:

```json
{
  "default_runner": "codex",
  "commands": {
    "codex": ["codex"],
    "claude": ["claude", "--dangerously-skip-permissions"]
  },
  "composer_keys": {
    "reply": "space",
    "cycle_runner": "tab",
    "cycle_root": "shift+tab"
  }
}
```

Commands are argv arrays, not shell strings. Fixed flags are placed before the
task arguments Shepherd adds. Callers select a runner, while the controller's
trusted config-backed resolver loads and resolves its argv immediately before
launch; a command action cannot supply arbitrary runner argv. The three
`composer_keys` fields may be omitted to keep the defaults shown above.
`reply` aims the empty composer at the selected session; `cycle_runner` and
`cycle_root` act whether or not the composer has text. All three are live at
once, so they may not share a key.
`Enter` is the single commit key and is not configurable — that is what keeps
the destination the one the composer displays.

Every key Shepherd already answers to is reserved and cannot be assigned to one
of the three: the organize chords, `Ctrl-G` resize mode, the help and settings
screens, and the composer's own editing keys such as `Shift-Enter`, `Ctrl-A`
and `Option-Left`. A composer binding is consulted before any of them, so
without the reservation the key would simply stop doing what it used to, with
nothing said about it at any point. A settings file that names a reserved key
fails to load with a message giving the key, what Shepherd already does with it,
and the default that deleting the field restores.
The removed `new_session` and `send_message` fields chose a commit key per
destination. A config still carrying either one fails to load with a message
naming `reply` as the replacement.

### Choosing what a brief shows

The `brief` block is two ordered lists of source names. Each slot takes the
first source with something to say. Omitting a slot keeps its default; an
explicitly empty `detail` is how you ask for a row that shows only its lead:

```json
{
  "brief": {
    "lead": ["title", "prompt", "runner"],
    "detail": []
  }
}
```

The built-in sources are:

| Source | What fills it |
| --- | --- |
| `title` | the durable title you gave the session |
| `automatic-title` | an ephemeral GPT-5.6 Luna title when explicitly enabled |
| `prompt` | the immutable task it was launched with |
| `latest` | the most recent message sent through Shepherd |
| `status` | bounded status published directly by the native runner |
| `activity` | what the runner last recorded the session doing |
| `runner` | `claude session`, as a last resort |

`activity` is the only one that goes and looks. It reads the tail of the
transcript Claude Code writes for the session — no network, no API key, nothing
to configure — and phrases the last record: `running make check`,
`editing observer.go`, `searching for BriefSource`, or
`replied · make check is green`. It is derived from another program's file, so
it always renders with the `~` mark, and it reads at most once every five
seconds per session and only after that session has shown terminal activity.
Naming it is what turns that on; drop it from the layout and Shepherd reads
nothing:

```json
{
  "brief": {
    "lead": ["title", "prompt", "runner"],
    "detail": ["latest", "prompt"]
  }
}
```

Anything that is not built in must be defined under `brief.sources` as a command
Shepherd runs:

```json
{
  "brief": {
    "lead": ["title", "prompt", "runner"],
    "detail": ["ci-status", "status", "activity", "latest"],
    "sources": {
      "ci-status": {
        "command": ["agent-status", "--porcelain"],
        "interval_seconds": 5,
        "timeout_seconds": 2
      }
    }
  }
}
```

### Optional automatic titles

Automatic titles are off by default. Opt in with the top-level setting:

```json
{
  "automatic_title": true
}
```

When `OPENAI_API_KEY` is present, the dashboard sends the first completed-turn
output once to `gpt-5.6-luna` and keeps the short result in memory. It never
rewrites the durable session title, and a title you set always wins. Without the
key, no transcript is read and no API call is attempted. The settings screen
shows whether the feature is active.

The command is argv, not a shell string. It runs once per session, is told
which session through `SHEPHERD_SESSION_ID`, `SHEPHERD_SESSION_RUNNER`,
`SHEPHERD_SESSION_STATE`, `SHEPHERD_SESSION_ROOT`, and `SHEPHERD_SESSION_TITLE`, and
prints one line to stdout. It is never given the session's prompt or messages.

A session is only re-run after `interval_seconds` **and** only if it has shown
terminal activity since the last look, so an idle dashboard costs nothing. Runs
are capped at four at a time and thirty-two per pass; a capped pass says how
many it deferred. Output is stripped of ANSI and control characters, reduced to
one line, and bounded. A source that fails or times out drops its text rather
than leaving a stale line that looks current.

Command output always renders with a leading `~`, and so does `activity`. A
command may well be reporting the truth, but Shepherd cannot check that; a phrase
read out of a transcript is a reading of another program's record rather than
something Shepherd watched happen. The mark is the difference between what it
observed and what it was told. Unknown source names,
duplicate entries, an empty `lead`, a source nothing refers to, and a timeout
longer than its interval are all load errors rather than surprises at runtime.
Brief changes apply as soon as settings are reloaded with `r`.
The settings pane displays the active bindings and reloads JSON changes with
`r`. Command changes affect new sessions; a changed `default_runner` applies
the next time the dashboard opens. `no-agent` is not configurable: it asks tmux
to start its default interactive shell without injecting the composer label.
Follow-up messages are then ordinary input to that shell, which makes it a
cheap transport test pane.

| Environment variable | Default | Purpose |
| --- | --- | --- |
| `SHEPHERD_DEFAULT_RUNNER` | `codex` | Initial runner in the composer |
| `SHEPHERD_TMUX_SOCKET` | `shepherd` | Private tmux socket name |
| `SHEPHERD_HOME` | `~/.shepherd` | Directory holding every Shepherd file |
| `SHEPHERD_CONFIG` | `~/.shepherd/config.json` | Settings file override |
| `SHEPHERD_STATE` | `~/.shepherd/state.json` | Durable application-state override |
| `SHEPHERD_DATA` | `~/.shepherd/workstreams` | Workstream artifact-directory base |
| `SHEPHERD_CODEX_BIN` | `codex` | Codex executable name or path |
| `SHEPHERD_CLAUDE_BIN` | `claude` | Claude executable name or path |

The dashboard also accepts `--runner`, `--root` / `-C`, and `--socket`.

## What Shepherd deliberately does not claim

An interactive agent process stays alive while it is thinking, waiting for
input, or simply sitting at its prompt. Tmux cannot distinguish those semantic
states. Shepherd reports process truth as `live`, `attached`, `exited`, or
`failed`, plus runtime, path, terminal activity, output preview, and an exit code
when tmux supplies one. New agent sessions may also carry a separate bounded
native status published by Claude or Codex; Shepherd never derives that status
from terminal text. Some retained dead panes—especially on older tmux
versions—omit `pane_dead_status`; Shepherd reports their process as exited with
an unknown outcome and never guesses zero or persists a successful exit.

Workstreams are organization, not autonomy. Shepherd has no manager role,
coordination grants, approvals, parent-child sessions, task graph, automatic
restart, queue, daemon, or MCP message bus. Durable session records survive a
tmux-server loss, but a missing pane without an already recorded terminal
outcome is reported as `unavailable`—never guessed to have exited. Running
multiple editing agents in one checkout can still cause conflicts; the selected
working directory remains intentionally prominent.

All current human mutations pass through one closed typed command plane with
an explicit actor and installation/workstream scope. Its active policy admits
only the local human; session actors are rejected until real manager grants and
authorization semantics exist. This is a future extension seam, not manager
mode.

See [the design document](docs/DESIGN.md) for the architecture, research notes,
extension seams, and next steps. Future product ideas live separately in the
[`todos`](todos/README.md) folder; the first note describes a pluggable
[composer module system](todos/composer-modules.md).

## Development

[`docs/DEVELOPMENT.md`](docs/DEVELOPMENT.md) is the full version. The short one:

```sh
make check
```

That runs the same gates as CI, in the same order, so a green `make check`
should mean a pull request has nothing left to discover.

**There is nothing to build before pushing.** No binaries, generated code, or
vendored dependencies are committed; users compile from source at `go install`,
and the agent instruction files reach the binary through `//go:embed` at compile
time. Editing `SKILL.md` or a help string is the whole change.

**Releasing is bumping `version` in [`cmd/shepherd/main.go`](cmd/shepherd/main.go).** `@latest`
resolves to the newest tag rather than to `main`, so a change merged without a
bump reaches nobody while nothing looks wrong. When CI passes on a `main` commit
whose `version` has no tag, the `Tag` workflow creates and pushes it; when the
version is unchanged it reports how far `main` has drifted ahead of the last
release. Nobody tags, builds, or uploads by hand.

The integration suite uses a randomly named private tmux server and a fake PTY
agent. It verifies caller-owned identity, lifecycle preservation, nonzero exits,
Unicode and paths with spaces, and literal delivery of shell-looking
prompt/message content. Controller tests cover durable-before-launch ordering,
failed launch retention, conservative reconciliation, orphan detection, and
explicit stop outcomes.

The end-to-end suite builds `shepherd` and drives it as a subprocess against a
throwaway `SHEPHERD_HOME` and a private tmux socket, so argument parsing, refusal
text, `--json` shape, and exit codes are tested as they ship. Alongside it, the
in-process suite drives the same verbs directly against a stub controller, which
is where the twenty-odd refusals and every `--json` key are checked without a
tmux server anywhere in sight.

The tmux-dependent suites skip themselves without tmux. Set
`SHEPHERD_TEST_REQUIRE_TMUX=1` — as CI and `make race` do — to turn that skip into
a failure, so a run cannot report green over a suite that never executed.

`internal/architecture` holds the module's shape: which package may import
which, and which concerns are allowed exactly one home. Adding a package means
placing it in that map, and an import that contradicts the layering fails there
rather than being discovered during a later refactor.
