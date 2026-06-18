// Package commands assembles the full 1ctl command tree.  Individual
// command groups live in isolated sub-packages (e.g. commands/pricing/).
// This file re-exports them so callers see one flat API surface:
//
//	commands.PricingCommand()
//
// without importing sub-packages directly.
package commands

import (
	"1ctl/internal/commands/pricing"

	"github.com/urfave/cli/v3"
)

// PricingCommand returns the "1ctl pricing" command tree.
func PricingCommand() *cli.Command { return pricing.Command() }
