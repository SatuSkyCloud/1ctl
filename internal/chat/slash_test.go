package chat

import (
	"strings"
	"testing"
)

func TestParseSlash(t *testing.T) {
	tests := []struct {
		name    string
		line    string
		wantCmd SlashCommand
		wantArg string
		isSlash bool
	}{
		{name: "connect with provider", line: "/connect openai", wantCmd: CmdConnect, wantArg: "openai", isSlash: true},
		{name: "connect no arg", line: "/connect", wantCmd: CmdConnect, wantArg: "", isSlash: true},
		{name: "switch", line: "/switch claude", wantCmd: CmdSwitch, wantArg: "claude", isSlash: true},
		{name: "disconnect", line: "/disconnect", wantCmd: CmdDisconnect, wantArg: "", isSlash: true},
		{name: "disconnect with arg", line: "/disconnect deepseek", wantCmd: CmdDisconnect, wantArg: "deepseek", isSlash: true},
		{name: "providers", line: "/providers", wantCmd: CmdProviders, wantArg: "", isSlash: true},
		{name: "tools on", line: "/tools on", wantCmd: CmdTools, wantArg: "on", isSlash: true},
		{name: "tools off", line: "/tools off", wantCmd: CmdTools, wantArg: "off", isSlash: true},
		{name: "tools bare", line: "/tools", wantCmd: CmdTools, wantArg: "", isSlash: true},
		{name: "ask", line: "/ask", wantCmd: CmdAsk, wantArg: "", isSlash: true},
		{name: "ask ignores extra args", line: "/ask now please", wantCmd: CmdAsk, wantArg: "now please", isSlash: true},
		{name: "go", line: "/go", wantCmd: CmdGo, wantArg: "", isSlash: true},
		{name: "skill show", line: "/skill", wantCmd: CmdSkill, wantArg: "", isSlash: true},
		{name: "skill load", line: "/skill custom.md", wantCmd: CmdSkill, wantArg: "custom.md", isSlash: true},
		{name: "model show", line: "/model", wantCmd: CmdModel, wantArg: "", isSlash: true},
		{name: "model set", line: "/model gpt-4o", wantCmd: CmdModel, wantArg: "gpt-4o", isSlash: true},
		{name: "export no path", line: "/export", wantCmd: CmdExport, wantArg: "", isSlash: true},
		{name: "export with path", line: "/export notes.md", wantCmd: CmdExport, wantArg: "notes.md", isSlash: true},
		{name: "clear", line: "/clear", wantCmd: CmdClear, wantArg: "", isSlash: true},
		{name: "help", line: "/help", wantCmd: CmdHelp, wantArg: "", isSlash: true},
		{name: "exit", line: "/exit", wantCmd: CmdExit, wantArg: "", isSlash: true},
		{name: "quit alias", line: "/quit", wantCmd: CmdExit, wantArg: "", isSlash: true},
		{name: "case insensitive", line: "/EXIT", wantCmd: CmdExit, wantArg: "", isSlash: true},
		{name: "leading whitespace", line: "  /clear  ", wantCmd: CmdClear, wantArg: "", isSlash: true},
		{name: "extra whitespace in arg", line: "/model   gpt-4o   ", wantCmd: CmdModel, wantArg: "gpt-4o", isSlash: true},
		{name: "plain message", line: "hello there", wantCmd: CmdUnknown, wantArg: "hello there", isSlash: false},
		{name: "empty line", line: "", wantCmd: CmdUnknown, wantArg: "", isSlash: false},
		{name: "unknown slash", line: "/frobnicate", wantCmd: CmdUnknown, wantArg: "/frobnicate", isSlash: true},
		{name: "slash only", line: "/", wantCmd: CmdUnknown, wantArg: "/", isSlash: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, arg, isSlash := ParseSlash(tt.line)
			if cmd != tt.wantCmd {
				t.Errorf("ParseSlash(%q) cmd = %v, want %v", tt.line, cmd, tt.wantCmd)
			}
			if arg != tt.wantArg {
				t.Errorf("ParseSlash(%q) arg = %q, want %q", tt.line, arg, tt.wantArg)
			}
			if isSlash != tt.isSlash {
				t.Errorf("ParseSlash(%q) isSlash = %v, want %v", tt.line, isSlash, tt.isSlash)
			}
		})
	}
}

func TestHelpTextListsCommands(t *testing.T) {
	for _, cmd := range []string{"/connect", "/switch", "/disconnect", "/providers", "/model", "/tools", "/ask", "/go", "/skill", "/export", "/clear", "/help", "/exit"} {
		if !strings.Contains(helpText, cmd) {
			t.Errorf("helpText does not mention %s", cmd)
		}
	}
}
