package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"1ctl/internal/utils"

	"github.com/urfave/cli/v3"
)

// CompletionCommand returns the completion command group.
//
// These scripts use v3's --generate-shell-completion flag which dynamically
// outputs commands/flags/descriptions based on the live command tree.
// No manual updates needed when commands are added or removed.
//
// Use `1ctl completion install` to auto-detect your shell and install.
func CompletionCommand() *cli.Command {
	return &cli.Command{
		Name:  "completion",
		Usage: "Generate and install shell completion scripts",
		Commands: []*cli.Command{
			{
				Name:   "install",
				Usage:  "Auto-detect shell and install completion (add one line to shell config)",
				Action: handleCompletionInstall,
			},
			{
				Name:   "bash",
				Usage:  "Generate bash completion script",
				Action: handleBashCompletion,
			},
			{
				Name:   "zsh",
				Usage:  "Generate zsh completion script",
				Action: handleZshCompletion,
			},
			{
				Name:   "fish",
				Usage:  "Generate fish completion script",
				Action: handleFishCompletion,
			},
			{
				Name:   "powershell",
				Usage:  "Generate PowerShell completion script",
				Action: handlePowerShellCompletion,
			},
		},
	}
}

func handleBashCompletion(ctx context.Context, cmd *cli.Command) error {
	appName := cmd.Root().Name
	script := fmt.Sprintf(`#/usr/bin/env bash

__%[1]s_init_completion() {
  COMPREPLY=()
  if declare -F _init_completion >/dev/null 2>&1; then
    _init_completion "$@"
  else
    COMPREPLY=()
    _get_comp_words_by_ref "$@" cur prev words cword
  fi
}

__%[1]s_build_completion_request() {
  local -a words_before_cursor=("${COMP_WORDS[@]:0:${COMP_CWORD}}")
  local current_word="${COMP_WORDS[COMP_CWORD]}"

  if [[ "${current_word}" == "-"* ]]; then
    printf '%%s %%s --generate-shell-completion' "${words_before_cursor[*]}" "${current_word}"
  else
    printf '%%s --generate-shell-completion' "${words_before_cursor[*]}"
  fi
}

_%[1]s_completions() {
  local cur prev words cword
  __%[1]s_init_completion -n "=:" || return

  local completion_request
  completion_request="$(__%[1]s_build_completion_request)"

  local line
  local longest=0
  local -a cli_tokens
  local -a cli_descriptions

  while IFS= read -r line; do
    local token="${line}"
    local description=""
    
    if [[ "${line}" == *:* ]]; then
      token="${line%%%%:*}"
      description="${line#*:}"
    fi

    if [[ -z "${token}" ]]; then
      continue
    fi

    cli_tokens+=("${token}")
    cli_descriptions+=("${description}")
    (( ${#token} > longest )) && longest=${#token}
  done < <(eval "${completion_request}" 2>/dev/null)

  local cur="${COMP_WORDS[COMP_CWORD]}"
  local -a matches=( $(compgen -W "${cli_tokens[*]}" -- "${cur}") )

  if [[ ${#matches[@]} -gt 0 ]]; then
    local candidate
    for candidate in "${matches[@]}"; do
      local idx=0
      for i in "${!cli_tokens[@]}"; do
        if [[ "${cli_tokens[$i]}" == "${candidate}" ]]; then
          idx=$i
          break
        fi
      done
      local desc="${cli_descriptions[$idx]}"
      if [[ -n "${desc}" ]]; then
        COMPREPLY+=("${candidate}")
      else
        COMPREPLY+=("${candidate}")
      fi
    done
  fi
}

complete -o bashdefault -o default -F _%[1]s_completions %[1]s
`, appName)
	_, err := fmt.Fprint(cmd.Root().Writer, script)
	return err
}

func handleZshCompletion(ctx context.Context, cmd *cli.Command) error {
	appName := cmd.Root().Name
	script := fmt.Sprintf(`#compdef %[1]s
compdef _%[1]s %[1]s

_%[1]s() {
  local -a opts
  local current
  local previous
  local executable
  current=${words[-1]}
  previous=${words[-2]}
  executable=${words[1]}
  if [[ "$current" == "-"* ]]; then
    opts=("${(@f)$(__1CTL_COMPLETE_CURRENT="${current}" __1CTL_COMPLETE_PREV="${previous}" "${executable}" ${words[2,$#words-1]} ${current} --generate-shell-completion 2>/dev/null)}")
  else
    opts=("${(@f)$(__1CTL_COMPLETE_CURRENT="${current}" __1CTL_COMPLETE_PREV="${previous}" "${executable}" ${words[2,$#words-1]} --generate-shell-completion 2>/dev/null)}")
  fi

  if [[ "${opts[1]}" != "" ]]; then
    _describe 'values' opts
  else
    _files
  fi
}

if [ "$funcstack[1]" = "_%[1]s" ]; then
  _%[1]s
fi
`, appName)
	_, err := fmt.Fprint(cmd.Root().Writer, script)
	return err
}

func handleFishCompletion(ctx context.Context, cmd *cli.Command) error {
	appName := cmd.Root().Name
	script := fmt.Sprintf(`# Fish completion for %[1]s

function __fish_%[1]s_complete
  set -l cmd (commandline -opc)
  set -e cmd[1]
  %[1]s $cmd --generate-shell-completion 2>/dev/null
end

complete -c %[1]s -f -a '(__fish_%[1]s_complete)'
`, appName)
	_, err := fmt.Fprint(cmd.Root().Writer, script)
	return err
}

func handlePowerShellCompletion(ctx context.Context, cmd *cli.Command) error {
	appName := cmd.Root().Name
	script := fmt.Sprintf(`using namespace System.Management.Automation
using namespace System.Management.Automation.Language

Register-ArgumentCompleter -Native -CommandName '%[1]s' -ScriptBlock {
    param($wordToComplete, $commandAst, $cursorPosition)

    $commandElements = $commandAst.CommandElements
    $command = @(
        '%[1]s'
        for ($i = 1; $i -lt $commandElements.Count; $i++) {
            $element = $commandElements[$i]
            if ($element -isnot [StringConstantExpressionAst] -or
                $element.StringConstantType -ne [StringConstantType]::BareWord -or
                $element.Value.StartsWith('-')) {
                break
            }
            $element.Value
        }
    ) -join ';'

    $completions = @(
        %[1]s --generate-shell-completion $commandElements[1..$($commandElements.Count - 1)] 2>$null | ForEach-Object {
            [CompletionResult]::new($_, $_, [CompletionResultType]::ParameterValue, $_)
        }
    )
    $completions
}
`, appName)
	_, err := fmt.Fprint(cmd.Root().Writer, script)
	return err
}

func handleCompletionInstall(ctx context.Context, cmd *cli.Command) error {
	appName := cmd.Root().Name
	shell := os.Getenv("SHELL")
	home, _ := os.UserHomeDir()

	type shellInfo struct {
		name        string
		dir         string
		file        string
		config      string
		scriptFunc  func(context.Context, *cli.Command) error
		postInstall string
	}

	var si shellInfo

	switch {
	case strings.Contains(shell, "zsh"):
		si = shellInfo{
			name:        "zsh",
			dir:         filepath.Join(home, ".zsh", "completions"),
			file:        "_" + appName,
			config:      fmt.Sprintf("fpath=(%s $fpath)", filepath.Join(home, ".zsh", "completions")),
			scriptFunc:  handleZshCompletion,
			postInstall: "rm -f ~/.zcompdump && compinit",
		}
	case strings.Contains(shell, "bash"):
		si = shellInfo{
			name:        "bash",
			dir:         filepath.Join(home, ".bash_completion.d"),
			file:        appName,
			config:      fmt.Sprintf("source %s/%s", filepath.Join(home, ".bash_completion.d"), appName),
			scriptFunc:  handleBashCompletion,
			postInstall: fmt.Sprintf("source %s/%s", filepath.Join(home, ".bash_completion.d"), appName),
		}
	case strings.Contains(shell, "fish"):
		installDir := filepath.Join(home, ".config", "fish", "completions")
		si = shellInfo{
			name:       "fish",
			dir:        installDir,
			file:       appName + ".fish",
			config:     "(auto-loaded by fish, nothing to add)",
			scriptFunc: handleFishCompletion,
		}
	case strings.Contains(shell, "pwsh") || strings.Contains(shell, "powershell"):
		utils.PrintInfo("PowerShell detected. Use:")
		fmt.Println()
		fmt.Println("  1ctl completion powershell >> $PROFILE")
		fmt.Println()
		utils.PrintInfo("Or run in zsh, bash, or fish.")
		return nil
	default:
		utils.PrintWarning("Cannot detect shell from SHELL=%s", shell)
		utils.PrintInfo("Supported: zsh, bash, fish. Use 1ctl completion <shell> to generate manually.")
		return nil
	}

	if err := os.MkdirAll(si.dir, 0755); err != nil {
		return utils.NewError(fmt.Sprintf("failed to create %s: %s", si.dir, err.Error()), nil)
	}

	installPath := filepath.Join(si.dir, si.file)
	f, err := os.Create(installPath)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to write %s: %s", installPath, err.Error()), nil)
	}
	defer f.Close()

	// Swap writer temporarily to write script into file
	origWriter := cmd.Root().Writer
	cmd.Root().Writer = f
	err = si.scriptFunc(ctx, cmd)
	cmd.Root().Writer = origWriter
	if err != nil {
		return err
	}

	utils.PrintSuccess("Installed %s completion at %s", si.name, installPath)
	fmt.Println()
	fmt.Printf("  # Add this ONE line to your ~/.%src and never touch it again:\n", si.name)
	fmt.Printf("  %s\n", si.config)
	fmt.Println()
	if si.postInstall != "" {
		utils.PrintInfo("Then run: %s", si.postInstall)
	}
	fmt.Println()
	utils.PrintInfo("Completions auto-update when 1ctl changes — no re-install needed.")
	return nil
}
