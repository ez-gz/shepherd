package learnshepherd

import (
	"strings"
	"testing"
)

// The Quickstart process and the dashboard temporarily hand control back and
// forth. If the guide drops one half of that round trip, a first-time user is
// stranded outside the agent that was meant to teach the next step.
func TestInstructionsCarryTheWholeDashboardRoundTrip(t *testing.T) {
	instructions := strings.Join(strings.Fields(Instructions), " ")
	required := []string{
		"`Ctrl-b`, release both keys, then press `d`",
		"With `Quickstart` selected and the composer empty, press `Space`",
		"type `I made it back`, and press `Enter`",
		"automatically clears the composer and releases the reply",
		"Do **not** press `Esc`",
		"Press `Enter` with the empty composer to attach to it again",
	}
	for _, text := range required {
		if !strings.Contains(instructions, text) {
			t.Errorf("Quickstart instructions lost required round-trip text %q", text)
		}
	}

	for _, obsolete := range []string{
		"Press `Esc` to leave reply mode",
		"reattach with `Esc` followed by `Enter`",
		"`Ctrl-C`, or `Esc` with an empty composer",
	} {
		if strings.Contains(instructions, obsolete) {
			t.Errorf("Quickstart instructions still contain obsolete advice %q", obsolete)
		}
	}
}

func TestInstructionsTeachOutcomeTopologyAndRenderedArtifacts(t *testing.T) {
	instructions := strings.Join(strings.Fields(Instructions), " ")
	for _, text := range []string{
		"workstream: one outcome or bundle of related work",
		"one or more directory routes",
		"Development, testing, QA",
		"`artifact_dir`",
		"`notes.md`",
		"`qa/checklist.md`",
		"`handoff.md`",
		"automatic titles are off by default",
		"keep asking this Quickstart agent to manage Shepherd",
	} {
		if !strings.Contains(instructions, text) {
			t.Errorf("Quickstart instructions lost onboarding contract %q", text)
		}
	}
}
