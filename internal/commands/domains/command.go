// Package domains defines the "1ctl domains" command tree — flag names,
// input structs, and CLI wiring.  Handler logic lives in handlers.go.
package domains

import (
	"context"

	"github.com/urfave/cli/v3"
)

// --- Flag name constants ------------------------------------------------

const (
	flagApp          = "app"
	flagPort         = "port"
	flagCustomDNS    = "custom-dns"
	flagWait         = "wait"
	flagNoWait       = "no-wait"
	flagWithWWW      = "with-www"
	flagYes          = "yes"
	flagProbe        = "probe"
	flagDomain       = "domain"
	flagPrice        = "price"
	flagName         = "name"
	flagTLD          = "tld"
	flagPeriod       = "period"
	flagIP           = "ip"
	flagType         = "type"
	flagData         = "data"
	flagTTL          = "ttl"
	flagPriority     = "priority"
	flagWeight       = "weight"
	flagFlags        = "flags"
	flagTag          = "tag"
	flagRecordID     = "record-id"
	flagIntentID     = "intent-id"
	flagFirstName    = "first-name"
	flagLastName     = "last-name"
	flagEmail        = "email"
	flagPhoneCode    = "phone-country-code"
	flagPhoneNumber  = "phone-number"
	flagStreet       = "street"
	flagStreetNumber = "street-number"
	flagPostalCode   = "postal-code"
	flagCity         = "city"
	flagState        = "state"
	flagCountry      = "country"
	flagCompany      = "company"
)

// --- Flag constructors --------------------------------------------------

func requiredString(name, usage string, dest *string, validate func(string) error) *cli.StringFlag {
	return &cli.StringFlag{
		Name:        name,
		Usage:       usage,
		Destination: dest,
		Required:    true,
		Validator:   validate,
	}
}

func optionalString(name, usage string, dest *string, validate func(string) error) *cli.StringFlag {
	return &cli.StringFlag{
		Name:        name,
		Usage:       usage,
		Destination: dest,
		Validator:   validate,
	}
}

// --- Input structs ------------------------------------------------------

type domainsAddInput struct {
	Domain   string
	App      string
	Port     int
	CustomDNS bool
	Wait     bool
	NoWait   bool
	WithWWW  bool
}

type domainsRemoveInput struct {
	Domain string
	App    string
	Yes    bool
}

type domainsCheckInput struct {
	Domain string
	Probe  bool
}

type domainsSetupInput struct {
	Domain string
}

type domainsAvailableInput struct {
	Domains []string
	Price   bool
}

type domainsSearchInput struct {
	Name   string
	TLD    []string
	Period int
}

type domainsManagedAddInput struct {
	Domain string
	IP     string
}

type domainsManagedVerifyInput struct {
	Domain string
}

type domainsManagedDeleteInput struct {
	Domain string
	Yes    bool
}

type dnsListInput struct {
	Domain string
}

type dnsCreateInput struct {
	Domain       string
	Type         string
	Name         string
	Data         string
	TTL          int
	Priority     int
	Port         int
	Weight       int
	Flags        int
	Tag          string
}

type dnsUpdateInput struct {
	Domain       string
	RecordID     string
	Type         string
	Name         string
	Data         string
	TTL          int
	Priority     int
	Port         int
	Weight       int
	Flags        int
	Tag          string
	TypeSet      bool
	NameSet      bool
	DataSet      bool
	TTLSet       bool
	PrioritySet  bool
	PortSet      bool
	WeightSet    bool
	FlagsSet     bool
	TagSet       bool
}

type dnsDeleteInput struct {
	Domain   string
	RecordID string
	Yes      bool
}

type domainsPurchaseInput struct {
	Domain           string
	Period           int
	FirstName        string
	LastName         string
	Email            string
	PhoneCountryCode string
	PhoneNumber      string
	Street           string
	StreetNumber     string
	PostalCode       string
	City             string
	State            string
	Country          string
	Company          string
}

type domainsPurchaseStatusInput struct {
	IntentID string
}

// --- Command tree -------------------------------------------------------

// Command returns the root domains command tree.
func Command() *cli.Command {
	return &cli.Command{
		Name:    "domains",
		Aliases: []string{"domain"},
		Usage:   "Add, list, and inspect custom domains for your apps",
		Commands: []*cli.Command{
			domainsListCommand(),
			domainsAddCommand(),
			domainsRemoveCommand(),
			domainsCheckCommand(),
			domainsSetupCommand(),
			domainsAvailableCommand(),
			domainsSearchCommand(),
			domainsManagedCommand(),
			domainsDNSCommand(),
			domainsPurchaseCommand(),
			domainsPurchaseStatusCommand(),
		},
	}
}

func domainsListCommand() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List all domains in the current organization",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return handleDomainsList(ctx)
		},
	}
}

func domainsAddCommand() *cli.Command {
	var in domainsAddInput
	return &cli.Command{
		Name:  "add",
		Usage: "Add a custom domain to an app",
		Flags: []cli.Flag{
			requiredString(flagDomain, "Domain name", &in.Domain, nil),
			requiredString(flagApp, "App name (the value of [app] name in satusky.toml, or --name on deploy)", &in.App, nil),
			&cli.IntFlag{Name: flagPort, Usage: "Target port on the deployment", Destination: &in.Port, Value: 8080},
			&cli.BoolFlag{Name: flagCustomDNS, Usage: "Treat the hostname as an external custom domain", Destination: &in.CustomDNS},
			&cli.BoolFlag{Name: flagWait, Usage: "Wait for DNS and TLS to become ready before returning", Destination: &in.Wait, Value: true},
			&cli.BoolFlag{Name: flagNoWait, Usage: "Skip waiting — return immediately after attaching", Destination: &in.NoWait},
			&cli.BoolFlag{Name: flagWithWWW, Usage: "Also configure a www redirect for apex domains when supported", Destination: &in.WithWWW},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return handleDomainsAdd(ctx, in)
		},
	}
}

func domainsRemoveCommand() *cli.Command {
	var in domainsRemoveInput
	return &cli.Command{
		Name:    "delete",
		Aliases: []string{"remove", "rm"},
		Usage:   "Remove a custom domain from an app",
		Flags: []cli.Flag{
			requiredString(flagDomain, "Domain name", &in.Domain, nil),
			requiredString(flagApp, "App name (the value of [app] name in satusky.toml)", &in.App, nil),
			&cli.BoolFlag{Name: flagYes, Aliases: []string{"y"}, Usage: "Skip confirmation prompt", Destination: &in.Yes},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return handleDomainsRemove(ctx, in)
		},
	}
}

func domainsCheckCommand() *cli.Command {
	var in domainsCheckInput
	return &cli.Command{
		Name:  "check",
		Usage: "Check backend, route, DNS, TLS, and HTTP status for a domain",
		Flags: []cli.Flag{
			requiredString(flagDomain, "Domain name", &in.Domain, nil),
			&cli.BoolFlag{Name: flagProbe, Usage: "Run an HTTP reachability probe", Destination: &in.Probe},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return handleDomainsCheck(ctx, in)
		},
	}
}

func domainsSetupCommand() *cli.Command {
	var in domainsSetupInput
	return &cli.Command{
		Name:  "setup",
		Usage: "Show exact DNS setup instructions for a domain",
		Flags: []cli.Flag{
			requiredString(flagDomain, "Domain name", &in.Domain, nil),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return handleDomainsSetup(ctx, in)
		},
	}
}

func domainsAvailableCommand() *cli.Command {
	var in domainsAvailableInput
	return &cli.Command{
		Name:  "available",
		Usage: "Check domain registration availability",
		Flags: []cli.Flag{
			&cli.StringSliceFlag{Name: flagDomain, Usage: "Domain to check (repeatable)", Destination: &in.Domains, Required: true},
			&cli.BoolFlag{Name: flagPrice, Usage: "Include registration pricing when available", Destination: &in.Price},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return handleDomainsAvailable(ctx, in)
		},
	}
}

func domainsSearchCommand() *cli.Command {
	var in domainsSearchInput
	return &cli.Command{
		Name:  "search",
		Usage: "Search available domains across TLDs",
		Flags: []cli.Flag{
			requiredString(flagName, "Name to search", &in.Name, nil),
			&cli.StringSliceFlag{Name: flagTLD, Usage: "TLD to search; can be repeated", Destination: &in.TLD},
			&cli.IntFlag{Name: flagPeriod, Usage: "Registration period in years", Destination: &in.Period, Value: 1},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return handleDomainsSearch(ctx, in)
		},
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
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return handleManagedDomainsList(ctx)
				},
			},
			func() *cli.Command {
				var in domainsManagedAddInput
				return &cli.Command{
					Name:  "add",
					Usage: "Add a domain zone to SatuSky DNS",
					Flags: []cli.Flag{
						requiredString(flagDomain, "Domain name", &in.Domain, nil),
						optionalString(flagIP, "Optional default apex A record", &in.IP, nil),
					},
					Action: func(ctx context.Context, cmd *cli.Command) error {
						return handleManagedDomainsAdd(ctx, in)
					},
				}
			}(),
			func() *cli.Command {
				var in domainsManagedVerifyInput
				return &cli.Command{
					Name:  "verify",
					Usage: "Verify nameservers for a managed domain",
					Flags: []cli.Flag{
						requiredString(flagDomain, "Domain name or ID", &in.Domain, nil),
					},
					Action: func(ctx context.Context, cmd *cli.Command) error {
						return handleManagedDomainsVerify(ctx, in)
					},
				}
			}(),
			func() *cli.Command {
				var in domainsManagedDeleteInput
				return &cli.Command{
					Name:    "delete",
					Aliases: []string{"rm"},
					Usage:   "Delete a managed domain zone",
					Flags: []cli.Flag{
						requiredString(flagDomain, "Domain name or ID", &in.Domain, nil),
						&cli.BoolFlag{Name: flagYes, Aliases: []string{"y"}, Usage: "Skip confirmation prompt", Destination: &in.Yes},
					},
					Action: func(ctx context.Context, cmd *cli.Command) error {
						return handleManagedDomainsDelete(ctx, in)
					},
				}
			}(),
		},
	}
}

func domainsDNSCommand() *cli.Command {
	return &cli.Command{
		Name:  "dns",
		Usage: "Manage DNS records for SatuSky-managed domains",
		Commands: []*cli.Command{
			func() *cli.Command {
				var in dnsListInput
				return &cli.Command{
					Name:  "list",
					Usage: "List DNS records",
					Flags: []cli.Flag{
						requiredString(flagDomain, "Domain name or ID", &in.Domain, nil),
					},
					Action: func(ctx context.Context, cmd *cli.Command) error {
						return handleDNSList(ctx, in)
					},
				}
			}(),
			func() *cli.Command {
				var in dnsCreateInput
				return &cli.Command{
					Name:    "create",
					Aliases: []string{"add"},
					Usage:   "Create a DNS record",
					Flags: []cli.Flag{
						requiredString(flagDomain, "Domain name or ID", &in.Domain, nil),
						requiredString(flagType, "DNS record type (A, AAAA, CNAME, MX, TXT, NS, SRV, CAA)", &in.Type, nil),
						requiredString(flagName, "Record name, such as @, www, or app", &in.Name, nil),
						requiredString(flagData, "Record value", &in.Data, nil),
						&cli.IntFlag{Name: flagTTL, Usage: "Record TTL in seconds (min 600)", Destination: &in.TTL},
						&cli.IntFlag{Name: flagPriority, Usage: "Record priority", Destination: &in.Priority},
						&cli.IntFlag{Name: flagPort, Usage: "SRV record port", Destination: &in.Port},
						&cli.IntFlag{Name: flagWeight, Usage: "SRV record weight", Destination: &in.Weight},
						&cli.IntFlag{Name: flagFlags, Usage: "CAA record flags", Destination: &in.Flags},
						optionalString(flagTag, "CAA record tag", &in.Tag, nil),
					},
					Action: func(ctx context.Context, cmd *cli.Command) error {
						return handleDNSCreate(ctx, in)
					},
				}
			}(),
			func() *cli.Command {
				var in dnsUpdateInput
				return &cli.Command{
					Name:  "update",
					Usage: "Update a DNS record",
					Flags: []cli.Flag{
						requiredString(flagDomain, "Domain name or ID", &in.Domain, nil),
						requiredString(flagRecordID, "DNS record ID", &in.RecordID, nil),
						optionalString(flagType, "DNS record type (A, AAAA, CNAME, MX, TXT, NS, SRV, CAA)", &in.Type, nil),
						optionalString(flagName, "Record name, such as @, www, or app", &in.Name, nil),
						optionalString(flagData, "Record value", &in.Data, nil),
						&cli.IntFlag{Name: flagTTL, Usage: "Record TTL in seconds (min 600)", Destination: &in.TTL},
						&cli.IntFlag{Name: flagPriority, Usage: "Record priority", Destination: &in.Priority},
						&cli.IntFlag{Name: flagPort, Usage: "SRV record port", Destination: &in.Port},
						&cli.IntFlag{Name: flagWeight, Usage: "SRV record weight", Destination: &in.Weight},
						&cli.IntFlag{Name: flagFlags, Usage: "CAA record flags", Destination: &in.Flags},
						optionalString(flagTag, "CAA record tag", &in.Tag, nil),
					},
					Before: func(ctx context.Context, cmd *cli.Command) (context.Context, error) {
						in.TypeSet = cmd.IsSet(flagType)
						in.NameSet = cmd.IsSet(flagName)
						in.DataSet = cmd.IsSet(flagData)
						in.TTLSet = cmd.IsSet(flagTTL)
						in.PrioritySet = cmd.IsSet(flagPriority)
						in.PortSet = cmd.IsSet(flagPort)
						in.WeightSet = cmd.IsSet(flagWeight)
						in.FlagsSet = cmd.IsSet(flagFlags)
						in.TagSet = cmd.IsSet(flagTag)
						return ctx, nil
					},
					Action: func(ctx context.Context, cmd *cli.Command) error {
						return handleDNSUpdate(ctx, in)
					},
				}
			}(),
			func() *cli.Command {
				var in dnsDeleteInput
				return &cli.Command{
					Name:    "delete",
					Aliases: []string{"rm"},
					Usage:   "Delete a DNS record",
					Flags: []cli.Flag{
						requiredString(flagDomain, "Domain name or ID", &in.Domain, nil),
						requiredString(flagRecordID, "DNS record ID", &in.RecordID, nil),
						&cli.BoolFlag{Name: flagYes, Aliases: []string{"y"}, Usage: "Skip confirmation prompt", Destination: &in.Yes},
					},
					Action: func(ctx context.Context, cmd *cli.Command) error {
						return handleDNSDelete(ctx, in)
					},
				}
			}(),
		},
	}
}

func domainsPurchaseCommand() *cli.Command {
	var in domainsPurchaseInput
	return &cli.Command{
		Name:  "purchase",
		Usage: "Purchase a domain using your credits balance",
		Flags: []cli.Flag{
			requiredString(flagDomain, "Domain name to purchase", &in.Domain, nil),
			&cli.IntFlag{Name: flagPeriod, Usage: "Registration period in years", Destination: &in.Period, Value: 1},
			requiredString(flagFirstName, "Registrant first name", &in.FirstName, nil),
			requiredString(flagLastName, "Registrant last name", &in.LastName, nil),
			requiredString(flagEmail, "Registrant email address", &in.Email, nil),
			requiredString(flagPhoneCode, "Phone country code (e.g. +60)", &in.PhoneCountryCode, nil),
			requiredString(flagPhoneNumber, "Phone subscriber number", &in.PhoneNumber, nil),
			requiredString(flagStreet, "Street name", &in.Street, nil),
			requiredString(flagStreetNumber, "Street or building number", &in.StreetNumber, nil),
			requiredString(flagPostalCode, "Postal code", &in.PostalCode, nil),
			requiredString(flagCity, "City", &in.City, nil),
			optionalString(flagState, "State or province (optional for some countries)", &in.State, nil),
			requiredString(flagCountry, "Two-letter ISO country code (e.g. MY, US)", &in.Country, nil),
			optionalString(flagCompany, "Company or organization name (optional)", &in.Company, nil),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return handleDomainsPurchase(ctx, in)
		},
	}
}

func domainsPurchaseStatusCommand() *cli.Command {
	var in domainsPurchaseStatusInput
	return &cli.Command{
		Name:  "purchase-status",
		Usage: "Check a domain purchase intent",
		Flags: []cli.Flag{
			requiredString(flagIntentID, "Purchase intent ID", &in.IntentID, nil),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return handleDomainsPurchaseStatus(ctx, in)
		},
	}
}
