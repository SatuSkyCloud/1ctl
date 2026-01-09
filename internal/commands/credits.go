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
	return nil
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
