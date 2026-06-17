package commands

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"
)

// CompletionCommand returns the completion command group.
//
// These scripts use v3's --generate-shell-completion flag which dynamically
// outputs commands/flags/descriptions based on the live command tree.
// No manual updates needed when commands are added or removed.
//
// Unlike v3's built-in embedded scripts (which have a zsh syntax bug in
// v3.10.0), these use proper zsh array syntax.
func CompletionCommand() *cli.Command {
	return &cli.Command{
		Name:  "completion",
		Usage: "Generate shell completion scripts",
		Commands: []*cli.Command{
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
  current=${words[-1]}
  if [[ "$current" == "-"* ]]; then
    opts=("${(@f)$(%[1]s ${words[2,$#words-1]} ${current} --generate-shell-completion 2>/dev/null)}")
  else
    opts=("${(@f)$(%[1]s ${words[2,$#words-1]} --generate-shell-completion 2>/dev/null)}")
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
