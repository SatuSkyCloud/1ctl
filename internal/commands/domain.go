package commands

import (
	"1ctl/internal/api"
	"1ctl/internal/context"
	"1ctl/internal/utils"
	"fmt"
	"strings"

	"github.com/urfave/cli/v2"
)

func DomainCommand() *cli.Command {
	return &cli.Command{
		Name:  "domain",
		Usage: "Manage custom domains and DNS records",
		Subcommands: []*cli.Command{
			domainListCommand(),
			domainGetCommand(),
			domainCreateCommand(),
			domainDeleteCommand(),
			domainVerifyCommand(),
			domainCheckCommand(),
			domainSearchCommand(),
			domainPurchaseCommand(),
			domainPurchaseStatusCommand(),
			domainContactCommand(),
			domainDNSCommand(),
		},
	}
}

// requireDomainContext validates userID and orgID are available from CLI context
func requireDomainContext() (string, string, error) {
	userID := context.GetUserID()
	if userID == "" {
		return "", "", utils.NewError("user ID not found. Please run '1ctl auth login' first", nil)
	}
	orgID := context.GetCurrentOrgID()
	if orgID == "" {
		return "", "", utils.NewError("organization ID not found. Please run '1ctl auth login' first", nil)
	}
	return userID, orgID, nil
}

func domainListCommand() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List all domains in the current organization",
		Action: func(c *cli.Context) error {
			userID, orgID, err := requireDomainContext()
			if err != nil {
				return err
			}
			domains, err := api.ListDomains(userID, orgID)
			if err != nil {
				return utils.NewError(fmt.Sprintf("failed to list domains: %s", err.Error()), nil)
			}
			if len(domains) == 0 {
				utils.PrintInfo("No domains found")
				return nil
			}
			utils.PrintHeader("Domains (%d)", len(domains))
			for _, d := range domains {
				printDomainSummary(&d)
				utils.PrintDivider()
			}
			return nil
		},
	}
}

func domainGetCommand() *cli.Command {
	return &cli.Command{
		Name:      "get",
		Usage:     "Get details for a domain",
		ArgsUsage: "<domain-id>",
		Action: func(c *cli.Context) error {
			userID, orgID, err := requireDomainContext()
			if err != nil {
				return err
			}
			domainID := c.Args().First()
			if domainID == "" {
				return utils.NewError("domain ID is required", nil)
			}
			d, err := api.GetDomain(userID, orgID, domainID)
			if err != nil {
				return utils.NewError(fmt.Sprintf("failed to get domain: %s", err.Error()), nil)
			}
			utils.PrintHeader("Domain: %s", d.Name)
			printDomainDetails(d)
			return nil
		},
	}
}

func domainCreateCommand() *cli.Command {
	return &cli.Command{
		Name:  "create",
		Usage: "Create a new domain",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "name",
				Aliases:  []string{"n"},
				Usage:    "Domain name (e.g. example.com)",
				Required: true,
			},
			&cli.StringFlag{
				Name:  "ip",
				Usage: "Optional IP address to create a default A record",
			},
		},
		Action: func(c *cli.Context) error {
			userID, orgID, err := requireDomainContext()
			if err != nil {
				return err
			}
			req := api.DomainCreateRequest{
				Name:      c.String("name"),
				IPAddress: c.String("ip"),
			}
			d, err := api.CreateDomain(userID, orgID, req)
			if err != nil {
				return utils.NewError(fmt.Sprintf("failed to create domain: %s", err.Error()), nil)
			}
			utils.PrintSuccess("Domain created successfully")
			printDomainDetails(d)
			return nil
		},
	}
}

func domainDeleteCommand() *cli.Command {
	return &cli.Command{
		Name:      "delete",
		Usage:     "Delete a domain and all its DNS records",
		ArgsUsage: "<domain-id>",
		Action: func(c *cli.Context) error {
			userID, orgID, err := requireDomainContext()
			if err != nil {
				return err
			}
			domainID := c.Args().First()
			if domainID == "" {
				return utils.NewError("domain ID is required", nil)
			}
			if err := api.DeleteDomain(userID, orgID, domainID); err != nil {
				return utils.NewError(fmt.Sprintf("failed to delete domain: %s", err.Error()), nil)
			}
			utils.PrintSuccess("Domain deleted successfully")
			return nil
		},
	}
}

func domainVerifyCommand() *cli.Command {
	return &cli.Command{
		Name:      "verify",
		Usage:     "Verify nameserver configuration for a domain",
		ArgsUsage: "<domain-id>",
		Action: func(c *cli.Context) error {
			userID, orgID, err := requireDomainContext()
			if err != nil {
				return err
			}
			domainID := c.Args().First()
			if domainID == "" {
				return utils.NewError("domain ID is required", nil)
			}
			d, ns, err := api.VerifyDomain(userID, orgID, domainID)
			if err != nil {
				return utils.NewError(fmt.Sprintf("failed to verify domain: %s", err.Error()), nil)
			}
			utils.PrintHeader("Nameserver Verification: %s", d.Name)
			utils.PrintStatusLine("Domain Status", colorDomainStatus(d.Status))
			if ns.Verified {
				utils.PrintStatusLine("Nameservers", utils.SuccessColor("verified"))
			} else {
				utils.PrintStatusLine("Nameservers", utils.WarnColor("not verified"))
			}
			utils.PrintStatusLine("Message", ns.Message)
			if len(ns.CurrentNameservers) > 0 {
				utils.PrintStatusLine("Current NSs", strings.Join(ns.CurrentNameservers, ", "))
			}
			utils.PrintStatusLine("Expected NSs", strings.Join(ns.ExpectedNameservers, ", "))
			return nil
		},
	}
}

func domainCheckCommand() *cli.Command {
	return &cli.Command{
		Name:  "check",
		Usage: "Check domain availability (requires OpenProvider integration)",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "domains",
				Aliases:  []string{"d"},
				Usage:    "Comma-separated list of domains to check (e.g. example.com,example.io)",
				Required: true,
			},
			&cli.BoolFlag{
				Name:  "price",
				Usage: "Include pricing information",
			},
		},
		Action: func(c *cli.Context) error {
			userID, orgID, err := requireDomainContext()
			if err != nil {
				return err
			}
			rawDomains := c.String("domains")
			domains := strings.Split(rawDomains, ",")
			for i, d := range domains {
				domains[i] = strings.TrimSpace(d)
			}
			req := api.DomainCheckRequest{
				Domains:   domains,
				WithPrice: c.Bool("price"),
			}
			results, err := api.CheckDomainAvailability(userID, orgID, req)
			if err != nil {
				return utils.NewError(fmt.Sprintf("failed to check domain availability: %s", err.Error()), nil)
			}
			utils.PrintHeader("Domain Availability")
			for _, r := range results {
				avail := utils.ErrorColor("taken")
				if r.Available {
					avail = utils.SuccessColor("available")
				}
				line := fmt.Sprintf("%-40s %s", r.Domain, avail)
				if r.Price != nil {
					line += fmt.Sprintf("  $%.2f %s/yr", r.Price.Amount, r.Price.Currency)
				}
				if r.IsPremium {
					line += utils.WarnColor("  [premium]")
				}
				fmt.Println(line)
			}
			return nil
		},
	}
}

func domainSearchCommand() *cli.Command {
	return &cli.Command{
		Name:  "search",
		Usage: "Search domain availability across TLDs (requires OpenProvider integration)",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "name",
				Aliases:  []string{"n"},
				Usage:    "Base domain name to search (e.g. myapp)",
				Required: true,
			},
			&cli.StringFlag{
				Name:  "ext",
				Usage: "Comma-separated TLD extensions to search (e.g. .com,.io,.net); defaults to popular TLDs",
			},
			&cli.IntFlag{
				Name:  "period",
				Usage: "Registration period in years (1-10)",
				Value: 1,
			},
		},
		Action: func(c *cli.Context) error {
			userID, orgID, err := requireDomainContext()
			if err != nil {
				return err
			}
			req := api.DomainSearchRequest{
				DomainName: c.String("name"),
				Period:     c.Int("period"),
			}
			if ext := c.String("ext"); ext != "" {
				parts := strings.Split(ext, ",")
				for i, p := range parts {
					parts[i] = strings.TrimSpace(p)
				}
				req.Extensions = parts
			}
			results, err := api.SearchDomains(userID, orgID, req)
			if err != nil {
				return utils.NewError(fmt.Sprintf("failed to search domains: %s", err.Error()), nil)
			}
			if len(results) == 0 {
				utils.PrintInfo("No results found")
				return nil
			}
			utils.PrintHeader("Domain Search Results")
			for _, r := range results {
				avail := utils.ErrorColor("taken")
				if r.Available {
					avail = utils.SuccessColor("available")
				}
				line := fmt.Sprintf("%-40s %s", r.Domain, avail)
				if r.Available && r.Price > 0 {
					line += fmt.Sprintf("  $%.2f %s/yr", r.Price, r.Currency)
				}
				if r.IsPremium {
					line += utils.WarnColor("  [premium]")
				}
				fmt.Println(line)
			}
			return nil
		},
	}
}

func domainPurchaseCommand() *cli.Command {
	return &cli.Command{
		Name:  "purchase",
		Usage: "Initiate a domain purchase via Stripe Checkout",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "domain", Aliases: []string{"d"}, Usage: "Domain name to purchase", Required: true},
			&cli.IntFlag{Name: "period", Usage: "Registration period in years (1-10)", Value: 1},
			&cli.StringFlag{Name: "first-name", Usage: "Contact first name", Required: true},
			&cli.StringFlag{Name: "last-name", Usage: "Contact last name", Required: true},
			&cli.StringFlag{Name: "email", Usage: "Contact email", Required: true},
			&cli.StringFlag{Name: "phone-code", Usage: "Phone country code (e.g. +1)", Required: true},
			&cli.StringFlag{Name: "phone", Usage: "Phone number", Required: true},
			&cli.StringFlag{Name: "street", Usage: "Street address", Required: true},
			&cli.StringFlag{Name: "street-number", Usage: "Street number", Required: true},
			&cli.StringFlag{Name: "postal-code", Usage: "Postal code", Required: true},
			&cli.StringFlag{Name: "city", Usage: "City", Required: true},
			&cli.StringFlag{Name: "state", Usage: "State/province (optional)"},
			&cli.StringFlag{Name: "country", Usage: "Country code (ISO 3166-1 alpha-2, e.g. US)", Required: true},
			&cli.StringFlag{Name: "company", Usage: "Company name (optional)"},
		},
		Action: func(c *cli.Context) error {
			userID, orgID, err := requireDomainContext()
			if err != nil {
				return err
			}
			contact := &api.DomainContactInfo{
				FirstName:        c.String("first-name"),
				LastName:         c.String("last-name"),
				Email:            c.String("email"),
				PhoneCountryCode: c.String("phone-code"),
				PhoneNumber:      c.String("phone"),
				Street:           c.String("street"),
				StreetNumber:     c.String("street-number"),
				PostalCode:       c.String("postal-code"),
				City:             c.String("city"),
				State:            c.String("state"),
				Country:          c.String("country"),
				CompanyName:      c.String("company"),
			}
			req := api.DomainPurchaseRequest{
				Domain:  c.String("domain"),
				Period:  c.Int("period"),
				Contact: contact,
			}
			intent, err := api.InitiateDomainPurchase(userID, orgID, req)
			if err != nil {
				return utils.NewError(fmt.Sprintf("failed to initiate domain purchase: %s", err.Error()), nil)
			}
			utils.PrintSuccess("Domain purchase intent created")
			utils.PrintStatusLine("Domain", intent.Domain)
			utils.PrintStatusLine("Price", fmt.Sprintf("%.2f %s/yr", intent.Price, intent.Currency))
			utils.PrintStatusLine("Intent ID", intent.IntentID)
			utils.PrintDivider()
			utils.PrintInfo("Complete your purchase in the browser:")
			fmt.Println(intent.RedirectURL)
			utils.PrintDivider()
			utils.PrintInfo("Run '1ctl domain purchase-status %s' to check payment status", intent.IntentID)
			return nil
		},
	}
}

func domainPurchaseStatusCommand() *cli.Command {
	return &cli.Command{
		Name:      "purchase-status",
		Usage:     "Check the status of a domain purchase intent",
		ArgsUsage: "<intent-id>",
		Action: func(c *cli.Context) error {
			userID, orgID, err := requireDomainContext()
			if err != nil {
				return err
			}
			intentID := c.Args().First()
			if intentID == "" {
				return utils.NewError("intent ID is required", nil)
			}
			status, err := api.GetPurchaseIntentStatus(userID, orgID, intentID)
			if err != nil {
				return utils.NewError(fmt.Sprintf("failed to get purchase intent status: %s", err.Error()), nil)
			}
			utils.PrintHeader("Purchase Intent: %s", status.Domain)
			utils.PrintStatusLine("Intent ID", status.IntentID)
			utils.PrintStatusLine("Domain", status.Domain)
			utils.PrintStatusLine("Status", colorPurchaseStatus(status.Status))
			return nil
		},
	}
}

func domainContactCommand() *cli.Command {
	return &cli.Command{
		Name:  "contact",
		Usage: "Show the saved contact info from the last domain purchase",
		Action: func(c *cli.Context) error {
			userID, orgID, err := requireDomainContext()
			if err != nil {
				return err
			}
			contact, err := api.GetSavedContact(userID, orgID)
			if err != nil {
				return utils.NewError(fmt.Sprintf("failed to get saved contact: %s", err.Error()), nil)
			}
			if contact == nil || contact.Email == "" {
				utils.PrintInfo("No saved contact found")
				return nil
			}
			utils.PrintHeader("Saved Contact")
			utils.PrintStatusLine("Name", fmt.Sprintf("%s %s", contact.FirstName, contact.LastName))
			utils.PrintStatusLine("Email", contact.Email)
			utils.PrintStatusLine("Phone", fmt.Sprintf("%s %s", contact.PhoneCountryCode, contact.PhoneNumber))
			utils.PrintStatusLine("Address", fmt.Sprintf("%s %s, %s, %s %s", contact.Street, contact.StreetNumber, contact.City, contact.PostalCode, contact.Country))
			if contact.CompanyName != "" {
				utils.PrintStatusLine("Company", contact.CompanyName)
			}
			return nil
		},
	}
}

func domainDNSCommand() *cli.Command {
	return &cli.Command{
		Name:  "dns",
		Usage: "Manage DNS records for a domain",
		Subcommands: []*cli.Command{
			domainDNSListCommand(),
			domainDNSCreateCommand(),
			domainDNSUpdateCommand(),
			domainDNSDeleteCommand(),
		},
	}
}

func domainDNSListCommand() *cli.Command {
	return &cli.Command{
		Name:      "list",
		Usage:     "List DNS records for a domain",
		ArgsUsage: "<domain-id>",
		Action: func(c *cli.Context) error {
			userID, orgID, err := requireDomainContext()
			if err != nil {
				return err
			}
			domainID := c.Args().First()
			if domainID == "" {
				return utils.NewError("domain ID is required", nil)
			}
			records, err := api.ListDNSRecords(userID, orgID, domainID)
			if err != nil {
				return utils.NewError(fmt.Sprintf("failed to list DNS records: %s", err.Error()), nil)
			}
			if len(records) == 0 {
				utils.PrintInfo("No DNS records found")
				return nil
			}
			utils.PrintHeader("DNS Records (%d)", len(records))
			for _, r := range records {
				printDNSRecord(&r)
				utils.PrintDivider()
			}
			return nil
		},
	}
}

func domainDNSCreateCommand() *cli.Command {
	return &cli.Command{
		Name:      "create",
		Usage:     "Create a DNS record",
		ArgsUsage: "<domain-id>",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "type", Aliases: []string{"t"}, Usage: "Record type (A, AAAA, CNAME, MX, TXT, NS, SRV, CAA)", Required: true},
			&cli.StringFlag{Name: "name", Aliases: []string{"n"}, Usage: "Record name (e.g. @ or subdomain)", Required: true},
			&cli.StringFlag{Name: "data", Aliases: []string{"d"}, Usage: "Record value", Required: true},
			&cli.IntFlag{Name: "ttl", Usage: "TTL in seconds (default: 3600)", Value: 3600},
			&cli.IntFlag{Name: "priority", Usage: "Priority (MX/SRV records)"},
			&cli.IntFlag{Name: "port", Usage: "Port (SRV records)"},
			&cli.IntFlag{Name: "weight", Usage: "Weight (SRV records)"},
		},
		Action: func(c *cli.Context) error {
			userID, orgID, err := requireDomainContext()
			if err != nil {
				return err
			}
			domainID := c.Args().First()
			if domainID == "" {
				return utils.NewError("domain ID is required", nil)
			}
			ttl := c.Int("ttl")
			req := api.DNSRecordCreateRequest{
				Type: strings.ToUpper(c.String("type")),
				Name: c.String("name"),
				Data: c.String("data"),
				TTL:  &ttl,
			}
			if c.IsSet("priority") {
				v := c.Int("priority")
				req.Priority = &v
			}
			if c.IsSet("port") {
				v := c.Int("port")
				req.Port = &v
			}
			if c.IsSet("weight") {
				v := c.Int("weight")
				req.Weight = &v
			}
			record, err := api.CreateDNSRecord(userID, orgID, domainID, req)
			if err != nil {
				return utils.NewError(fmt.Sprintf("failed to create DNS record: %s", err.Error()), nil)
			}
			utils.PrintSuccess("DNS record created")
			printDNSRecord(record)
			return nil
		},
	}
}

func domainDNSUpdateCommand() *cli.Command {
	return &cli.Command{
		Name:      "update",
		Usage:     "Update a DNS record",
		ArgsUsage: "<domain-id> <record-id>",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "type", Aliases: []string{"t"}, Usage: "New record type"},
			&cli.StringFlag{Name: "name", Aliases: []string{"n"}, Usage: "New record name"},
			&cli.StringFlag{Name: "data", Aliases: []string{"d"}, Usage: "New record value"},
			&cli.IntFlag{Name: "ttl", Usage: "New TTL in seconds"},
			&cli.IntFlag{Name: "priority", Usage: "New priority"},
		},
		Action: func(c *cli.Context) error {
			userID, orgID, err := requireDomainContext()
			if err != nil {
				return err
			}
			args := c.Args()
			if args.Len() < 2 {
				return utils.NewError("domain ID and record ID are required", nil)
			}
			domainID := args.Get(0)
			recordID := args.Get(1)
			req := api.DNSRecordUpdateRequest{}
			if c.IsSet("type") {
				v := strings.ToUpper(c.String("type"))
				req.Type = &v
			}
			if c.IsSet("name") {
				v := c.String("name")
				req.Name = &v
			}
			if c.IsSet("data") {
				v := c.String("data")
				req.Data = &v
			}
			if c.IsSet("ttl") {
				v := c.Int("ttl")
				req.TTL = &v
			}
			if c.IsSet("priority") {
				v := c.Int("priority")
				req.Priority = &v
			}
			record, err := api.UpdateDNSRecord(userID, orgID, domainID, recordID, req)
			if err != nil {
				return utils.NewError(fmt.Sprintf("failed to update DNS record: %s", err.Error()), nil)
			}
			utils.PrintSuccess("DNS record updated")
			printDNSRecord(record)
			return nil
		},
	}
}

func domainDNSDeleteCommand() *cli.Command {
	return &cli.Command{
		Name:      "delete",
		Usage:     "Delete a DNS record",
		ArgsUsage: "<domain-id> <record-id>",
		Action: func(c *cli.Context) error {
			userID, orgID, err := requireDomainContext()
			if err != nil {
				return err
			}
			args := c.Args()
			if args.Len() < 2 {
				return utils.NewError("domain ID and record ID are required", nil)
			}
			if err := api.DeleteDNSRecord(userID, orgID, args.Get(0), args.Get(1)); err != nil {
				return utils.NewError(fmt.Sprintf("failed to delete DNS record: %s", err.Error()), nil)
			}
			utils.PrintSuccess("DNS record deleted")
			return nil
		},
	}
}

// --- display helpers ---

func printDomainSummary(d *api.Domain) {
	utils.PrintStatusLine("ID", d.DomainID)
	utils.PrintStatusLine("Name", d.Name)
	utils.PrintStatusLine("Status", colorDomainStatus(d.Status))
	utils.PrintStatusLine("TTL", fmt.Sprintf("%d", d.TTL))
}

func printDomainDetails(d *api.Domain) {
	utils.PrintStatusLine("Domain ID", d.DomainID)
	utils.PrintStatusLine("Name", d.Name)
	utils.PrintStatusLine("Status", colorDomainStatus(d.Status))
	utils.PrintStatusLine("TTL", fmt.Sprintf("%d", d.TTL))
	utils.PrintStatusLine("Created", d.CreatedAt.Format("2006-01-02 15:04:05"))
	utils.PrintStatusLine("Updated", d.UpdatedAt.Format("2006-01-02 15:04:05"))
}

func printDNSRecord(r *api.DNSRecord) {
	utils.PrintStatusLine("Record ID", r.RecordID)
	utils.PrintStatusLine("Type", r.Type)
	utils.PrintStatusLine("Name", r.Name)
	utils.PrintStatusLine("Data", r.Data)
	utils.PrintStatusLine("TTL", fmt.Sprintf("%d", r.TTL))
	if r.Priority != nil {
		utils.PrintStatusLine("Priority", fmt.Sprintf("%d", *r.Priority))
	}
	if r.Port != nil {
		utils.PrintStatusLine("Port", fmt.Sprintf("%d", *r.Port))
	}
	if r.Weight != nil {
		utils.PrintStatusLine("Weight", fmt.Sprintf("%d", *r.Weight))
	}
}

func colorDomainStatus(status string) string {
	switch status {
	case "active":
		return utils.SuccessColor(status)
	case "failed":
		return utils.ErrorColor(status)
	case "pending", "verifying":
		return utils.WarnColor(status)
	default:
		return status
	}
}

func colorPurchaseStatus(status string) string {
	switch status {
	case "completed":
		return utils.SuccessColor(status)
	case "failed", "canceled":
		return utils.ErrorColor(status)
	case "pending":
		return utils.WarnColor(status)
	default:
		return status
	}
}
