package chat

import "strings"

// SlashCommand identifies a REPL slash command.
type SlashCommand int

const (
	// CmdUnknown is a non-slash line or an unrecognized slash command.
	CmdUnknown SlashCommand = iota
	CmdConnect
	CmdSwitch
	CmdDisconnect
	CmdProviders
	CmdModel
	CmdStatus
	CmdTools
	CmdAsk
	CmdGo
	CmdSkill
	CmdExport
	CmdClear
	CmdHelp
	CmdExit
)

// helpText lists every Phase 1+2 slash command (see plan §2.6).
const helpText = `Commands:
  /connect [provider]     Connect or reconnect a provider (openai, claude, deepseek)
  /switch <provider>      Switch the active provider (auto-tests if never verified)
  /disconnect [provider]  Remove a provider's API key (default: active provider)
  /providers              Show providers, models and connection status
  /model [name]           Show the active model, or set a new one
  /status                 Refresh and show the SatuSky state snapshot (profile, apps, databases, domains, credits)
  /tools on|off           Toggle workspace tools (files, shell, SatuSky) — default on
  /ask                    Ask clarifying questions before acting (this session)
  /go                     Skip questions and act directly (this session)
  /skill [path]           Show the loaded skill, or load an alternate SKILL.md
  /export [path]          Save the conversation transcript as markdown
  /clear                  Reset the conversation (keeps the connection)
  /help                   Show this help
  /exit  (/quit)          Leave the chat

Anything else is sent to the model as a message.`

// ParseSlash parses a line into a slash command and its argument. The bool
// reports whether the line was slash-prefixed: CmdUnknown with isSlash=true
// means an unknown slash command; isSlash=false means a plain message.
func ParseSlash(line string) (cmd SlashCommand, arg string, isSlash bool) {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "/") {
		return CmdUnknown, line, false
	}
	rest := strings.TrimSpace(strings.TrimPrefix(trimmed, "/"))
	name, arg := rest, ""
	if i := strings.IndexAny(rest, " \t"); i >= 0 {
		name, arg = strings.TrimSpace(rest[:i]), strings.TrimSpace(rest[i+1:])
	}
	switch strings.ToLower(name) {
	case "connect":
		return CmdConnect, arg, true
	case "switch":
		return CmdSwitch, arg, true
	case "disconnect":
		return CmdDisconnect, arg, true
	case "providers":
		return CmdProviders, arg, true
	case "model":
		return CmdModel, arg, true
	case "status":
		return CmdStatus, arg, true
	case "tools":
		return CmdTools, arg, true
	case "ask":
		return CmdAsk, arg, true
	case "go":
		return CmdGo, arg, true
	case "skill":
		return CmdSkill, arg, true
	case "export":
		return CmdExport, arg, true
	case "clear":
		return CmdClear, arg, true
	case "help":
		return CmdHelp, arg, true
	case "exit", "quit":
		return CmdExit, arg, true
	default:
		return CmdUnknown, line, true
	}
}
