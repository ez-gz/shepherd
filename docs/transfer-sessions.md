# Transfer Heikou sessions to Shepherd

Shepherd is a clean rename. It does not automatically inspect `~/.heikou`,
honor `HEIKOU_*` variables, or discover sessions on Heikou's tmux socket. This
runbook is the opt-in path for someone who wants to carry durable history into
Shepherd.

The transfer carries workstreams, notes, titles, prompts, outcomes, and
registered native conversation IDs. It does **not** move a running process,
tmux pane, terminal screen, or in-flight message. Finish or stop live Heikou
sessions before transferring them. If a live pane must be kept, leave the old
`h` binary and its tmux server alone until that work is finished.

## Before the transfer

1. Keep a copy of the old `h` binary.
2. Save `h list --json` somewhere outside `~/.heikou`.
3. For every session that may need to continue, run
   `h conversation SESSION --json`. A result with `registered: false` cannot be
   transferred reliably; do not guess its runner conversation ID.
4. Export any history the user wants with `h history SESSION --json`.
5. Close Heikou and make a complete backup of `~/.heikou`. Do not let Heikou or
   Shepherd write the state while it is being copied.

## Transfer procedure for an agent

Work in a temporary directory and install the result as `~/.shepherd` only
after validation.

1. Copy `state.json`, `config.json` when present, and the complete
   `workstreams/` tree. Do not copy generated `AGENTS.md`, `CLAUDE.md`, or the
   old embedded skills; Shepherd will install its own versions.
2. Transform the copied `state.json` structurally, not with a global text
   replacement:
   - resolve both home directories to absolute paths, then rewrite the exact
     leading `$HOME/.heikou` prefix to `$HOME/.shepherd` in every
     `workstreams[].artifact_dir`, `workstreams[].roots[]`, and
     `sessions[].initial_root`;
   - rename the exact seeded workstream name `heikou-managers` to
     `shepherd-managers` and update its stock description;
   - preserve IDs, revisions, timestamps, memberships, outcomes, prompts,
     titles, and `conversation` objects exactly;
   - leave old `launch.binding` values unchanged. They describe historical
     Heikou runtimes and must not be relabeled as Shepherd runtimes.
3. Do not rewrite user-authored notes, titles, prompts, or arbitrary directory
   names merely because they contain the word “Heikou.”
4. Validate the copied tree by pointing `SHEPHERD_HOME` at the temporary
   directory and running `shepherd list --json`. Treat any schema or path error
   as a failed transfer; restore from the backup instead of repairing the live
   copy in place.
5. Install the validated directory as `~/.shepherd`, then run
   `shepherd init --force` to write the current manager instructions.
6. Continue a transferred conversation with
   `shepherd resume SESSION MESSAGE`. This creates a new Shepherd session and
   leaves the transferred Heikou record as history.

Keep the backup and old binary until the user has checked their workstreams,
notes, and every conversation they intended to resume. Removing either is a
separate, explicit cleanup step.
