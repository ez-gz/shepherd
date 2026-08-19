# Session ordering within a workstream

Status: included in 0.8.0.

## Contract

`Shift-Up` / `Shift-Down` on a named workstream reorders that workstream.
The same chord on a member session reorders the session inside its current
named workstream. A reorder never changes membership; `Ctrl-T` remains the
explicit dashboard move/adopt flow.

The CLI equivalent is:

```sh
shepherd reorder SESSION --up|--down
```

Ungrouped sessions have no membership and no durable position. They keep the
existing live-first, newest-first projection. A session newly launched,
adopted, or moved into a named workstream appends to the bottom.

## Durable shape

State schema v4 adds a dense, zero-based `Position` to `Membership`. Positions
are unique within a workstream. Boundary reorders are no-ops and do not advance
the state revision; successful reorders atomically swap neighboring positions.
Moving or deleting a member compacts its source workstream, and moving into a
workstream appends after its last member.

The v3-to-v4 migration orders existing members newest-created first, matching
the old projection's durable fallback. `JoinedAt` and legacy membership order
are deterministic tie-breakers. Runtime liveness is ephemeral and is never
written into the migration. The migration itself does not manufacture a domain
revision.
