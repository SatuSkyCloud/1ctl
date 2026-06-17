package commands

import (
	"context"
	"fmt"
	"strings"

	"1ctl/internal/api"
	satuskyctx "1ctl/internal/context"
	"1ctl/internal/utils"

	"github.com/urfave/cli/v3"
)

func domainsAvailableCommand() *cli.Command {
	return &cli.Command{
		Name:      "available",
		Usage:     "Check domain registration availability",
		ArgsUsage: "<domain> [domain...]",
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "price", Usage: "Include registration pricing when available"},
		},
		Action: handleDomainsAvailable,
	}
}

func domainsSearchCommand() *cli.Command {
	return &cli.Command{
		Name:      "search",
		Usage:     "Search available domains across TLDs",
		ArgsUsage: "<name>",
		Flags: []cli.Flag{
			&cli.StringSliceFlag{Name: "tld", Usage: "TLD to search; can be repeated"},
			&cli.IntFlag{Name: "period", Usage: "Registration period in years", Value: 1},
		},
		Action: handleDomainsSearch,
	}
}

func domainsManagedCommand() *cli.Command {
	return &cli.Command{
		Name:  "managed",
		Usage: "Manage domains owned or delegated to SatuSky DNS",
		Commands: []*cli.Command{
			{
				Name:   "list",
				Usage:  "List managed domains",
				Action: handleManagedDomainsList,
			},
			{
				Name:      "add",
				Usage:     "Add a domain zone to SatuSky DNS",
				ArgsUsage: "<domain>",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "ip", Usage: "Optional default apex A record"},
				},
				Action: handleManagedDomainsAdd,
			},
			{
				Name:      "verify",
				Usage:     "Verify nameservers for a managed domain",
				ArgsUsage: "<domain|domain-id>",
				Action:    handleManagedDomainsVerify,
			},
			{
				Name:      "delete",
				Aliases:   []string{"rm"},
				Usage:     "Delete a managed domain zone",
				ArgsUsage: "<domain|domain-id>",
				Flags: []cli.Flag{
					&cli.BoolFlag{Name: "yes", Aliases: []string{"y"}, Usage: "Skip confirmation prompt"},
				},
				Action: handleManagedDomainsDelete,
			},
		},
	}
}

func domainsDNSCommand() *cli.Command {
	recordFlags := []cli.Flag{
		&cli.StringFlag{Name: "type", Usage: "DNS record type (A, AAAA, CNAME, MX, TXT, NS, SRV, CAA) — required"},
		&cli.StringFlag{Name: "name", Usage: "Record name, such as @, www, or app — required"},
		&cli.StringFlag{Name: "data", Usage: "Record value — required"},
		&cli.IntFlag{Name: "ttl", Usage: "Record TTL in seconds (min 600)", DefaultText: "600"},
		&cli.IntFlag{Name: "priority", Usage: "Record priority"},
		&cli.IntFlag{Name: "port", Usage: "SRV record port"},
		&cli.IntFlag{Name: "weight", Usage: "SRV record weight"},
		&cli.IntFlag{Name: "flags", Usage: "CAA record flags"},
		&cli.StringFlag{Name: "tag", Usage: "CAA record tag"},
	}
	updateFlags := []cli.Flag{
		&cli.StringFlag{Name: "type", Usage: "DNS record type (A, AAAA, CNAME, MX, TXT, NS, SRV, CAA)"},
		&cli.StringFlag{Name: "name", Usage: "Record name, such as @, www, or app"},
		&cli.StringFlag{Name: "data", Usage: "Record value"},
		&cli.IntFlag{Name: "ttl", Usage: "Record TTL in seconds (min 600)"},
		&cli.IntFlag{Name: "priority", Usage: "Record priority"},
		&cli.IntFlag{Name: "port", Usage: "SRV record port"},
		&cli.IntFlag{Name: "weight", Usage: "SRV record weight"},
		&cli.IntFlag{Name: "flags", Usage: "CAA record flags"},
		&cli.StringFlag{Name: "tag", Usage: "CAA record tag"},
	}
	return &cli.Command{
		Name:  "dns",
		Usage: "Manage DNS records for SatuSky-managed domains",
		Commands: []*cli.Command{
			{
				Name:      "list",
				Usage:     "List DNS records",
				ArgsUsage: "<domain|domain-id>",
				Action:    handleDNSList,
			},
			{
				Name:      "create",
				Aliases:   []string{"add"},
				Usage:     "Create a DNS record",
				ArgsUsage: "<domain|domain-id>",
				Flags:     recordFlags,
				Action:    handleDNSCreate,
			},
			{
				Name:      "update",
				Usage:     "Update a DNS record",
				ArgsUsage: "<domain|domain-id> <record-id>",
				Arguments: []cli.Argument{
					&cli.StringArgs{Name: "domain", Min: 1, Max: 1},
					&cli.StringArgs{Name: "record-id", Min: 1, Max: 1},
				},
				Flags:     updateFlags,
				Action:    handleDNSUpdate,
			},
			{
				Name:      "delete",
				Aliases:   []string{"rm"},
				Usage:     "Delete a DNS record",
				ArgsUsage: "<domain|domain-id> [<record-id>]",
				Flags: []cli.Flag{
					&cli.BoolFlag{Name: "yes", Aliases: []string{"y"}, Usage: "Skip confirmation prompt"},
					&cli.StringFlag{Name: "record-id", Usage: "DNS record ID (alternative to positional arg)"},
				},
				Action: handleDNSDelete,
			},
		},
	}
}

func domainsPurchaseCommand() *cli.Command {
	return &cli.Command{
		Name:      "purchase",
		Usage:     "Purchase a domain using your credits balance",
		ArgsUsage: "<domain>",
		Flags: []cli.Flag{
			&cli.IntFlag{Name: "period", Usage: "Registration period in years", Value: 1},
			&cli.StringFlag{Name: "first-name", Usage: "Registrant first name", Required: true},
			&cli.StringFlag{Name: "last-name", Usage: "Registrant last name", Required: true},
			&cli.StringFlag{Name: "email", Usage: "Registrant email address", Required: true},
			&cli.StringFlag{Name: "phone-country-code", Usage: "Phone country code (e.g. +60)", Required: true},
			&cli.StringFlag{Name: "phone-number", Usage: "Phone subscriber number", Required: true},
			&cli.StringFlag{Name: "street", Usage: "Street name", Required: true},
			&cli.StringFlag{Name: "street-number", Usage: "Street or building number", Required: true},
			&cli.StringFlag{Name: "postal-code", Usage: "Postal code", Required: true},
			&cli.StringFlag{Name: "city", Usage: "City", Required: true},
			&cli.StringFlag{Name: "state", Usage: "State or province (optional for some countries)"},
			&cli.StringFlag{Name: "country", Usage: "Two-letter ISO country code (e.g. MY, US)", Required: true},
			&cli.StringFlag{Name: "company", Usage: "Company or organization name (optional)"},
		},
		Action: handleDomainsPurchase,
	}
}

func domainsPurchaseStatusCommand() *cli.Command {
	return &cli.Command{
		Name:      "purchase-status",
		Usage:     "Check a domain purchase intent",
		ArgsUsage: "<intent-id>",
		Action:    handleDomainsPurchaseStatus,
	}
}

func handleDomainsAvailable(ctx context.Context, cmd *cli.Command) error {
	if cmd.NArg() == 0 {
		return utils.NewError("at least one domain is required", nil)
	}
	userID, orgID, err := domainAPIScope()
	if err != nil {
		return err
	}
	domains := make([]string, 0, cmd.NArg())
	for i := 0; i < cmd.NArg(); i++ {
		domain, err := normalizeDomainArg(cmd.Args().Get(i))
		if err != nil {
			return err
		}
		domains = append(domains, domain)
	}
	results, err := api.CheckDomainAvailability(userID, orgID, api.DomainCheckRequest{Domains: domains, WithPrice: cmd.Bool("price")})
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to check availability: %s", err.Error()), nil)
	}
	if utils.TryPrintJSON(results) {
		return nil
	}
	rows := make([][]string, 0, len(results))
	for _, r := range results {
		price := ""
		if r.Price != nil {
			price = fmt.Sprintf("%.2f %s", r.Price.Amount, r.Price.Currency)
		}
		rows = append(rows, []string{r.Domain, boolText(r.Available), r.Status, price})
	}
	utils.PrintTable([]string{"DOMAIN", "AVAILABLE", "STATUS", "PRICE"}, rows)
	return nil
}

func handleDomainsSearch(ctx context.Context, cmd *cli.Command) error {
	if cmd.NArg() < 1 {
		return utils.NewError("name is required. Usage: 1ctl domains search <name>", nil)
	}
	userID, orgID, err := domainAPIScope()
	if err != nil {
		return err
	}
	results, err := api.SearchDomains(userID, orgID, api.DomainSearchRequest{
		DomainName: cmd.Args().First(),
		Extensions: cmd.StringSlice("tld"),
		Period:     cmd.Int("period"),
	})
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to search domains: %s", err.Error()), nil)
	}
	if utils.TryPrintJSON(results) {
		return nil
	}
	rows := make([][]string, 0, len(results))
	for _, r := range results {
		rows = append(rows, []string{
			r.Domain,
			boolText(r.Available),
			r.Status,
			fmt.Sprintf("%.2f %s", r.Price, r.Currency),
		})
	}
	utils.PrintTable([]string{"DOMAIN", "AVAILABLE", "STATUS", "PRICE"}, rows)
	return nil
}

func handleManagedDomainsList(ctx context.Context, cmd *cli.Command) error {
	userID, orgID, err := domainAPIScope()
	if err != nil {
		return err
	}
	domains, err := api.ListManagedDomains(userID, orgID)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to list managed domains: %s", err.Error()), nil)
	}
	if utils.TryPrintJSON(domains) {
		return nil
	}
	printManagedDomains(domains)
	return nil
}

func handleManagedDomainsAdd(ctx context.Context, cmd *cli.Command) error {
	if cmd.NArg() < 1 {
		return utils.NewError("domain is required. Usage: 1ctl domains managed add <domain>", nil)
	}
	userID, orgID, err := domainAPIScope()
	if err != nil {
		return err
	}
	domain, err := normalizeDomainArg(cmd.Args().First())
	if err != nil {
		return err
	}
	created, ns, err := api.CreateManagedDomain(userID, orgID, api.DomainCreateRequest{Name: domain, IPAddress: cmd.String("ip")})
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to add managed domain: %s", err.Error()), nil)
	}
	if utils.TryPrintJSON(map[string]interface{}{"domain": created, "nameserver_status": ns}) {
		return nil
	}
	utils.PrintSuccess("Managed domain %s added", created.Name)
	printManagedDomain(*created)
	printNameserverStatus(ns)
	return nil
}

func handleManagedDomainsVerify(ctx context.Context, cmd *cli.Command) error {
	if cmd.NArg() < 1 {
		return utils.NewError("domain is required. Usage: 1ctl domains managed verify <domain|domain-id>", nil)
	}
	userID, orgID, err := domainAPIScope()
	if err != nil {
		return err
	}
	domainID, err := resolveManagedDomainID(userID, orgID, cmd.Args().First())
	if err != nil {
		return err
	}
	domain, ns, err := api.VerifyManagedDomain(userID, orgID, domainID)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to verify managed domain: %s", err.Error()), nil)
	}
	if utils.TryPrintJSON(map[string]interface{}{"domain": domain, "nameserver_status": ns}) {
		return nil
	}
	printManagedDomain(*domain)
	printNameserverStatus(ns)
	return nil
}

func handleManagedDomainsDelete(ctx context.Context, cmd *cli.Command) error {
	if cmd.NArg() < 1 {
		return utils.NewError("domain is required. Usage: 1ctl domains managed delete <domain|domain-id>", nil)
	}
	userID, orgID, err := domainAPIScope()
	if err != nil {
		return err
	}
	domainID, err := resolveManagedDomainID(userID, orgID, cmd.Args().First())
	if err != nil {
		return err
	}
	if !utils.Confirm(fmt.Sprintf("Delete managed domain %s?", cmd.Args().First()), cmd.Bool("yes")) {
		fmt.Println("Aborted.")
		return nil
	}
	if err := api.DeleteManagedDomain(userID, orgID, domainID); err != nil {
		return utils.NewError(fmt.Sprintf("failed to delete managed domain: %s", err.Error()), nil)
	}
	utils.PrintSuccess("Managed domain deleted")
	return nil
}

func handleDNSList(ctx context.Context, cmd *cli.Command) error {
	if cmd.NArg() < 1 {
		return utils.NewError("domain is required. Usage: 1ctl domains dns list <domain|domain-id>", nil)
	}
	userID, orgID, err := domainAPIScope()
	if err != nil {
		return err
	}
	domainID, err := resolveManagedDomainID(userID, orgID, cmd.Args().First())
	if err != nil {
		return err
	}
	records, err := api.ListDNSRecords(userID, orgID, domainID)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to list DNS records: %s", err.Error()), nil)
	}
	if utils.TryPrintJSON(records) {
		return nil
	}
	printDNSRecords(records)
	return nil
}

func handleDNSCreate(ctx context.Context, cmd *cli.Command) error {
	if cmd.NArg() < 1 {
		return utils.NewError("domain is required. Usage: 1ctl domains dns create <domain|domain-id>", nil)
	}
	// In v3, flags can appear anywhere in the argument list, so cmd.String()
	// works for flags after positional args.
	if cmd.String("type") == "" {
		return utils.NewError("--type is required. Usage: 1ctl domains dns create <domain> --type A --name @ --data <value>", nil)
	}
	if cmd.String("name") == "" {
		return utils.NewError("--name is required. Usage: 1ctl domains dns create <domain> --type A --name @ --data <value>", nil)
	}
	if cmd.String("data") == "" {
		return utils.NewError("--data is required. Usage: 1ctl domains dns create <domain> --type A --name @ --data <value>", nil)
	}
	userID, orgID, err := domainAPIScope()
	if err != nil {
		return err
	}
	domainID, err := resolveManagedDomainID(userID, orgID, cmd.Args().First())
	if err != nil {
		return err
	}
	record, err := api.CreateDNSRecord(userID, orgID, domainID, dnsCreateRequestFromFlags(cmd))
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to create DNS record: %s", err.Error()), nil)
	}
	if utils.TryPrintJSON(record) {
		return nil
	}
	utils.PrintSuccess("DNS record created")
	printDNSRecords([]api.DNSRecord{*record})
	return nil
}

func handleDNSUpdate(ctx context.Context, cmd *cli.Command) error {
	userID, orgID, err := domainAPIScope()
	if err != nil {
		return err
	}
	domain := cmd.StringArgs("domain")
	recordID := cmd.StringArgs("record-id")
	domainID, err := resolveManagedDomainID(userID, orgID, domain[0])
	if err != nil {
		return err
	}
	record, err := api.UpdateDNSRecord(userID, orgID, domainID, recordID[0], dnsUpdateRequestFromFlags(cmd))
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to update DNS record: %s", err.Error()), nil)
	}
	if utils.TryPrintJSON(record) {
		return nil
	}
	utils.PrintSuccess("DNS record updated")
	printDNSRecords([]api.DNSRecord{*record})
	return nil
}

func handleDNSDelete(ctx context.Context, cmd *cli.Command) error {
	userID, orgID, err := domainAPIScope()
	if err != nil {
		return err
	}
	domain := cmd.StringArgs("domain")
	domainID, err := resolveManagedDomainID(userID, orgID, domain[0])
	if err != nil {
		return err
	}
	recordID := cmd.String("record-id")
	if recordID == "" {
		posRecord := cmd.StringArgs("record-id")
		if len(posRecord) == 0 {
			return utils.NewError("record ID is required. Usage: 1ctl domains dns delete <domain|domain-id> [<record-id>] [--record-id <id>]", nil)
		}
		recordID = posRecord[0]
	}
	if !utils.Confirm(fmt.Sprintf("Delete DNS record %s?", recordID), cmd.Bool("yes")) {
		fmt.Println("Aborted.")
		return nil
	}
	if err := api.DeleteDNSRecord(userID, orgID, domainID, recordID); err != nil {
		return utils.NewError(fmt.Sprintf("failed to delete DNS record: %s", err.Error()), nil)
	}
	utils.PrintSuccess("DNS record deleted")
	return nil
}

func handleDomainsPurchase(ctx context.Context, cmd *cli.Command) error {
	if cmd.NArg() < 1 {
		return utils.NewError("domain is required. Usage: 1ctl domains purchase <domain>", nil)
	}
	userID, orgID, err := domainAPIScope()
	if err != nil {
		return err
	}
	domain, err := normalizeDomainArg(cmd.Args().First())
	if err != nil {
		return err
	}
	resp, err := api.PurchaseDomain(userID, orgID, api.DomainPurchaseRequest{
		Domain: domain,
		Period: cmd.Int("period"),
		Contact: &api.DomainContactInfo{
			FirstName:        cmd.String("first-name"),
			LastName:         cmd.String("last-name"),
			Email:            cmd.String("email"),
			PhoneCountryCode: cmd.String("phone-country-code"),
			PhoneNumber:      cmd.String("phone-number"),
			Street:           cmd.String("street"),
			StreetNumber:     cmd.String("street-number"),
			PostalCode:       cmd.String("postal-code"),
			City:             cmd.String("city"),
			State:            cmd.String("state"),
			Country:          strings.ToUpper(cmd.String("country")),
			CompanyName:      cmd.String("company"),
		},
	})
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to start domain purchase: %s", err.Error()), nil)
	}
	if utils.TryPrintJSON(resp) {
		return nil
	}
	utils.PrintSuccess("Domain purchase checkout created")
	utils.PrintStatusLine("Domain", resp.Domain)
	utils.PrintStatusLine("Price", fmt.Sprintf("%.2f %s", resp.Price, resp.Currency))
	utils.PrintStatusLine("Intent ID", resp.IntentID)
	utils.PrintStatusLine("Checkout", resp.RedirectURL)
	return nil
}

func handleDomainsPurchaseStatus(ctx context.Context, cmd *cli.Command) error {
	if cmd.NArg() < 1 {
		return utils.NewError("intent ID is required. Usage: 1ctl domains purchase-status <intent-id>", nil)
	}
	userID, orgID, err := domainAPIScope()
	if err != nil {
		return err
	}
	status, err := api.GetDomainPurchaseStatus(userID, orgID, cmd.Args().First())
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to get purchase status: %s", err.Error()), nil)
	}
	if utils.TryPrintJSON(status) {
		return nil
	}
	utils.PrintStatusLine("Intent ID", status.IntentID)
	utils.PrintStatusLine("Domain", status.Domain)
	utils.PrintStatusLine("Status", status.Status)
	return nil
}

func domainAPIScope() (userID, orgID string, err error) {
	userID = strings.TrimSpace(satuskyctx.GetUserID())
	orgID = strings.TrimSpace(satuskyctx.GetCurrentOrgID())
	if userID == "" {
		return "", "", utils.NewError("no current user ID is set. Run: 1ctl auth login", nil)
	}
	if orgID == "" {
		return "", "", utils.NewError("no current organization ID is set. Run: 1ctl org switch --id <org-id>", nil)
	}
	return userID, orgID, nil
}

func resolveManagedDomainID(userID, orgID, value string) (string, error) {
	if _, err := api.ParseUUID(value); err == nil {
		return value, nil
	}
	name, err := normalizeDomainArg(value)
	if err != nil {
		return "", err
	}
	domains, err := api.ListManagedDomains(userID, orgID)
	if err != nil {
		return "", utils.NewError(fmt.Sprintf("failed to resolve managed domain: %s", err.Error()), nil)
	}
	for _, domain := range domains {
		if strings.EqualFold(domain.Name, name) {
			return domain.DomainID, nil
		}
	}
	return "", utils.NewError(fmt.Sprintf("managed domain %q not found", value), nil)
}

func dnsCreateRequestFromFlags(cmd *cli.Command) api.DNSRecordCreateRequest {
	req := api.DNSRecordCreateRequest{
		Type: strings.ToUpper(cmd.String("type")),
		Name: cmd.String("name"),
		Data: cmd.String("data"),
	}
	if cmd.IsSet("ttl") {
		v := cmd.Int("ttl")
		req.TTL = &v
	}
	if cmd.IsSet("priority") {
		v := cmd.Int("priority")
		req.Priority = &v
	}
	if cmd.IsSet("port") {
		v := cmd.Int("port")
		req.Port = &v
	}
	if cmd.IsSet("weight") {
		v := cmd.Int("weight")
		req.Weight = &v
	}
	if cmd.IsSet("flags") {
		v := cmd.Int("flags")
		req.Flags = &v
	}
	if cmd.IsSet("tag") {
		v := cmd.String("tag")
		req.Tag = &v
	}
	return req
}

func dnsUpdateRequestFromFlags(cmd *cli.Command) api.DNSRecordUpdateRequest {
	req := api.DNSRecordUpdateRequest{}
	if cmd.IsSet("type") {
		v := strings.ToUpper(cmd.String("type"))
		req.Type = &v
	}
	if cmd.IsSet("name") {
		v := cmd.String("name")
		req.Name = &v
	}
	if cmd.IsSet("data") {
		v := cmd.String("data")
		req.Data = &v
	}
	if cmd.IsSet("ttl") {
		v := cmd.Int("ttl")
		req.TTL = &v
	}
	if cmd.IsSet("priority") {
		v := cmd.Int("priority")
		req.Priority = &v
	}
	if cmd.IsSet("port") {
		v := cmd.Int("port")
		req.Port = &v
	}
	if cmd.IsSet("weight") {
		v := cmd.Int("weight")
		req.Weight = &v
	}
	if cmd.IsSet("flags") {
		v := cmd.Int("flags")
		req.Flags = &v
	}
	if cmd.IsSet("tag") {
		v := cmd.String("tag")
		req.Tag = &v
	}
	return req
}

func printManagedDomains(domains []api.Domain) {
	rows := make([][]string, 0, len(domains))
	for _, domain := range domains {
		rows = append(rows, []string{
			domain.Name,
			domain.DomainID,
			domain.Status,
			fmt.Sprintf("%d", domain.TTL),
			api.FormatTimeAgo(domain.CreatedAt),
		})
	}
	utils.PrintTable([]string{"DOMAIN", "ID", "STATUS", "TTL", "CREATED"}, rows)
}

func printManagedDomain(domain api.Domain) {
	utils.PrintStatusLine("Domain", domain.Name)
	utils.PrintStatusLine("Domain ID", domain.DomainID)
	utils.PrintStatusLine("Status", domain.Status)
	utils.PrintStatusLine("TTL", fmt.Sprintf("%d", domain.TTL))
}

func printNameserverStatus(status *api.NameserverStatus) {
	if status == nil {
		return
	}
	utils.PrintStatusLine("Nameservers", boolText(status.Verified))
	if len(status.CurrentNameservers) > 0 {
		utils.PrintStatusLine("Current NS", strings.Join(status.CurrentNameservers, ", "))
	}
	if len(status.ExpectedNameservers) > 0 {
		utils.PrintStatusLine("Expected NS", strings.Join(status.ExpectedNameservers, ", "))
	}
	if status.Message != "" {
		utils.PrintStatusLine("Message", status.Message)
	}
}

func printDNSRecords(records []api.DNSRecord) {
	rows := make([][]string, 0, len(records))
	for _, r := range records {
		name := r.Name
		if name == "" {
			name = "@"
		}
		rows = append(rows, []string{r.RecordID, r.Type, name, r.Data, fmt.Sprintf("%d", r.TTL)})
	}
	utils.PrintTable([]string{"ID", "TYPE", "NAME", "DATA", "TTL"}, rows)
}

func boolText(v bool) string {
	if v {
		return "yes"
	}
	return "no"
}
