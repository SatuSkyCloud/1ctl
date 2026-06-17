package commands

import (
	"context"
	"1ctl/internal/api"
	satuskyctx "1ctl/internal/context"
	"1ctl/internal/utils"
	"fmt"
	"time"

	"github.com/urfave/cli/v3"
)

func CreditsCommand() *cli.Command {
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
		Action: handleCreditsBalance,
	}
}

func creditsTransactionsCommand() *cli.Command {
	return &cli.Command{
		Name:  "transactions",
		Usage: "Show transaction history",
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:  "limit",
				Usage: "Number of transactions to show",
				Value: 10,
			},
			&cli.IntFlag{
				Name:  "offset",
				Usage: "Offset for pagination",
				Value: 0,
			},
		},
		Action: handleCreditsTransactions,
	}
}

func creditsUsageCommand() *cli.Command {
	return &cli.Command{
		Name:  "usage",
		Usage: "Show machine usage history",
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:  "days",
				Usage: "Number of days to show usage for",
				Value: 7,
			},
		},
		Action: handleCreditsUsage,
	}
}

func handleCreditsBalance(ctx context.Context, cmd *cli.Command) error {
	orgID := satuskyctx.GetCurrentOrgID()
	if orgID == "" {
		return utils.NewError("organization ID not found. Please run '1ctl auth login' first", nil)
	}

	balance, err := api.GetCreditBalance(orgID)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to get credit balance: %s", err.Error()), nil)
	}

	if utils.TryPrintJSON(balance) {
		return nil
	}

	utils.PrintHeader("Credit Balance")
	utils.PrintStatusLine("Organization ID", balance.OrganizationID.String())
	utils.PrintStatusLine("Balance", fmt.Sprintf("$%.2f %s", balance.Balance, balance.Currency))
	utils.PrintStatusLine("Last Updated", formatTimeAgo(balance.UpdatedAt))

	// Display tier information if available
	if balance.Tier != nil {
		fmt.Println()
		utils.PrintHeader("Tier Status")
		utils.PrintStatusLine("Current Tier", utils.SuccessColor(formatTierDisplayName(balance.Tier.CurrentTier)))

		// Show peak tier if different from current
		if balance.Tier.HighestTier != "" && balance.Tier.HighestTier != balance.Tier.CurrentTier {
			utils.PrintStatusLine("Peak Tier", utils.SuccessColor(formatTierDisplayName(balance.Tier.HighestTier))+" (achieved)")
		}

		// Show current limits
		fmt.Println()
		utils.PrintHeader("Resource Limits")
		utils.PrintStatusLine("CPU", balance.Tier.CurrentLimits.CPU)
		utils.PrintStatusLine("Memory", balance.Tier.CurrentLimits.Memory)
		utils.PrintStatusLine("Pods", fmt.Sprintf("%d", balance.Tier.CurrentLimits.Pods))
		utils.PrintStatusLine("PVCs", fmt.Sprintf("%d", balance.Tier.CurrentLimits.PVCs))

		// Show upgrade path if available
		if balance.Tier.CanUpgrade && balance.Tier.NextTier != "" {
			fmt.Println()
			utils.PrintHeader("Upgrade Path")
			utils.PrintStatusLine("Next Tier", formatTierDisplayName(balance.Tier.NextTier))
			utils.PrintStatusLine("Credits Needed", fmt.Sprintf("$%.2f", balance.Tier.CreditsToNextTier))
			if balance.Tier.NextTierLimits != nil {
				utils.PrintStatusLine("Next Tier CPU", balance.Tier.NextTierLimits.CPU)
				utils.PrintStatusLine("Next Tier Memory", balance.Tier.NextTierLimits.Memory)
				utils.PrintStatusLine("Next Tier Pods", fmt.Sprintf("%d", balance.Tier.NextTierLimits.Pods))
			}
		}
	}

	return nil
}

// formatTierDisplayName converts tier ID to display name
func formatTierDisplayName(tier string) string {
	switch tier {
	case "free":
		return "Free"
	case "starter":
		return "Starter"
	case "pro":
		return "Pro"
	case "enterprise":
		return "Enterprise"
	default:
		return tier
	}
}

func handleCreditsTransactions(ctx context.Context, cmd *cli.Command) error {
	orgID := satuskyctx.GetCurrentOrgID()
	if orgID == "" {
		return utils.NewError("organization ID not found. Please run '1ctl auth login' first", nil)
	}

	limit := cmd.Int("limit")
	offset := cmd.Int("offset")

	transactions, err := api.GetCreditTransactions(orgID, limit, offset)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to get transactions: %s", err.Error()), nil)
	}

	if utils.PrintListOrJSON(transactions, "No transactions found") {
		return nil
	}

	utils.PrintHeader("Transaction History")
	for _, tx := range transactions {
		amountStr := fmt.Sprintf("$%.2f", tx.Amount)
		if tx.Amount > 0 {
			amountStr = "+" + amountStr
		}
		utils.PrintStatusLine("ID", tx.TransactionID.String())
		utils.PrintStatusLine("Amount", amountStr)
		utils.PrintStatusLine("Type", tx.TransactionType)
		utils.PrintStatusLine("Description", tx.Description)
		utils.PrintStatusLine("Date", formatTimeAgo(tx.CreatedAt))
		utils.PrintDivider()
	}
	return nil
}

func handleCreditsUsage(ctx context.Context, cmd *cli.Command) error {
	orgID := satuskyctx.GetCurrentOrgID()
	if orgID == "" {
		return utils.NewError("organization ID not found. Please run '1ctl auth login' first", nil)
	}

	days := cmd.Int("days")

	usages, err := api.GetMachineUsageHistory(orgID, days)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to get machine usage: %s", err.Error()), nil)
	}

	if utils.PrintListOrJSON(usages, "No machine usage found for the last 7 days") {
		return nil
	}

	utils.PrintHeader("Machine Usage History")
	var totalCost float64
	for _, usage := range usages {
		utils.PrintStatusLine("Machine", usage.MachineName)
		utils.PrintStatusLine("Hours", fmt.Sprintf("%.1f", usage.HoursUsed))
		utils.PrintStatusLine("Cost", fmt.Sprintf("$%.2f", usage.Cost))
		utils.PrintStatusLine("Status", usage.Status)
		utils.PrintStatusLine("Period", fmt.Sprintf("Last %d days", days))
		utils.PrintDivider()
		totalCost += usage.Cost
	}
	fmt.Printf("\nTotal Cost: $%.2f\n", totalCost)
	return nil
}

// formatTimeAgo formats a time as a human-readable "X ago" string
func formatTimeAgo(t time.Time) string {
	duration := time.Since(t)

	switch {
	case duration < time.Minute:
		return "just now"
	case duration < time.Hour:
		minutes := int(duration.Minutes())
		if minutes == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", minutes)
	case duration < 24*time.Hour:
		hours := int(duration.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	case duration < 7*24*time.Hour:
		days := int(duration.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	case duration < 30*24*time.Hour:
		weeks := int(duration.Hours() / 24 / 7)
		if weeks == 1 {
			return "1 week ago"
		}
		return fmt.Sprintf("%d weeks ago", weeks)
	default:
		months := int(duration.Hours() / 24 / 30)
		if months == 1 {
			return "1 month ago"
		}
		return fmt.Sprintf("%d months ago", months)
	}
}
