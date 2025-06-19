package commands

import (
	"1ctl/internal/utils"

	"github.com/urfave/cli/v2"
)

func CompletionCommand() *cli.Command {
	return &cli.Command{
		Name:  "completion",
		Usage: "Generate shell completion scripts",
		Subcommands: []*cli.Command{
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

func handleBashCompletion(c *cli.Context) error {
	script := `#/usr/bin/env bash

_satusky_cli_completion() {
    local cur prev opts cmd
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"
    cmd="${COMP_WORDS[1]}"

    # Top level commands
    if [[ $COMP_CWORD == 1 ]]; then
        opts="auth deploy service secret ingress issuer environment machine completion --help --version"
        COMPREPLY=( $(compgen -W "${opts}" -- ${cur}) )
        return 0
    fi

    # Handle flag values
    case "${prev}" in
        --machine)
            # TODO: Fetch actual machine list from API
            local machines=$(1ctl machine list --quiet 2>/dev/null)
            COMPREPLY=( $(compgen -W "${machines}" -- ${cur}) )
            return 0
            ;;
        --cpu)
            COMPREPLY=( $(compgen -W "0.5 1 2 4 8" -- ${cur}) )
            return 0
            ;;
        --memory)
            COMPREPLY=( $(compgen -W "512Mi 1Gi 2Gi 4Gi 8Gi 16Gi" -- ${cur}) )
            return 0
            ;;
        --volume-size)
            COMPREPLY=( $(compgen -W "1Gi 5Gi 10Gi 20Gi 50Gi 100Gi" -- ${cur}) )
            return 0
            ;;
        --port)
            COMPREPLY=( $(compgen -W "80 443 3000 8080 8443" -- ${cur}) )
            return 0
            ;;
    esac

    # Handle subcommands and their flags
    case "${cmd}" in
        auth)
            COMPREPLY=( $(compgen -W "login logout status" -- ${cur}) )
            ;;
        deploy)
            if [[ ${COMP_WORDS[2]} == "create" ]]; then
                local flags="--cpu --memory --machine --domain --organization --dockerfile --port --env --volume-size --volume-mount"
                COMPREPLY=( $(compgen -W "${flags}" -- ${cur}) )
            elif [[ ${COMP_WORDS[2]} == "get" || ${COMP_WORDS[2]} == "delete" ]]; then
                local flags="--deployment-id"
                COMPREPLY=( $(compgen -W "${flags}" -- ${cur}) )
            elif [[ ${COMP_WORDS[2]} == "list" ]]; then
                local flags="--namespace --quiet"
                COMPREPLY=( $(compgen -W "${flags}" -- ${cur}) )
            elif [[ ${COMP_WORDS[2]} == "status" ]]; then
                local flags="--deployment-id --watch"
                COMPREPLY=( $(compgen -W "${flags}" -- ${cur}) )
            elif [[ ${COMP_WORDS[2]} == "" ]]; then
                COMPREPLY=( $(compgen -W "create list delete get status" -- ${cur}) )
            fi
            ;;
        service)
            if [[ ${COMP_WORDS[2]} == "" ]]; then
                COMPREPLY=( $(compgen -W "list delete" -- ${cur}) )
            fi
            ;;
        secret)
            if [[ ${COMP_WORDS[2]} == "" ]]; then
                COMPREPLY=( $(compgen -W "create list delete" -- ${cur}) )
            fi
            ;;
        ingress)
            if [[ ${COMP_WORDS[2]} == "" ]]; then
                COMPREPLY=( $(compgen -W "list delete" -- ${cur}) )
            fi
            ;;
        issuer)
            if [[ ${COMP_WORDS[2]} == "" ]]; then
                COMPREPLY=( $(compgen -W "create list delete" -- ${cur}) )
            fi
            ;;
        environment)
            if [[ ${COMP_WORDS[2]} == "" ]]; then
                COMPREPLY=( $(compgen -W "create list delete" -- ${cur}) )
            fi
            ;;
        machine)
            if [[ ${COMP_WORDS[2]} == "" ]]; then
                COMPREPLY=( $(compgen -W "list get" -- ${cur}) )
            elif [[ ${COMP_WORDS[2]} == "get" ]]; then
                local flags="--machine-id --name"
                COMPREPLY=( $(compgen -W "${flags}" -- ${cur}) )
            elif [[ ${COMP_WORDS[2]} == "list" ]]; then
                local flags="--quiet"
                COMPREPLY=( $(compgen -W "${flags}" -- ${cur}) )
            fi
            ;;
        completion)
            if [[ ${COMP_WORDS[2]} == "" ]]; then
                COMPREPLY=( $(compgen -W "bash zsh fish powershell" -- ${cur}) )
            fi
            ;;
    esac
    return 0
}

complete -F _satusky_cli_completion 1ctl`

	utils.PrintInfo("%s", script)
	utils.PrintInfo("\n# Add this to ~/.bashrc:")
	utils.PrintInfo("# source <(1ctl completion bash)")
	return nil
}

func handleZshCompletion(c *cli.Context) error {
	script := `#compdef 1ctl

_satusky_cli() {
    local curcontext="$curcontext" state line
    typeset -A opt_args

    local -a common_flags
    common_flags=(
        '--help[Show help]'
        '--version[Show version]'
    )

    local -a deploy_create_flags deploy_list_flags deploy_get_flags deploy_status_flags
    deploy_create_flags=(
        '--cpu[CPU cores allocation (e.g., '"'"'2'"'"')]:cpu:(0.5 1 2 4 8)'
        '--memory[Memory allocation (e.g., '"'"'512Mi'"'"', '"'"'2Gi'"'"')]:memory:(512Mi 1Gi 2Gi 4Gi 8Gi 16Gi)'
        '--machine[Machine name to deploy to]:machine:($(_machine_list))'
        '--domain[Custom domain (default: *.satusky.com)]'
        '--organization[Organization name to organize your resources]'
        '--dockerfile[Path to Dockerfile (default: ./Dockerfile)]:dockerfile:_files'
        '--port[Application port (default: 8080)]:port:(80 443 3000 8080 8443)'
        '--env[Environment variables (format: KEY=VALUE)]'
        '--volume-size[Storage size (e.g., '"'"'10Gi'"'"')]:size:(1Gi 5Gi 10Gi 20Gi 50Gi 100Gi)'
        '--volume-mount[Storage mount path]:path:_files -/'
    )

    deploy_list_flags=(
        '--namespace[Filter by namespace]'
        '--quiet[Only show names]'
    )

    deploy_get_flags=(
        '--deployment-id[Deployment ID to get details for]'
    )

    deploy_status_flags=(
        '--deployment-id[Deployment ID to check]'
        '--watch[Watch deployment status in real-time]'
    )

    local -a machine_get_flags machine_list_flags
    machine_get_flags=(
        '--machine-id[Machine ID to get details for]'
        '--name[Machine name to get details for]'
    )

    machine_list_flags=(
        '--quiet[Only show names]'
    )

    _arguments -C \
        $common_flags \
        ': :->command' \
        '*:: :->args' && ret=0

    function _machine_list() {
        local machines
        machines=(${(f)"$(1ctl machine list --quiet 2>/dev/null)"})
        _values 'machines' $machines
    }

    case $state in
        command)
            local -a commands
            commands=(
                'auth:Display commands for authentication'
                'deploy:Manage deployments'
                'service:Manage services'
                'secret:Manage secrets'
                'ingress:Manage ingresses'
                'issuer:Manage issuers'
                'environment:Manage environments'
                'machine:Manage machines'
                'completion:Generate shell completion scripts'
            )
            _describe -t commands 'commands' commands
            ;;
        args)
            case $words[1] in
                auth)
                    local -a subcommands=(
                        'login:Authenticate with Satusky'
                        'logout:Remove stored authentication'
                        'status:View authentication status'
                    )
                    _describe -t subcommands 'auth subcommands' subcommands
                    ;;
                deploy)
                    if (( CURRENT == 2 )); then
                        local -a subcommands=(
                            'list:List deployments'
                            'get:Get deployment details'
                            'status:Check deployment status'
                        )
                        _describe -t subcommands 'deploy subcommands' subcommands
                    else
                        case $words[2] in
                            list)
                                _arguments $deploy_list_flags
                                ;;
                            get)
                                _arguments $deploy_get_flags
                                ;;
                            status)
                                _arguments $deploy_status_flags
                                ;;
                            *)
                                _arguments $deploy_create_flags
                                ;;
                        esac
                    fi
                    ;;
                machine)
                    if (( CURRENT == 2 )); then
                        local -a subcommands=(
                            'list:List machines'
                            'get:Get machine details'
                        )
                        _describe -t subcommands 'machine subcommands' subcommands
                    else
                        case $words[2] in
                            get)
                                _arguments $machine_get_flags
                                ;;
                            list)
                                _arguments $machine_list_flags
                                ;;
                        esac
                    fi
                    ;;
                # ... similar patterns for other commands ...
            esac
            ;;
    esac
}

compdef _satusky_cli 1ctl`

	utils.PrintInfo("%s", script)
	utils.PrintInfo("\n# Add this to ~/.zshrc:")
	utils.PrintInfo("# source <(1ctl completion zsh)")
	return nil
}

func handleFishCompletion(c *cli.Context) error {
	script := `function __fish_1ctl_no_subcommand
    for i in (commandline -opc)
        if contains -- $i auth deploy service secret ingress issuer environment machine completion
            return 1
        end
    end
    return 0
end

function __fish_1ctl_using_command
    set -l cmd (commandline -opc)
    if [ (count $cmd) -gt 1 ]
        if [ $argv[1] = $cmd[2] ]
            return 0
        end
    end
    return 1
end

function __fish_1ctl_using_subcommand
    set -l cmd (commandline -opc)
    if [ (count $cmd) -gt 2 ]
        if [ $argv[1] = $cmd[2] ] && [ $argv[2] = $cmd[3] ]
            return 0
        end
    end
    return 1
end

# Machine name completion helper
function __fish_1ctl_machines
    1ctl machine list --quiet 2>/dev/null
end

# Common value completions
complete -c 1ctl -n '__fish_1ctl_using_command deploy' -l cpu -xa '0.5 1 2 4 8'
complete -c 1ctl -n '__fish_1ctl_using_command deploy' -l memory -xa '512Mi 1Gi 2Gi 4Gi 8Gi 16Gi'
complete -c 1ctl -n '__fish_1ctl_using_command deploy' -l machine -xa '(__fish_1ctl_machines)'
complete -c 1ctl -n '__fish_1ctl_using_command deploy' -l port -xa '80 443 3000 8080 8443'
complete -c 1ctl -n '__fish_1ctl_using_command deploy' -l volume-size -xa '1Gi 5Gi 10Gi 20Gi 50Gi 100Gi'

# Top level commands
complete -c 1ctl -f -n __fish_1ctl_no_subcommand -a auth -d 'Display commands for authentication'
complete -c 1ctl -f -n __fish_1ctl_no_subcommand -a deploy -d 'Manage deployments'
complete -c 1ctl -f -n __fish_1ctl_no_subcommand -a service -d 'Manage services'
complete -c 1ctl -f -n __fish_1ctl_no_subcommand -a secret -d 'Manage secrets'
complete -c 1ctl -f -n __fish_1ctl_no_subcommand -a ingress -d 'Manage ingresses'
complete -c 1ctl -f -n __fish_1ctl_no_subcommand -a issuer -d 'Manage certificate issuers'
complete -c 1ctl -f -n __fish_1ctl_no_subcommand -a environment -d 'Manage environments'
complete -c 1ctl -f -n __fish_1ctl_no_subcommand -a machine -d 'Manage machines'
complete -c 1ctl -f -n __fish_1ctl_no_subcommand -a completion -d 'Generate shell completion scripts'

# Auth subcommands
complete -c 1ctl -f -n '__fish_1ctl_using_command auth' -a 'login' -d 'Authenticate with Satusky'
complete -c 1ctl -f -n '__fish_1ctl_using_command auth' -a 'logout' -d 'Remove stored authentication'
complete -c 1ctl -f -n '__fish_1ctl_using_command auth' -a 'status' -d 'View authentication status'

# Deploy subcommands and flags
complete -c 1ctl -f -n '__fish_1ctl_using_command deploy' -a 'list' -d 'List deployments'
complete -c 1ctl -f -n '__fish_1ctl_using_command deploy' -a 'get' -d 'Get deployment details'
complete -c 1ctl -f -n '__fish_1ctl_using_command deploy' -a 'status' -d 'Check deployment status'

# Deploy flags
complete -c 1ctl -f -n '__fish_1ctl_using_subcommand deploy' -l cpu -d 'CPU cores allocation'
complete -c 1ctl -f -n '__fish_1ctl_using_subcommand deploy' -l memory -d 'Memory allocation'
complete -c 1ctl -f -n '__fish_1ctl_using_subcommand deploy' -l machine -d 'Machine name to deploy to'
complete -c 1ctl -f -n '__fish_1ctl_using_subcommand deploy' -l domain -d 'Custom domain'
complete -c 1ctl -f -n '__fish_1ctl_using_subcommand deploy' -l organization -d 'Organization name'
complete -c 1ctl -f -n '__fish_1ctl_using_subcommand deploy' -l dockerfile -d 'Path to Dockerfile'
complete -c 1ctl -f -n '__fish_1ctl_using_subcommand deploy' -l port -d 'Application port'
complete -c 1ctl -f -n '__fish_1ctl_using_subcommand deploy' -l env -d 'Environment variables'
complete -c 1ctl -f -n '__fish_1ctl_using_subcommand deploy' -l volume-size -d 'Storage size'
complete -c 1ctl -f -n '__fish_1ctl_using_subcommand deploy' -l volume-mount -d 'Storage mount path'

# Deploy list flags
complete -c 1ctl -f -n '__fish_1ctl_using_subcommand deploy list' -l namespace -d 'Filter by namespace'
complete -c 1ctl -f -n '__fish_1ctl_using_subcommand deploy list' -l quiet -d 'Only show names'

# Deploy get/delete flags
complete -c 1ctl -f -n '__fish_1ctl_using_subcommand deploy get' -l deployment-id -d 'Deployment ID to get details for'
complete -c 1ctl -f -n '__fish_1ctl_using_subcommand deploy delete' -l deployment-id -d 'Deployment ID to delete'

# Deploy status flags
complete -c 1ctl -f -n '__fish_1ctl_using_subcommand deploy status' -l deployment-id -d 'Deployment ID to check'
complete -c 1ctl -f -n '__fish_1ctl_using_subcommand deploy status' -l watch -d 'Watch deployment status in real-time'

# Machine subcommands and flags
complete -c 1ctl -f -n '__fish_1ctl_using_command machine' -a 'list' -d 'List machines'
complete -c 1ctl -f -n '__fish_1ctl_using_command machine' -a 'get' -d 'Get machine details'

complete -c 1ctl -f -n '__fish_1ctl_using_subcommand machine get' -l machine-id -d 'Machine ID to get details for'
complete -c 1ctl -f -n '__fish_1ctl_using_subcommand machine get' -l name -d 'Machine name to get details for'
complete -c 1ctl -f -n '__fish_1ctl_using_subcommand machine list' -l quiet -d 'Only show names'

# Completion subcommands
complete -c 1ctl -f -n '__fish_1ctl_using_command completion' -a 'bash' -d 'Generate bash completion script'
complete -c 1ctl -f -n '__fish_1ctl_using_command completion' -a 'zsh' -d 'Generate zsh completion script'
complete -c 1ctl -f -n '__fish_1ctl_using_command completion' -a 'fish' -d 'Generate fish completion script'
complete -c 1ctl -f -n '__fish_1ctl_using_command completion' -a 'powershell' -d 'Generate PowerShell completion script'`

	utils.PrintInfo("%s", script)
	utils.PrintInfo("\n# Add this to ~/.config/fish/completions/1ctl.fish")
	return nil
}

func handlePowerShellCompletion(c *cli.Context) error {
	script := `using namespace System.Management.Automation
using namespace System.Management.Automation.Language

Register-ArgumentCompleter -Native -CommandName 1ctl -ScriptBlock {
    param($wordToComplete, $commandAst, $cursorPosition)
    
    $commandElements = $commandAst.CommandElements
    $command = @(
        '1ctl'
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

    $completions = @(switch ($command) {
        '1ctl' {
            [CompletionResult]::new('auth', 'auth', [CompletionResultType]::ParameterValue, 'Display commands for authentication')
            [CompletionResult]::new('deploy', 'deploy', [CompletionResultType]::ParameterValue, 'Manage deployments')
            [CompletionResult]::new('service', 'service', [CompletionResultType]::ParameterValue, 'Manage services')
            [CompletionResult]::new('secret', 'secret', [CompletionResultType]::ParameterValue, 'Manage secrets')
            [CompletionResult]::new('ingress', 'ingress', [CompletionResultType]::ParameterValue, 'Manage ingresses')
            [CompletionResult]::new('issuer', 'issuer', [CompletionResultType]::ParameterValue, 'Manage certificate issuers')
            [CompletionResult]::new('environment', 'environment', [CompletionResultType]::ParameterValue, 'Manage environments')
            [CompletionResult]::new('machine', 'machine', [CompletionResultType]::ParameterValue, 'Manage machines')
            [CompletionResult]::new('completion', 'completion', [CompletionResultType]::ParameterValue, 'Generate shell completion scripts')
            break
        }
        '1ctl;auth' {
            [CompletionResult]::new('login', 'login', [CompletionResultType]::ParameterValue, 'Authenticate with Satusky')
            [CompletionResult]::new('logout', 'logout', [CompletionResultType]::ParameterValue, 'Remove stored authentication')
            [CompletionResult]::new('status', 'status', [CompletionResultType]::ParameterValue, 'View authentication status')
            break
        }
        '1ctl;deploy' {
            [CompletionResult]::new('create', 'create', [CompletionResultType]::ParameterValue, 'Create a new deployment')
            [CompletionResult]::new('list', 'list', [CompletionResultType]::ParameterValue, 'List deployments')
            [CompletionResult]::new('delete', 'delete', [CompletionResultType]::ParameterValue, 'Delete a deployment')
            [CompletionResult]::new('get', 'get', [CompletionResultType]::ParameterValue, 'Get deployment details')
            [CompletionResult]::new('status', 'status', [CompletionResultType]::ParameterValue, 'Check deployment status')
            break
        }
        '1ctl;service' {
            [CompletionResult]::new('list', 'list', [CompletionResultType]::ParameterValue, 'List services')
            [CompletionResult]::new('delete', 'delete', [CompletionResultType]::ParameterValue, 'Delete a service')
            break
        }
        '1ctl;secret' {
            [CompletionResult]::new('create', 'create', [CompletionResultType]::ParameterValue, 'Create a new secret')
            [CompletionResult]::new('list', 'list', [CompletionResultType]::ParameterValue, 'List secrets')
            [CompletionResult]::new('delete', 'delete', [CompletionResultType]::ParameterValue, 'Delete a secret')
            break
        }
        '1ctl;ingress' {
            [CompletionResult]::new('list', 'list', [CompletionResultType]::ParameterValue, 'List ingresses')
            [CompletionResult]::new('delete', 'delete', [CompletionResultType]::ParameterValue, 'Delete an ingress')
            break
        }
        '1ctl;issuer' {
            [CompletionResult]::new('create', 'create', [CompletionResultType]::ParameterValue, 'Create a new issuer')
            [CompletionResult]::new('list', 'list', [CompletionResultType]::ParameterValue, 'List issuers')
            [CompletionResult]::new('delete', 'delete', [CompletionResultType]::ParameterValue, 'Delete an issuer')
            break
        }
        '1ctl;environment' {
            [CompletionResult]::new('create', 'create', [CompletionResultType]::ParameterValue, 'Create a new environment')
            [CompletionResult]::new('list', 'list', [CompletionResultType]::ParameterValue, 'List environments')
            [CompletionResult]::new('delete', 'delete', [CompletionResultType]::ParameterValue, 'Delete an environment')
            break
        }
        '1ctl;machine' {
            [CompletionResult]::new('list', 'list', [CompletionResultType]::ParameterValue, 'List machines')
            [CompletionResult]::new('get', 'get', [CompletionResultType]::ParameterValue, 'Get machine details')
            break
        }
        '1ctl;completion' {
            [CompletionResult]::new('bash', 'bash', [CompletionResultType]::ParameterValue, 'Generate bash completion script')
            [CompletionResult]::new('zsh', 'zsh', [CompletionResultType]::ParameterValue, 'Generate zsh completion script')
            [CompletionResult]::new('fish', 'fish', [CompletionResultType]::ParameterValue, 'Generate fish completion script')
            [CompletionResult]::new('powershell', 'powershell', [CompletionResultType]::ParameterValue, 'Generate PowerShell completion script')
            break
        }
        '1ctl;deploy;create' {
            [CompletionResult]::new('--cpu', 'cpu', [CompletionResultType]::ParameterName, 'CPU cores allocation')
            [CompletionResult]::new('--memory', 'memory', [CompletionResultType]::ParameterName, 'Memory allocation')
            [CompletionResult]::new('--machine', 'machine', [CompletionResultType]::ParameterName, 'Machine name to deploy to')
            [CompletionResult]::new('--domain', 'domain', [CompletionResultType]::ParameterName, 'Custom domain')
            [CompletionResult]::new('--organization', 'organization', [CompletionResultType]::ParameterName, 'Organization name')
            [CompletionResult]::new('--dockerfile', 'dockerfile', [CompletionResultType]::ParameterName, 'Path to Dockerfile')
            [CompletionResult]::new('--port', 'port', [CompletionResultType]::ParameterName, 'Application port')
            [CompletionResult]::new('--env', 'env', [CompletionResultType]::ParameterName, 'Environment variables')
            [CompletionResult]::new('--volume-size', 'volume-size', [CompletionResultType]::ParameterName, 'Storage size')
            [CompletionResult]::new('--volume-mount', 'volume-mount', [CompletionResultType]::ParameterName, 'Storage mount path')
            break
        }
    })

    $completions.Where{ $_.CompletionText -like "$wordToComplete*" } |
        Sort-Object -Property ListItemText
}`

	utils.PrintInfo("%s", script)
	utils.PrintInfo("\n# Add this to your PowerShell profile:")
	utils.PrintInfo("# . ./1ctl-completion.ps1")
	return nil
}
