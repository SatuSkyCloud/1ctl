package commands

import (
	"1ctl/internal/api"
	"1ctl/internal/context"
	"1ctl/internal/utils"
	"fmt"
	"time"

	"github.com/urfave/cli/v2"
)

func CreditsCommand() *cli.Command {
	return &cli.Command{
		Name:    "credits",
		Aliases: []string{"billing"},
		Usage:   "Manage credits and billing",
		Subcommands: []*cli.Command{
			creditsBalanceCommand(),
			creditsTransactionsCommand(),
			creditsUsageCommand(),
			creditsTopupCommand(),
			creditsInvoicesCommand(),
			creditsAutoTopupCommand(),
			creditsNotificationsCommand(),
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

func creditsTopupCommand() *cli.Command {
	return &cli.Command{
		Name:  "topup",
		Usage: "Initiate a credit topup",
		Flags: []cli.Flag{
			&cli.Float64Flag{
				Name:     "amount",
				Usage:    "Amount to top up (in USD)",
				Required: true,
			},
		},
		Action: handleCreditsTopup,
	}
}

func creditsInvoicesCommand() *cli.Command {
	return &cli.Command{
		Name:  "invoices",
		Usage: "Manage invoices",
		Subcommands: []*cli.Command{
			{
				Name:   "list",
				Usage:  "List all invoices",
				Action: handleInvoicesList,
			},
			{
				Name:  "get",
				Usage: "Get invoice details",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "invoice-id",
						Usage:    "Invoice ID to retrieve",
						Required: true,
					},
				},
				Action: handleInvoiceGet,
			},
			{
				Name:   "summary",
				Usage:  "Show invoice summary",
				Action: handleInvoiceSummary,
			},
			{
				Name:  "generate",
				Usage: "Generate a new invoice",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "start-date",
						Usage:    "Start date (YYYY-MM-DD)",
						Required: true,
					},
					&cli.StringFlag{
						Name:     "end-date",
						Usage:    "End date (YYYY-MM-DD)",
						Required: true,
					},
				},
				Action: handleInvoiceGenerate,
			},
		},
		Action: handleInvoicesList, // Default action shows list
	}
}

func handleCreditsBalance(c *cli.Context) error {
	orgID := context.GetCurrentOrgID()
	if orgID == "" {
		return utils.NewError("organization ID not found. Please run '1ctl auth login' first", nil)
	}

	balance, err := api.GetCreditBalance(orgID)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to get credit balance: %s", err.Error()), nil)
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

func handleCreditsTransactions(c *cli.Context) error {
	orgID := context.GetCurrentOrgID()
	if orgID == "" {
		return utils.NewError("organization ID not found. Please run '1ctl auth login' first", nil)
	}

	limit := c.Int("limit")
	offset := c.Int("offset")

	transactions, err := api.GetCreditTransactions(orgID, limit, offset)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to get transactions: %s", err.Error()), nil)
	}

	if len(transactions) == 0 {
		utils.PrintInfo("No transactions found")
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

func handleCreditsUsage(c *cli.Context) error {
	orgID := context.GetCurrentOrgID()
	if orgID == "" {
		return utils.NewError("organization ID not found. Please run '1ctl auth login' first", nil)
	}

	days := c.Int("days")

	usages, err := api.GetMachineUsageHistory(orgID, days)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to get machine usage: %s", err.Error()), nil)
	}

	if len(usages) == 0 {
		utils.PrintInfo("No machine usage found for the last %d days", days)
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

func handleCreditsTopup(c *cli.Context) error {
	orgID := context.GetCurrentOrgID()
	if orgID == "" {
		return utils.NewError("organization ID not found. Please run '1ctl auth login' first", nil)
	}

	amount := c.Float64("amount")
	if amount <= 0 {
		return utils.NewError("amount must be greater than 0", nil)
	}

	result, err := api.InitiateTopup(orgID, amount)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to initiate topup: %s", err.Error()), nil)
	}

	utils.PrintSuccess("Top-up initiated successfully!")
	utils.PrintStatusLine("Amount", fmt.Sprintf("$%.2f USD", result.Amount))
	utils.PrintStatusLine("Status", result.Status)
	if result.PaymentURL != "" {
		utils.PrintStatusLine("Payment URL", result.PaymentURL)
	}
	return nil
}

func handleInvoicesList(c *cli.Context) error {
	orgID := context.GetCurrentOrgID()
	if orgID == "" {
		return utils.NewError("organization ID not found. Please run '1ctl auth login' first", nil)
	}

	invoices, err := api.GetInvoices(orgID)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to get invoices: %s", err.Error()), nil)
	}

	if len(invoices) == 0 {
		utils.PrintInfo("No invoices found")
		return nil
	}

	utils.PrintHeader("Invoices")
	for _, inv := range invoices {
		utils.PrintStatusLine("Invoice #", inv.InvoiceNumber)
		utils.PrintStatusLine("Amount", fmt.Sprintf("$%.2f", inv.Amount))
		utils.PrintStatusLine("Status", inv.Status)
		utils.PrintStatusLine("Period", fmt.Sprintf("%s - %s", inv.PeriodStart.Format("Jan 2006"), inv.PeriodEnd.Format("Jan 2006")))
		utils.PrintStatusLine("Created", formatTimeAgo(inv.CreatedAt))
		utils.PrintDivider()
	}
	return nil
}

func handleInvoiceGet(c *cli.Context) error {
	orgID := context.GetCurrentOrgID()
	if orgID == "" {
		return utils.NewError("organization ID not found. Please run '1ctl auth login' first", nil)
	}

	invoiceID := c.String("invoice-id")
	if invoiceID == "" {
		return utils.NewError("--invoice-id is required", nil)
	}

	invoice, err := api.GetInvoiceByID(orgID, invoiceID)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to get invoice: %s", err.Error()), nil)
	}

	utils.PrintHeader("Invoice Details")
	utils.PrintStatusLine("Invoice ID", invoice.InvoiceID.String())
	utils.PrintStatusLine("Invoice #", invoice.InvoiceNumber)
	utils.PrintStatusLine("Amount", fmt.Sprintf("$%.2f", invoice.Amount))
	utils.PrintStatusLine("Status", invoice.Status)
	utils.PrintStatusLine("Period Start", invoice.PeriodStart.Format("2006-01-02"))
	utils.PrintStatusLine("Period End", invoice.PeriodEnd.Format("2006-01-02"))
	if !invoice.DueDate.IsZero() {
		utils.PrintStatusLine("Due Date", invoice.DueDate.Format("2006-01-02"))
	}
	if !invoice.PaidAt.IsZero() {
		utils.PrintStatusLine("Paid At", invoice.PaidAt.Format("2006-01-02"))
	}
	utils.PrintStatusLine("Created", formatTimeAgo(invoice.CreatedAt))
	return nil
}

func handleInvoiceSummary(c *cli.Context) error {
	orgID := context.GetCurrentOrgID()
	if orgID == "" {
		return utils.NewError("organization ID not found. Please run '1ctl auth login' first", nil)
	}

	summary, err := api.GetInvoiceSummary(orgID)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to get invoice summary: %s", err.Error()), nil)
	}

	utils.PrintHeader("Invoice Summary")
	utils.PrintStatusLine("Total Invoices", fmt.Sprintf("%d", summary.TotalInvoices))
	utils.PrintStatusLine("Total Amount", fmt.Sprintf("$%.2f", summary.TotalAmount))
	utils.PrintStatusLine("Paid Amount", fmt.Sprintf("$%.2f", summary.PaidAmount))
	utils.PrintStatusLine("Pending Amount", fmt.Sprintf("$%.2f", summary.PendingAmount))
	utils.PrintStatusLine("Overdue Amount", fmt.Sprintf("$%.2f", summary.OverdueAmount))
	return nil
}

func handleInvoiceGenerate(c *cli.Context) error {
	orgID := context.GetCurrentOrgID()
	if orgID == "" {
		return utils.NewError("organization ID not found. Please run '1ctl auth login' first", nil)
	}

	startDateStr := c.String("start-date")
	endDateStr := c.String("end-date")

	startDate, err := time.Parse("2006-01-02", startDateStr)
	if err != nil {
		return utils.NewError(fmt.Sprintf("invalid start-date format. Use YYYY-MM-DD: %s", err.Error()), nil)
	}

	endDate, err := time.Parse("2006-01-02", endDateStr)
	if err != nil {
		return utils.NewError(fmt.Sprintf("invalid end-date format. Use YYYY-MM-DD: %s", err.Error()), nil)
	}

	invoice, err := api.GenerateInvoice(orgID, startDate, endDate)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to generate invoice: %s", err.Error()), nil)
	}

	utils.PrintSuccess("Invoice generated successfully!")
	utils.PrintStatusLine("Invoice ID", invoice.InvoiceID.String())
	utils.PrintStatusLine("Invoice #", invoice.InvoiceNumber)
	utils.PrintStatusLine("Amount", fmt.Sprintf("$%.2f", invoice.Amount))
	utils.PrintStatusLine("Status", invoice.Status)
	return nil
}

// ============================================================
// Auto-Topup Settings
// ============================================================

func creditsAutoTopupCommand() *cli.Command {
	return &cli.Command{
		Name:  "auto-topup",
		Usage: "Manage auto-topup settings",
		Subcommands: []*cli.Command{
			{
				Name:   "get",
				Usage:  "Show current auto-topup settings",
				Action: handleAutoTopupGet,
			},
			{
				Name:  "set",
				Usage: "Configure auto-topup settings",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "enabled",
						Usage: "Enable auto-topup",
					},
					&cli.Float64Flag{
						Name:  "threshold",
						Usage: "Balance threshold that triggers auto-topup (USD)",
					},
					&cli.Float64Flag{
						Name:  "amount",
						Usage: "Amount to top up when triggered (USD)",
					},
				},
				Action: handleAutoTopupSet,
			},
		},
	}
}

func handleAutoTopupGet(c *cli.Context) error {
	orgID := context.GetCurrentOrgID()
	if orgID == "" {
		return utils.NewError("organization ID not found. Please run '1ctl auth login' first", nil)
	}

	settings, err := api.GetAutoTopupSettings(orgID)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to get auto-topup settings: %s", err.Error()), nil)
	}

	utils.PrintHeader("Auto-Topup Settings")
	enabledStr := utils.WarnColor("disabled")
	if settings.Enabled {
		enabledStr = utils.SuccessColor("enabled")
	}
	utils.PrintStatusLine("Status", enabledStr)
	utils.PrintStatusLine("Threshold", fmt.Sprintf("$%.2f", settings.ThresholdAmount))
	utils.PrintStatusLine("Topup Amount", fmt.Sprintf("$%.2f", settings.TopupAmount))
	if settings.HasPaymentMethod {
		pm := fmt.Sprintf("%s •••• %s", settings.PaymentMethodBrand, settings.PaymentMethodLast4)
		utils.PrintStatusLine("Payment Method", pm)
	} else {
		utils.PrintStatusLine("Payment Method", utils.WarnColor("none configured"))
	}
	if settings.LastTopupAt != nil {
		utils.PrintStatusLine("Last Topup", *settings.LastTopupAt)
	}
	return nil
}

func handleAutoTopupSet(c *cli.Context) error {
	orgID := context.GetCurrentOrgID()
	if orgID == "" {
		return utils.NewError("organization ID not found. Please run '1ctl auth login' first", nil)
	}

	// Fetch current settings to use as defaults for unset flags
	current, err := api.GetAutoTopupSettings(orgID)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to get current settings: %s", err.Error()), nil)
	}

	req := api.AutoTopupSettingsRequest{
		Enabled:         current.Enabled,
		ThresholdAmount: current.ThresholdAmount,
		TopupAmount:     current.TopupAmount,
	}
	if c.IsSet("enabled") {
		req.Enabled = c.Bool("enabled")
	}
	if c.IsSet("threshold") {
		req.ThresholdAmount = c.Float64("threshold")
	}
	if c.IsSet("amount") {
		req.TopupAmount = c.Float64("amount")
	}

	updated, err := api.UpdateAutoTopupSettings(orgID, req)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to update auto-topup settings: %s", err.Error()), nil)
	}

	utils.PrintSuccess("Auto-topup settings updated")
	enabledStr := utils.WarnColor("disabled")
	if updated.Enabled {
		enabledStr = utils.SuccessColor("enabled")
	}
	utils.PrintStatusLine("Status", enabledStr)
	utils.PrintStatusLine("Threshold", fmt.Sprintf("$%.2f", updated.ThresholdAmount))
	utils.PrintStatusLine("Topup Amount", fmt.Sprintf("$%.2f", updated.TopupAmount))
	return nil
}

// ============================================================
// Notification Preferences
// ============================================================

func creditsNotificationsCommand() *cli.Command {
	return &cli.Command{
		Name:  "notifications",
		Usage: "Manage billing notification preferences",
		Subcommands: []*cli.Command{
			{
				Name:   "get",
				Usage:  "Show current notification preferences",
				Action: handleNotificationsGet,
			},
			{
				Name:  "set",
				Usage: "Update notification preferences",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "email",
						Usage: "Enable email notifications",
					},
					&cli.BoolFlag{
						Name:  "in-app",
						Usage: "Enable in-app notifications",
					},
					&cli.BoolFlag{
						Name:  "webhook",
						Usage: "Enable webhook notifications",
					},
					&cli.StringFlag{
						Name:  "webhook-url",
						Usage: "Webhook URL for notifications",
					},
					&cli.BoolFlag{
						Name:  "digest",
						Usage: "Enable digest notifications",
					},
					&cli.StringFlag{
						Name:  "digest-frequency",
						Usage: "Digest frequency (daily, weekly)",
					},
					&cli.Float64Flag{
						Name:  "low-balance-amount",
						Usage: "Alert when balance falls below this amount (USD)",
					},
				},
				Action: handleNotificationsSet,
			},
		},
	}
}

func handleNotificationsGet(c *cli.Context) error {
	orgID := context.GetCurrentOrgID()
	if orgID == "" {
		return utils.NewError("organization ID not found. Please run '1ctl auth login' first", nil)
	}

	prefs, err := api.GetNotificationPreferences(orgID)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to get notification preferences: %s", err.Error()), nil)
	}

	utils.PrintHeader("Notification Preferences")
	utils.PrintStatusLine("Email", formatBoolEnabled(prefs.EmailEnabled))
	utils.PrintStatusLine("In-App", formatBoolEnabled(prefs.InAppEnabled))
	utils.PrintStatusLine("Webhook", formatBoolEnabled(prefs.WebhookEnabled))
	if prefs.WebhookEnabled && prefs.WebhookURL != "" {
		utils.PrintStatusLine("Webhook URL", prefs.WebhookURL)
	}
	utils.PrintStatusLine("Digest", formatBoolEnabled(prefs.DigestEnabled))
	if prefs.DigestEnabled && prefs.DigestFrequency != "" {
		utils.PrintStatusLine("Digest Frequency", prefs.DigestFrequency)
	}
	utils.PrintStatusLine("Low Balance Alert", fmt.Sprintf("$%.2f", prefs.LowBalanceThresholdAmount))
	utils.PrintStatusLine("Alert Cooldown", fmt.Sprintf("%d hours", prefs.AlertCooldownHours))
	return nil
}

func handleNotificationsSet(c *cli.Context) error {
	orgID := context.GetCurrentOrgID()
	if orgID == "" {
		return utils.NewError("organization ID not found. Please run '1ctl auth login' first", nil)
	}

	// Fetch current to use as defaults for unset flags
	current, err := api.GetNotificationPreferences(orgID)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to get current preferences: %s", err.Error()), nil)
	}

	req := api.NotificationPreferencesRequest{
		EmailEnabled:              current.EmailEnabled,
		InAppEnabled:              current.InAppEnabled,
		WebhookEnabled:            current.WebhookEnabled,
		WebhookURL:                current.WebhookURL,
		DigestEnabled:             current.DigestEnabled,
		DigestFrequency:           current.DigestFrequency,
		LowBalanceThresholdAmount: current.LowBalanceThresholdAmount,
		AlertCooldownHours:        current.AlertCooldownHours,
	}
	if c.IsSet("email") {
		req.EmailEnabled = c.Bool("email")
	}
	if c.IsSet("in-app") {
		req.InAppEnabled = c.Bool("in-app")
	}
	if c.IsSet("webhook") {
		req.WebhookEnabled = c.Bool("webhook")
	}
	if c.IsSet("webhook-url") {
		req.WebhookURL = c.String("webhook-url")
	}
	if c.IsSet("digest") {
		req.DigestEnabled = c.Bool("digest")
	}
	if c.IsSet("digest-frequency") {
		req.DigestFrequency = c.String("digest-frequency")
	}
	if c.IsSet("low-balance-amount") {
		req.LowBalanceThresholdAmount = c.Float64("low-balance-amount")
	}

	updated, err := api.UpdateNotificationPreferences(orgID, req)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to update notification preferences: %s", err.Error()), nil)
	}

	utils.PrintSuccess("Notification preferences updated")
	utils.PrintStatusLine("Email", formatBoolEnabled(updated.EmailEnabled))
	utils.PrintStatusLine("In-App", formatBoolEnabled(updated.InAppEnabled))
	utils.PrintStatusLine("Webhook", formatBoolEnabled(updated.WebhookEnabled))
	utils.PrintStatusLine("Digest", formatBoolEnabled(updated.DigestEnabled))
	return nil
}

func formatBoolEnabled(b bool) string {
	if b {
		return utils.SuccessColor("enabled")
	}
	return utils.WarnColor("disabled")
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
