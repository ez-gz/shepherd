# Attached terminal interaction

Status: active UX backlog for 0.8.0 consideration.

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

The dashboard remains terminal-native and does not request the mouse.

## First experiment

Test a private-server tmux binding that always routes `MouseDrag1Pane` into copy
mode and copies on drag end, even when the foreground application has requested
mouse events. Preserve ordinary clicks and wheel events. On macOS, verify both
tmux-buffer copying and the system clipboard rather than assuming OSC 52 support.

Exercise the matrix in Terminal.app and iTerm2 against live Codex and Claude
sessions, including alternate-screen output, borders, wrapped lines, scrolling,
detach/reattach, and sessions launched before the bootstrap change.

## Deliberately not

- Do not inject undocumented provider settings to force a preferred mouse mode.
- Do not infer behavior from the runner name; inspect terminal/tmux behavior.
- Do not make users learn different copy instructions for Codex and Claude.
- Do not replace tmux or build a terminal emulator inside Shepherd.
