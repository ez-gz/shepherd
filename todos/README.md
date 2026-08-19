# Shepherd ideas

This folder holds product and architecture ideas that are intentionally outside
the current implementation. One idea gets one document so the main design does
not become an undifferentiated backlog.

| Idea | Status |
| --- | --- |
| [Code-quality and reliability audit](code-quality-audit.md) | Active backlog |
| [Shepherd pilot](shepherd-pilot.md) | CLI verbs and agent instructions shipped; pilot loop and UI deferred |
| [Session history](session-history.md) | `shepherd history` shipped for Claude; Codex parity is optional compatibility cleanup, not a roadmap feature |
| [Session resume](session-resume.md) | Shipped; Codex enforces one live writer, resume reuses its owner, and fork is explicit |
| [Composable composer modules](composer-modules.md) | Proposed |
| [Configurable brief sources](brief-sources.md) | Shipped |
| [What a runner exposes about what it is doing](runner-activity.md) | Transcript `activity` source shipped; Claude's per-process status file documented and unbuilt |
| [Session status, titles, and recency](session-status-titles.md) | Durable and optional automatic titles plus native status shipped; returned/seen attention deferred |
| [Session ordering within a workstream](session-ordering.md) | Shipped with durable schema-v4 ordering, dashboard chords, and the `reorder` CLI |
| [Attached terminal interaction](terminal-interaction.md) | Active UX backlog: make selection, copying, clicking, and scrolling predictable across runners |

An idea should move into `docs/DESIGN.md` only when it becomes part of the
committed architecture or an active implementation.
