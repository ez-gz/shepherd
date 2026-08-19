package ui

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/ez-gz/shepherd/internal/control"
	"github.com/ez-gz/shepherd/internal/shepherd"
	"github.com/ez-gz/shepherd/internal/workstream"
)

func TestEveryDashboardSurfaceRequestsCellMouseEvents(t *testing.T) {
	model, _ := newTestModel("/tmp", shepherd.BackendCodex)
	model.width, model.height = 80, 24
	for _, configure := range []func(*Model){
		func(*Model) {},
		func(m *Model) { m.screen = screenSettings },
		func(m *Model) { m.overlay = overlayHelp },
	} {
		candidate := model
		configure(&candidate)
		if got := candidate.View().MouseMode; got != tea.MouseModeCellMotion {
			t.Fatalf("mouse mode = %v, want cell motion", got)
		}
	}
}

func TestMouseClickSelectsVisibleRowWithoutAttaching(t *testing.T) {
	model, _ := mouseTestModel(t)
	rows := model.rows()
	target := rows[2]
	updated, cmd := model.Update(tea.MouseClickMsg(tea.Mouse{X: 12, Y: dashboardListTop + 2, Button: tea.MouseLeft}))
	model = updated.(Model)
	if model.selected != target.key {
		t.Fatalf("selected = %q, want %q", model.selected, target.key)
	}
	if model.busy || strings.Contains(model.notice, "opening terminal") {
		t.Fatal("a selection click inherited Enter's attach behavior")
	}
	// Selecting a live session may request its preview, but it must never return
	// an ExecProcess attach command. Running the command here is safe because
	// the fake controller only returns a preview message.
	for _, message := range collectMessages(cmd) {
		if _, ok := message.(attachReadyMsg); ok {
			t.Fatal("selection click requested an attach")
		}
	}
}

func TestMouseClickUsesScrolledListViewport(t *testing.T) {
	model, _ := mouseTestModel(t)
	model.height = 12
	rows := model.rows()
	model.cursor = len(rows) - 1
	model.selected = rows[model.cursor].key
	start := model.listViewportStart(rows)

	updated, _ := model.Update(tea.MouseClickMsg(tea.Mouse{X: 10, Y: dashboardListTop, Button: tea.MouseLeft}))
	model = updated.(Model)
	if model.selected != rows[start].key {
		t.Fatalf("scrolled click selected %q, want viewport row %q", model.selected, rows[start].key)
	}
}

func TestDisclosureClickTogglesOnlyWorkstreamTriangle(t *testing.T) {
	model, _ := mouseTestModel(t)
	rows := model.rows()
	header := rows[0]

	updated, _ := model.Update(tea.MouseClickMsg(tea.Mouse{X: 2, Y: dashboardListTop, Button: tea.MouseLeft}))
	model = updated.(Model)
	if model.selected != header.key || !model.collapsed[header.key] {
		t.Fatalf("triangle click selected=%q collapsed=%v", model.selected, model.collapsed[header.key])
	}

	updated, _ = model.Update(tea.MouseClickMsg(tea.Mouse{X: 12, Y: dashboardListTop, Button: tea.MouseLeft}))
	model = updated.(Model)
	if !model.collapsed[header.key] {
		t.Fatal("ordinary row click toggled the disclosure")
	}
}

func TestMouseWheelMovesDashboardSelectionAndPanelViewports(t *testing.T) {
	model, _ := mouseTestModel(t)
	before := model.cursor
	updated, _ := model.Update(tea.MouseWheelMsg(tea.Mouse{X: 5, Y: dashboardListTop, Button: tea.MouseWheelDown}))
	model = updated.(Model)
	if model.cursor <= before {
		t.Fatalf("dashboard wheel left cursor at %d from %d", model.cursor, before)
	}

	model.overlay = overlayHelp
	model.width, model.height = 40, 12
	updated, _ = model.Update(tea.MouseWheelMsg(tea.Mouse{X: 5, Y: 5, Button: tea.MouseWheelDown}))
	model = updated.(Model)
	if model.helpOffset == 0 {
		t.Fatal("help wheel did not scroll")
	}

	model.overlay, model.screen = overlayNone, screenSettings
	updated, _ = model.Update(tea.MouseWheelMsg(tea.Mouse{X: 5, Y: 5, Button: tea.MouseWheelDown}))
	model = updated.(Model)
	if model.settingsOffset == 0 {
		t.Fatal("settings wheel did not scroll")
	}
}

func TestMouseCannotRedirectPinnedComposer(t *testing.T) {
	model, _ := mouseTestModel(t)
	rows := model.rows()
	model.selected = rows[1].key
	model.restoreSelection()
	selected, ok := model.selectedSession()
	if !ok {
		t.Fatal("fixture row is not a session")
	}
	model.replyTarget = selected.ID

	updated, _ := model.Update(tea.MouseClickMsg(tea.Mouse{X: 10, Y: dashboardListTop + 2, Button: tea.MouseLeft}))
	model = updated.(Model)
	if model.selected != rows[1].key || !strings.Contains(model.notice, "replying to") {
		t.Fatalf("pinned click selected=%q notice=%q", model.selected, model.notice)
	}
}

func TestPointerGestureDisarmsDestructiveConfirmations(t *testing.T) {
	model, _ := mouseTestModel(t)
	model.confirmStop = "session-mouse-a"
	model.confirmDelete = "session-mouse-b"
	model.confirmArchive = "workstream-mouse"

	updated, _ := model.Update(tea.MouseWheelMsg(tea.Mouse{X: 5, Y: dashboardListTop, Button: tea.MouseWheelDown}))
	model = updated.(Model)
	if model.confirmStop != "" || model.confirmDelete != "" || model.confirmArchive != "" {
		t.Fatalf("mouse left confirmations armed: stop=%q delete=%q archive=%q",
			model.confirmStop, model.confirmDelete, model.confirmArchive)
	}
}

func mouseTestModel(t *testing.T) (Model, *fakeController) {
	t.Helper()
	model, controller := newTestModel("/tmp", shepherd.BackendCodex)
	model.width, model.height = 80, 24
	now := time.Now()
	container := testWorkstream("workstream-mouse", "Mouse", []string{"/tmp"}, now)
	sessions := []control.Session{
		testDurableSession("session-mouse-a", container.ID, shepherd.BackendCodex, "first", "/tmp", now),
		testDurableSession("session-mouse-b", container.ID, shepherd.BackendClaude, "second", "/tmp", now),
		testDurableSession("session-mouse-c", container.ID, shepherd.BackendNoAgent, "third", "/tmp", now),
	}
	model.setSnapshot(control.Snapshot{Workstreams: []workstream.Workstream{container}, Sessions: sessions})
	model.selected = workstreamRowKey(container.ID)
	model.restoreSelection()
	return model, controller
}
