package ui

import tea "charm.land/bubbletea/v2"

const (
	// The dashboard list follows the one-line header and rule. Keeping the
	// coordinate in one place makes rendering and hit testing share the same
	// layout instead of growing parallel magic numbers.
	dashboardListTop = 2
	mouseWheelStep   = 3
)

// handleMouse gives the dashboard a deliberately small pointer contract. A
// click selects but never attaches or mutates durable state; the keyboard keeps
// those consequential verbs. Wheel events move the active viewport. Dragging
// is intentionally not interpreted here, so the terminal's bypass modifier
// remains available for native selection.
func (m Model) handleMouse(message tea.MouseMsg) (tea.Model, tea.Cmd) {
	switch message.(type) {
	case tea.MouseClickMsg, tea.MouseWheelMsg:
		// A pointer gesture is just as capable of coming between two confirmation
		// presses as a key. It must disarm destructive actions rather than let the
		// next chord confirm a decision made before the gesture.
		m.confirmStop, m.confirmDelete, m.confirmArchive = "", "", ""
	default:
		return m, nil
	}

	mouse := message.Mouse()
	if wheel, ok := message.(tea.MouseWheelMsg); ok {
		return m.handleMouseWheel(wheel.Mouse())
	}
	if click, ok := message.(tea.MouseClickMsg); ok && mouse.Button == tea.MouseLeft {
		return m.handleMouseClick(click.Mouse())
	}
	return m, nil
}

func (m Model) handleMouseWheel(mouse tea.Mouse) (tea.Model, tea.Cmd) {
	delta := 0
	switch mouse.Button {
	case tea.MouseWheelUp:
		delta = -mouseWheelStep
	case tea.MouseWheelDown:
		delta = mouseWheelStep
	default:
		return m, nil
	}

	if m.overlay == overlayHelp {
		m.helpOffset += delta
		m.clampHelpOffset()
		return m, nil
	}
	if m.screen == screenSettings {
		m.settingsOffset += delta
		m.clampSettingsOffset()
		return m, nil
	}
	if m.busy || !m.dashboardListContains(mouse.Y) {
		return m, nil
	}
	m.resizeMode = false
	m.errorText = ""
	if m.selectionLocked() {
		return m.reportSelectionLock()
	}
	return m.moveSelection(delta)
}

func (m Model) handleMouseClick(mouse tea.Mouse) (tea.Model, tea.Cmd) {
	if m.overlay == overlayHelp || m.screen == screenSettings || m.busy {
		return m, nil
	}
	index, row, ok := m.listRowAt(mouse.Y)
	if !ok {
		return m, nil
	}
	if m.selectionLocked() && row.key != m.selected {
		return m.reportSelectionLock()
	}

	m.resizeMode = false
	m.errorText = ""
	updated, cmd := m.moveSelection(index - m.cursor)
	m = updated.(Model)

	// Workstream rows draw their disclosure triangle in column two. Only that
	// small target toggles the group; clicking the rest of a row merely selects
	// it, and no click ever inherits Enter's attach behavior.
	if mouse.X == 2 && (row.kind == rowWorkstream || row.kind == rowOrphanHeader) {
		m.collapsed[row.key] = !m.collapsed[row.key]
		m.restoreSelection()
	}
	return m, cmd
}

func (m Model) dashboardListContains(y int) bool {
	return y >= dashboardListTop && y < dashboardListTop+m.listHeight()
}

func (m Model) listRowAt(y int) (int, listRow, bool) {
	if !m.dashboardListContains(y) {
		return 0, listRow{}, false
	}
	rows := m.rows()
	index := m.listViewportStart(rows) + y - dashboardListTop
	if index < 0 || index >= len(rows) {
		return 0, listRow{}, false
	}
	return index, rows[index], true
}

func (m Model) listViewportStart(rows []listRow) int {
	height := m.listHeight()
	start := 0
	if m.cursor >= height {
		start = m.cursor - height + 1
	}
	if start+height > len(rows) {
		start = max(0, len(rows)-height)
	}
	return start
}
