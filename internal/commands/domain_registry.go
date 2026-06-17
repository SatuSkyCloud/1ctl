package commands

import (
	"fmt"
	"strconv"
	"strings"

	"1ctl/internal/api"
	"1ctl/internal/context"
	"1ctl/internal/utils"

	"github.com/urfave/cli/v2"
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
		Subcommands: []*cli.Command{
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
		Subcommands: []*cli.Command{
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

func handleDomainsAvailable(c *cli.Context) error {
	if c.NArg() == 0 {
		return utils.NewError("at least one domain is required", nil)
	}
	userID, orgID, err := domainAPIScope()
	if err != nil {
		return err
	}
	domains := make([]string, 0, c.NArg())
	for i := 0; i < c.NArg(); i++ {
		domain, err := normalizeDomainArg(c.Args().Get(i))
		if err != nil {
			return err
		}
		domains = append(domains, domain)
	}
	results, err := api.CheckDomainAvailability(userID, orgID, api.DomainCheckRequest{Domains: domains, WithPrice: c.Bool("price")})
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

func handleDomainsSearch(c *cli.Context) error {
	if c.NArg() < 1 {
		return utils.NewError("name is required. Usage: 1ctl domains search <name>", nil)
	}
	userID, orgID, err := domainAPIScope()
	if err != nil {
		return err
	}
	results, err := api.SearchDomains(userID, orgID, api.DomainSearchRequest{
		DomainName: c.Args().First(),
		Extensions: c.StringSlice("tld"),
		Period:     c.Int("period"),
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

func handleManagedDomainsList(c *cli.Context) error {
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

func handleManagedDomainsAdd(c *cli.Context) error {
	if c.NArg() < 1 {
		return utils.NewError("domain is required. Usage: 1ctl domains managed add <domain>", nil)
	}
	userID, orgID, err := domainAPIScope()
	if err != nil {
		return err
	}
	domain, err := normalizeDomainArg(c.Args().First())
	if err != nil {
		return err
	}
	created, ns, err := api.CreateManagedDomain(userID, orgID, api.DomainCreateRequest{Name: domain, IPAddress: c.String("ip")})
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

func handleManagedDomainsVerify(c *cli.Context) error {
	if c.NArg() < 1 {
		return utils.NewError("domain is required. Usage: 1ctl domains managed verify <domain|domain-id>", nil)
	}
	userID, orgID, err := domainAPIScope()
	if err != nil {
		return err
	}
	domainID, err := resolveManagedDomainID(userID, orgID, c.Args().First())
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

func handleManagedDomainsDelete(c *cli.Context) error {
	if c.NArg() < 1 {
		return utils.NewError("domain is required. Usage: 1ctl domains managed delete <domain|domain-id>", nil)
	}
	userID, orgID, err := domainAPIScope()
	if err != nil {
		return err
	}
	domainID, err := resolveManagedDomainID(userID, orgID, c.Args().First())
	if err != nil {
		return err
	}
	if !utils.Confirm(fmt.Sprintf("Delete managed domain %s?", c.Args().First()), c.Bool("yes")) {
		fmt.Println("Aborted.")
		return nil
	}
	if err := api.DeleteManagedDomain(userID, orgID, domainID); err != nil {
		return utils.NewError(fmt.Sprintf("failed to delete managed domain: %s", err.Error()), nil)
	}
	utils.PrintSuccess("Managed domain deleted")
	return nil
}

func handleDNSList(c *cli.Context) error {
	if c.NArg() < 1 {
		return utils.NewError("domain is required. Usage: 1ctl domains dns list <domain|domain-id>", nil)
	}
	userID, orgID, err := domainAPIScope()
	if err != nil {
		return err
	}
	domainID, err := resolveManagedDomainID(userID, orgID, c.Args().First())
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

func handleDNSCreate(c *cli.Context) error {
	if c.NArg() < 1 {
		return utils.NewError("domain is required. Usage: 1ctl domains dns create <domain|domain-id>", nil)
	}
	// Manually validate required flags (handles flags after positional args that
	// Go's flag.FlagSet doesn't parse)
	if flagValueFromArgs(c, "type") == "" {
		return utils.NewError("--type is required. Usage: 1ctl domains dns create <domain> --type A --name @ --data <value>", nil)
	}
	if flagValueFromArgs(c, "name") == "" {
		return utils.NewError("--name is required. Usage: 1ctl domains dns create <domain> --type A --name @ --data <value>", nil)
	}
	if flagValueFromArgs(c, "data") == "" {
		return utils.NewError("--data is required. Usage: 1ctl domains dns create <domain> --type A --name @ --data <value>", nil)
	}
	userID, orgID, err := domainAPIScope()
	if err != nil {
		return err
	}
	domainID, err := resolveManagedDomainID(userID, orgID, c.Args().First())
	if err != nil {
		return err
	}
	record, err := api.CreateDNSRecord(userID, orgID, domainID, dnsCreateRequestFromFlags(c))
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

func handleDNSUpdate(c *cli.Context) error {
	if c.NArg() < 2 {
		return utils.NewError("domain and record ID are required. Usage: 1ctl domains dns update <domain|domain-id> <record-id>", nil)
	}
	userID, orgID, err := domainAPIScope()
	if err != nil {
		return err
	}
	domainID, err := resolveManagedDomainID(userID, orgID, c.Args().First())
	if err != nil {
		return err
	}
	record, err := api.UpdateDNSRecord(userID, orgID, domainID, c.Args().Get(1), dnsUpdateRequestFromFlags(c))
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

func handleDNSDelete(c *cli.Context) error {
	if c.NArg() < 1 {
		return utils.NewError("domain is required. Usage: 1ctl domains dns delete <domain|domain-id> [<record-id>] [--record-id <id>]", nil)
	}
	userID, orgID, err := domainAPIScope()
	if err != nil {
		return err
	}
	domainID, err := resolveManagedDomainID(userID, orgID, c.Args().First())
	if err != nil {
		return err
	}
	recordID := flagValueFromArgs(c, "record-id")
	if recordID == "" {
		if c.NArg() < 2 {
			return utils.NewError("record ID is required. Usage: 1ctl domains dns delete <domain|domain-id> [<record-id>] [--record-id <id>]", nil)
		}
		recordID = c.Args().Get(1)
	}
	if !utils.Confirm(fmt.Sprintf("Delete DNS record %s?", recordID), c.Bool("yes")) {
		fmt.Println("Aborted.")
		return nil
	}
	if err := api.DeleteDNSRecord(userID, orgID, domainID, recordID); err != nil {
		return utils.NewError(fmt.Sprintf("failed to delete DNS record: %s", err.Error()), nil)
	}
	utils.PrintSuccess("DNS record deleted")
	return nil
}

func handleDomainsPurchase(c *cli.Context) error {
	if c.NArg() < 1 {
		return utils.NewError("domain is required. Usage: 1ctl domains purchase <domain>", nil)
	}
	userID, orgID, err := domainAPIScope()
	if err != nil {
		return err
	}
	domain, err := normalizeDomainArg(c.Args().First())
	if err != nil {
		return err
	}
	resp, err := api.PurchaseDomain(userID, orgID, api.DomainPurchaseRequest{
		Domain: domain,
		Period: c.Int("period"),
		Contact: &api.DomainContactInfo{
			FirstName:        c.String("first-name"),
			LastName:         c.String("last-name"),
			Email:            c.String("email"),
			PhoneCountryCode: c.String("phone-country-code"),
			PhoneNumber:      c.String("phone-number"),
			Street:           c.String("street"),
			StreetNumber:     c.String("street-number"),
			PostalCode:       c.String("postal-code"),
			City:             c.String("city"),
			State:            c.String("state"),
			Country:          strings.ToUpper(c.String("country")),
			CompanyName:      c.String("company"),
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

func handleDomainsPurchaseStatus(c *cli.Context) error {
	if c.NArg() < 1 {
		return utils.NewError("intent ID is required. Usage: 1ctl domains purchase-status <intent-id>", nil)
	}
	userID, orgID, err := domainAPIScope()
	if err != nil {
		return err
	}
	status, err := api.GetDomainPurchaseStatus(userID, orgID, c.Args().First())
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
	userID = strings.TrimSpace(context.GetUserID())
	orgID = strings.TrimSpace(context.GetCurrentOrgID())
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

// flagValueFromArgs extracts a flag's value from the full args slice (including flags
// that appear after positional args, which Go's flag.FlagSet.Parse() doesn't parse).
func flagValueFromArgs(c *cli.Context, name string) string {
	if c.IsSet(name) {
		return c.String(name)
	}
	args := c.Args().Slice()
	for i := 0; i < len(args); i++ {
		if args[i] == "--"+name {
			if i+1 < len(args) {
				return args[i+1]
			}
		}
		if strings.HasPrefix(args[i], "--"+name+"=") {
			return strings.SplitN(args[i], "=", 2)[1]
		}
	}
	return ""
}

// flagIsSetInArgs checks whether a flag was set, scanning both parsed flags
// and any flags that appear after positional arguments.
func flagIsSetInArgs(c *cli.Context, name string) bool {
	if c.IsSet(name) {
		return true
	}
	args := c.Args().Slice()
	for _, arg := range args {
		if arg == "--"+name || strings.HasPrefix(arg, "--"+name+"=") {
			return true
		}
	}
	return false
}

// flagIntFromArgs extracts an int flag value from full args, falling back to c.Int().
func flagIntFromArgs(c *cli.Context, name string) int {
	if c.IsSet(name) {
		return c.Int(name)
	}
	v := flagValueFromArgs(c, name)
	if v == "" {
		return 0
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0
	}
	return n
}

func dnsCreateRequestFromFlags(c *cli.Context) api.DNSRecordCreateRequest {
	req := api.DNSRecordCreateRequest{
		Type: strings.ToUpper(flagValueFromArgs(c, "type")),
		Name: flagValueFromArgs(c, "name"),
		Data: flagValueFromArgs(c, "data"),
	}
	if flagIsSetInArgs(c, "ttl") {
		v := flagIntFromArgs(c, "ttl")
		req.TTL = &v
	}
	if flagIsSetInArgs(c, "priority") {
		v := flagIntFromArgs(c, "priority")
		req.Priority = &v
	}
	if flagIsSetInArgs(c, "port") {
		v := flagIntFromArgs(c, "port")
		req.Port = &v
	}
	if flagIsSetInArgs(c, "weight") {
		v := flagIntFromArgs(c, "weight")
		req.Weight = &v
	}
	if flagIsSetInArgs(c, "flags") {
		v := flagIntFromArgs(c, "flags")
		req.Flags = &v
	}
	if flagIsSetInArgs(c, "tag") {
		v := flagValueFromArgs(c, "tag")
		req.Tag = &v
	}
	return req
}

func dnsUpdateRequestFromFlags(c *cli.Context) api.DNSRecordUpdateRequest {
	req := api.DNSRecordUpdateRequest{}
	if flagIsSetInArgs(c, "type") {
		v := strings.ToUpper(flagValueFromArgs(c, "type"))
		req.Type = &v
	}
	if flagIsSetInArgs(c, "name") {
		v := flagValueFromArgs(c, "name")
		req.Name = &v
	}
	if flagIsSetInArgs(c, "data") {
		v := flagValueFromArgs(c, "data")
		req.Data = &v
	}
	if flagIsSetInArgs(c, "ttl") {
		v := flagIntFromArgs(c, "ttl")
		req.TTL = &v
	}
	if flagIsSetInArgs(c, "priority") {
		v := flagIntFromArgs(c, "priority")
		req.Priority = &v
	}
	if flagIsSetInArgs(c, "port") {
		v := flagIntFromArgs(c, "port")
		req.Port = &v
	}
	if flagIsSetInArgs(c, "weight") {
		v := flagIntFromArgs(c, "weight")
		req.Weight = &v
	}
	if flagIsSetInArgs(c, "flags") {
		v := flagIntFromArgs(c, "flags")
		req.Flags = &v
	}
	if flagIsSetInArgs(c, "tag") {
		v := flagValueFromArgs(c, "tag")
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
