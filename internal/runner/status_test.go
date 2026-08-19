package runner

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/ez-gz/shepherd/internal/shepherd"
)

func TestRunnerLaunchesInstallSessionLocalNativeStatus(t *testing.T) {
	codex, _ := AdapterFor(shepherd.BackendCodex)
	codexArgs := codex.Arguments(Launch{Prompt: "task"})
	if len(codexArgs) < 2 || codexArgs[0] != "-c" || codexArgs[1] != CodexTerminalTitleOverride {
		t.Fatalf("codex instrumentation = %#v", codexArgs)
	}
	if strings.Contains(strings.Join(codexArgs, " "), "notify") {
		t.Fatalf("codex instrumentation replaced the user's notify callback: %#v", codexArgs)
	}

	claude, _ := AdapterFor(shepherd.BackendClaude)
	claudeArgs := claude.Arguments(Launch{Prompt: "task", Callback: "/Applications/Shepherd App/shepherd"})
	if len(claudeArgs) < 2 || claudeArgs[0] != "--settings" {
		t.Fatalf("claude instrumentation = %#v", claudeArgs)
	}
	var settings struct {
		StatusLine map[string]any              `json:"statusLine"`
		Hooks      map[string][]map[string]any `json:"hooks"`
	}
	if err := json.Unmarshal([]byte(claudeArgs[1]), &settings); err != nil {
		t.Fatal(err)
	}
	if command, _ := settings.StatusLine["command"].(string); !strings.Contains(command, "__statusline claude") ||
		!strings.Contains(command, "Shepherd App") {
		t.Fatalf("statusLine command = %q", command)
	}
	for _, event := range []string{
		"SessionStart", "UserPromptSubmit", "PostToolUse", "PermissionRequest", "Notification",
		"Elicitation", "ElicitationResult", "Stop", "StopFailure", "SessionEnd",
	} {
		if len(settings.Hooks[event]) != 1 {
			t.Fatalf("hook %s = %#v", event, settings.Hooks[event])
		}
	}
}

func TestClaudeStatusPublishesMetricsAndLifecycleSeparately(t *testing.T) {
	metrics, err := ReadClaudeStatus(strings.NewReader(`{
		"model":{"display_name":"Opus"},"effort":{"level":"high"},
		"context_window":{"used_percentage":18.4,"total_input_tokens":15000,"total_output_tokens":1700},
		"cost":{"total_cost_usd":0.01234}
	}`))
	if err != nil {
		t.Fatal(err)
	}
	if metrics.State != "" || metrics.Metrics != "Opus · high · 18% ctx · 16.7k tok · $0.0123" || metrics.Display != metrics.Metrics {
		t.Fatalf("metrics update = %#v", metrics)
	}

	for event, want := range map[string]string{
		"SessionStart": "Ready", "UserPromptSubmit": "Working", "PostToolUse": "Working",
		"PermissionRequest": "Needs input", "Elicitation": "Needs input", "ElicitationResult": "Working",
		"StopFailure": "Error", "SessionEnd": "Done", "Stop": "Ready",
	} {
		update, err := ReadClaudeStatus(strings.NewReader(`{"hook_event_name":"` + event + `"}`))
		if err != nil {
			t.Fatal(err)
		}
		if update.State != want || update.Metrics != "" || update.Display != "" {
			t.Fatalf("%s update = %#v, want state %q", event, update, want)
		}
	}
	needsInput, err := ReadClaudeStatus(strings.NewReader(`{"hook_event_name":"Notification","notification_type":"permission_prompt"}`))
	if err != nil || needsInput.State != "Needs input" {
		t.Fatalf("permission notification = %#v, err = %v", needsInput, err)
	}
	completed, err := ReadClaudeStatus(strings.NewReader(`{"hook_event_name":"Notification","notification_type":"agent_completed"}`))
	if err != nil || completed.State != "" {
		t.Fatalf("non-input notification = %#v, err = %v", completed, err)
	}
}

func TestNativeStatusIsSingleLineAndRuneBounded(t *testing.T) {
	got := BoundedNativeStatus("\x1b[31mWorking\x1b[0m\n" + strings.Repeat("界", 300))
	if strings.Contains(got, "\x1b") || strings.Contains(got, "\n") {
		t.Fatalf("status retained terminal controls: %q", got)
	}
	if len([]rune(got)) != MaxNativeStatusRunes {
		t.Fatalf("status has %d runes, want %d", len([]rune(got)), MaxNativeStatusRunes)
	}
}

func TestInteractiveEnvironmentRemovesOnlyNoColor(t *testing.T) {
	input := []string{"NO_COLOR=1", "OPENAI_API_KEY=secret", "MY_NO_COLOR=keep", "NO_COLORISH=keep"}
	got := interactiveEnvironment(input)
	if slices.Contains(got, "NO_COLOR=1") {
		t.Fatalf("NO_COLOR survived: %#v", got)
	}
	for _, want := range input[1:] {
		if !slices.Contains(got, want) {
			t.Fatalf("interactive environment lost %q: %#v", want, got)
		}
	}
}

func TestForkArgumentsAreExplicit(t *testing.T) {
	launch := Launch{Prompt: "branch here", Resume: "019e6d0c-14bd-7792-91d2-f684a8dc6e80", Fork: true}
	for backend, want := range map[shepherd.Backend][]string{
		shepherd.BackendCodex:  {"fork", launch.Resume, "--", launch.Prompt},
		shepherd.BackendClaude: {"--resume", launch.Resume, "--fork-session", "--name", "", "--", launch.Prompt},
	} {
		adapter, _ := AdapterFor(backend)
		got := withoutInstrumentation(backend, adapter.Arguments(launch))
		if !slices.Equal(got, want) {
			t.Fatalf("%s fork argv = %#v, want %#v", backend, got, want)
		}
	}
}
