// Package learnshepherd exposes the agent-readable onboarding skill to the
// installed Shepherd binary.
package learnshepherd

import _ "embed"

// Instructions is the canonical onboarding skill used by shepherd quickstart.
//
//go:embed SKILL.md
var Instructions string
