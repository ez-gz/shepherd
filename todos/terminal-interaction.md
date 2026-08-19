# Attached terminal interaction

Status: included in 0.8.0.

## The problem

An attached session crosses three independent input owners:

1. the user's terminal emulator,
2. Shepherd's private tmux server, and
3. the native runner TUI.

Shepherd currently enables tmux mouse handling and terminal clipboard forwarding
globally. A drag may therefore become terminal-native selection, tmux copy mode,
or a mouse event delivered to the runner, depending on the terminal and the
runner's active terminal modes. In iTerm2, holding Option bypasses those mouse
modes and forces terminal-native selection. That workaround is evidence that the
plain drag was claimed before the terminal could select it.

The result is provider-dependent: a drag that visibly selects and copies in one
runner may scroll, click, do nothing, or require Option in another. This is a
Shepherd interaction-contract problem even when the underlying difference comes
from the runner.

## Product contract

Attached sessions should have one short, discoverable contract:

- Drag selects and copies text from the pane through tmux, independent of runner.
- Clicks may still reach runners that expose clickable controls.
- The wheel scrolls the pane's tmux history unless an explicit mode says otherwise.
- A documented modifier bypasses tmux for terminal-native, cross-pane selection.
- Keyboard-only copy mode remains available and visible in help.

The dashboard requests cell-motion mouse events. A click selects a row without
attaching or acting on it, clicking a disclosure triangle collapses or expands
a workstream, and the wheel moves the dashboard list, settings, or help. Mouse
input disarms destructive confirmations. The terminal's bypass modifier remains
the native-selection path on the dashboard as well as in attached sessions.

## What shipped

The private tmux server always routes `MouseDrag1Pane` into copy mode and copies
on drag end, even when the foreground application requested mouse events.
Ordinary clicks and wheel events keep their default routing. On macOS,
`copy-command` uses `/usr/bin/pbcopy`; tmux's paste buffer and `set-clipboard`
remain enabled for portable behavior.

Keyboard copy is also independent of tmux's mode-key choice: `Ctrl-b [` enters
copy mode, `Space` begins the selection, `Enter` copies and exits, and `Esc`
leaves copy mode. Shepherd installs the selection and copy keys in both the
emacs and vi tables.

The dashboard handles cell-motion events without giving a click any keyboard
verb's consequences. Hit testing uses the same scrolled list window as rendering,
and a pinned composer refuses mouse redirection. Focused UI tests cover dashboard,
settings, help, scrolled rows, disclosure clicks, and reply ownership. Real tmux
integration tests pin the unconditional drag binding, both copy-mode tables, the
clipboard options, and bootstrap-version migration.

Manual release verification still exercises Terminal.app and iTerm2 against live
Codex and Claude sessions, including alternate-screen output, borders, wrapped
lines, scrolling, detach/reattach, and a server created before the bootstrap
version changed.

## Deliberately not

- Do not inject undocumented provider settings to force a preferred mouse mode.
- Do not infer behavior from the runner name; inspect terminal/tmux behavior.
- Do not make users learn different copy instructions for Codex and Claude.
- Do not replace tmux or build a terminal emulator inside Shepherd.
