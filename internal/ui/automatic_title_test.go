package ui

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/ez-gz/shepherd/internal/config"
	"github.com/ez-gz/shepherd/internal/control"
	"github.com/ez-gz/shepherd/internal/shepherd"
)

func automaticTitleTestModel(t *testing.T) (Model, control.Session) {
	t.Helper()
	model, _ := newTestModel(t.TempDir(), shepherd.BackendCodex)
	model.settings = config.Default()
	model.settings.AutomaticTitle = true
	session := testDurableSession(
		"018f0000-0000-4000-8000-0000000000a1", "", shepherd.BackendCodex,
		"implement status", t.TempDir(), time.Now(),
	)
	model.setSnapshot(control.Snapshot{Sessions: []control.Session{session}})
	return model, session
}

func TestAutomaticTitleIsOneAsynchronousAttemptAndNeverRewritesState(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "test-key")
	model, session := automaticTitleTestModel(t)
	calls := 0
	model.generateAutomaticTitle = func(context.Context, control.Session) (string, bool, error) {
		calls++
		return "Native Runner Status", true, nil
	}

	command := model.requestAutomaticTitle()
	if command == nil || calls != 0 {
		t.Fatalf("request command = %v, synchronous calls = %d", command != nil, calls)
	}
	updated, next := model.Update(command())
	after := updated.(Model)
	if calls != 1 || next != nil {
		t.Fatalf("calls = %d, follow-up command = %v", calls, next != nil)
	}
	if after.automaticTitles[session.ID] != "Native Runner Status" || !after.automaticTitleTried[session.ID] {
		t.Fatalf("automatic title state = %#v, tried = %#v", after.automaticTitles, after.automaticTitleTried)
	}
	if after.snapshot.Sessions[0].Record.Title != "" {
		t.Fatalf("automatic title became durable: %q", after.snapshot.Sessions[0].Record.Title)
	}
	if retry := after.requestAutomaticTitle(); retry != nil {
		t.Fatal("completed output triggered a second title attempt")
	}
}

func TestAutomaticTitleMakesNoCallWithoutAKey(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "")
	model, _ := automaticTitleTestModel(t)
	calls := 0
	model.generateAutomaticTitle = func(context.Context, control.Session) (string, bool, error) {
		calls++
		return "must not happen", true, nil
	}
	if command := model.requestAutomaticTitle(); command != nil || calls != 0 {
		t.Fatalf("command = %v, calls = %d", command != nil, calls)
	}
}

func TestHumanTitleWinsEvenIfAutomaticTitleAlreadyArrived(t *testing.T) {
	model, session := automaticTitleTestModel(t)
	model.automaticTitles[session.ID] = "Generated Label"
	session.Record.Title = "Human Label"

	item := model.sessionBrief(session)
	if item.Lead.Text != "Human Label" || item.Lead.Source != "title" {
		t.Fatalf("brief lead = %#v", item.Lead)
	}
	if strings.Contains(item.Lead.Text, "Generated") {
		t.Fatalf("generated title beat the human title: %#v", item.Lead)
	}
}

func TestAutomaticTitleIsStrictlyOptIn(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "test-key")
	model, _ := automaticTitleTestModel(t)
	model.settings.AutomaticTitle = false
	if command := model.requestAutomaticTitle(); command != nil {
		t.Fatal("default-off automatic title scheduled work")
	}
}
