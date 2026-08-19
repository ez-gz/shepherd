// Package env names every environment variable Shepherd reads or sets. Keeping
// them together lets configuration, runner launch, tmux refresh, and hermetic
// tests share one contract instead of repeating security-sensitive strings.
//
// This package is a leaf: it imports nothing else in the module, so any layer
// can name a variable without a dependency detour. An architecture test asserts
// that no SHEPHERD_ literal is written anywhere else in non-test source.
package env

import (
	"os"
	"strings"
)

const (
	// Home overrides the single directory holding every Shepherd file.
	Home = "SHEPHERD_HOME"
	// Config, State and Data override individual paths inside the home
	// directory for tests and custom layouts.
	Config = "SHEPHERD_CONFIG"
	State  = "SHEPHERD_STATE"
	Data   = "SHEPHERD_DATA"

	// TmuxSocket names the private tmux server. Tests rely on it to keep a run
	// off the developer's own sessions.
	TmuxSocket = "SHEPHERD_TMUX_SOCKET"

	// DefaultRunner, CodexBinary and ClaudeBinary override settings that
	// otherwise come from config.json.
	DefaultRunner = "SHEPHERD_DEFAULT_RUNNER"
	CodexBinary   = "SHEPHERD_CODEX_BIN"
	ClaudeBinary  = "SHEPHERD_CLAUDE_BIN"
	// OpenAIAPIKey opts an explicitly enabled automatic-title observer into one
	// Responses API call per session. Without it Shepherd makes no network call.
	OpenAIAPIKey = "OPENAI_API_KEY"

	// SessionID is set by Shepherd into an agent's environment rather than read
	// from the user, so a running agent can identify its own session.
	SessionID = "SHEPHERD_SESSION_ID"

	// SessionRunner, SessionState, SessionRoot and SessionTitle describe one
	// session to a configured brief source. They are written outward only, into
	// that command's environment, and are never read back.
	//
	// The set is deliberately small. A brief source is told which session it is
	// describing, not what was said in it: the initial prompt and the messages
	// are the user's content, and wanting a status line in a row is not a reason
	// to hand what someone typed to another program on a timer.
	SessionRunner = "SHEPHERD_SESSION_RUNNER"
	SessionState  = "SHEPHERD_SESSION_STATE"
	SessionRoot   = "SHEPHERD_SESSION_ROOT"
	SessionTitle  = "SHEPHERD_SESSION_TITLE"
)

// Names lists every variable above, so that a test can assert this file is the
// only place in non-test source where a SHEPHERD_ name is written, and a reader
// can answer "which variables does Shepherd honour" from one list.
//
// It is deliberately not the tmux server's environment-refresh list. Those are
// different questions: that list is about which values must be re-read from an
// attaching client, and answering it wholesale from here would refresh
// SHEPHERD_SESSION_ID, replacing a session's identity with the attacher's.
var Names = []string{
	Home, Config, State, Data,
	TmuxSocket,
	DefaultRunner, CodexBinary, ClaudeBinary,
	OpenAIAPIKey,
	SessionID, SessionRunner, SessionState, SessionRoot, SessionTitle,
}

// Value reads a variable and trims it, treating whitespace as unset. Every
// caller wanted this; several wrote it out by hand.
func Value(name string) string {
	return strings.TrimSpace(os.Getenv(name))
}

// ValueOr reads a variable, falling back when it is unset or blank.
func ValueOr(name, fallback string) string {
	if value := Value(name); value != "" {
		return value
	}
	return fallback
}
