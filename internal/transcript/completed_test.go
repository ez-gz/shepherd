package transcript

import (
	"os"
	"testing"
	"time"

	"github.com/ez-gz/shepherd/internal/shepherd"
)

func TestClaudeCompletedOutputRequiresTheRunnersTurnBoundary(t *testing.T) {
	completed := record(t, map[string]any{
		"type": "assistant", "message": map[string]any{
			"role": "assistant", "stop_reason": "end_turn",
			"content": []map[string]any{{"type": "text", "text": "Implemented native status"}},
		},
	})
	projects := writeTranscript(t, "-work", userRecord(t, "do it"), completed)

	text, found, err := (Reader{ClaudeProjects: projects}).FirstCompletedOutput(CompletedOutputRequest{
		Runner: shepherd.BackendClaude, SessionID: testSessionID, Root: "/work",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !found || text != "Implemented native status" {
		t.Fatalf("completed output = %q, found = %t", text, found)
	}

	incompleteProjects := writeTranscript(t, "-work", userRecord(t, "do it"), assistantRecord(t, textBlock("still working")))
	if text, found, err := (Reader{ClaudeProjects: incompleteProjects}).FirstCompletedOutput(CompletedOutputRequest{
		Runner: shepherd.BackendClaude, SessionID: testSessionID, Root: "/work",
	}); err != nil || found || text != "" {
		t.Fatalf("incomplete output = %q, found = %t, err = %v", text, found, err)
	}
}

func TestClaudeCompletedOutputStartsAtThisLaunchNotTheWholeResumedConversation(t *testing.T) {
	startedAt := time.Date(2026, 8, 18, 10, 0, 0, 0, time.UTC)
	assistant := func(at time.Time, text string) string {
		return record(t, map[string]any{
			"type": "assistant", "timestamp": at.Format(time.RFC3339Nano),
			"message": map[string]any{
				"role": "assistant", "stop_reason": "end_turn",
				"content": []map[string]any{{"type": "text", "text": text}},
			},
		})
	}
	projects := writeTranscript(t, "-work",
		assistant(startedAt.Add(-time.Hour), "Old conversation output"),
		assistant(startedAt.Add(time.Second), "This launch output"),
	)

	text, found, err := (Reader{ClaudeProjects: projects}).FirstCompletedOutput(CompletedOutputRequest{
		Runner: shepherd.BackendClaude, SessionID: testSessionID, Root: "/work", StartedAt: startedAt,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !found || text != "This launch output" {
		t.Fatalf("completed output = %q, found = %t", text, found)
	}
}

func TestCodexCompletedOutputUsesTaskCompleteNotTerminalText(t *testing.T) {
	const id = "019e6d0c-14bd-7792-91d2-f684a8dc6e80"
	sessions := rollout(t, "", id, "/work/repo", launchedAt.Add(2*time.Second), "ship the release")
	conversation, err := find(sessions, "/work/repo", "ship the release", launchedAt)
	if err != nil {
		t.Fatal(err)
	}
	file, err := os.OpenFile(conversation.Path, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	_, writeErr := file.WriteString(encode(t, map[string]any{
		"timestamp": launchedAt.Add(10 * time.Second).Format(time.RFC3339Nano),
		"type":      "event_msg", "payload": map[string]any{
			"type": "task_complete", "last_agent_message": "Shipped the release",
		},
	}) + "\n")
	closeErr := file.Close()
	if writeErr != nil {
		t.Fatal(writeErr)
	}
	if closeErr != nil {
		t.Fatal(closeErr)
	}

	text, found, err := (Reader{CodexSessions: sessions}).FirstCompletedOutput(CompletedOutputRequest{
		Runner: shepherd.BackendCodex, Root: "/work/repo", StartedAt: launchedAt, Prompt: "ship the release",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !found || text != "Shipped the release" {
		t.Fatalf("completed output = %q, found = %t", text, found)
	}
}
