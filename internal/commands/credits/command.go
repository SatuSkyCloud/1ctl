// Package credits defines the "1ctl credits" command tree — flag names,
// input structs, and CLI wiring.  Handler logic lives in handlers.go.
package credits

import (
	"context"

	"github.com/urfave/cli/v3"
)

// --- Flag name constants ------------------------------------------------

const (
	flagLimit  = "limit"
	flagOffset = "offset"
	flagDays   = "days"
)

// --- Input structs ------------------------------------------------------

type creditsTransactionsInput struct {
	Limit  int
	Offset int
}

type creditsUsageInput struct {
	Days int
}

// --- Flag constructors --------------------------------------------------

func optionalInt(name, usage string, dest *int, defaultValue int) *cli.IntFlag {
	return &cli.IntFlag{
		Name:        name,
		Usage:       usage,
		Destination: dest,
		Value:       defaultValue,
	}
}

// --- Command tree -------------------------------------------------------

// Command returns the root credits command tree.
func Command() *cli.Command {
	return &cli.Command{
		Name:    "credits",
		Aliases: []string{"billing"},
		Usage:   "Manage credits and billing",
		Commands: []*cli.Command{
			creditsBalanceCommand(),
			creditsTransactionsCommand(),
			creditsUsageCommand(),
		},
	}
}

func creditsBalanceCommand() *cli.Command {
	return &cli.Command{
		Name:   "balance",
		Usage:  "Show current credit balance",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return handleCreditsBalance(ctx)
		},
	}
}

func creditsTransactionsCommand() *cli.Command {
	var in creditsTransactionsInput
	return &cli.Command{
		Name:  "transactions",
		Usage: "Show transaction history",
		Flags: []cli.Flag{
			optionalInt(flagLimit, "Number of transactions to show", &in.Limit, 10),
			optionalInt(flagOffset, "Offset for pagination", &in.Offset, 0),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return handleCreditsTransactions(ctx, in)
		},
	}
}

func creditsUsageCommand() *cli.Command {
	var in creditsUsageInput
	return &cli.Command{
		Name:  "usage",
		Usage: "Show machine usage history",
		Flags: []cli.Flag{
			optionalInt(flagDays, "Number of days to show usage for", &in.Days, 7),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return handleCreditsUsage(ctx, in)
		},
	}
}
