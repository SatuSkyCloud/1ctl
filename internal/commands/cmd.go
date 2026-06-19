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
	"1ctl/internal/commands/cluster"
	"1ctl/internal/commands/credits"
	"1ctl/internal/commands/logs"
	"1ctl/internal/commands/marketplace"
	"1ctl/internal/commands/notifications"
	"1ctl/internal/commands/pricing"
	"1ctl/internal/commands/token"
	"1ctl/internal/commands/user"

	"github.com/urfave/cli/v3"
)

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
