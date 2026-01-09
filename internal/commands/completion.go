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
    local cur prev opts cmd subcmd
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"
    cmd="${COMP_WORDS[1]}"
    subcmd="${COMP_WORDS[2]}"

    # Top level commands
    if [[ $COMP_CWORD == 1 ]]; then
        opts="auth deploy service secret ingress issuer environment machine org github notifications user token marketplace audit talos admin credits storage logs completion --help --version"
        COMPREPLY=( $(compgen -W "${opts}" -- ${cur}) )
        return 0
    fi

    # Handle flag values
    case "${prev}" in
        --machine|--hostname)
            local machines=$(1ctl machine list --quiet 2>/dev/null)
            COMPREPLY=( $(compgen -W "${machines}" -- ${cur}) )
            return 0
            ;;
        --cpu)
            COMPREPLY=( $(compgen -W "0.5 1 2 4 8 16" -- ${cur}) )
            return 0
            ;;
        --memory)
            COMPREPLY=( $(compgen -W "512Mi 1Gi 2Gi 4Gi 8Gi 16Gi 32Gi" -- ${cur}) )
            return 0
            ;;
        --volume-size|--storage-size)
            COMPREPLY=( $(compgen -W "1Gi 5Gi 10Gi 20Gi 50Gi 100Gi" -- ${cur}) )
            return 0
            ;;
        --port)
            COMPREPLY=( $(compgen -W "80 443 3000 8080 8443" -- ${cur}) )
            return 0
            ;;
        --role)
            COMPREPLY=( $(compgen -W "admin member owner" -- ${cur}) )
            return 0
            ;;
        --format)
            COMPREPLY=( $(compgen -W "json csv yaml" -- ${cur}) )
            return 0
            ;;
        --multicluster-mode)
            COMPREPLY=( $(compgen -W "active-active active-passive" -- ${cur}) )
            return 0
            ;;
    esac

    # Handle subcommands and their flags
    case "${cmd}" in
        auth)
            COMPREPLY=( $(compgen -W "login logout status" -- ${cur}) )
            ;;
        deploy)
            if [[ ${COMP_CWORD} == 2 ]]; then
                COMPREPLY=( $(compgen -W "list get status delete" -- ${cur}) )
            else
                case "${subcmd}" in
                    list)
                        COMPREPLY=( $(compgen -W "--namespace --quiet --live" -- ${cur}) )
                        ;;
                    get|status|delete)
                        COMPREPLY=( $(compgen -W "--deployment-id --watch --live" -- ${cur}) )
                        ;;
                esac
            fi
            ;;
        service)
            if [[ ${COMP_CWORD} == 2 ]]; then
                COMPREPLY=( $(compgen -W "list delete get" -- ${cur}) )
            fi
            ;;
        secret)
            if [[ ${COMP_CWORD} == 2 ]]; then
                COMPREPLY=( $(compgen -W "create list delete" -- ${cur}) )
            fi
            ;;
        ingress)
            if [[ ${COMP_CWORD} == 2 ]]; then
                COMPREPLY=( $(compgen -W "list delete get" -- ${cur}) )
            fi
            ;;
        issuer)
            if [[ ${COMP_CWORD} == 2 ]]; then
                COMPREPLY=( $(compgen -W "create list delete" -- ${cur}) )
            fi
            ;;
        environment)
            if [[ ${COMP_CWORD} == 2 ]]; then
                COMPREPLY=( $(compgen -W "create list delete get" -- ${cur}) )
            fi
            ;;
        machine)
            if [[ ${COMP_CWORD} == 2 ]]; then
                COMPREPLY=( $(compgen -W "list available get recommended hardware labels iso siderolink talos" -- ${cur}) )
            else
                case "${subcmd}" in
                    list|available)
                        COMPREPLY=( $(compgen -W "--quiet --region --zone --min-cpu --min-memory --gpu --recommended --pricing-tier" -- ${cur}) )
                        ;;
                    get)
                        COMPREPLY=( $(compgen -W "--machine-id --name" -- ${cur}) )
                        ;;
                    hardware)
                        COMPREPLY=( $(compgen -W "refresh" -- ${cur}) )
                        ;;
                    labels)
                        COMPREPLY=( $(compgen -W "set" -- ${cur}) )
                        ;;
                    iso)
                        COMPREPLY=( $(compgen -W "generate download" -- ${cur}) )
                        ;;
                    talos)
                        COMPREPLY=( $(compgen -W "status metadata" -- ${cur}) )
                        ;;
                esac
            fi
            ;;
        org)
            if [[ ${COMP_CWORD} == 2 ]]; then
                COMPREPLY=( $(compgen -W "list current switch create delete team" -- ${cur}) )
            else
                case "${subcmd}" in
                    switch)
                        COMPREPLY=( $(compgen -W "--org-id --org-name" -- ${cur}) )
                        ;;
                    create)
                        COMPREPLY=( $(compgen -W "--name --description" -- ${cur}) )
                        ;;
                    team)
                        if [[ ${COMP_CWORD} == 3 ]]; then
                            COMPREPLY=( $(compgen -W "list add role remove" -- ${cur}) )
                        else
                            case "${COMP_WORDS[3]}" in
                                add)
                                    COMPREPLY=( $(compgen -W "--email --role" -- ${cur}) )
                                    ;;
                                role)
                                    COMPREPLY=( $(compgen -W "--role" -- ${cur}) )
                                    ;;
                            esac
                        fi
                        ;;
                esac
            fi
            ;;
        github)
            if [[ ${COMP_CWORD} == 2 ]]; then
                COMPREPLY=( $(compgen -W "status connect disconnect repos deploy installation" -- ${cur}) )
            else
                case "${subcmd}" in
                    repos)
                        if [[ ${COMP_CWORD} == 3 ]]; then
                            COMPREPLY=( $(compgen -W "sync get --page --limit" -- ${cur}) )
                        fi
                        ;;
                    deploy)
                        COMPREPLY=( $(compgen -W "--repo --namespace --cpu --memory" -- ${cur}) )
                        ;;
                    installation)
                        if [[ ${COMP_CWORD} == 3 ]]; then
                            COMPREPLY=( $(compgen -W "info revoke" -- ${cur}) )
                        fi
                        ;;
                esac
            fi
            ;;
        notifications)
            if [[ ${COMP_CWORD} == 2 ]]; then
                COMPREPLY=( $(compgen -W "list count read delete watch" -- ${cur}) )
            else
                case "${subcmd}" in
                    list)
                        COMPREPLY=( $(compgen -W "--unread --limit" -- ${cur}) )
                        ;;
                    read)
                        COMPREPLY=( $(compgen -W "--all" -- ${cur}) )
                        ;;
                esac
            fi
            ;;
        user)
            if [[ ${COMP_CWORD} == 2 ]]; then
                COMPREPLY=( $(compgen -W "me update password permissions sessions" -- ${cur}) )
            else
                case "${subcmd}" in
                    update)
                        COMPREPLY=( $(compgen -W "--name --email" -- ${cur}) )
                        ;;
                    sessions)
                        if [[ ${COMP_CWORD} == 3 ]]; then
                            COMPREPLY=( $(compgen -W "revoke" -- ${cur}) )
                        fi
                        ;;
                esac
            fi
            ;;
        token)
            if [[ ${COMP_CWORD} == 2 ]]; then
                COMPREPLY=( $(compgen -W "list create get enable disable delete" -- ${cur}) )
            else
                case "${subcmd}" in
                    create)
                        COMPREPLY=( $(compgen -W "--name --expires" -- ${cur}) )
                        ;;
                esac
            fi
            ;;
        marketplace)
            if [[ ${COMP_CWORD} == 2 ]]; then
                COMPREPLY=( $(compgen -W "list get deploy" -- ${cur}) )
            else
                case "${subcmd}" in
                    list)
                        COMPREPLY=( $(compgen -W "--limit --offset --sort" -- ${cur}) )
                        ;;
                    deploy)
                        COMPREPLY=( $(compgen -W "--name --hostname --cpu --memory --domain --storage-size --storage-class --multicluster --multicluster-mode" -- ${cur}) )
                        ;;
                esac
            fi
            ;;
        audit)
            if [[ ${COMP_CWORD} == 2 ]]; then
                COMPREPLY=( $(compgen -W "list get export" -- ${cur}) )
            else
                case "${subcmd}" in
                    list)
                        COMPREPLY=( $(compgen -W "--limit --action --user" -- ${cur}) )
                        ;;
                    export)
                        COMPREPLY=( $(compgen -W "--format --output" -- ${cur}) )
                        ;;
                esac
            fi
            ;;
        talos)
            if [[ ${COMP_CWORD} == 2 ]]; then
                COMPREPLY=( $(compgen -W "generate apply history network" -- ${cur}) )
            else
                case "${subcmd}" in
                    generate)
                        COMPREPLY=( $(compgen -W "--machine-id --cluster-name --role --output" -- ${cur}) )
                        ;;
                    apply)
                        COMPREPLY=( $(compgen -W "--machine-id --config-file" -- ${cur}) )
                        ;;
                esac
            fi
            ;;
        admin)
            if [[ ${COMP_CWORD} == 2 ]]; then
                COMPREPLY=( $(compgen -W "usage credits namespaces cluster-roles cleanup" -- ${cur}) )
            else
                case "${subcmd}" in
                    usage)
                        if [[ ${COMP_CWORD} == 3 ]]; then
                            COMPREPLY=( $(compgen -W "unbilled machine bill" -- ${cur}) )
                        fi
                        ;;
                    credits)
                        if [[ ${COMP_CWORD} == 3 ]]; then
                            COMPREPLY=( $(compgen -W "add refund" -- ${cur}) )
                        else
                            COMPREPLY=( $(compgen -W "--amount --description" -- ${cur}) )
                        fi
                        ;;
                    cleanup)
                        COMPREPLY=( $(compgen -W "--label --namespace" -- ${cur}) )
                        ;;
                esac
            fi
            ;;
        credits)
            if [[ ${COMP_CWORD} == 2 ]]; then
                COMPREPLY=( $(compgen -W "balance transactions usage topup invoices" -- ${cur}) )
            else
                case "${subcmd}" in
                    transactions|usage)
                        COMPREPLY=( $(compgen -W "--limit --offset --days" -- ${cur}) )
                        ;;
                    topup)
                        COMPREPLY=( $(compgen -W "--amount" -- ${cur}) )
                        ;;
                    invoices)
                        if [[ ${COMP_CWORD} == 3 ]]; then
                            COMPREPLY=( $(compgen -W "get download generate" -- ${cur}) )
                        else
                            COMPREPLY=( $(compgen -W "--output --start-date --end-date" -- ${cur}) )
                        fi
                        ;;
                esac
            fi
            ;;
        storage)
            if [[ ${COMP_CWORD} == 2 ]]; then
                COMPREPLY=( $(compgen -W "list get create delete buckets files upload download presign usage" -- ${cur}) )
            else
                case "${subcmd}" in
                    create)
                        COMPREPLY=( $(compgen -W "--name --type --size" -- ${cur}) )
                        ;;
                    buckets)
                        if [[ ${COMP_CWORD} == 3 ]]; then
                            COMPREPLY=( $(compgen -W "create delete" -- ${cur}) )
                        else
                            COMPREPLY=( $(compgen -W "--name" -- ${cur}) )
                        fi
                        ;;
                    download)
                        COMPREPLY=( $(compgen -W "--output" -- ${cur}) )
                        ;;
                    presign)
                        COMPREPLY=( $(compgen -W "--file --expires" -- ${cur}) )
                        ;;
                esac
            fi
            ;;
        logs)
            COMPREPLY=( $(compgen -W "--deployment-id -d --follow -f --stats --tail" -- ${cur}) )
            ;;
        completion)
            if [[ ${COMP_CWORD} == 2 ]]; then
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
                'auth:Authentication commands'
                'deploy:Manage deployments'
                'service:Manage services'
                'secret:Manage secrets'
                'ingress:Manage ingresses'
                'issuer:Manage certificate issuers'
                'environment:Manage environments'
                'machine:Manage machines'
                'org:Manage organizations'
                'github:GitHub integration'
                'notifications:Manage notifications'
                'user:Manage user profile'
                'token:Manage API tokens'
                'marketplace:Browse and deploy marketplace apps'
                'audit:View audit logs'
                'talos:Talos Linux configuration'
                'admin:Admin operations'
                'credits:Manage credits and billing'
                'storage:Manage S3/object storage'
                'logs:View and stream pod logs'
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
                            'delete:Delete a deployment'
                        )
                        _describe -t subcommands 'deploy subcommands' subcommands
                    fi
                    ;;
                machine)
                    if (( CURRENT == 2 )); then
                        local -a subcommands=(
                            'list:List owned machines'
                            'available:List available machines for rent'
                            'get:Get machine details'
                            'recommended:List recommended machines'
                            'hardware:View machine hardware info'
                            'labels:Manage machine labels'
                            'iso:Generate/download ISO'
                            'siderolink:View siderolink connections'
                            'talos:View Talos status'
                        )
                        _describe -t subcommands 'machine subcommands' subcommands
                    fi
                    ;;
                org)
                    if (( CURRENT == 2 )); then
                        local -a subcommands=(
                            'list:List organizations'
                            'current:Show current organization'
                            'switch:Switch organization'
                            'create:Create organization'
                            'delete:Delete organization'
                            'team:Manage team members'
                        )
                        _describe -t subcommands 'org subcommands' subcommands
                    elif [[ $words[2] == "team" ]] && (( CURRENT == 3 )); then
                        local -a teamcmds=(
                            'list:List team members'
                            'add:Add team member'
                            'role:Update member role'
                            'remove:Remove team member'
                        )
                        _describe -t subcommands 'team subcommands' teamcmds
                    fi
                    ;;
                github)
                    if (( CURRENT == 2 )); then
                        local -a subcommands=(
                            'status:Check GitHub connection'
                            'connect:Connect to GitHub'
                            'disconnect:Disconnect from GitHub'
                            'repos:Manage repositories'
                            'deploy:Deploy from GitHub'
                            'installation:Manage GitHub installation'
                        )
                        _describe -t subcommands 'github subcommands' subcommands
                    fi
                    ;;
                notifications)
                    if (( CURRENT == 2 )); then
                        local -a subcommands=(
                            'list:List notifications'
                            'count:Get unread count'
                            'read:Mark as read'
                            'delete:Delete notification'
                            'watch:Watch notifications'
                        )
                        _describe -t subcommands 'notifications subcommands' subcommands
                    fi
                    ;;
                user)
                    if (( CURRENT == 2 )); then
                        local -a subcommands=(
                            'me:View current user'
                            'update:Update profile'
                            'password:Change password'
                            'permissions:View permissions'
                            'sessions:Manage sessions'
                        )
                        _describe -t subcommands 'user subcommands' subcommands
                    fi
                    ;;
                token)
                    if (( CURRENT == 2 )); then
                        local -a subcommands=(
                            'list:List API tokens'
                            'create:Create token'
                            'get:Get token details'
                            'enable:Enable token'
                            'disable:Disable token'
                            'delete:Delete token'
                        )
                        _describe -t subcommands 'token subcommands' subcommands
                    fi
                    ;;
                marketplace)
                    if (( CURRENT == 2 )); then
                        local -a subcommands=(
                            'list:List marketplace apps'
                            'get:Get app details'
                            'deploy:Deploy marketplace app'
                        )
                        _describe -t subcommands 'marketplace subcommands' subcommands
                    fi
                    ;;
                audit)
                    if (( CURRENT == 2 )); then
                        local -a subcommands=(
                            'list:List audit logs'
                            'get:Get log details'
                            'export:Export audit logs'
                        )
                        _describe -t subcommands 'audit subcommands' subcommands
                    fi
                    ;;
                talos)
                    if (( CURRENT == 2 )); then
                        local -a subcommands=(
                            'generate:Generate Talos config'
                            'apply:Apply Talos config'
                            'history:View config history'
                            'network:View network info'
                        )
                        _describe -t subcommands 'talos subcommands' subcommands
                    fi
                    ;;
                admin)
                    if (( CURRENT == 2 )); then
                        local -a subcommands=(
                            'usage:Manage machine usage'
                            'credits:Manage credits'
                            'namespaces:List namespaces'
                            'cluster-roles:List cluster roles'
                            'cleanup:Cleanup resources'
                        )
                        _describe -t subcommands 'admin subcommands' subcommands
                    fi
                    ;;
                credits)
                    if (( CURRENT == 2 )); then
                        local -a subcommands=(
                            'balance:View credit balance'
                            'transactions:View transactions'
                            'usage:View usage history'
                            'topup:Top up credits'
                            'invoices:Manage invoices'
                        )
                        _describe -t subcommands 'credits subcommands' subcommands
                    fi
                    ;;
                storage)
                    if (( CURRENT == 2 )); then
                        local -a subcommands=(
                            'list:List storage configs'
                            'get:Get storage details'
                            'create:Create storage'
                            'delete:Delete storage'
                            'buckets:Manage buckets'
                            'files:List files'
                            'upload:Upload file'
                            'download:Download file'
                            'presign:Get presigned URL'
                            'usage:View storage usage'
                        )
                        _describe -t subcommands 'storage subcommands' subcommands
                    fi
                    ;;
                logs)
                    _arguments \
                        '(-d --deployment-id)'{-d,--deployment-id}'[Deployment ID]:id:' \
                        '(-f --follow)'{-f,--follow}'[Stream logs]' \
                        '--stats[Show log statistics]' \
                        '--tail[Number of lines]:lines:'
                    ;;
                completion)
                    if (( CURRENT == 2 )); then
                        local -a subcommands=(
                            'bash:Generate bash completion'
                            'zsh:Generate zsh completion'
                            'fish:Generate fish completion'
                            'powershell:Generate PowerShell completion'
                        )
                        _describe -t subcommands 'completion subcommands' subcommands
                    fi
                    ;;
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
	script := `# Fish completion for 1ctl

function __fish_1ctl_no_subcommand
    for i in (commandline -opc)
        if contains -- $i auth deploy service secret ingress issuer environment machine org github notifications user token marketplace audit talos admin credits storage logs completion
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
complete -c 1ctl -l cpu -xa '0.5 1 2 4 8 16'
complete -c 1ctl -l memory -xa '512Mi 1Gi 2Gi 4Gi 8Gi 16Gi 32Gi'
complete -c 1ctl -l machine -xa '(__fish_1ctl_machines)'
complete -c 1ctl -l hostname -xa '(__fish_1ctl_machines)'
complete -c 1ctl -l port -xa '80 443 3000 8080 8443'
complete -c 1ctl -l volume-size -xa '1Gi 5Gi 10Gi 20Gi 50Gi 100Gi'
complete -c 1ctl -l storage-size -xa '1Gi 5Gi 10Gi 20Gi 50Gi 100Gi'
complete -c 1ctl -l role -xa 'admin member owner'
complete -c 1ctl -l format -xa 'json csv yaml'
complete -c 1ctl -l multicluster-mode -xa 'active-active active-passive'

# Top level commands
complete -c 1ctl -f -n __fish_1ctl_no_subcommand -a auth -d 'Authentication commands'
complete -c 1ctl -f -n __fish_1ctl_no_subcommand -a deploy -d 'Manage deployments'
complete -c 1ctl -f -n __fish_1ctl_no_subcommand -a service -d 'Manage services'
complete -c 1ctl -f -n __fish_1ctl_no_subcommand -a secret -d 'Manage secrets'
complete -c 1ctl -f -n __fish_1ctl_no_subcommand -a ingress -d 'Manage ingresses'
complete -c 1ctl -f -n __fish_1ctl_no_subcommand -a issuer -d 'Manage certificate issuers'
complete -c 1ctl -f -n __fish_1ctl_no_subcommand -a environment -d 'Manage environments'
complete -c 1ctl -f -n __fish_1ctl_no_subcommand -a machine -d 'Manage machines'
complete -c 1ctl -f -n __fish_1ctl_no_subcommand -a org -d 'Manage organizations'
complete -c 1ctl -f -n __fish_1ctl_no_subcommand -a github -d 'GitHub integration'
complete -c 1ctl -f -n __fish_1ctl_no_subcommand -a notifications -d 'Manage notifications'
complete -c 1ctl -f -n __fish_1ctl_no_subcommand -a user -d 'Manage user profile'
complete -c 1ctl -f -n __fish_1ctl_no_subcommand -a token -d 'Manage API tokens'
complete -c 1ctl -f -n __fish_1ctl_no_subcommand -a marketplace -d 'Browse and deploy marketplace apps'
complete -c 1ctl -f -n __fish_1ctl_no_subcommand -a audit -d 'View audit logs'
complete -c 1ctl -f -n __fish_1ctl_no_subcommand -a talos -d 'Talos Linux configuration'
complete -c 1ctl -f -n __fish_1ctl_no_subcommand -a admin -d 'Admin operations'
complete -c 1ctl -f -n __fish_1ctl_no_subcommand -a credits -d 'Manage credits and billing'
complete -c 1ctl -f -n __fish_1ctl_no_subcommand -a storage -d 'Manage S3/object storage'
complete -c 1ctl -f -n __fish_1ctl_no_subcommand -a logs -d 'View and stream pod logs'
complete -c 1ctl -f -n __fish_1ctl_no_subcommand -a completion -d 'Generate shell completion scripts'

# Auth subcommands
complete -c 1ctl -f -n '__fish_1ctl_using_command auth' -a 'login' -d 'Authenticate with Satusky'
complete -c 1ctl -f -n '__fish_1ctl_using_command auth' -a 'logout' -d 'Remove stored authentication'
complete -c 1ctl -f -n '__fish_1ctl_using_command auth' -a 'status' -d 'View authentication status'

# Deploy subcommands
complete -c 1ctl -f -n '__fish_1ctl_using_command deploy' -a 'list' -d 'List deployments'
complete -c 1ctl -f -n '__fish_1ctl_using_command deploy' -a 'get' -d 'Get deployment details'
complete -c 1ctl -f -n '__fish_1ctl_using_command deploy' -a 'status' -d 'Check deployment status'
complete -c 1ctl -f -n '__fish_1ctl_using_command deploy' -a 'delete' -d 'Delete a deployment'

# Machine subcommands
complete -c 1ctl -f -n '__fish_1ctl_using_command machine' -a 'list' -d 'List owned machines'
complete -c 1ctl -f -n '__fish_1ctl_using_command machine' -a 'available' -d 'List available machines'
complete -c 1ctl -f -n '__fish_1ctl_using_command machine' -a 'get' -d 'Get machine details'
complete -c 1ctl -f -n '__fish_1ctl_using_command machine' -a 'recommended' -d 'List recommended machines'
complete -c 1ctl -f -n '__fish_1ctl_using_command machine' -a 'hardware' -d 'View machine hardware'
complete -c 1ctl -f -n '__fish_1ctl_using_command machine' -a 'labels' -d 'Manage machine labels'
complete -c 1ctl -f -n '__fish_1ctl_using_command machine' -a 'iso' -d 'Generate/download ISO'
complete -c 1ctl -f -n '__fish_1ctl_using_command machine' -a 'siderolink' -d 'View siderolink connections'
complete -c 1ctl -f -n '__fish_1ctl_using_command machine' -a 'talos' -d 'View Talos status'

# Org subcommands
complete -c 1ctl -f -n '__fish_1ctl_using_command org' -a 'list' -d 'List organizations'
complete -c 1ctl -f -n '__fish_1ctl_using_command org' -a 'current' -d 'Show current organization'
complete -c 1ctl -f -n '__fish_1ctl_using_command org' -a 'switch' -d 'Switch organization'
complete -c 1ctl -f -n '__fish_1ctl_using_command org' -a 'create' -d 'Create organization'
complete -c 1ctl -f -n '__fish_1ctl_using_command org' -a 'delete' -d 'Delete organization'
complete -c 1ctl -f -n '__fish_1ctl_using_command org' -a 'team' -d 'Manage team members'

# GitHub subcommands
complete -c 1ctl -f -n '__fish_1ctl_using_command github' -a 'status' -d 'Check GitHub connection'
complete -c 1ctl -f -n '__fish_1ctl_using_command github' -a 'connect' -d 'Connect to GitHub'
complete -c 1ctl -f -n '__fish_1ctl_using_command github' -a 'disconnect' -d 'Disconnect from GitHub'
complete -c 1ctl -f -n '__fish_1ctl_using_command github' -a 'repos' -d 'Manage repositories'
complete -c 1ctl -f -n '__fish_1ctl_using_command github' -a 'deploy' -d 'Deploy from GitHub'
complete -c 1ctl -f -n '__fish_1ctl_using_command github' -a 'installation' -d 'Manage installation'

# Notifications subcommands
complete -c 1ctl -f -n '__fish_1ctl_using_command notifications' -a 'list' -d 'List notifications'
complete -c 1ctl -f -n '__fish_1ctl_using_command notifications' -a 'count' -d 'Get unread count'
complete -c 1ctl -f -n '__fish_1ctl_using_command notifications' -a 'read' -d 'Mark as read'
complete -c 1ctl -f -n '__fish_1ctl_using_command notifications' -a 'delete' -d 'Delete notification'
complete -c 1ctl -f -n '__fish_1ctl_using_command notifications' -a 'watch' -d 'Watch notifications'

# User subcommands
complete -c 1ctl -f -n '__fish_1ctl_using_command user' -a 'me' -d 'View current user'
complete -c 1ctl -f -n '__fish_1ctl_using_command user' -a 'update' -d 'Update profile'
complete -c 1ctl -f -n '__fish_1ctl_using_command user' -a 'password' -d 'Change password'
complete -c 1ctl -f -n '__fish_1ctl_using_command user' -a 'permissions' -d 'View permissions'
complete -c 1ctl -f -n '__fish_1ctl_using_command user' -a 'sessions' -d 'Manage sessions'

# Token subcommands
complete -c 1ctl -f -n '__fish_1ctl_using_command token' -a 'list' -d 'List API tokens'
complete -c 1ctl -f -n '__fish_1ctl_using_command token' -a 'create' -d 'Create token'
complete -c 1ctl -f -n '__fish_1ctl_using_command token' -a 'get' -d 'Get token details'
complete -c 1ctl -f -n '__fish_1ctl_using_command token' -a 'enable' -d 'Enable token'
complete -c 1ctl -f -n '__fish_1ctl_using_command token' -a 'disable' -d 'Disable token'
complete -c 1ctl -f -n '__fish_1ctl_using_command token' -a 'delete' -d 'Delete token'

# Marketplace subcommands
complete -c 1ctl -f -n '__fish_1ctl_using_command marketplace' -a 'list' -d 'List marketplace apps'
complete -c 1ctl -f -n '__fish_1ctl_using_command marketplace' -a 'get' -d 'Get app details'
complete -c 1ctl -f -n '__fish_1ctl_using_command marketplace' -a 'deploy' -d 'Deploy marketplace app'

# Audit subcommands
complete -c 1ctl -f -n '__fish_1ctl_using_command audit' -a 'list' -d 'List audit logs'
complete -c 1ctl -f -n '__fish_1ctl_using_command audit' -a 'get' -d 'Get log details'
complete -c 1ctl -f -n '__fish_1ctl_using_command audit' -a 'export' -d 'Export audit logs'

# Talos subcommands
complete -c 1ctl -f -n '__fish_1ctl_using_command talos' -a 'generate' -d 'Generate Talos config'
complete -c 1ctl -f -n '__fish_1ctl_using_command talos' -a 'apply' -d 'Apply Talos config'
complete -c 1ctl -f -n '__fish_1ctl_using_command talos' -a 'history' -d 'View config history'
complete -c 1ctl -f -n '__fish_1ctl_using_command talos' -a 'network' -d 'View network info'

# Admin subcommands
complete -c 1ctl -f -n '__fish_1ctl_using_command admin' -a 'usage' -d 'Manage machine usage'
complete -c 1ctl -f -n '__fish_1ctl_using_command admin' -a 'credits' -d 'Manage credits'
complete -c 1ctl -f -n '__fish_1ctl_using_command admin' -a 'namespaces' -d 'List namespaces'
complete -c 1ctl -f -n '__fish_1ctl_using_command admin' -a 'cluster-roles' -d 'List cluster roles'
complete -c 1ctl -f -n '__fish_1ctl_using_command admin' -a 'cleanup' -d 'Cleanup resources'

# Credits subcommands
complete -c 1ctl -f -n '__fish_1ctl_using_command credits' -a 'balance' -d 'View credit balance'
complete -c 1ctl -f -n '__fish_1ctl_using_command credits' -a 'transactions' -d 'View transactions'
complete -c 1ctl -f -n '__fish_1ctl_using_command credits' -a 'usage' -d 'View usage history'
complete -c 1ctl -f -n '__fish_1ctl_using_command credits' -a 'topup' -d 'Top up credits'
complete -c 1ctl -f -n '__fish_1ctl_using_command credits' -a 'invoices' -d 'Manage invoices'

# Storage subcommands
complete -c 1ctl -f -n '__fish_1ctl_using_command storage' -a 'list' -d 'List storage configs'
complete -c 1ctl -f -n '__fish_1ctl_using_command storage' -a 'get' -d 'Get storage details'
complete -c 1ctl -f -n '__fish_1ctl_using_command storage' -a 'create' -d 'Create storage'
complete -c 1ctl -f -n '__fish_1ctl_using_command storage' -a 'delete' -d 'Delete storage'
complete -c 1ctl -f -n '__fish_1ctl_using_command storage' -a 'buckets' -d 'Manage buckets'
complete -c 1ctl -f -n '__fish_1ctl_using_command storage' -a 'files' -d 'List files'
complete -c 1ctl -f -n '__fish_1ctl_using_command storage' -a 'upload' -d 'Upload file'
complete -c 1ctl -f -n '__fish_1ctl_using_command storage' -a 'download' -d 'Download file'
complete -c 1ctl -f -n '__fish_1ctl_using_command storage' -a 'presign' -d 'Get presigned URL'
complete -c 1ctl -f -n '__fish_1ctl_using_command storage' -a 'usage' -d 'View storage usage'

# Logs flags
complete -c 1ctl -f -n '__fish_1ctl_using_command logs' -s d -l deployment-id -d 'Deployment ID'
complete -c 1ctl -f -n '__fish_1ctl_using_command logs' -s f -l follow -d 'Stream logs'
complete -c 1ctl -f -n '__fish_1ctl_using_command logs' -l stats -d 'Show log statistics'
complete -c 1ctl -f -n '__fish_1ctl_using_command logs' -l tail -d 'Number of lines'

# Completion subcommands
complete -c 1ctl -f -n '__fish_1ctl_using_command completion' -a 'bash' -d 'Generate bash completion'
complete -c 1ctl -f -n '__fish_1ctl_using_command completion' -a 'zsh' -d 'Generate zsh completion'
complete -c 1ctl -f -n '__fish_1ctl_using_command completion' -a 'fish' -d 'Generate fish completion'
complete -c 1ctl -f -n '__fish_1ctl_using_command completion' -a 'powershell' -d 'Generate PowerShell completion'`

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
            [CompletionResult]::new('auth', 'auth', [CompletionResultType]::ParameterValue, 'Authentication commands')
            [CompletionResult]::new('deploy', 'deploy', [CompletionResultType]::ParameterValue, 'Manage deployments')
            [CompletionResult]::new('service', 'service', [CompletionResultType]::ParameterValue, 'Manage services')
            [CompletionResult]::new('secret', 'secret', [CompletionResultType]::ParameterValue, 'Manage secrets')
            [CompletionResult]::new('ingress', 'ingress', [CompletionResultType]::ParameterValue, 'Manage ingresses')
            [CompletionResult]::new('issuer', 'issuer', [CompletionResultType]::ParameterValue, 'Manage certificate issuers')
            [CompletionResult]::new('environment', 'environment', [CompletionResultType]::ParameterValue, 'Manage environments')
            [CompletionResult]::new('machine', 'machine', [CompletionResultType]::ParameterValue, 'Manage machines')
            [CompletionResult]::new('org', 'org', [CompletionResultType]::ParameterValue, 'Manage organizations')
            [CompletionResult]::new('github', 'github', [CompletionResultType]::ParameterValue, 'GitHub integration')
            [CompletionResult]::new('notifications', 'notifications', [CompletionResultType]::ParameterValue, 'Manage notifications')
            [CompletionResult]::new('user', 'user', [CompletionResultType]::ParameterValue, 'Manage user profile')
            [CompletionResult]::new('token', 'token', [CompletionResultType]::ParameterValue, 'Manage API tokens')
            [CompletionResult]::new('marketplace', 'marketplace', [CompletionResultType]::ParameterValue, 'Browse and deploy marketplace apps')
            [CompletionResult]::new('audit', 'audit', [CompletionResultType]::ParameterValue, 'View audit logs')
            [CompletionResult]::new('talos', 'talos', [CompletionResultType]::ParameterValue, 'Talos Linux configuration')
            [CompletionResult]::new('admin', 'admin', [CompletionResultType]::ParameterValue, 'Admin operations')
            [CompletionResult]::new('credits', 'credits', [CompletionResultType]::ParameterValue, 'Manage credits and billing')
            [CompletionResult]::new('storage', 'storage', [CompletionResultType]::ParameterValue, 'Manage S3/object storage')
            [CompletionResult]::new('logs', 'logs', [CompletionResultType]::ParameterValue, 'View and stream pod logs')
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
            [CompletionResult]::new('list', 'list', [CompletionResultType]::ParameterValue, 'List deployments')
            [CompletionResult]::new('get', 'get', [CompletionResultType]::ParameterValue, 'Get deployment details')
            [CompletionResult]::new('status', 'status', [CompletionResultType]::ParameterValue, 'Check deployment status')
            [CompletionResult]::new('delete', 'delete', [CompletionResultType]::ParameterValue, 'Delete a deployment')
            break
        }
        '1ctl;machine' {
            [CompletionResult]::new('list', 'list', [CompletionResultType]::ParameterValue, 'List owned machines')
            [CompletionResult]::new('available', 'available', [CompletionResultType]::ParameterValue, 'List available machines')
            [CompletionResult]::new('get', 'get', [CompletionResultType]::ParameterValue, 'Get machine details')
            [CompletionResult]::new('recommended', 'recommended', [CompletionResultType]::ParameterValue, 'List recommended machines')
            [CompletionResult]::new('hardware', 'hardware', [CompletionResultType]::ParameterValue, 'View machine hardware')
            [CompletionResult]::new('labels', 'labels', [CompletionResultType]::ParameterValue, 'Manage machine labels')
            [CompletionResult]::new('iso', 'iso', [CompletionResultType]::ParameterValue, 'Generate/download ISO')
            [CompletionResult]::new('siderolink', 'siderolink', [CompletionResultType]::ParameterValue, 'View siderolink connections')
            [CompletionResult]::new('talos', 'talos', [CompletionResultType]::ParameterValue, 'View Talos status')
            break
        }
        '1ctl;org' {
            [CompletionResult]::new('list', 'list', [CompletionResultType]::ParameterValue, 'List organizations')
            [CompletionResult]::new('current', 'current', [CompletionResultType]::ParameterValue, 'Show current organization')
            [CompletionResult]::new('switch', 'switch', [CompletionResultType]::ParameterValue, 'Switch organization')
            [CompletionResult]::new('create', 'create', [CompletionResultType]::ParameterValue, 'Create organization')
            [CompletionResult]::new('delete', 'delete', [CompletionResultType]::ParameterValue, 'Delete organization')
            [CompletionResult]::new('team', 'team', [CompletionResultType]::ParameterValue, 'Manage team members')
            break
        }
        '1ctl;org;team' {
            [CompletionResult]::new('list', 'list', [CompletionResultType]::ParameterValue, 'List team members')
            [CompletionResult]::new('add', 'add', [CompletionResultType]::ParameterValue, 'Add team member')
            [CompletionResult]::new('role', 'role', [CompletionResultType]::ParameterValue, 'Update member role')
            [CompletionResult]::new('remove', 'remove', [CompletionResultType]::ParameterValue, 'Remove team member')
            break
        }
        '1ctl;github' {
            [CompletionResult]::new('status', 'status', [CompletionResultType]::ParameterValue, 'Check GitHub connection')
            [CompletionResult]::new('connect', 'connect', [CompletionResultType]::ParameterValue, 'Connect to GitHub')
            [CompletionResult]::new('disconnect', 'disconnect', [CompletionResultType]::ParameterValue, 'Disconnect from GitHub')
            [CompletionResult]::new('repos', 'repos', [CompletionResultType]::ParameterValue, 'Manage repositories')
            [CompletionResult]::new('deploy', 'deploy', [CompletionResultType]::ParameterValue, 'Deploy from GitHub')
            [CompletionResult]::new('installation', 'installation', [CompletionResultType]::ParameterValue, 'Manage installation')
            break
        }
        '1ctl;notifications' {
            [CompletionResult]::new('list', 'list', [CompletionResultType]::ParameterValue, 'List notifications')
            [CompletionResult]::new('count', 'count', [CompletionResultType]::ParameterValue, 'Get unread count')
            [CompletionResult]::new('read', 'read', [CompletionResultType]::ParameterValue, 'Mark as read')
            [CompletionResult]::new('delete', 'delete', [CompletionResultType]::ParameterValue, 'Delete notification')
            [CompletionResult]::new('watch', 'watch', [CompletionResultType]::ParameterValue, 'Watch notifications')
            break
        }
        '1ctl;user' {
            [CompletionResult]::new('me', 'me', [CompletionResultType]::ParameterValue, 'View current user')
            [CompletionResult]::new('update', 'update', [CompletionResultType]::ParameterValue, 'Update profile')
            [CompletionResult]::new('password', 'password', [CompletionResultType]::ParameterValue, 'Change password')
            [CompletionResult]::new('permissions', 'permissions', [CompletionResultType]::ParameterValue, 'View permissions')
            [CompletionResult]::new('sessions', 'sessions', [CompletionResultType]::ParameterValue, 'Manage sessions')
            break
        }
        '1ctl;token' {
            [CompletionResult]::new('list', 'list', [CompletionResultType]::ParameterValue, 'List API tokens')
            [CompletionResult]::new('create', 'create', [CompletionResultType]::ParameterValue, 'Create token')
            [CompletionResult]::new('get', 'get', [CompletionResultType]::ParameterValue, 'Get token details')
            [CompletionResult]::new('enable', 'enable', [CompletionResultType]::ParameterValue, 'Enable token')
            [CompletionResult]::new('disable', 'disable', [CompletionResultType]::ParameterValue, 'Disable token')
            [CompletionResult]::new('delete', 'delete', [CompletionResultType]::ParameterValue, 'Delete token')
            break
        }
        '1ctl;marketplace' {
            [CompletionResult]::new('list', 'list', [CompletionResultType]::ParameterValue, 'List marketplace apps')
            [CompletionResult]::new('get', 'get', [CompletionResultType]::ParameterValue, 'Get app details')
            [CompletionResult]::new('deploy', 'deploy', [CompletionResultType]::ParameterValue, 'Deploy marketplace app')
            break
        }
        '1ctl;audit' {
            [CompletionResult]::new('list', 'list', [CompletionResultType]::ParameterValue, 'List audit logs')
            [CompletionResult]::new('get', 'get', [CompletionResultType]::ParameterValue, 'Get log details')
            [CompletionResult]::new('export', 'export', [CompletionResultType]::ParameterValue, 'Export audit logs')
            break
        }
        '1ctl;talos' {
            [CompletionResult]::new('generate', 'generate', [CompletionResultType]::ParameterValue, 'Generate Talos config')
            [CompletionResult]::new('apply', 'apply', [CompletionResultType]::ParameterValue, 'Apply Talos config')
            [CompletionResult]::new('history', 'history', [CompletionResultType]::ParameterValue, 'View config history')
            [CompletionResult]::new('network', 'network', [CompletionResultType]::ParameterValue, 'View network info')
            break
        }
        '1ctl;admin' {
            [CompletionResult]::new('usage', 'usage', [CompletionResultType]::ParameterValue, 'Manage machine usage')
            [CompletionResult]::new('credits', 'credits', [CompletionResultType]::ParameterValue, 'Manage credits')
            [CompletionResult]::new('namespaces', 'namespaces', [CompletionResultType]::ParameterValue, 'List namespaces')
            [CompletionResult]::new('cluster-roles', 'cluster-roles', [CompletionResultType]::ParameterValue, 'List cluster roles')
            [CompletionResult]::new('cleanup', 'cleanup', [CompletionResultType]::ParameterValue, 'Cleanup resources')
            break
        }
        '1ctl;credits' {
            [CompletionResult]::new('balance', 'balance', [CompletionResultType]::ParameterValue, 'View credit balance')
            [CompletionResult]::new('transactions', 'transactions', [CompletionResultType]::ParameterValue, 'View transactions')
            [CompletionResult]::new('usage', 'usage', [CompletionResultType]::ParameterValue, 'View usage history')
            [CompletionResult]::new('topup', 'topup', [CompletionResultType]::ParameterValue, 'Top up credits')
            [CompletionResult]::new('invoices', 'invoices', [CompletionResultType]::ParameterValue, 'Manage invoices')
            break
        }
        '1ctl;storage' {
            [CompletionResult]::new('list', 'list', [CompletionResultType]::ParameterValue, 'List storage configs')
            [CompletionResult]::new('get', 'get', [CompletionResultType]::ParameterValue, 'Get storage details')
            [CompletionResult]::new('create', 'create', [CompletionResultType]::ParameterValue, 'Create storage')
            [CompletionResult]::new('delete', 'delete', [CompletionResultType]::ParameterValue, 'Delete storage')
            [CompletionResult]::new('buckets', 'buckets', [CompletionResultType]::ParameterValue, 'Manage buckets')
            [CompletionResult]::new('files', 'files', [CompletionResultType]::ParameterValue, 'List files')
            [CompletionResult]::new('upload', 'upload', [CompletionResultType]::ParameterValue, 'Upload file')
            [CompletionResult]::new('download', 'download', [CompletionResultType]::ParameterValue, 'Download file')
            [CompletionResult]::new('presign', 'presign', [CompletionResultType]::ParameterValue, 'Get presigned URL')
            [CompletionResult]::new('usage', 'usage', [CompletionResultType]::ParameterValue, 'View storage usage')
            break
        }
        '1ctl;completion' {
            [CompletionResult]::new('bash', 'bash', [CompletionResultType]::ParameterValue, 'Generate bash completion')
            [CompletionResult]::new('zsh', 'zsh', [CompletionResultType]::ParameterValue, 'Generate zsh completion')
            [CompletionResult]::new('fish', 'fish', [CompletionResultType]::ParameterValue, 'Generate fish completion')
            [CompletionResult]::new('powershell', 'powershell', [CompletionResultType]::ParameterValue, 'Generate PowerShell completion')
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
