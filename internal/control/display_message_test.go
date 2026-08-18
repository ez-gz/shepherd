package control

import (
	"testing"

	"github.com/ez-gz/shepherd/internal/shepherd"
	"github.com/ez-gz/shepherd/internal/workstream"
)

func TestProjectOneUsesLatestRuntimeUserMessageForDisplay(t *testing.T) {
	record := workstream.SessionRecord{ID: "018f0000-0000-4000-8000-000000000070", InitialPrompt: "initial task"}
	runtime := shepherd.Session{LastUserMessage: "latest follow-up"}

	session := projectOne(workstream.EmptyState(), record, &runtime)
	if got := session.DisplayMessage(); got != "latest follow-up" {
		t.Fatalf("DisplayMessage() = %q, want latest follow-up", got)
	}

	session = projectOne(workstream.EmptyState(), record, nil)
	if got := session.DisplayMessage(); got != "initial task" {
		t.Fatalf("DisplayMessage() without runtime = %q, want initial task", got)
	}
}
