// Package commands assembles the full 1ctl command tree.  Individual
// command groups live in isolated sub-packages (e.g. commands/pricing/).
// This file re-exports them so callers see one flat API surface:
//
//	commands.PricingCommand()
//
// without importing sub-packages directly.
package commands

import (
	"1ctl/internal/commands/audit"
	"1ctl/internal/commands/auth"
	"1ctl/internal/commands/cluster"
	"1ctl/internal/commands/completion"
	"1ctl/internal/commands/credits"
	deploycmd "1ctl/internal/commands/deploy"
	"1ctl/internal/commands/doctor"
	"1ctl/internal/commands/domains"
	"1ctl/internal/commands/environment"
	initcmd "1ctl/internal/commands/init"
	"1ctl/internal/commands/ingress"
	"1ctl/internal/commands/issuer"
	"1ctl/internal/commands/launch"
	"1ctl/internal/commands/logs"
	"1ctl/internal/commands/machine"
	"1ctl/internal/commands/marketplace"
	"1ctl/internal/commands/postgres"
	"1ctl/internal/commands/notifications"
	"1ctl/internal/commands/org"
	"1ctl/internal/commands/pricing"
	"1ctl/internal/commands/profile"
	"1ctl/internal/commands/secret"
	"1ctl/internal/commands/service"
	"1ctl/internal/commands/token"
	"1ctl/internal/commands/user"
	"1ctl/internal/commands/volumes"

	"github.com/urfave/cli/v3"
)

// AuthCommand returns the "1ctl auth" command tree.
func AuthCommand() *cli.Command { return auth.Command() }

// ProfileCommand returns the "1ctl profile" command tree.
func ProfileCommand() *cli.Command { return profile.Command() }

// InitCommand returns the "1ctl init" command tree.
func InitCommand() *cli.Command { return initcmd.Command() }

// LaunchCommand returns the "1ctl launch" command tree.
func LaunchCommand() *cli.Command { return launch.Command() }

// CompletionCommand returns the "1ctl completion" command tree.
func CompletionCommand() *cli.Command { return completion.Command() }

// PricingCommand returns the "1ctl pricing" command tree.
func PricingCommand() *cli.Command { return pricing.Command() }

// CreditsCommand returns the "1ctl credits" command tree.
func CreditsCommand() *cli.Command { return credits.Command() }

// LogsCommand returns the "1ctl logs" command tree.
func LogsCommand() *cli.Command { return logs.Command() }

// NotificationsCommand returns the "1ctl notifications" command tree.
func NotificationsCommand() *cli.Command { return notifications.Command() }

// UserCommand returns the "1ctl user" command tree.
func UserCommand() *cli.Command { return user.Command() }

// TokenCommand returns the "1ctl token" command tree.
func TokenCommand() *cli.Command { return token.Command() }

// MarketplaceCommand returns the "1ctl marketplace" command tree.
func MarketplaceCommand() *cli.Command { return marketplace.Command() }

// AuditCommand returns the "1ctl audit" command tree.
func AuditCommand() *cli.Command { return audit.Command() }

// ClusterCommand returns the "1ctl cluster" command tree.
func ClusterCommand() *cli.Command { return cluster.Command() }

// ServiceCommand returns the "1ctl service" command tree.
func ServiceCommand() *cli.Command { return service.Command() }

// IngressCommand returns the "1ctl ingress" command tree.
func IngressCommand() *cli.Command { return ingress.Command() }

// IssuerCommand returns the "1ctl issuer" command tree.
func IssuerCommand() *cli.Command { return issuer.Command() }

// VolumesCommand returns the "1ctl volumes" command tree.
func VolumesCommand() *cli.Command { return volumes.Command() }

// OrgCommand returns the "1ctl org" command tree.
func OrgCommand() *cli.Command { return org.Command() }

// DoctorCommand returns the "1ctl doctor" command tree.
func DoctorCommand() *cli.Command { return doctor.Command() }

// SecretCommand returns the "1ctl secret" command tree.
func SecretCommand() *cli.Command { return secret.Command() }

// EnvironmentCommand returns the "1ctl env" command tree.
func EnvironmentCommand() *cli.Command { return environment.Command() }

// ConfigCommand returns the "1ctl config" command tree (alias for env).
func ConfigCommand() *cli.Command {
	cmd := environment.Command()
	cmd.Name = "config"
	cmd.Usage = "Manage environment variables"
	cmd.Aliases = []string{"env", "environment"}
	cmd.Description = `Manage non-sensitive environment variables.
Secrets are managed separately via "1ctl secret".`
	return cmd
}

// DomainsCommand returns the "1ctl domains" command tree.
func DomainsCommand() *cli.Command { return domains.Command() }

// MachineCommand returns the "1ctl machine" command tree.
func MachineCommand() *cli.Command { return machine.Command() }

// DeployCommand returns the "1ctl deploy" command tree.
func DeployCommand() *cli.Command { return deploycmd.Command() }

// AppCommand returns the "1ctl app" command tree.
func AppCommand() *cli.Command { return deploycmd.AppCommand() }

// PostgresCommand returns the "1ctl postgres" command tree.
func PostgresCommand() *cli.Command { return postgres.Command() }
