// Package skill embeds the 1ctl chat agent skill (SKILL.md), which is
// injected into the chat session as the system prompt. The full skill
// (tools, SatuSky copilot rules) lands in Phase 3/4; this is the v1 draft.
package skill

import _ "embed"

//go:embed SKILL.md
var skillMD string

// Load returns the embedded SKILL.md content.
func Load() string { return skillMD }
