package runner

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"strings"
	"unicode"

	"github.com/charmbracelet/x/ansi"
)

const (
	// MaxNativeStatusRunes is the transport bound shared by runner publishers
	// and the tmux projection. Keeping the limit at both ends means neither a
	// changed runner nor hand-edited pane metadata can grow a dashboard row.
	MaxNativeStatusRunes = 240

	// CodexTerminalTitleOverride asks Codex itself to publish only documented,
	// bounded run metadata into the pane title. The supervisor accepts that
	// title only on panes it marked for this launch contract.
	CodexTerminalTitleOverride = `tui.terminal_title=["run-state","model-with-reasoning","context-used","used-tokens"]`
)

// NativeStatusUpdate is one Claude callback's bounded contribution. Hooks and
// the status line use separate pane options so a frequent metrics refresh does
// not erase the lifecycle state, and a lifecycle edge does not erase metrics.
type NativeStatusUpdate struct {
	State   string
	Metrics string
	// Display is written back to Claude only for statusLine callbacks. Hook
	// callbacks must stay silent because stdout from some hooks enters context.
	Display string
}

// ClaudeSettings returns an inline, session-local settings layer. It is passed
// through --settings for this launch only; Shepherd never edits the user's
// settings file and never rewrites an already-running process.
func ClaudeSettings(callback string) string {
	if strings.TrimSpace(callback) == "" {
		callback = "shepherd"
	}
	command := shellQuote(callback) + " __statusline claude"
	hook := func() []map[string]any {
		entry := map[string]any{"type": "command", "command": command}
		return []map[string]any{{"hooks": []map[string]any{entry}}}
	}
	settings := map[string]any{
		"statusLine": map[string]any{"type": "command", "command": command},
		"hooks": map[string]any{
			"SessionStart":      hook(),
			"UserPromptSubmit":  hook(),
			"PostToolUse":       hook(),
			"PermissionRequest": hook(),
			"Notification":      hook(),
			"Elicitation":       hook(),
			"ElicitationResult": hook(),
			"Stop":              hook(),
			"StopFailure":       hook(),
			"SessionEnd":        hook(),
		},
	}
	data, _ := json.Marshal(settings)
	return string(data)
}

// ReadClaudeStatus interprets the documented subset shared by Claude's
// statusLine and lifecycle-hook payloads. Unknown events publish nothing;
// instrumentation must fail soft when a runner adds an event or field.
func ReadClaudeStatus(reader io.Reader) (NativeStatusUpdate, error) {
	decoder := json.NewDecoder(io.LimitReader(reader, 256<<10))
	var payload struct {
		HookEventName    string `json:"hook_event_name"`
		NotificationType string `json:"notification_type"`
		Model            struct {
			DisplayName string `json:"display_name"`
		} `json:"model"`
		Effort struct {
			Level string `json:"level"`
		} `json:"effort"`
		Cost struct {
			TotalCostUSD float64 `json:"total_cost_usd"`
		} `json:"cost"`
		ContextWindow struct {
			TotalInputTokens  int64   `json:"total_input_tokens"`
			TotalOutputTokens int64   `json:"total_output_tokens"`
			UsedPercentage    float64 `json:"used_percentage"`
		} `json:"context_window"`
	}
	if err := decoder.Decode(&payload); err != nil {
		return NativeStatusUpdate{}, fmt.Errorf("decode claude status: %w", err)
	}
	if payload.HookEventName != "" {
		return NativeStatusUpdate{State: claudeLifecycleState(payload.HookEventName, payload.NotificationType)}, nil
	}

	parts := make([]string, 0, 5)
	if model := nativeStatusText(payload.Model.DisplayName); model != "" {
		parts = append(parts, model)
	}
	if effort := nativeStatusText(payload.Effort.Level); effort != "" {
		parts = append(parts, effort)
	}
	if percentage := payload.ContextWindow.UsedPercentage; percentage > 0 && !math.IsNaN(percentage) && !math.IsInf(percentage, 0) {
		parts = append(parts, fmt.Sprintf("%.0f%% ctx", percentage))
	}
	if tokens := payload.ContextWindow.TotalInputTokens + payload.ContextWindow.TotalOutputTokens; tokens > 0 {
		parts = append(parts, tokenCount(tokens)+" tok")
	}
	if cost := payload.Cost.TotalCostUSD; cost > 0 && !math.IsNaN(cost) && !math.IsInf(cost, 0) {
		parts = append(parts, fmt.Sprintf("$%.4f", cost))
	}
	metrics := BoundedNativeStatus(strings.Join(parts, " · "))
	return NativeStatusUpdate{Metrics: metrics, Display: metrics}, nil
}

func claudeLifecycleState(event, notificationType string) string {
	switch event {
	case "SessionStart", "Stop":
		return "Ready"
	case "UserPromptSubmit", "PostToolUse", "ElicitationResult":
		return "Working"
	case "PermissionRequest", "Elicitation":
		return "Needs input"
	case "Notification":
		switch notificationType {
		case "permission_prompt", "idle_prompt", "elicitation_dialog", "elicitation_url_dialog", "agent_needs_input":
			return "Needs input"
		default:
			return ""
		}
	case "StopFailure":
		return "Error"
	case "SessionEnd":
		return "Done"
	default:
		return ""
	}
}

// BoundedNativeStatus makes runner-owned text safe for a one-line terminal
// projection and caps it by runes rather than bytes.
func BoundedNativeStatus(value string) string {
	value = nativeStatusText(value)
	runes := []rune(value)
	if len(runes) > MaxNativeStatusRunes {
		value = string(runes[:MaxNativeStatusRunes])
	}
	return value
}

func nativeStatusText(value string) string {
	value = ansi.Strip(value)
	value = strings.Map(func(character rune) rune {
		if unicode.IsControl(character) {
			return ' '
		}
		return character
	}, value)
	return strings.Join(strings.Fields(value), " ")
}

func tokenCount(value int64) string {
	switch {
	case value >= 1_000_000:
		return strings.TrimSuffix(fmt.Sprintf("%.1f", float64(value)/1_000_000), ".0") + "m"
	case value >= 1_000:
		return strings.TrimSuffix(fmt.Sprintf("%.1f", float64(value)/1_000), ".0") + "k"
	default:
		return fmt.Sprintf("%d", value)
	}
}

// shellQuote quotes one argv element for the shell Claude uses to execute hook
// commands. The callback itself never contains user content, but an installed
// application path may contain spaces or a single quote.
func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'"
}
