// Package tools implements the workspace tools available to the chat
// agent: read_file, write_file and list_dir for the filesystem, and
// run_shell for commands. Execution is sandboxed to the chat working
// directory (see Executor): absolute paths and ".." traversal are
// rejected, symlink escapes are refused, run_shell always requires
// confirmation, and overwriting an existing file requires confirmation.
package tools

import (
	openai "github.com/sashabaranov/go-openai"
)

// ToolDef describes one workspace tool: its name, a description the model
// sees, and its JSON-schema parameters.
type ToolDef struct {
	Name        string
	Description string
	Parameters  map[string]any
}

// Definitions returns the workspace tool set as go-openai Tool objects,
// ready to attach to a ChatCompletionRequest.
func Definitions() []openai.Tool {
	parameters := func(required []string, props map[string]any) map[string]any {
		schema := map[string]any{"type": "object", "properties": props}
		if len(required) > 0 {
			schema["required"] = required
		}
		return schema
	}
	stringProp := func(desc string) map[string]any { return map[string]any{"type": "string", "description": desc} }
	intProp := func(desc string) map[string]any { return map[string]any{"type": "integer", "description": desc} }
	stringArrayProp := func(desc string) map[string]any {
		return map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": desc}
	}

	defs := []ToolDef{
		{
			Name:        "read_file",
			Description: "Read a file from the chat working directory. Returns the file content (capped at 100k characters), optionally restricted to a line range. All paths are relative to the chat working directory.",
			Parameters: parameters([]string{"path"}, map[string]any{
				"path":   stringProp("Path to the file, relative to the chat working directory"),
				"offset": intProp("1-based starting line (default 1)"),
				"limit":  intProp("Maximum number of lines to return (default: the whole file)"),
			}),
		},
		{
			Name:        "write_file",
			Description: "Create or overwrite a file in the chat working directory. Parent directories are created automatically. Writing to an existing file requires user confirmation. All paths are relative to the chat working directory.",
			Parameters: parameters([]string{"path", "content"}, map[string]any{
				"path":    stringProp("Path to the file, relative to the chat working directory"),
				"content": stringProp("The full content of the file"),
			}),
		},
		{
			Name:        "list_dir",
			Description: "List the entries of a directory in the chat working directory: one line per entry with kind (dir/file) and size. All paths are relative to the chat working directory.",
			Parameters: parameters(nil, map[string]any{
				"path": stringProp("Directory to list, relative to the chat working directory (default: the working directory itself)"),
			}),
		},
		{
			Name:        "run_shell",
			Description: "Run a shell command (sh -c) in the chat working directory with a 60s timeout. Captures stdout and stderr (each capped at 32k characters) and returns the exit code. ALWAYS requires user confirmation; destructive commands are refused outright.",
			Parameters: parameters([]string{"command"}, map[string]any{
				"command": stringProp("The shell command to run"),
				"cwd":     stringProp("Working directory for the command, relative to the chat working directory (optional)"),
			}),
		},
		{
			Name:        "satusky_status",
			Description: "Inspect the current SatuSky state: authentication, profile, org, namespace, apps, managed databases, domains and credits. Refreshes the state snapshot via read-only 1ctl commands and returns a compact digest. Call this first before advising on or proposing any SatuSky action, so advice matches the user's real setup.",
			Parameters:  parameters(nil, map[string]any{}),
		},
		{
			Name:        "satusky_run",
			Description: "Run a real 1ctl command (1ctl <args>) and return its output. Read-only commands (list, status, get, doctor, ...) run immediately; mutating commands (create, delete, set, deploy, ...) require user confirmation with a preview first, and destructive operations carry a warning. Pass a JSON array of argument strings: {\"args\":[\"postgres\",\"list\"]} or {\"args\":[\"app\",\"get\",\"my-app\"]}. Never invent subcommands — when unsure, run {\"args\":[\"<cmd>\",\"--help\"]}. Do not pass shell metacharacters or pipes; each argument is passed verbatim to 1ctl.",
			Parameters: parameters(nil, map[string]any{
				"args":    stringArrayProp("1ctl arguments as a JSON array of strings, e.g. [\"postgres\",\"list\"]"),
				"command": stringProp("Alternative to args: a single command string parsed with shell-like splitting, e.g. \"postgres list\""),
			}),
		},
	}

	tools := make([]openai.Tool, 0, len(defs))
	for _, d := range defs {
		tools = append(tools, openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        d.Name,
				Description: d.Description,
				Parameters:  d.Parameters,
			},
		})
	}
	return tools
}
