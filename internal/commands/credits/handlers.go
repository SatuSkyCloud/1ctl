package credits

import (
	"context"
	"fmt"

	"1ctl/internal/api"
	satuskyctx "1ctl/internal/context"
	"1ctl/internal/utils"
)

// --- Handlers -----------------------------------------------------------

func handleCreditsBalance(ctx context.Context) error {
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
	utils.PrintStatusLine("Last Updated", utils.FormatTimeAgo(balance.UpdatedAt))

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

func handleCreditsTransactions(ctx context.Context, in creditsTransactionsInput) error {
	orgID := satuskyctx.GetCurrentOrgID()
	if orgID == "" {
		return utils.NewError("organization ID not found. Please run '1ctl auth login' first", nil)
	}

	transactions, err := api.GetCreditTransactions(orgID, in.Limit, in.Offset)
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
		utils.PrintStatusLine("Date", utils.FormatTimeAgo(tx.CreatedAt))
		utils.PrintDivider()
	}
	return nil
}

func handleCreditsUsage(ctx context.Context, in creditsUsageInput) error {
	orgID := satuskyctx.GetCurrentOrgID()
	if orgID == "" {
		return utils.NewError("organization ID not found. Please run '1ctl auth login' first", nil)
	}

	usages, err := api.GetMachineUsageHistory(orgID, in.Days)
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
		utils.PrintStatusLine("Period", fmt.Sprintf("Last %d days", in.Days))
		utils.PrintDivider()
		totalCost += usage.Cost
	}
	fmt.Printf("\nTotal Cost: $%.2f\n", totalCost)
	return nil
}

// --- Shared helpers -----------------------------------------------------

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
