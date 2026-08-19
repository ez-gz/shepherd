package shepherd

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Backend identifies a native coding-agent CLI.
type Backend string

const (
	BackendCodex   Backend = "codex"
	BackendClaude  Backend = "claude"
	BackendNoAgent Backend = "no-agent"
)

// MaxInteractivePayloadBytes bounds text routed through durable state, tmux
// command construction, or pane input. 64 KiB is ample for an interactive turn
// while staying comfortably below platform argv and tmux transport limits even
// after launch metadata is base64 encoded.
const MaxInteractivePayloadBytes = 64 << 10

func ValidateInteractivePayload(kind, value string) error {
	if len([]byte(value)) > MaxInteractivePayloadBytes {
		return fmt.Errorf("%s is %d bytes; maximum is %d bytes", kind, len([]byte(value)), MaxInteractivePayloadBytes)
	}
	return nil
}

func ParseBackend(value string) (Backend, error) {
	switch Backend(strings.ToLower(strings.TrimSpace(value))) {
	case BackendCodex:
		return BackendCodex, nil
	case BackendClaude:
		return BackendClaude, nil
	case BackendNoAgent:
		return BackendNoAgent, nil
	default:
		return "", fmt.Errorf("unknown runner %q (want codex, claude, or no-agent)", value)
	}
}

func (b Backend) Next() Backend {
	switch b {
	case BackendCodex:
		return BackendClaude
	case BackendClaude:
		return BackendNoAgent
	default:
		return BackendCodex
	}
}

type Status string

const (
	StatusLive   Status = "live"
	StatusExited Status = "exited"
	StatusFailed Status = "failed"
)

// Session is the runner-neutral projection of one tmux-owned agent process and
// its bounded tmux-scoped presentation metadata. Lifecycle fields contain
// process truth only; NativeStatus is a separate runner-published observation.
type Session struct {
	ID      string
	Name    string
	PaneID  string
	Backend Backend
	Prompt  string
	// LastUserMessage is a bounded preview of the most recent message routed
	// through Shepherd. Messages typed in an attached native TUI are not observed.
	LastUserMessage string
	// NativeStatus is bounded, ephemeral runner-published presentation data.
	// It comes from launch-time instrumentation and is never inferred from the
	// terminal or written to durable workstream state.
	NativeStatus   string
	Root           string
	CurrentPath    string
	CurrentCommand string
	Status         Status
	StartedAt      time.Time
	EndedAt        time.Time
	LastActivityAt time.Time
	// ExitCode is nil when the runtime is live or when tmux retained a dead
	// pane without reporting pane_dead_status. A non-nil zero is therefore a
	// known successful exit, distinct from an unknown terminal status.
	ExitCode        *int
	AttachedClients int
	PaneInMode      bool
	InputDisabled   bool
	// ObservationError explains why Shepherd could identify the pane but could
	// not trust all of its metadata. Degraded observations are deliberately
	// visible and operable for attach/capture/stop, but must never be used to
	// rewrite durable lifecycle state or receive injected input.
	ObservationError string
}

func (s Session) Alive() bool    { return s.Status == StatusLive }
func (s Session) Degraded() bool { return strings.TrimSpace(s.ObservationError) != "" }

func (s Session) Runtime(now time.Time) time.Duration {
	end := now
	if !s.EndedAt.IsZero() {
		end = s.EndedAt
	}
	if s.StartedAt.IsZero() || end.Before(s.StartedAt) {
		return 0
	}
	return end.Sub(s.StartedAt)
}

type StartRequest struct {
	// ID is allocated by the durable controller before launch. Runtime
	// implementations must use it verbatim and must never allocate identity.
	ID      string
	Backend Backend
	Prompt  string
	Root    string
	// Command is an argv prefix for agent backends. The executable is the first
	// element and fixed flags follow it. Supervisor implementations append the
	// runner-specific task/session arguments without invoking a shell. It is
	// ignored for no-agent sessions, which start tmux's default shell directly.
	Command []string
	// Resume names a native runner conversation this launch should continue
	// rather than begin. Empty starts a fresh conversation. It travels as argv
	// to the runner like every other launch value and is never interpolated.
	Resume string
	// Fork starts a new native conversation from Resume. It is explicit because
	// resume means one continuing writer, while fork means a new branch.
	Fork bool
}

// Supervisor is the deliberately small boundary between the dashboard and
// today's tmux runtime. A daemon or MCP-backed runtime can implement the same
// contract later without replacing the UI.
type Supervisor interface {
	Bootstrap(context.Context) error
	Sessions(context.Context) ([]Session, error)
	Find(context.Context, string) (Session, error)
	// RuntimeExists is the fail-closed lifecycle check for a caller-owned ID.
	// Implementations must use the durable ID and, when supplied, its bound
	// runtime name without depending on optional presentation metadata.
	RuntimeExists(context.Context, string, string) (bool, error)
	Start(context.Context, StartRequest) (Session, error)
	Send(context.Context, Session, string) error
	Capture(context.Context, Session, int) (string, error)
	Stop(context.Context, Session) error
	AttachCommand(Session) *exec.Cmd
}
