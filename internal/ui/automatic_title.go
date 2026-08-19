package ui

import (
	"context"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/ez-gz/shepherd/internal/autotitle"
	"github.com/ez-gz/shepherd/internal/control"
	"github.com/ez-gz/shepherd/internal/format"
	"github.com/ez-gz/shepherd/internal/shepherd"
	"github.com/ez-gz/shepherd/internal/transcript"
)

const (
	automaticTitleTimeout      = 20 * time.Second
	automaticTitleScanInterval = 5 * time.Second
)

type automaticTitleFunc func(context.Context, control.Session) (title string, outputFound bool, err error)

type automaticTitleMsg struct {
	id          string
	title       string
	outputFound bool
	err         error
}

func defaultAutomaticTitle(ctx context.Context, session control.Session) (string, bool, error) {
	output, found, err := (transcript.Reader{}).FirstCompletedOutput(transcript.CompletedOutputRequest{
		Runner: session.Backend, SessionID: session.ConversationID(), Root: session.Record.InitialRoot,
		StartedAt: session.CreatedAt, Prompt: session.Prompt,
	})
	if err != nil || !found {
		return "", found, err
	}
	title, err := autotitle.FromEnvironment().Generate(ctx, output)
	return title, true, err
}

func (m *Model) requestAutomaticTitle() tea.Cmd {
	if !m.settings.AutomaticTitle || m.automaticTitleActive != "" ||
		!autotitle.FromEnvironment().Configured() {
		return nil
	}
	if m.automaticTitles == nil {
		m.automaticTitles = make(map[string]string)
	}
	if m.automaticTitleTried == nil {
		m.automaticTitleTried = make(map[string]bool)
	}
	if m.automaticTitleScanned == nil {
		m.automaticTitleScanned = make(map[string]time.Time)
	}
	now := m.clock()
	sessions := append(append([]control.Session(nil), m.snapshot.Sessions...), m.snapshot.Orphans...)
	for _, session := range sessions {
		if !automaticTitleEligible(session) || strings.TrimSpace(session.Record.Title) != "" ||
			m.automaticTitles[session.ID] != "" || m.automaticTitleTried[session.ID] {
			continue
		}
		if scanned := m.automaticTitleScanned[session.ID]; !scanned.IsZero() && now.Sub(scanned) < automaticTitleScanInterval {
			continue
		}
		m.automaticTitleActive = session.ID
		generator := m.generateAutomaticTitle
		if generator == nil {
			generator = defaultAutomaticTitle
		}
		return func() tea.Msg {
			ctx, cancel := context.WithTimeout(context.Background(), automaticTitleTimeout)
			defer cancel()
			title, found, err := generator(ctx, session)
			return automaticTitleMsg{id: session.ID, title: title, outputFound: found, err: err}
		}
	}
	return nil
}

func automaticTitleEligible(session control.Session) bool {
	if !session.Durable || session.Orphaned || session.Backend == shepherd.BackendNoAgent {
		return false
	}
	return true
}

func (m Model) automaticTitleSettingsLine() string {
	status := "off"
	if m.settings.AutomaticTitle {
		status = "on · " + autotitle.Model
		if !autotitle.FromEnvironment().Configured() {
			status += " · OPENAI_API_KEY missing, no calls"
		}
		if failure := strings.TrimSpace(m.automaticTitleFailure); failure != "" {
			status += " · " + failure
		}
	}
	return mutedStyle.Render(" auto title ") + truncatePlain(format.OneLine(status), max(1, m.width-12))
}
