package commands

import (
	"1ctl/internal/utils"

	"github.com/urfave/cli/v2"
)

// CompletionCommand returns the completion command group.
//
// Maintenance note: when adding or removing commands/subcommands/flags in 1ctl,
// update ALL four shell templates below (bash, zsh, fish, powershell) to keep
// tab-completion in sync. Each template lists the full command inventory.
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
        opts="auth org deploy service secret ingress issuer environment machine domain credits storage logs notifications user token marketplace audit talos admin pricing cluster completion --help --version"
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
        --zone)
            local zones=$(1ctl cluster zones 2>/dev/null | awk 'NR>2 {print $1}')
            COMPREPLY=( $(compgen -W "${zones}" -- ${cur}) )
            return 0
            ;;
        --backup-schedule)
            COMPREPLY=( $(compgen -W "hourly daily weekly" -- ${cur}) )
            return 0
            ;;
        --backup-retention)
            COMPREPLY=( $(compgen -W "24h 72h 168h 720h" -- ${cur}) )
            return 0
            ;;
        --pdb-type)
            COMPREPLY=( $(compgen -W "auto fixed percent" -- ${cur}) )
            return 0
            ;;
        --vpa-mode)
            COMPREPLY=( $(compgen -W "Off Initial Auto" -- ${cur}) )
            return 0
            ;;
        --pricing-tier)
            COMPREPLY=( $(compgen -W "basic premium" -- ${cur}) )
            return 0
            ;;
    esac

    # Handle subcommands and their flags
    case "${cmd}" in
        auth)
            COMPREPLY=( $(compgen -W "login logout status" -- ${cur}) )
            ;;
        org|organization)
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
        deploy)
            if [[ ${COMP_CWORD} == 2 ]]; then
                COMPREPLY=( $(compgen -W "list get status --cpu --memory --machine --domain --image --dockerfile --env --port --volume-size --volume-mount --zone --multicluster --multicluster-mode --backup-enabled --backup-schedule --backup-retention --backup-priority-cluster --replicas --pdb --pdb-type --pdb-min-available --pdb-percent --hpa --hpa-min-replicas --hpa-max-replicas --hpa-cpu-target --hpa-memory-target --vpa --vpa-mode --vpa-min-cpu --vpa-max-cpu --vpa-min-memory --vpa-max-memory --wait-for" -- ${cur}) )
            else
                case "${subcmd}" in
                    list)
                        COMPREPLY=( $(compgen -W "--namespace --quiet" -- ${cur}) )
                        ;;
                    get|status)
                        COMPREPLY=( $(compgen -W "--deployment-id --watch" -- ${cur}) )
                        ;;
                esac
            fi
            ;;
        service)
            if [[ ${COMP_CWORD} == 2 ]]; then
                COMPREPLY=( $(compgen -W "list delete" -- ${cur}) )
            fi
            ;;
        secret)
            if [[ ${COMP_CWORD} == 2 ]]; then
                COMPREPLY=( $(compgen -W "create list delete" -- ${cur}) )
            fi
            ;;
        ingress)
            if [[ ${COMP_CWORD} == 2 ]]; then
                COMPREPLY=( $(compgen -W "list delete" -- ${cur}) )
            fi
            ;;
        issuer)
            if [[ ${COMP_CWORD} == 2 ]]; then
                COMPREPLY=( $(compgen -W "create list delete" -- ${cur}) )
            fi
            ;;
        env|environment)
            if [[ ${COMP_CWORD} == 2 ]]; then
                COMPREPLY=( $(compgen -W "create list delete" -- ${cur}) )
            fi
            ;;
        machine)
            if [[ ${COMP_CWORD} == 2 ]]; then
                COMPREPLY=( $(compgen -W "list available vm usage" -- ${cur}) )
            else
                case "${subcmd}" in
                    list)
                        COMPREPLY=( $(compgen -W "--quiet" -- ${cur}) )
                        ;;
                    available)
                        COMPREPLY=( $(compgen -W "--quiet --region --zone --min-cpu --min-memory --gpu --recommended --pricing-tier" -- ${cur}) )
                        ;;
                    vm)
                        if [[ ${COMP_CWORD} == 3 ]]; then
                            COMPREPLY=( $(compgen -W "status start stop reboot resize apply-config console" -- ${cur}) )
                        fi
                        ;;
                esac
            fi
            ;;
        domain)
            if [[ ${COMP_CWORD} == 2 ]]; then
                COMPREPLY=( $(compgen -W "list get create delete verify check search purchase purchase-status contact dns" -- ${cur}) )
            fi
            ;;
        credits|billing)
            if [[ ${COMP_CWORD} == 2 ]]; then
                COMPREPLY=( $(compgen -W "balance transactions usage topup invoices auto-topup notifications" -- ${cur}) )
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
        storage|s3|spaces)
            if [[ ${COMP_CWORD} == 2 ]]; then
                COMPREPLY=( $(compgen -W "list get buckets files usage presign delete" -- ${cur}) )
            else
                case "${subcmd}" in
                    buckets)
                        if [[ ${COMP_CWORD} == 3 ]]; then
                            COMPREPLY=( $(compgen -W "list create delete" -- ${cur}) )
                        fi
                        ;;
                    presign)
                        COMPREPLY=( $(compgen -W "--file --expires" -- ${cur}) )
                        ;;
                esac
            fi
            ;;
        logs)
            if [[ ${COMP_CWORD} == 2 ]]; then
                COMPREPLY=( $(compgen -W "stream stats delete" -- ${cur}) )
            else
                case "${subcmd}" in
                    stream)
                        COMPREPLY=( $(compgen -W "--deployment-id --follow --tail" -- ${cur}) )
                        ;;
                esac
            fi
            ;;
        notifications|notif)
            if [[ ${COMP_CWORD} == 2 ]]; then
                COMPREPLY=( $(compgen -W "list count read delete" -- ${cur}) )
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
        user|profile)
            if [[ ${COMP_CWORD} == 2 ]]; then
                COMPREPLY=( $(compgen -W "me update password permissions sessions" -- ${cur}) )
            else
                case "${subcmd}" in
                    update)
                        COMPREPLY=( $(compgen -W "--name --email" -- ${cur}) )
                        ;;
                esac
            fi
            ;;
        token|api-token)
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
        marketplace|market|apps)
            if [[ ${COMP_CWORD} == 2 ]]; then
                COMPREPLY=( $(compgen -W "list get deploy" -- ${cur}) )
            else
                case "${subcmd}" in
                    list)
                        COMPREPLY=( $(compgen -W "--limit --offset --sort" -- ${cur}) )
                        ;;
                    deploy)
                        COMPREPLY=( $(compgen -W "--name --hostname --cpu --memory --domain --storage-size --storage-class --multicluster --multicluster-mode --zone" -- ${cur}) )
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
        pricing|price)
            if [[ ${COMP_CWORD} == 2 ]]; then
                COMPREPLY=( $(compgen -W "list get lookup calculate" -- ${cur}) )
            else
                case "${subcmd}" in
                    lookup)
                        COMPREPLY=( $(compgen -W "--region --type --sla" -- ${cur}) )
                        ;;
                esac
            fi
            ;;
        cluster)
            if [[ ${COMP_CWORD} == 2 ]]; then
                COMPREPLY=( $(compgen -W "zones list" -- ${cur}) )
            fi
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
                'org:Manage organizations'
                'deploy:Deploy applications'
                'service:Manage services'
                'secret:Manage secrets'
                'ingress:Manage ingresses'
                'issuer:Manage certificate issuers'
                'environment:Manage environments'
                'machine:Manage machines'
                'domain:Manage custom domains'
                'credits:Manage credits and billing'
                'storage:Manage S3/object storage'
                'logs:View and stream pod logs'
                'notifications:Manage notifications'
                'user:Manage user profile'
                'token:Manage API tokens'
                'marketplace:Browse and deploy marketplace apps'
                'audit:View audit logs'
                'talos:Talos Linux configuration'
                'admin:Admin operations'
                'pricing:View machine pricing'
                'cluster:View cluster and zone information'
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
                    fi
                    ;;
                service)
                    if (( CURRENT == 2 )); then
                        local -a subcommands=(
                            'list:List services'
                            'delete:Delete a service'
                        )
                        _describe -t subcommands 'service subcommands' subcommands
                    fi
                    ;;
                secret)
                    if (( CURRENT == 2 )); then
                        local -a subcommands=(
                            'create:Create a secret'
                            'list:List secrets'
                            'delete:Delete a secret'
                        )
                        _describe -t subcommands 'secret subcommands' subcommands
                    fi
                    ;;
                ingress)
                    if (( CURRENT == 2 )); then
                        local -a subcommands=(
                            'list:List ingresses'
                            'delete:Delete an ingress'
                        )
                        _describe -t subcommands 'ingress subcommands' subcommands
                    fi
                    ;;
                issuer)
                    if (( CURRENT == 2 )); then
                        local -a subcommands=(
                            'create:Create an issuer'
                            'list:List issuers'
                            'delete:Delete an issuer'
                        )
                        _describe -t subcommands 'issuer subcommands' subcommands
                    fi
                    ;;
                environment)
                    if (( CURRENT == 2 )); then
                        local -a subcommands=(
                            'create:Create an environment'
                            'list:List environments'
                            'delete:Delete an environment'
                        )
                        _describe -t subcommands 'env subcommands' subcommands
                    fi
                    ;;
                machine)
                    if (( CURRENT == 2 )); then
                        local -a subcommands=(
                            'list:List owned machines'
                            'available:List available machines for rent'
                            'vm:Manage Mac agent VM lifecycle'
                            'usage:View machine usage'
                        )
                        _describe -t subcommands 'machine subcommands' subcommands
                    elif [[ $words[2] == "vm" ]] && (( CURRENT == 3 )); then
                        local -a vmcmds=(
                            'status:Show VM state'
                            'start:Start a VM'
                            'stop:Stop a VM'
                            'reboot:Reboot a VM'
                            'resize:Resize a VM'
                            'apply-config:Apply Talos config'
                            'console:Enable/disable console streaming'
                        )
                        _describe -t subcommands 'vm subcommands' vmcmds
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
                domain)
                    if (( CURRENT == 2 )); then
                        local -a subcommands=(
                            'list:List domains'
                            'get:Get domain details'
                            'create:Register a domain'
                            'delete:Delete a domain'
                            'verify:Verify domain ownership'
                            'check:Check domain availability'
                            'search:Search available domains'
                            'purchase:Create purchase intent'
                            'purchase-status:Check purchase status'
                            'contact:Manage contact details'
                            'dns:Manage DNS records'
                        )
                        _describe -t subcommands 'domain subcommands' subcommands
                    fi
                    ;;
                notifications)
                    if (( CURRENT == 2 )); then
                        local -a subcommands=(
                            'list:List notifications'
                            'count:Get unread count'
                            'read:Mark as read'
                            'delete:Delete notification'
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
                            'auto-topup:Manage auto top-up'
                            'notifications:Manage billing notifications'
                        )
                        _describe -t subcommands 'credits subcommands' subcommands
                    fi
                    ;;
                storage)
                    if (( CURRENT == 2 )); then
                        local -a subcommands=(
                            'list:List storage configs'
                            'get:Get storage details'
                            'delete:Delete storage'
                            'buckets:Manage buckets'
                            'files:List files'
                            'presign:Get presigned URL'
                            'usage:View storage usage'
                        )
                        _describe -t subcommands 'storage subcommands' subcommands
                    fi
                    ;;
                logs)
                    if (( CURRENT == 2 )); then
                        local -a subcommands=(
                            'stream:Stream pod logs'
                            'stats:Show log statistics'
                            'delete:Delete stored logs'
                        )
                        _describe -t subcommands 'logs subcommands' subcommands
                    fi
                    ;;
                pricing)
                    if (( CURRENT == 2 )); then
                        local -a subcommands=(
                            'list:List pricing configs'
                            'get:Get pricing details'
                            'lookup:Lookup price by region/type/SLA'
                            'calculate:Calculate deployment cost'
                        )
                        _describe -t subcommands 'pricing subcommands' subcommands
                    fi
                    ;;
                cluster)
                    if (( CURRENT == 2 )); then
                        local -a subcommands=(
                            'zones:List available deployment zones'
                            'list:List enabled clusters'
                        )
                        _describe -t subcommands 'cluster subcommands' subcommands
                    fi
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
        if contains -- $i auth org deploy service secret ingress issuer environment machine domain credits storage logs notifications user token marketplace audit talos admin pricing cluster completion
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

# Zone completion helper
function __fish_1ctl_zones
    1ctl cluster zones 2>/dev/null | awk 'NR>2 {print $1}'
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
complete -c 1ctl -l backup-schedule -xa 'hourly daily weekly'
complete -c 1ctl -l backup-retention -xa '24h 72h 168h 720h'
complete -c 1ctl -l pdb-type -xa 'auto fixed percent'
complete -c 1ctl -l vpa-mode -xa 'Off Initial Auto'
complete -c 1ctl -l pricing-tier -xa 'basic premium'
complete -c 1ctl -l zone -xa '(__fish_1ctl_zones)'

# Top level commands
complete -c 1ctl -f -n __fish_1ctl_no_subcommand -a auth -d 'Authentication commands'
complete -c 1ctl -f -n __fish_1ctl_no_subcommand -a org -d 'Manage organizations'
complete -c 1ctl -f -n __fish_1ctl_no_subcommand -a deploy -d 'Deploy applications'
complete -c 1ctl -f -n __fish_1ctl_no_subcommand -a service -d 'Manage services'
complete -c 1ctl -f -n __fish_1ctl_no_subcommand -a secret -d 'Manage secrets'
complete -c 1ctl -f -n __fish_1ctl_no_subcommand -a ingress -d 'Manage ingresses'
complete -c 1ctl -f -n __fish_1ctl_no_subcommand -a issuer -d 'Manage certificate issuers'
complete -c 1ctl -f -n __fish_1ctl_no_subcommand -a environment -d 'Manage environments'
complete -c 1ctl -f -n __fish_1ctl_no_subcommand -a machine -d 'Manage machines'
complete -c 1ctl -f -n __fish_1ctl_no_subcommand -a domain -d 'Manage custom domains'
complete -c 1ctl -f -n __fish_1ctl_no_subcommand -a credits -d 'Manage credits and billing'
complete -c 1ctl -f -n __fish_1ctl_no_subcommand -a storage -d 'Manage S3/object storage'
complete -c 1ctl -f -n __fish_1ctl_no_subcommand -a logs -d 'View and stream pod logs'
complete -c 1ctl -f -n __fish_1ctl_no_subcommand -a notifications -d 'Manage notifications'
complete -c 1ctl -f -n __fish_1ctl_no_subcommand -a user -d 'Manage user profile'
complete -c 1ctl -f -n __fish_1ctl_no_subcommand -a token -d 'Manage API tokens'
complete -c 1ctl -f -n __fish_1ctl_no_subcommand -a marketplace -d 'Browse and deploy marketplace apps'
complete -c 1ctl -f -n __fish_1ctl_no_subcommand -a audit -d 'View audit logs'
complete -c 1ctl -f -n __fish_1ctl_no_subcommand -a talos -d 'Talos Linux configuration'
complete -c 1ctl -f -n __fish_1ctl_no_subcommand -a admin -d 'Admin operations'
complete -c 1ctl -f -n __fish_1ctl_no_subcommand -a pricing -d 'View machine pricing'
complete -c 1ctl -f -n __fish_1ctl_no_subcommand -a cluster -d 'View cluster and zone information'
complete -c 1ctl -f -n __fish_1ctl_no_subcommand -a completion -d 'Generate shell completion scripts'

# Auth subcommands
complete -c 1ctl -f -n '__fish_1ctl_using_command auth' -a 'login' -d 'Authenticate with Satusky'
complete -c 1ctl -f -n '__fish_1ctl_using_command auth' -a 'logout' -d 'Remove stored authentication'
complete -c 1ctl -f -n '__fish_1ctl_using_command auth' -a 'status' -d 'View authentication status'

# Org subcommands
complete -c 1ctl -f -n '__fish_1ctl_using_command org' -a 'list' -d 'List organizations'
complete -c 1ctl -f -n '__fish_1ctl_using_command org' -a 'current' -d 'Show current organization'
complete -c 1ctl -f -n '__fish_1ctl_using_command org' -a 'switch' -d 'Switch organization'
complete -c 1ctl -f -n '__fish_1ctl_using_command org' -a 'create' -d 'Create organization'
complete -c 1ctl -f -n '__fish_1ctl_using_command org' -a 'delete' -d 'Delete organization'
complete -c 1ctl -f -n '__fish_1ctl_using_command org' -a 'team' -d 'Manage team members'

# Deploy subcommands
complete -c 1ctl -f -n '__fish_1ctl_using_command deploy' -a 'list' -d 'List deployments'
complete -c 1ctl -f -n '__fish_1ctl_using_command deploy' -a 'get' -d 'Get deployment details'
complete -c 1ctl -f -n '__fish_1ctl_using_command deploy' -a 'status' -d 'Check deployment status'

# Service subcommands
complete -c 1ctl -f -n '__fish_1ctl_using_command service' -a 'list' -d 'List services'
complete -c 1ctl -f -n '__fish_1ctl_using_command service' -a 'delete' -d 'Delete a service'

# Secret subcommands
complete -c 1ctl -f -n '__fish_1ctl_using_command secret' -a 'create' -d 'Create a secret'
complete -c 1ctl -f -n '__fish_1ctl_using_command secret' -a 'list' -d 'List secrets'
complete -c 1ctl -f -n '__fish_1ctl_using_command secret' -a 'delete' -d 'Delete a secret'

# Ingress subcommands
complete -c 1ctl -f -n '__fish_1ctl_using_command ingress' -a 'list' -d 'List ingresses'
complete -c 1ctl -f -n '__fish_1ctl_using_command ingress' -a 'delete' -d 'Delete an ingress'

# Issuer subcommands
complete -c 1ctl -f -n '__fish_1ctl_using_command issuer' -a 'create' -d 'Create an issuer'
complete -c 1ctl -f -n '__fish_1ctl_using_command issuer' -a 'list' -d 'List issuers'
complete -c 1ctl -f -n '__fish_1ctl_using_command issuer' -a 'delete' -d 'Delete an issuer'

# Environment subcommands
complete -c 1ctl -f -n '__fish_1ctl_using_command environment' -a 'create' -d 'Create an environment'
complete -c 1ctl -f -n '__fish_1ctl_using_command environment' -a 'list' -d 'List environments'
complete -c 1ctl -f -n '__fish_1ctl_using_command environment' -a 'delete' -d 'Delete an environment'

# Machine subcommands
complete -c 1ctl -f -n '__fish_1ctl_using_command machine' -a 'list' -d 'List owned machines'
complete -c 1ctl -f -n '__fish_1ctl_using_command machine' -a 'available' -d 'List available machines'
complete -c 1ctl -f -n '__fish_1ctl_using_command machine' -a 'vm' -d 'Manage Mac agent VM lifecycle'
complete -c 1ctl -f -n '__fish_1ctl_using_command machine' -a 'usage' -d 'View machine usage'

# Domain subcommands
complete -c 1ctl -f -n '__fish_1ctl_using_command domain' -a 'list' -d 'List domains'
complete -c 1ctl -f -n '__fish_1ctl_using_command domain' -a 'get' -d 'Get domain details'
complete -c 1ctl -f -n '__fish_1ctl_using_command domain' -a 'create' -d 'Register a domain'
complete -c 1ctl -f -n '__fish_1ctl_using_command domain' -a 'delete' -d 'Delete a domain'
complete -c 1ctl -f -n '__fish_1ctl_using_command domain' -a 'verify' -d 'Verify domain ownership'
complete -c 1ctl -f -n '__fish_1ctl_using_command domain' -a 'check' -d 'Check domain availability'
complete -c 1ctl -f -n '__fish_1ctl_using_command domain' -a 'search' -d 'Search available domains'
complete -c 1ctl -f -n '__fish_1ctl_using_command domain' -a 'purchase' -d 'Create purchase intent'
complete -c 1ctl -f -n '__fish_1ctl_using_command domain' -a 'purchase-status' -d 'Check purchase status'
complete -c 1ctl -f -n '__fish_1ctl_using_command domain' -a 'contact' -d 'Manage contact details'
complete -c 1ctl -f -n '__fish_1ctl_using_command domain' -a 'dns' -d 'Manage DNS records'

# Credits subcommands
complete -c 1ctl -f -n '__fish_1ctl_using_command credits' -a 'balance' -d 'View credit balance'
complete -c 1ctl -f -n '__fish_1ctl_using_command credits' -a 'transactions' -d 'View transactions'
complete -c 1ctl -f -n '__fish_1ctl_using_command credits' -a 'usage' -d 'View usage history'
complete -c 1ctl -f -n '__fish_1ctl_using_command credits' -a 'topup' -d 'Top up credits'
complete -c 1ctl -f -n '__fish_1ctl_using_command credits' -a 'invoices' -d 'Manage invoices'
complete -c 1ctl -f -n '__fish_1ctl_using_command credits' -a 'auto-topup' -d 'Manage auto top-up'
complete -c 1ctl -f -n '__fish_1ctl_using_command credits' -a 'notifications' -d 'Manage billing notifications'

# Storage subcommands
complete -c 1ctl -f -n '__fish_1ctl_using_command storage' -a 'list' -d 'List storage configs'
complete -c 1ctl -f -n '__fish_1ctl_using_command storage' -a 'get' -d 'Get storage details'
complete -c 1ctl -f -n '__fish_1ctl_using_command storage' -a 'delete' -d 'Delete storage'
complete -c 1ctl -f -n '__fish_1ctl_using_command storage' -a 'buckets' -d 'Manage buckets'
complete -c 1ctl -f -n '__fish_1ctl_using_command storage' -a 'files' -d 'List files'
complete -c 1ctl -f -n '__fish_1ctl_using_command storage' -a 'presign' -d 'Get presigned URL'
complete -c 1ctl -f -n '__fish_1ctl_using_command storage' -a 'usage' -d 'View storage usage'

# Logs subcommands
complete -c 1ctl -f -n '__fish_1ctl_using_command logs' -a 'stream' -d 'Stream pod logs'
complete -c 1ctl -f -n '__fish_1ctl_using_command logs' -a 'stats' -d 'Show log statistics'
complete -c 1ctl -f -n '__fish_1ctl_using_command logs' -a 'delete' -d 'Delete stored logs'

# Notifications subcommands
complete -c 1ctl -f -n '__fish_1ctl_using_command notifications' -a 'list' -d 'List notifications'
complete -c 1ctl -f -n '__fish_1ctl_using_command notifications' -a 'count' -d 'Get unread count'
complete -c 1ctl -f -n '__fish_1ctl_using_command notifications' -a 'read' -d 'Mark as read'
complete -c 1ctl -f -n '__fish_1ctl_using_command notifications' -a 'delete' -d 'Delete notification'

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

# Pricing subcommands
complete -c 1ctl -f -n '__fish_1ctl_using_command pricing' -a 'list' -d 'List pricing configs'
complete -c 1ctl -f -n '__fish_1ctl_using_command pricing' -a 'get' -d 'Get pricing details'
complete -c 1ctl -f -n '__fish_1ctl_using_command pricing' -a 'lookup' -d 'Lookup price'
complete -c 1ctl -f -n '__fish_1ctl_using_command pricing' -a 'calculate' -d 'Calculate cost'

# Cluster subcommands
complete -c 1ctl -f -n '__fish_1ctl_using_command cluster' -a 'zones' -d 'List available deployment zones'
complete -c 1ctl -f -n '__fish_1ctl_using_command cluster' -a 'list' -d 'List enabled clusters'

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
            [CompletionResult]::new('org', 'org', [CompletionResultType]::ParameterValue, 'Manage organizations')
            [CompletionResult]::new('deploy', 'deploy', [CompletionResultType]::ParameterValue, 'Deploy applications')
            [CompletionResult]::new('service', 'service', [CompletionResultType]::ParameterValue, 'Manage services')
            [CompletionResult]::new('secret', 'secret', [CompletionResultType]::ParameterValue, 'Manage secrets')
            [CompletionResult]::new('ingress', 'ingress', [CompletionResultType]::ParameterValue, 'Manage ingresses')
            [CompletionResult]::new('issuer', 'issuer', [CompletionResultType]::ParameterValue, 'Manage certificate issuers')
            [CompletionResult]::new('environment', 'environment', [CompletionResultType]::ParameterValue, 'Manage environments')
            [CompletionResult]::new('machine', 'machine', [CompletionResultType]::ParameterValue, 'Manage machines')
            [CompletionResult]::new('domain', 'domain', [CompletionResultType]::ParameterValue, 'Manage custom domains')
            [CompletionResult]::new('credits', 'credits', [CompletionResultType]::ParameterValue, 'Manage credits and billing')
            [CompletionResult]::new('storage', 'storage', [CompletionResultType]::ParameterValue, 'Manage S3/object storage')
            [CompletionResult]::new('logs', 'logs', [CompletionResultType]::ParameterValue, 'View and stream pod logs')
            [CompletionResult]::new('notifications', 'notifications', [CompletionResultType]::ParameterValue, 'Manage notifications')
            [CompletionResult]::new('user', 'user', [CompletionResultType]::ParameterValue, 'Manage user profile')
            [CompletionResult]::new('token', 'token', [CompletionResultType]::ParameterValue, 'Manage API tokens')
            [CompletionResult]::new('marketplace', 'marketplace', [CompletionResultType]::ParameterValue, 'Browse and deploy marketplace apps')
            [CompletionResult]::new('audit', 'audit', [CompletionResultType]::ParameterValue, 'View audit logs')
            [CompletionResult]::new('talos', 'talos', [CompletionResultType]::ParameterValue, 'Talos Linux configuration')
            [CompletionResult]::new('admin', 'admin', [CompletionResultType]::ParameterValue, 'Admin operations')
            [CompletionResult]::new('pricing', 'pricing', [CompletionResultType]::ParameterValue, 'View machine pricing')
            [CompletionResult]::new('cluster', 'cluster', [CompletionResultType]::ParameterValue, 'View cluster and zone information')
            [CompletionResult]::new('completion', 'completion', [CompletionResultType]::ParameterValue, 'Generate shell completion scripts')
            break
        }
        '1ctl;auth' {
            [CompletionResult]::new('login', 'login', [CompletionResultType]::ParameterValue, 'Authenticate with Satusky')
            [CompletionResult]::new('logout', 'logout', [CompletionResultType]::ParameterValue, 'Remove stored authentication')
            [CompletionResult]::new('status', 'status', [CompletionResultType]::ParameterValue, 'View authentication status')
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
        '1ctl;deploy' {
            [CompletionResult]::new('list', 'list', [CompletionResultType]::ParameterValue, 'List deployments')
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
            [CompletionResult]::new('create', 'create', [CompletionResultType]::ParameterValue, 'Create a secret')
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
            [CompletionResult]::new('create', 'create', [CompletionResultType]::ParameterValue, 'Create an issuer')
            [CompletionResult]::new('list', 'list', [CompletionResultType]::ParameterValue, 'List issuers')
            [CompletionResult]::new('delete', 'delete', [CompletionResultType]::ParameterValue, 'Delete an issuer')
            break
        }
        '1ctl;environment' {
            [CompletionResult]::new('create', 'create', [CompletionResultType]::ParameterValue, 'Create an environment')
            [CompletionResult]::new('list', 'list', [CompletionResultType]::ParameterValue, 'List environments')
            [CompletionResult]::new('delete', 'delete', [CompletionResultType]::ParameterValue, 'Delete an environment')
            break
        }
        '1ctl;machine' {
            [CompletionResult]::new('list', 'list', [CompletionResultType]::ParameterValue, 'List owned machines')
            [CompletionResult]::new('available', 'available', [CompletionResultType]::ParameterValue, 'List available machines')
            [CompletionResult]::new('vm', 'vm', [CompletionResultType]::ParameterValue, 'Manage Mac agent VM lifecycle')
            [CompletionResult]::new('usage', 'usage', [CompletionResultType]::ParameterValue, 'View machine usage')
            break
        }
        '1ctl;machine;vm' {
            [CompletionResult]::new('status', 'status', [CompletionResultType]::ParameterValue, 'Show VM state')
            [CompletionResult]::new('start', 'start', [CompletionResultType]::ParameterValue, 'Start a VM')
            [CompletionResult]::new('stop', 'stop', [CompletionResultType]::ParameterValue, 'Stop a VM')
            [CompletionResult]::new('reboot', 'reboot', [CompletionResultType]::ParameterValue, 'Reboot a VM')
            [CompletionResult]::new('resize', 'resize', [CompletionResultType]::ParameterValue, 'Resize a VM')
            [CompletionResult]::new('apply-config', 'apply-config', [CompletionResultType]::ParameterValue, 'Apply Talos config')
            [CompletionResult]::new('console', 'console', [CompletionResultType]::ParameterValue, 'Enable/disable console streaming')
            break
        }
        '1ctl;domain' {
            [CompletionResult]::new('list', 'list', [CompletionResultType]::ParameterValue, 'List domains')
            [CompletionResult]::new('get', 'get', [CompletionResultType]::ParameterValue, 'Get domain details')
            [CompletionResult]::new('create', 'create', [CompletionResultType]::ParameterValue, 'Register a domain')
            [CompletionResult]::new('delete', 'delete', [CompletionResultType]::ParameterValue, 'Delete a domain')
            [CompletionResult]::new('verify', 'verify', [CompletionResultType]::ParameterValue, 'Verify domain ownership')
            [CompletionResult]::new('check', 'check', [CompletionResultType]::ParameterValue, 'Check availability')
            [CompletionResult]::new('search', 'search', [CompletionResultType]::ParameterValue, 'Search available domains')
            [CompletionResult]::new('purchase', 'purchase', [CompletionResultType]::ParameterValue, 'Create purchase intent')
            [CompletionResult]::new('purchase-status', 'purchase-status', [CompletionResultType]::ParameterValue, 'Check purchase status')
            [CompletionResult]::new('contact', 'contact', [CompletionResultType]::ParameterValue, 'Manage contact details')
            [CompletionResult]::new('dns', 'dns', [CompletionResultType]::ParameterValue, 'Manage DNS records')
            break
        }
        '1ctl;credits' {
            [CompletionResult]::new('balance', 'balance', [CompletionResultType]::ParameterValue, 'View credit balance')
            [CompletionResult]::new('transactions', 'transactions', [CompletionResultType]::ParameterValue, 'View transactions')
            [CompletionResult]::new('usage', 'usage', [CompletionResultType]::ParameterValue, 'View usage history')
            [CompletionResult]::new('topup', 'topup', [CompletionResultType]::ParameterValue, 'Top up credits')
            [CompletionResult]::new('invoices', 'invoices', [CompletionResultType]::ParameterValue, 'Manage invoices')
            [CompletionResult]::new('auto-topup', 'auto-topup', [CompletionResultType]::ParameterValue, 'Manage auto top-up')
            [CompletionResult]::new('notifications', 'notifications', [CompletionResultType]::ParameterValue, 'Manage billing notifications')
            break
        }
        '1ctl;storage' {
            [CompletionResult]::new('list', 'list', [CompletionResultType]::ParameterValue, 'List storage configs')
            [CompletionResult]::new('get', 'get', [CompletionResultType]::ParameterValue, 'Get storage details')
            [CompletionResult]::new('delete', 'delete', [CompletionResultType]::ParameterValue, 'Delete storage')
            [CompletionResult]::new('buckets', 'buckets', [CompletionResultType]::ParameterValue, 'Manage buckets')
            [CompletionResult]::new('files', 'files', [CompletionResultType]::ParameterValue, 'List files')
            [CompletionResult]::new('presign', 'presign', [CompletionResultType]::ParameterValue, 'Get presigned URL')
            [CompletionResult]::new('usage', 'usage', [CompletionResultType]::ParameterValue, 'View storage usage')
            break
        }
        '1ctl;logs' {
            [CompletionResult]::new('stream', 'stream', [CompletionResultType]::ParameterValue, 'Stream pod logs')
            [CompletionResult]::new('stats', 'stats', [CompletionResultType]::ParameterValue, 'Show log statistics')
            [CompletionResult]::new('delete', 'delete', [CompletionResultType]::ParameterValue, 'Delete stored logs')
            break
        }
        '1ctl;notifications' {
            [CompletionResult]::new('list', 'list', [CompletionResultType]::ParameterValue, 'List notifications')
            [CompletionResult]::new('count', 'count', [CompletionResultType]::ParameterValue, 'Get unread count')
            [CompletionResult]::new('read', 'read', [CompletionResultType]::ParameterValue, 'Mark as read')
            [CompletionResult]::new('delete', 'delete', [CompletionResultType]::ParameterValue, 'Delete notification')
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
        '1ctl;pricing' {
            [CompletionResult]::new('list', 'list', [CompletionResultType]::ParameterValue, 'List pricing configs')
            [CompletionResult]::new('get', 'get', [CompletionResultType]::ParameterValue, 'Get pricing details')
            [CompletionResult]::new('lookup', 'lookup', [CompletionResultType]::ParameterValue, 'Lookup price')
            [CompletionResult]::new('calculate', 'calculate', [CompletionResultType]::ParameterValue, 'Calculate cost')
            break
        }
        '1ctl;cluster' {
            [CompletionResult]::new('zones', 'zones', [CompletionResultType]::ParameterValue, 'List available deployment zones')
            [CompletionResult]::new('list', 'list', [CompletionResultType]::ParameterValue, 'List enabled clusters')
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
