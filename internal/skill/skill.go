// Package skill embeds the 1ctl chat agent skill (SKILL.md), which is
// injected into the chat session as the system prompt. The embedded copy
// is the default; SATUSKY_CHAT_SKILL (a file path) and /skill override it
// at runtime. Name() reports the active source for /skill display.
package skill

import (
	_ "embed"
	"fmt"
	"os"
)

//go:embed SKILL.md
var skillMD string

// embeddedName is the display name of the default embedded skill.
const embeddedName = "embedded SKILL.md"

// name tracks the source of the most recently loaded skill.
var name = embeddedName

// Load returns the skill content for the chat system prompt: the embedded
// SKILL.md by default, or the file at SATUSKY_CHAT_SKILL when that env var
// is set. A missing or unreadable override file is an error (the chat
// refuses to start with a broken skill rather than silently falling back).
func Load() (string, error) {
	if path := os.Getenv("SATUSKY_CHAT_SKILL"); path != "" {
		content, err := LoadPath(path)
		if err != nil {
			return "", err
		}
		return content, nil
	}
	name = embeddedName
	return skillMD, nil
}

// LoadPath reads a skill file from disk and records it as the active
// source. Used by the SATUSKY_CHAT_SKILL override and /skill <path>.
func LoadPath(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("load skill %s: %w", path, err)
	}
	name = path
	return string(content), nil
}

// Name returns the source of the currently loaded skill ("embedded
// SKILL.md" or the file path) for /skill display.
func Name() string { return name }
